// Package contrib provides functionality for getting digests from
// registries supported by docker-lock's community.
package contrib

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/safe-waters/docker-lock/pkg/generate/registry"
)

// MCRWrapper is a registry wrapper for Microsoft Container Registry.
type MCRWrapper struct {
	client *registry.HTTPClient
}

// NewMCRWrapper creates an MCRWrapper.
func NewMCRWrapper(client *registry.HTTPClient) *MCRWrapper {
	w := &MCRWrapper{}

	if client == nil {
		w.client = &registry.HTTPClient{
			Client:      &http.Client{},
			RegistryURL: fmt.Sprintf("https://%sv2", w.Prefix()),
		}
	}

	return w
}

// init registers MCRWrapper for use by docker-lock.
func init() { // nolint: gochecknoinits
	constructor := func(
		client *registry.HTTPClient,
		_ string,
	) (registry.Wrapper, error) {
		return NewMCRWrapper(client), nil
	}

	constructors = append(constructors, constructor)
}

// Digest queries the container registry for the digest given a repo and ref.
func (m *MCRWrapper) Digest(repo string, ref string) (string, error) {
	repo = strings.Replace(repo, m.Prefix(), "", 1)

	r, err := registry.NewV2(m.client)
	if err != nil {
		return "", err
	}

	return r.Digest(repo, ref, "")
}

// Prefix returns the registry prefix that identifies MCR.
func (m *MCRWrapper) Prefix() string {
	return "mcr.microsoft.com/"
}
