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

func (w *DockerWrapper) GetDigest(name string, tag string) (string, error) {
	// Docker-Content-Digest is the root of the hash chain
	// https://github.com/docker/distribution/issues/1662
	username, password, err := w.getAuthCredentials()
	if err != nil {
		return "", err
	}
	var names []string
	if strings.Contains(name, "/") {
		names = []string{name, "library/" + name}
	} else {
		names = []string{"library/" + name, name}
	}
	for _, name := range names {
		token, err := w.getToken(name, username, password)
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

func (w *DockerWrapper) getToken(name string, username string, password string) (string, error) {
	client := &http.Client{}
	url := "https://auth.docker.io/token?scope=repository:" + name + ":pull&service=registry.docker.io"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
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

func (w *DockerWrapper) getAuthCredentials() (string, string, error) {
	username := os.Getenv("DOCKER_USERNAME")
	password := os.Getenv("DOCKER_PASSWORD")
	if username != "" && password != "" {
		return username, password, nil
	}
	if w.ConfigFile == "" {
		return "", "", nil
	}
	confByt, err := ioutil.ReadFile(w.ConfigFile)
	if err != nil {
		return "", "", err
	}
	var conf config
	if err = json.Unmarshal(confByt, &conf); err != nil {
		return "", "", err
	}
	authByt, err := base64.StdEncoding.DecodeString(conf.Auths.Index.Auth)
	if err != nil {
		return "", "", err
	}
	authString := string(authByt)
	if authString != "" {
		auth := strings.Split(authString, ":")
		username = auth[0]
		password = auth[1]
	} else if conf.CredsStore != "" {
		username, password, err = w.getAuthCredentialsFromCredsStore(conf.CredsStore)
		if err != nil {
			fmt.Fprintln(os.Stderr, `docker's config.json found, but unable to get auth credentials.
Proceeding as if user not logged in, so private repositories will be unavailable.
Try logging in with "docker login" to have access to private repositories.`)
			return "", "", nil
		}
	}
	return username, password, nil
}

// Works for “osxkeychain” on macOS, “wincred” on windows, and “pass” on Linux.
func (w *DockerWrapper) getAuthCredentialsFromCredsStore(credsStore string) (username string, password string, err error) {
	credsStore = fmt.Sprintf("%s-%s", "docker-credential", credsStore)
	defer func() {
		if err := recover(); err != nil {
			username, password = "", ""
			return
		}
	}()
	p := c.NewShellProgramFunc(credsStore)
	credResponse, err := c.Get(p, "https://index.docker.io/v1/")
	if err != nil {
		return
	}
	username, password = credResponse.Username, credResponse.Secret
	return username, password, err
}

func (w *DockerWrapper) Prefix() string {
	return ""
}
