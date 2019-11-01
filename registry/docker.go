package registry

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	c "github.com/docker/docker-credential-helpers/client"
)

type DockerWrapper struct {
	ConfigFile string
	authCreds  authCredentials
}

type authCredentials struct {
	username string
	password string
}

type dockerTokenResponse struct {
	Token string `json:"token"`
}

type config struct {
	Auths struct {
		Index struct {
			Auth string `json:"auth"`
		} `json:"https://index.docker.io/v1/"`
	} `json:"auths"`
	CredsStore string `json:"credsStore"`
}

func NewDockerWrapper(configFile string) (*DockerWrapper, error) {
	w := &DockerWrapper{ConfigFile: configFile}
	authCreds, err := w.getAuthCredentials()
	if err != nil {
		return nil, err
	}
	w.authCreds = authCreds
	return w, nil
}

func (w *DockerWrapper) GetDigest(name string, tag string) (string, error) {
	// Docker-Content-Digest is the root of the hash chain
	// https://github.com/docker/distribution/issues/1662
	var names []string
	if strings.Contains(name, "/") {
		names = []string{name, "library/" + name}
	} else {
		names = []string{"library/" + name, name}
	}
	for _, name := range names {
		token, err := w.getToken(name)
		if err != nil {
			return "", err
		}
		registryURL := "https://registry-1.docker.io/v2/" + name + "/manifests/" + tag
		req, err := http.NewRequest("GET", registryURL, nil)
		if err != nil {
			return "", err
		}
		req.Header.Add("Authorization", "Bearer "+token)
		req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		digest := resp.Header.Get("Docker-Content-Digest")
		if digest != "" {
			return strings.TrimPrefix(digest, "sha256:"), nil
		}
	}
	return "", errors.New("No digest found")
}

func (w *DockerWrapper) getToken(name string) (string, error) {
	client := &http.Client{}
	url := "https://auth.docker.io/token?scope=repository:" + name + ":pull&service=registry.docker.io"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	if w.authCreds.username != "" && w.authCreds.password != "" {
		req.SetBasicAuth(w.authCreds.username, w.authCreds.password)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	var t dockerTokenResponse
	if err = decoder.Decode(&t); err != nil {
		return "", err
	}
	return t.Token, nil
}

func (w *DockerWrapper) getAuthCredentials() (authCredentials, error) {
	username := os.Getenv("DOCKER_USERNAME")
	password := os.Getenv("DOCKER_PASSWORD")
	if username != "" && password != "" {
		return authCredentials{username: username, password: password}, nil
	}
	if w.ConfigFile == "" {
		return authCredentials{}, nil
	}
	confByt, err := ioutil.ReadFile(w.ConfigFile)
	if err != nil {
		return authCredentials{}, err
	}
	var conf config
	if err = json.Unmarshal(confByt, &conf); err != nil {
		return authCredentials{}, err
	}
	authByt, err := base64.StdEncoding.DecodeString(conf.Auths.Index.Auth)
	if err != nil {
		return authCredentials{}, err
	}
	authString := string(authByt)
	if authString != "" {
		auth := strings.Split(authString, ":")
		return authCredentials{username: auth[0], password: auth[1]}, nil
	} else if conf.CredsStore != "" {
		authCreds, err := w.getAuthCredentialsFromCredsStore(conf.CredsStore)
		if err != nil {
			return authCredentials{}, nil
		}
		return authCreds, nil
	}
	return authCredentials{}, nil
}

// Works for “osxkeychain” on macOS, “wincred” on windows, and “pass” on Linux.
func (w *DockerWrapper) getAuthCredentialsFromCredsStore(credsStore string) (authCreds authCredentials, err error) {
	credsStore = fmt.Sprintf("%s-%s", "docker-credential", credsStore)
	defer func() {
		if err := recover(); err != nil {
			authCreds = authCredentials{}
			return
		}
	}()
	p := c.NewShellProgramFunc(credsStore)
	credResponse, err := c.Get(p, "https://index.docker.io/v1/")
	if err != nil {
		return authCreds, err
	}
	return authCredentials{username: credResponse.Username, password: credResponse.Secret}, nil
}

func (w *DockerWrapper) Prefix() string {
	return ""
}
