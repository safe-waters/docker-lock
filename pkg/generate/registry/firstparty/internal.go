package firstparty

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/safe-waters/docker-lock/pkg/generate/registry"
)

// InternalWrapper is a registry wrapper for internal registries.
type InternalWrapper struct {
	client      *registry.HTTPClient
	prefix      string
	stripPrefix bool
}

// init registers InternalWrapper for use by docker-lock
// if INTERNAL_REGISTRY_URL is set.
func init() { // nolint: gochecknoinits
	constructor := func(
		client *registry.HTTPClient,
		_ string,
	) (registry.Wrapper, error) {
		stripPrefix, err := strconv.ParseBool(
			os.Getenv("INTERNAL_STRIP_PREFIX"),
		)
		if err != nil {
			err = fmt.Errorf("cannot register InternalWrapper: %s", err)
			return nil, err
		}

		w, err := NewInternalWrapper(
			client, os.Getenv("INTERNAL_PREFIX"), stripPrefix,
			os.Getenv("INTERNAL_REGISTRY_URL"), os.Getenv("INTERNAL_TOKEN_URL"),
		)
		if err != nil {
			err = fmt.Errorf("cannot register InternalWrapper: %s", err)
		}

		return w, err
	}

	constructors = append(constructors, constructor)
}

// NewInternalWrapper creates an InternalWrapper. If tokenURL is blank,
// the wrapper will not acquire a token before requesting the digest.
// If stripPrefix is true, the prefix will not be considered part of
// the repo name in API calls. registryURL must be set.
func NewInternalWrapper(
	client *registry.HTTPClient,
	prefix string,
	stripPrefix bool,
	registryURL string,
	tokenURL string,
) (*InternalWrapper, error) {
	if registryURL == "" {
		return nil, fmt.Errorf("internal registry url is empty")
	}

	w := &InternalWrapper{
		prefix:      prefix,
		stripPrefix: stripPrefix,
	}

	if client == nil {
		w.client = &registry.HTTPClient{
			Client:      &http.Client{},
			RegistryURL: fmt.Sprintf("%s/v2", registryURL),
			TokenURL:    tokenURL,
		}
	}

	return w, nil
}

// Digest queries the container registry for the digest given a repo and ref.
func (i *InternalWrapper) Digest(repo string, ref string) (string, error) {
	if i.stripPrefix {
		repo = strings.Replace(repo, i.Prefix(), "", 1)
	}

	r, err := registry.NewV2(i.client)
	if err != nil {
		return "", err
	}

	token := ""

	if i.client.TokenURL != "" {
		tokenURL := strings.ReplaceAll(i.client.TokenURL, "<REPO>", repo)

		var err error

		token, err = r.Token(
			tokenURL, "", "", &registry.DefaultTokenExtractor{},
		)
		if err != nil {
			return "", err
		}
	}

	return r.Digest(repo, ref, token)
}

// Prefix returns the registry prefix that identifies the internal
// registry.
func (i *InternalWrapper) Prefix() string {
	return i.prefix
}
