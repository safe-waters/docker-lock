package registry

import (
	"net/http"
)

// HTTPClient overrides base urls to get digests and auth tokens.
type HTTPClient struct {
	*http.Client
	RegistryURL string
	TokenURL    string
}

// WrapperConstructor is a type for a function that can create a wrapper.
// Each Wrapper has an init function that registers a WrapperConstructor so
// it can be found at runtime.
type WrapperConstructor func(
	client *HTTPClient,
	configPath string,
) (Wrapper, error)

// AllWrappers returns all wrappers constructed from constructor functions.
func AllWrappers(
	client *HTTPClient,
	configPath string,
	constructors []WrapperConstructor,
) []Wrapper {
	var wrappers []Wrapper // nolint:prealloc

	for _, c := range constructors {
		w, err := c(client, configPath)
		if err != nil {
			continue
		}

		wrappers = append(wrappers, w)
	}

	return wrappers
}
