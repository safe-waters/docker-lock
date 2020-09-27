package firstparty

import (
	"os"

	"github.com/safe-waters/docker-lock/pkg/generate/registry"
)

// constructors holds all constructor functions for registries. It is
// the intention that these will be created in an init function for each
// registry as a way to register the registry.
var constructors []registry.WrapperConstructor // nolint: gochecknoglobals

// DefaultWrapper returns a wrapper for images without a prefix.
func DefaultWrapper(
	client *registry.HTTPClient,
	configPath string,
) (registry.Wrapper, error) {
	username := os.Getenv("DOCKER_USERNAME")
	password := os.Getenv("DOCKER_PASSWORD")

	return NewDockerWrapper(client, configPath, username, password)
}

// AllWrappers returns all wrappers officially supported
// by the maintainers of docker-lock, that match the caller's environment.
func AllWrappers(
	client *registry.HTTPClient,
	configPath string,
) []registry.Wrapper {
	return registry.AllWrappers(client, configPath, constructors)
}
