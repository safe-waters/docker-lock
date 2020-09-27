package contrib

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/safe-waters/docker-lock/pkg/generate/registry"
)

// ElasticWrapper is a registry wrapper for the Elasticsearch registry.
type ElasticWrapper struct {
	client *registry.HTTPClient
}

// NewElasticWrapper creates an ElasticWrapper.
func NewElasticWrapper(client *registry.HTTPClient) *ElasticWrapper {
	w := &ElasticWrapper{}

	if client == nil {
		w.client = &registry.HTTPClient{
			Client:      &http.Client{},
			RegistryURL: fmt.Sprintf("https://%sv2", w.Prefix()),
			TokenURL: fmt.Sprintf(
				"https://docker-auth.elastic.co/auth%s",
				"?scope=repository:%s:pull&service=token-service",
			),
		}
	}

	return w
}

// init registers ElasticWrapper for use by docker-lock.
func init() { // nolint: gochecknoinits
	constructor := func(
		client *registry.HTTPClient,
		_ string,
	) (registry.Wrapper, error) {
		return NewElasticWrapper(client), nil
	}

	constructors = append(constructors, constructor)
}

// Digest queries the container registry for the digest given a repo and ref.
func (e *ElasticWrapper) Digest(repo string, ref string) (string, error) {
	repo = strings.Replace(repo, e.Prefix(), "", 1)

	tokenURL := fmt.Sprintf(e.client.TokenURL, repo)

	r, err := registry.NewV2(e.client)
	if err != nil {
		return "", err
	}

	token, err := r.Token(tokenURL, "", "", &registry.DefaultTokenExtractor{})
	if err != nil {
		return "", err
	}

	return r.Digest(repo, ref, token)
}

// Prefix returns the registry prefix that identifies the Elasticsearch
// registry.
func (e *ElasticWrapper) Prefix() string {
	return "docker.elastic.co/"
}
