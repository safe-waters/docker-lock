package firstparty

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/safe-waters/docker-lock/pkg/generate/registry"
)

// ACRWrapper is a registry wrapper for Azure Container Registry.
type ACRWrapper struct {
	client *registry.HTTPClient
	*registry.AuthCredentials
	registryName string
}

// acrTokenExtractor extracts tokens, using a unique implementation to ACR.
type acrTokenExtractor struct{}

// acrTokenResponse contains the bearer token required to
// query ACR for a digest.
type acrTokenResponse struct {
	Token string `json:"access_token"`
}

// acrAuthExtractor is used to extract the base64 encoded auth string from
// docker's config.
type acrAuthExtractor struct {
	registryName string
}

// acrConfig represents the section in docker's config.json for ACR.
type acrConfig struct {
	Auths      map[string]map[string]string `json:"auths"`
	CredsStore string                       `json:"credsStore"`
}

// init registers ACRWrapper for use by docker-lock
// if ACR_REGISTRY_NAME is set.
func init() { // nolint: gochecknoinits
	constructor := func(
		client *registry.HTTPClient,
		configPath string,
	) (registry.Wrapper, error) {
		w, err := NewACRWrapper(
			client, configPath, os.Getenv("ACR_USERNAME"),
			os.Getenv("ACR_PASSWORD"), os.Getenv("ACR_REGISTRY_NAME"),
		)
		if err != nil {
			err = fmt.Errorf("cannot register ACRWrapper: %s", err)
		}

		return w, err
	}

	constructors = append(constructors, constructor)
}

// NewACRWrapper creates an ACRWrapper or returns an error if not possible.
//
// registryName must not be empty. If your image is referenced by the line
// myregistry.azurecr.io/myimage, "myregistry" should be registryName's value.
//
// If username and password are defined, then they will be used for
// authentication. Otherwise, the username and password will be obtained
// from docker's config.json. For this to work, please login using
// 'docker login' such as 'docker login myregistry.azurecr.io'.
//
// If using the cli, to set the registry name, username, and password, ensure
// ACR_REGISTRY_NAME, ACR_USERNAME, and ACR_PASSWORD are set. This can
// be achieved automatically via a .env file or manually by exporting the
// environment variables. configPath can be set via cli flags.
func NewACRWrapper(
	client *registry.HTTPClient,
	configPath string,
	username string,
	password string,
	registryName string,
) (*ACRWrapper, error) {
	if registryName == "" {
		return nil, fmt.Errorf("acr registry name is empty")
	}

	w := &ACRWrapper{}
	w.registryName = registryName

	if client == nil {
		w.client = &registry.HTTPClient{
			Client:      &http.Client{},
			RegistryURL: fmt.Sprintf("https://%sv2", w.Prefix()),
			TokenURL: fmt.Sprintf(
				"https://%soauth2/token%s", w.Prefix(),
				"?service=%s.azurecr.io&scope=repository:%s:pull",
			),
		}
	}

	ac, err := registry.NewAuthCredentials(
		username, password, configPath,
		&acrAuthExtractor{registryName: registryName},
	)
	if err != nil {
		return nil, err
	}

	w.AuthCredentials = ac

	return w, nil
}

// Digest queries the container registry for the digest given a repo and ref.
func (a *ACRWrapper) Digest(repo string, ref string) (string, error) {
	repo = strings.Replace(repo, a.Prefix(), "", 1)

	tokenURL := fmt.Sprintf(a.client.TokenURL, a.registryName, repo)

	r, err := registry.NewV2(a.client)
	if err != nil {
		return "", err
	}

	token, err := r.Token(
		tokenURL, a.Username, a.Password, &acrTokenExtractor{},
	)
	if err != nil {
		return "", err
	}

	return r.Digest(repo, ref, token)
}

// Prefix returns the registry prefix that identifies ACR.
func (a *ACRWrapper) Prefix() string {
	return fmt.Sprintf("%s.azurecr.io/", a.registryName)
}

// FromBody parses the token from the registry response body.
func (*acrTokenExtractor) FromBody(body io.ReadCloser) (string, error) {
	decoder := json.NewDecoder(body)

	t := acrTokenResponse{}
	if err := decoder.Decode(&t); err != nil {
		return "", err
	}

	return t.Token, nil
}

// ExtractAuthStr returns the base64 encoded auth string and creds store. An
// error will occur if you used 'acr login' rather than 'docker login'.
func (a *acrAuthExtractor) ExtractAuthStr(
	confByt []byte,
) (string, string, error) {
	conf := acrConfig{}
	if err := json.Unmarshal(confByt, &conf); err != nil {
		return "", "", err
	}

	for serverName, authInfo := range conf.Auths {
		if serverName == fmt.Sprintf("%s.azurecr.io", a.registryName) {
			if _, ok := authInfo["identitytoken"]; ok {
				log.Printf("ACRWrapper found 'identitytoken' in docker "+
					"config for registry '%s'. This method of logging in "+
					"is unsupported. It occurs if you logged in via "+
					"'az acr login'. To successfully use docker-lock, "+
					"please login via 'docker login' with a username and "+
					"password or create a .env file with ACR_USERNAME "+
					"and ACR_PASSWORD or export ACR_USERNAME and ACR_PASSWORD.",
					a.registryName,
				)

				return "", "", fmt.Errorf(
					"invalid login method for '%s'", serverName,
				)
			}

			return authInfo["auth"], conf.CredsStore, nil
		}
	}

	return "", "", nil
}

// ServerURL returns the login server for ACR to be used by the creds store.
func (a *acrAuthExtractor) ServerURL() string {
	return fmt.Sprintf("%s.azurecr.io", a.registryName)
}
