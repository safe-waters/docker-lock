# Bring Your Own Registry

If you find that `docker-lock` does not support the registry you are using,
you may want to add your own.

> Note: If you are using an internal registry, please try the provided
[internal registry wrapper](./internal-registry.md). It likely covers your
scenario, especially if your registry implements the
[Docker Registry HTTP API V2 Specification](https://docs.docker.com/registry/spec/api/).

`docker-lock` has two types of registries, first party and contributed.
First party registries are officially supported by `docker-lock` and run as
part of the integration tests. Contributed registries are supported by the
community and can be (but do not have to be) run as part of the integration
tests. This split exists because many registries require paid accounts, which
`docker-lock`'s maintainers do not have, and there are too many registries to
be able to officially support them all.

Fortunately, it is very easy to bring your own registry. To do so:
* Create a struct in `contrib` or `firstparty` that implements the
[registry.Wrapper interface](../../pkg/generate/registry/wrapper.go).
* Register your Wrapper with `docker-lock` in an init function.

The `registry.Wrapper` interface has 2 methods,
`Digest(repo string, ref string) (string, error)` and `Prefix() string`.

To understand them, consider the scenario where you woud like to get the digest
from `myregistry/myimage:tag`. `Digest` would take `myimage` and `tag` as the
two arguments and return the digest (hash) as a string. `Prefix` would return
`myregistry/`, which tells `docker-lock` to use that wrapper whenever it
encounters an image with the prefix `myregistry/`.

To register your wrapper, in an init function, append a `constructor` (a
function that returns your wrapper) to the `constructors` slice. For a good
example, checkout the
[DockerWrapper init function](../../pkg/generate/registry/firstparty/docker.go).