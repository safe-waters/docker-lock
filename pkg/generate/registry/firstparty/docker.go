// Package firstparty provides functionality for getting digests from
// registries supported by docker-lock's maintainers.
package firstparty

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/safe-waters/docker-lock/pkg/generate/registry"
)

// DockerWrapper is a registry wrapper for Docker Hub. It supports public
// and private repositories.
type DockerWrapper struct {
	client *registry.HTTPClient
	*registry.AuthCredentials
}

// dockerAuthExtractor is used to extract the base64 encoded auth string from
// docker's config.
type dockerAuthExtractor struct{}

// dockerConfig represents the section in docker's config.json for Docker Hub.
type dockerConfig struct {
	Auths struct {
		Index struct {
			Auth string `json:"auth"`
		} `json:"https://index.docker.io/v1/"`
	} `json:"auths"`
	CredsStore string `json:"credsStore"`
}

// init registers DockerWrapper for use by docker-lock.
func init() { // nolint: gochecknoinits
	constructor := func(
		client *registry.HTTPClient,
		configPath string,
	) (registry.Wrapper, error) {
		w, err := NewDockerWrapper(
			client, configPath, os.Getenv("DOCKER_USERNAME"),
			os.Getenv("DOCKER_PASSWORD"),
		)
		if err != nil {
			err = fmt.Errorf("cannot register DockerWrapper: %s", err)
		}

		return w, err
	}

	constructors = append(constructors, constructor)
}

// NewDockerWrapper creates a DockerWrapper or returns an error
// if not possible.
//
// If username and password are defined, then they will be used for
// authentication. Otherwise, the username and password will be obtained
// from docker's config.json. For this to work, please login using
// 'docker login'.
//
// If using the cli, to set the username and password, ensure
// DOCKER_USERNAME and DOCKER_PASSWORD are set. This can be achieved
// automatically via a .env file or manually by exporting the
// environment variables. configPath can be set via cli flags.
func NewDockerWrapper(
	client *registry.HTTPClient,
	configPath string,
	username string,
	password string,
) (*DockerWrapper, error) {
	if client == nil {
		client = &registry.HTTPClient{
			Client:      &http.Client{},
			RegistryURL: "https://registry-1.docker.io/v2",
			TokenURL: fmt.Sprintf(
				"https://auth.docker.io/token%s",
				"?scope=repository:%s:pull&service=registry.docker.io",
			),
		}
	}

	w := &DockerWrapper{client: client}

	ac, err := registry.NewAuthCredentials(
		username, password, configPath, &dockerAuthExtractor{},
	)
	if err != nil {
		return nil, err
	}

	w.AuthCredentials = ac

	return w, nil
}

// Digest queries the container registry for the digest given a repo and ref.
func (d *DockerWrapper) Digest(repo string, ref string) (string, error) {
	// Docker-Content-Digest is the root of the hash chain
	// https://github.com/docker/distribution/issues/1662
	var repos []string

	if strings.Contains(repo, "/") {
		repos = []string{repo, "library/" + repo}
	} else {
		repos = []string{"library/" + repo, repo}
	}

	for _, repo := range repos {
		tokenURL := fmt.Sprintf(d.client.TokenURL, repo)

		r, err := registry.NewV2(d.client)
		if err != nil {
			return "", err
		}

		token, err := r.Token(
			tokenURL, d.Username, d.Password, &registry.DefaultTokenExtractor{},
		)
		if err != nil {
			return "", err
		}

		digest, _ := r.Digest(repo, ref, token)
		if digest != "" {
			return digest, nil
		}
	}

	return "", fmt.Errorf("no digest found for '%s:%s'", repo, ref)
}

// Prefix returns an empty string since images on Docker Hub do not use a
// prefix, unlike third party registries.
func (d *DockerWrapper) Prefix() string {
	return ""
}

// ExtractAuthStr returns the base64 encoded auth string and creds store.
func (d *dockerAuthExtractor) ExtractAuthStr(
	confByt []byte,
) (string, string, error) {
	conf := dockerConfig{}
	if err := json.Unmarshal(confByt, &conf); err != nil {
		return "", "", err
	}

	return conf.Auths.Index.Auth, conf.CredsStore, nil
}

// ServerURL returns the login server for DockerHub to be used
// by the creds store.
func (d *dockerAuthExtractor) ServerURL() string {
	return "https://index.docker.io/v1/"
}
