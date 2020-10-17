![Docker-Lock-Banner](./docs/assets/readme-banner.png)
# About
[![Go Report Card](https://goreportcard.com/badge/github.com/safe-waters/docker-lock)](https://goreportcard.com/report/github.com/safe-waters/docker-lock)
[![Build Status](https://dev.azure.com/michaelsethperel/docker-lock/_apis/build/status/safe-waters.docker-lock?branchName=master)](https://dev.azure.com/michaelsethperel/docker-lock/_build/latest?definitionId=6&branchName=master)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/safe-waters/docker-lock)
<!-- https://github.com/golang/go/issues/40506 -->
<!-- [![PkgGoDev](https://pkg.go.dev/badge/github.com/safe-waters/docker-lock)](https://pkg.go.dev/github.com/safe-waters/docker-lock) -->

`docker-lock` is a cli tool that automates managing image digests by tracking
them in a separate Lockfile (think package-lock.json or Pipfile.lock). With
`docker-lock`, you can refer to images in Dockerfiles or 
docker-compose files by mutable tags (as in `python:3.6`) yet receive the same 
benefits as if you had specified immutable digests (as in `python:3.6@sha256:25a189a536ae4d7c77dd5d0929da73057b85555d6b6f8a66bfbcc1a7a7de094b`).

> Note: If you are unsure about the differences between tags and digests,
refer to this [quick summary](./docs/tutorials/tags-vs-digests.md).

`docker-lock` ships with 3 commands that take you from development 
to production:

* `docker lock generate` finds base images in your Dockerfiles and docker-compose
files and generates a Lockfile containing digests that correspond to their tags.
* `docker lock verify` lets you know if there are more recent digests 
than those last recorded in the Lockfile.
* `docker lock rewrite` rewrites Dockerfiles and docker-compose files 
to include digests.

`docker-lock` ships with support for [Docker Hub](https://hub.docker.com/),
[Azure Container Registry](https://azure.microsoft.com/en-us/services/container-registry/),
[internal registries](https://docs.docker.com/registry/deploying/),
and a variety of other registries. If your registry is not supported
out of the box, do not worry. `docker-lock` was designed to be
[easily extensible](./docs/tutorials/bring-your-own-registry.md) to any
container registry.

`docker-lock` is most commonly used as a
[cli-plugin](https://github.com/docker/cli/issues/1534) for `docker` so `lock`
can be used as subcommand of `docker` as in `docker lock`. However,
`docker-lock` does not require `docker` at all. Instead, it can be called
manually as in `docker-lock lock`. This is especially convenient if the proper
version of `docker` is unavailable or you would prefer to use another
container technology such as [podman](https://podman.io/).

# Demo
Consider a project with a multi-stage build Dockerfile at its root:
```
FROM ubuntu AS base
# ...
FROM mperel/log:v1
# ...
FROM python:3.6
# ...
```
Running `docker lock generate` from the root queries each images' 
registry to produce a Lockfile, `docker-lock.json`.

![Generate GIF](./docs/assets/generate.gif)

Note that the Lockfile records image digests so you do not have to 
manually specify them.

Running `docker lock verify` ensures that the image digests are the 
same as those on the registry for the same tags.

![Verify Success GIF](./docs/assets/verify_success.gif)

Now, assume that a change to `mperel/log:v1` has been pushed to the registry.

Running `docker lock verify` shows that the image digest in the Lockfile 
is out-of-date because it differs from the newer image's digest on the registry.

![Verify Fail GIF](./docs/assets/verify_fail.gif)

While developing, it can be useful to generate a Lockfile, commit it to 
source control, and verify it periodically (for instance on PR merges). In 
this way, developers can be notified when images change, and if a bug related 
to a change in an image crops up, it will be easy to identify.

Finally, lets assume the Dockerfile is ready to be built and shared.

Running `docker lock rewrite` will add digests from the Lockfile 
to all of the images.

![Rewrite GIF](./docs/assets/rewrite.gif)

At this point, the Dockerfile will contain all of the digest information 
from the Lockfile, so it will always maintain the same, known behavior 
in the future.

# Install
`docker-lock` can be installed as a
[cli-plugin](https://github.com/docker/cli/issues/1534) for `docker`, as a
standalone tool if you do not want to install the `docker` cli, or as a
docker image.

## Cli-plugin
Ensure `docker` cli version >= 19.03 is installed by running `docker --version`.

### Linux / Mac
* `mkdir -p ~/.docker/cli-plugins`
* `curl -fsSL "https://github.com/safe-waters/docker-lock/releases/download/v${VERSION}/docker-lock_${VERSION}_${OS}_${ARCH}.tar.gz" | tar -xz -C "${HOME}/.docker/cli-plugins"`
* `chmod +x "${HOME}/.docker/cli-plugins/docker-lock"`

### Windows
* Create the folder `%USERPROFILE%\.docker\cli-plugins`
* Download the Windows release from the releases page.
* Unzip the release.
* Move `docker-lock.exe` into `%USERPROFILE%\.docker\cli-plugins`

To verify that `docker-lock` was installed as a cli-plugin, run
```
docker lock --help
```

## Standalone tool
* Follow the same instructions as in the
[cli-plugin section](#cli-plugin) except place the `docker-lock` executable in
your `PATH`.
* To use `docker-lock`, replace any `docker` command such as `docker lock` with
the name of the executable, `docker-lock`, as in `docker-lock lock`.
* To verify that `docker-lock` was installed, run:
```
docker-lock lock --help
```

## Docker Image
* Instead of installing `docker-lock` on your machine, you can use a container
hosted on Dockerhub
* If you would like to use the container on Linux/Mac:
```
docker run -v "${PWD}":/run safewaters/docker-lock:${VERSION} [commands]
```
* If you would like to use the container on Windows:
```
docker run -v "%cd%":/run safewaters/docker-lock:${VERSION} [commands]
```
* If you leave off the `${VERSION}` tag, you will use the latest, nightly build.
* If you would like the container to use your docker config on Linux/Mac:
```
docker run -v "${HOME}/.docker/config.json":/.docker/config.json:ro -v "${PWD}":/run safewaters/docker-lock:${VERSION} [commands]
```
* If you would like the container to use your docker config on Windows:
```
docker run -v "%USERPROFILE%\.docker\config.json":/.docker/config.json:ro -v "%cd%":/run safewaters/docker-lock:${VERSION} [commands]
```
> Note: If your host machine uses a credential helper such as osxkeychain,
> wincred, or pass, the credentials will not be available to the container

# Build From Source
If you would like to install `docker-lock` from source, ensure `go` is
installed or use the [supplied development container](#Development-Environment).
From the root of the project, run:

```
go build ./cmd/docker-lock
```

If on Mac or Linux, make the output binary executable:

```
chmod +x docker-lock
```

Finally, move the binary to the cli-plugins folder or add it to your PATH,
as described in the [installation section](#Install-Pre-built-Binary).

If you would like to cross-compile for another operating system
or architecture, from the root of the project, run:

```
CGO_ENABLED=0 GOOS=<your os> GOARCH=<your arch> go build ./cmd/docker-lock
```

# Contributing

## Development Environment
A development container based on `ubuntu:bionic` has been provided,
so ensure docker is installed and the docker daemon is running.

* Open the project in [VSCode](https://code.visualstudio.com/).
* Install VSCode's [Remote Development Extension - Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.vscode-remote-extensionpack).
* In the command palette (ctrl+shift+p on Windows/Linux,
command+shift+p on Mac), type "Reopen in Container".
* In the command palette type: "Go: Install/Update Tools" and select all.
* When all tools are finished installing, in the command palette type:
"Developer: Reload Window".
* The docker daemon is mapped from the host into the dev container,
so you can use docker and docker-compose commands from within the container
as if they were run on the host.

## Code Quality and Correctness
Unit tests, integration tests, and linting run in the
[CI pipeline](https://dev.azure.com/michaelsethperel/docker-lock/_build)
on pull requests. Locally, you can run quality checks for everything except for integration tests.
* To format your code: `./scripts/format.sh`
* To lint your code: `./scripts/lint.sh`
* To run unit tests: `./scripts/unittest.sh`
* To generate a coverage report: `./scripts/coverage.sh`
* To view the coverage report on your browser, open a console, but not in
docker, run:
```
go tool cover -html=coverage.out
```

# Tutorials
* [Command Line Flags/Configuration File](./docs/tutorials/command-line-flags-configuration-file.md)
* [Using Internal Registries](./docs/tutorials/internal-registry.md)
* [Bring Your Own Registry](./docs/tutorials/bring-your-own-registry.md)
* [Tags Vs. Digests](./docs/tutorials/tags-vs-digests.md)
