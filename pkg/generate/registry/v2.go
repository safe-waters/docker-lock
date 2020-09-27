package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// V2 provides methods to get digests and tokens according to the
// HTTP API V2 specification:
// https://docs.docker.com/registry/spec/api/#docker-registry-http-api-v2
type V2 struct {
	Client *HTTPClient
}

// TokenExtractor allows registry wrappers to implement their own logic
// to extract tokens from from a registry's response. For instance,
// Dockerhub returns json with the key "token" whereas ACR returns json
// with the key "access_token". Since this functionality varies per registry,
// it is up to the registry wrapper to define how to extract the token from
// the registry's response.
type TokenExtractor interface {
	FromBody(io.ReadCloser) (string, error)
}

// DefaultTokenExtractor provides a concrete implementation for registries
// whose json response returns the token with the key "token".
type DefaultTokenExtractor struct{}

// defaultTokenResponse contains the bearer token required to query some
// container registries.
type defaultTokenResponse struct {
	Token string `json:"token"`
}

// NewV2 returns a *V2 with a client initialized or an error if the client is
// nil.
func NewV2(client *HTTPClient) (*V2, error) {
	if client == nil {
		return nil, errors.New("client cannot be nil")
	}

	return &V2{Client: client}, nil
}

// Digest queries the container registry for the digest given a repo, ref, and
// token. If a token is not required, leave it empty.
func (v *V2) Digest(repo, ref, token string) (string, error) {
	url := fmt.Sprintf("%s/%s/manifests/%s", v.Client.RegistryURL, repo, ref)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	req.Header.Add(
		"Accept", "application/vnd.docker.distribution.manifest.v2+json",
	)
	req.Header.Add(
		"Accept", "application/vnd.docker.distribution.manifest.list.v2+json",
	)

	resp, err := v.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	digest := resp.Header.Get("Docker-Content-Digest")

	if digest == "" {
		return "", fmt.Errorf("no digest found for '%s:%s'", repo, ref)
	}

	return strings.TrimPrefix(digest, "sha256:"), nil
}

// Token queries the container registry for a bearer token that is later
// required to query the container registry for a digest.
func (v *V2) Token(
	url string,
	username string,
	password string,
	extractor TokenExtractor,
) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	resp, err := v.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	token, err := extractor.FromBody(resp.Body)
	if err != nil {
		return "", err
	}

	return token, nil
}

// FromBody decodes and returns the token from the json body.
func (*DefaultTokenExtractor) FromBody(body io.ReadCloser) (string, error) {
	decoder := json.NewDecoder(body)

	t := defaultTokenResponse{}
	if err := decoder.Decode(&t); err != nil {
		return "", err
	}

	return t.Token, nil
}
