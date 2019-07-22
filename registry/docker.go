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
}

func (w *DockerWrapper) GetDigest(name string, tag string) (string, error) {
	// Docker-Content-Digest is the root of the hash chain
	// https://github.com/docker/distribution/issues/1662
	token, err := w.getToken(name)
	if err != nil {
		return "", err
	}
	registryUrl := "https://registry-1.docker.io/v2/" + name + "/manifests/" + tag
	req, err := http.NewRequest("GET", registryUrl, nil)
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
	if digest == "" && !strings.HasPrefix(name, "library/") {
		name = "library/" + name
		return w.GetDigest(name, tag)
	}
	if digest == "" {
		return "", errors.New("No digest found")
	}
	return strings.TrimPrefix(digest, "sha256:"), nil
}

func (w *DockerWrapper) getToken(name string) (string, error) {
	client := &http.Client{}
	url := "https://auth.docker.io/token?scope=repository:" + name + ":pull&service=registry.docker.io"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	username, password, err := w.getAuthCredentials()
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
	auth := strings.Split(string(authByt), ":")
	if len(auth) != 2 {
		return "", "", fmt.Errorf("Unable to get username and password from config file '%s'.", w.ConfigFile)
	}
	username = auth[0]
	password = auth[1]
	return username, password, nil
}

func (w *DockerWrapper) Prefix() string {
	return ""
}
