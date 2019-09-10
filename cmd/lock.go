package cmd

import (
	"github.com/spf13/cobra"
)

func NewLockCmd() *cobra.Command {
	lockCmd := &cobra.Command{
		Use:   "lock",
		Short: "Umbrella command for generating Lockfiles and verifying and rewriting base images.",
		Long: `docker lock can generate a Lockfile for base images referenced in Dockerfiles and docker-compose files,
verify that all base images in files referenced in the Lockfile exist in the Lockfile and have up-to-date digests, and
rewrite Dockerfiles and docker-compose files to use image digests from the Lockfile rather than tags.

With docker lock, developers can reference base images by their tags,
yet receive the same benefits as referencing them by digest.

Example workflow:
* A developer writes Dockerfiles and docker-compose files using the common imagename:tag syntax,
such as "FROM python:3.6".
* Next, the developer generates a Lockfile by running "docker lock generate". The Lockfile will
contain imagename:tag:digest.
* During development, if a bug occurs, the developer can use the Lockfile to check if a change occurred to
an imagename:tag by running "docker lock verify". If one of the base images has been updated,
console output will show the changed image.
* Prior to deployment, the developer can use the Lockfile to rewrite all Dockerfiles and docker-compose files
to use image digests rather than tags by running "docker lock rewrite". In this way, all future deployments
will be repeatable even if a base image changes.
`,
	}
	return lockCmd
}
