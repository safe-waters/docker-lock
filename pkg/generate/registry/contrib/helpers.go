package contrib

import (
	"github.com/safe-waters/docker-lock/pkg/generate/registry"
)

// constructors hold all constructor functions for registries. It is
// the intention that these will be created in an init function for each
// registry as a way to register the registry.
var constructors []registry.WrapperConstructor // nolint: gochecknoglobals

// AllWrappers returns all wrappers supported by the community, that match
// the caller's environment.
func AllWrappers(
	client *registry.HTTPClient,
	configPath string,
) []registry.Wrapper {
	return registry.AllWrappers(client, configPath, constructors)
}
