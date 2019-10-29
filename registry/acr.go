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

type ACRWrapper struct {
	ConfigFile   string
	authCreds    authCredentials
	RegistryName string
}

type acrTokenResponse struct {
	Token string `json:"access_token"`
}

type Auth struct {
	Auth string `json:"auth"`
}

type acrConfig struct {
	Auths struct {
		Index map[string]Auth
	} `json:"auths"`
	CredsStore string `json:"credsStore"`
}

func NewACRWrapper(configFile string) (*ACRWrapper, error) {
	w := &ACRWrapper{ConfigFile: configFile}
	authCreds, err := w.getAuthCredentials()
	if err != nil {
		return nil, err
	}
	w.authCreds = authCreds
	return w, nil
}

func (w *ACRWrapper) GetDigest(name string, tag string) (string, error) {
	prefix := w.Prefix()
	name = strings.Replace(name, prefix, "", 1)
	token, err := w.getToken(name)
	registryURL := "https://" + prefix + "v2/" + name + "/manifests/" + tag
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
	if digest == "" {
		return "", errors.New("No digest found")
	}
	return strings.TrimPrefix(digest, "sha256:"), nil
}

func (w *ACRWrapper) getToken(name string) (string, error) {
	prefix := w.Prefix()
	client := &http.Client{}
	url := "https://" + prefix + "oauth2/token?service=" + w.RegistryName + ".azurecr.io" + "&scope=repository:" + name + ":pull"

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
	var t acrTokenResponse
	if err = decoder.Decode(&t); err != nil {
		return "", err
	}
	return t.Token, nil
}

func (w *ACRWrapper) getAuthCredentials() (authCredentials, error) {
	username := os.Getenv("DOCKER_USERNAME")
	password := os.Getenv("DOCKER_PASSWORD")
	registryName := os.Getenv("ACR_REGISTRY_NAME")
	w.RegistryName = registryName

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
	var conf acrConfig
	if err = json.Unmarshal(confByt, &conf); err != nil {
		return authCredentials{}, err
	}
	var authByt []byte
	var decodeErr error
	for server := range conf.Auths.Index {
		serverName := w.RegistryName + ".azurecr.io"
		if server == serverName {
			authByt, decodeErr = base64.StdEncoding.DecodeString(conf.Auths.Index[serverName].Auth)
		}
	}
	if decodeErr != nil {
		return authCredentials{}, err
	}
	authString := string(authByt)
	if authString != "" {
		auth := strings.Split(authString, ":")
		return authCredentials{username: auth[0], password: auth[1]}, nil
	} else if conf.CredsStore != "" {
		authCreds, err := w.getAuthCredentialsFromCredsStore(conf.CredsStore)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: '%s' found, but unable to get auth credentials. "+
				"Proceeding as if user not logged in, so private repositories will be unavailable. "+
				"Try logging in with 'docker login' to have access to private repositories.\n",
				w.ConfigFile)
			return authCredentials{}, nil
		}
		return authCreds, nil
	}
	return authCredentials{}, nil
}

// Works for “osxkeychain” on macOS, “wincred” on windows, and “pass” on Linux.
func (w *ACRWrapper) getAuthCredentialsFromCredsStore(credsStore string) (authCreds authCredentials, err error) {
	credsStore = fmt.Sprintf("%s-%s", "docker-credential", credsStore)
	defer func() {
		if err := recover(); err != nil {
			authCreds = authCredentials{}
			return
		}
	}()
	p := c.NewShellProgramFunc(credsStore)
	credResponse, err := c.Get(p, w.RegistryName+".azurecr.io")
	if err != nil {
		return authCreds, err
	}
	return authCredentials{username: credResponse.Username, password: credResponse.Secret}, nil
}

func (w *ACRWrapper) Prefix() string {
	return w.RegistryName + ".azurecr.io/"
}
