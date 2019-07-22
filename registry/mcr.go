package registry

import (
	"errors"
	"net/http"
	"strings"
)

type MCRWrapper struct{}

func (w *MCRWrapper) GetDigest(name string, tag string) (string, error) {
	prefix := w.Prefix()
	name = strings.Replace(name, prefix, "", 1)
	registryUrl := "https://" + prefix + "v2/" + name + "/manifests/" + tag
	req, err := http.NewRequest("GET", registryUrl, nil)
	if err != nil {
		return "", err
	}
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

func (w *MCRWrapper) Prefix() string {
	return "mcr.microsoft.com/"
}
