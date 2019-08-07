# About
`docker-lock` is a [cli-plugin](https://github.com/docker/cli/issues/1534) that generates and verifies lockfiles (think `package-lock.json` or `Pipfile.lock`) for docker and docker-compose. `docker-lock` allows developers to refer to images by tags, yet still receive all the benefits of referring to images by digest.

# Motivation
Docker image tags are mutable. This means an image maintainer can push changes to an image without changing the tag. For instance, consider the `python:3.6` image hosted on Dockerhub. Recently, its maintainers changed the underlying linux distribution and pushed the updated image to Dockerhub with the same tag. 

Mutable image tags are particularly useful for receiving updates. However, they jeopardize repeatable builds. An update to an image that an application builds upon could break the application, even if the application code does not change.

Image tags are by far the most common way to refer to an image. However, an image can also be identified by its digest, a unique hash that always refers to the same image. For instance, at the time of writing this README, the current `python:3.6` image could also be specified with the name and digest `python@sha256:25a189a536ae4d7c77dd5d0929da73057b85555d6b6f8a66bfbcc1a7a7de094b`. If the `3.6` tag receives an update, the digest would still refer to the older image.

Although specifying digests ensures that updates to a base image will not break the application, doing so comes with a host of problems. Namely:
* The application will no longer benefit from updates (security updates, performance updates, etc.).
* Dockerfiles will become stale.
* Digests are considerably less readable.
* It is unclear why an image is tied to a specific digest. Is it because future changes are incompatible, is it just to be safe, or does the developer prefer the digest over the tag?
* Keeping digests up to date can become unwieldly in projects with many services.
* Specifying the correct digest is complicated. Local digests may differ from remote digests, and there are many different types of digests (manifest digests, layer digests, etc.)

# How to use
`docker-lock` ships with two commmands `generate` and `verify`:
* `docker lock generate` generates a lockfile.
* `docker lock verify` verifies that the lockfile digests are the same as the ones in the registry.

## Demo
Consider a project with a multi-stage build Dockerfile at its root:
```
FROM ubuntu AS base
# ...
FROM mperel/log:v1
# ...
FROM python:3.6
# ...
```
Running `docker lock generate` from the root queries each images' registry to produce a lockfile, `docker-lock.json`.
![Generate GIF](gifs/generate.gif)

Note that the lockfile records image digests. Running `docker lock verify` ensures that the image digests are the same as those on the registry for the same tags.

Now, assume that a change to `mperel/log:v1` has been pushed to the registry. Running `docker lock verify` shows that the image digest in the lockfile is out of date because it differs from the newer image's digest on the registry.

![Verify GIF](gifs/verify.gif)

# Use cases
## CI/CD pipelines
`docker lock` is particularly useful in CI/CD pipelines to ensure that base images have not changed after testing but before deployment. Consider the following CI/CD pipeline:
```
docker lock generate
# build images
# run tests
# tag images
# push images to registry
docker lock verify
```
`docker lock generate` will generate a lockfile. Running `docker lock verify` after deployment (in this case, pushing the built/tagged images to a registry) ensures that the base images upon which the built/tagged images rely have not changed in between testing and deployment. If `docker lock verify` fails, a change to a base image could have occurred before deployment.

## Development
While developing, it can be useful to generate a lockfile, commit it to source control, and verify it periodically (for instance on PR merges). In this way, developers can be notified when base images change, and if a bug related to a change in a base image crops up, it will be easy to identify.

# Features
* Supports docker-compose (including build args, .env, etc.).
* Supports private images on Dockerhub, via the standard `docker login` command or via environment variables.
* Has CLI flags for common tasks such as selecting Dockerfiles/docker-compose files by globs.
* Smart defaults such as including `Dockerfile`, `docker-compose.yml` and `docker-compose.yaml` without configuration during generation so typically there is no need to learn any CLI flags.
* Lightning fast - uses goroutine's to process files/make http calls concurrently.
* Supports registries compliant with the [Docker Registry HTTP API V2](https://docs.docker.com/registry/spec/api/) (coming soon).

# Install
***
Warning: Currently, only linux is supported, with mac and windows support coming down the road.
Warning: The instructions below rely on `go` being installed. In the future, you will be able to download the binary without having to compile yourself.
***
If you are using docker >= 19.03, `docker-lock` should be installed as a cli-plugin. If you are using an older version of docker, `docker-lock` can be used as a standalone cli tool.
### cli-plugin
* Ensure docker version >= 19.03
* Clone this repo.
* `cd docker-lock`
* `mkdir -p ~/.docker/cli-plugins`
* `go get`
* `go build -o ~/.docker/cli-plugins/docker-lock`

Typing the command, `docker`, should reveal the `lock` subcommand in the list of possible subcommands.

### standalone tool
* `go get github.com/michaelperel/docker-lock`

`docker-lock` should appear in the `bin/` in your `GOPATH`.

# Coming soon
* `docker lock rewrite` to rewrite Dockerfiles and docker-compose files to include the digest (useful for CI/CD).
