package registry

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type ElasticWrapper struct{}

type elasticTokenResponse struct {
	Token string `json:"token"`
}

func (w *ElasticWrapper) GetDigest(name string, tag string) (string, error) {
	prefix := w.Prefix()
	name = strings.Replace(name, prefix, "", 1)
	token, err := w.getToken(name)
	if err != nil {
		return "", err
	}
	registryUrl := "https://" + prefix + "v2/" + name + "/manifests/" + tag
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
	if digest == "" {
		return "", errors.New("No digest found")
	}
	return strings.TrimPrefix(digest, "sha256:"), nil
}

func (w *ElasticWrapper) getToken(name string) (string, error) {
	// example name -> "elasticsearch/elasticsearch-oss"
	url := "https://docker-auth.elastic.co/auth?scope=repository:" + name + ":pull&service=token-service"
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	var t elasticTokenResponse
	if err = decoder.Decode(&t); err != nil {
		return "", err
	}
	return t.Token, nil
}

func (w *ElasticWrapper) Prefix() string {
	return "docker.elastic.co/"
}
