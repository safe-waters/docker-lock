![Docker-Lock-Banner](./docs/assets/readme-banner.png)
# About
![ci](https://github.com/safe-waters/docker-lock/workflows/ci/badge.svg)
![cd-master](https://github.com/safe-waters/docker-lock/workflows/cd-master/badge.svg)
![cd-tag](https://github.com/safe-waters/docker-lock/workflows/cd-tag/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/safe-waters/docker-lock)](https://goreportcard.com/report/github.com/safe-waters/docker-lock)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/safe-waters/docker-lock)](https://pkg.go.dev/github.com/safe-waters/docker-lock)

`docker-lock` is a cli tool that automates managing image digests by tracking
them in a separate Lockfile (think package-lock.json or Pipfile.lock). With
`docker-lock`, you can refer to images in **Dockerfiles**,
**docker-compose V3 files**, and **Kubernetes manifests** by
mutable tags (as in `python:3.6`) yet receive the same 
benefits as if you had specified immutable digests (as in `python:3.6@sha256:25a189a536ae4d7c77dd5d0929da73057b85555d6b6f8a66bfbcc1a7a7de094b`).

> Note: If you are unsure about the differences between tags and digests,
refer to this [quick summary](./docs/tutorials/tags-vs-digests.md).

`docker-lock` ships with 3 commands that take you from development 
to production:

* `docker lock generate` finds images in your `Dockerfiles`,
`docker-compose` files, and `Kubernetes` manifests and generates
a Lockfile containing digests that correspond to their tags.
* `docker lock verify` lets you know if there are more recent digests 
than those last recorded in the Lockfile.
* `docker lock rewrite` rewrites `Dockerfiles`, `docker-compose` files,
and `Kubernetes` manifests to include digests.

`docker-lock` is most commonly used as a
[cli-plugin](https://github.com/docker/cli/issues/1534) for `docker` so `lock`
can be used as subcommand of `docker` as in `docker lock`. However,
`docker-lock` does not require `docker` at all. Instead, it can be called
manually as a standalone executable as in `docker-lock lock`. 
This is especially convenient if the proper version of `docker` is unavailable
or you would prefer to use another container technology such as
[podman](https://podman.io/).

# Demo
Consider a project with a multi-stage build `Dockerfile` at its root:
```Dockerfile
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

Finally, lets assume the `Dockerfile` is ready to be built and shared.

Running `docker lock rewrite` will add digests from the Lockfile 
to all of the images.

![Rewrite GIF](./docs/assets/rewrite.gif)

At this point, the `Dockerfile` will contain all of the digest information 
from the Lockfile, so it will always maintain the same, known behavior 
in the future.

# Install
`docker-lock` can be run as a
* [cli-plugin](https://github.com/docker/cli/issues/1534) for `docker`
* standalone executable without `docker`
* prebuilt [image from Dockerhub](https://hub.docker.com/repository/docker/safewaters/docker-lock)

## Cli-plugin
Ensure `docker` cli version >= 19.03 is installed by running `docker --version`.

### Linux / Mac
```bash
$ mkdir -p "${HOME}/.docker/cli-plugins"
$ curl -fsSL "https://github.com/safe-waters/docker-lock/releases/download/v${VERSION}/docker-lock_${VERSION}_${OS}_${ARCH}.tar.gz" | tar -xz -C "${HOME}/.docker/cli-plugins" "docker-lock"
$ chmod +x "${HOME}/.docker/cli-plugins/docker-lock"
```

### Windows
* Create the folder `%USERPROFILE%\.docker\cli-plugins`
* Download the Windows release from the releases page.
* Unzip the release.
* Move `docker-lock.exe` into `%USERPROFILE%\.docker\cli-plugins`

## Standalone tool
* Follow the same instructions as in the
[cli-plugin section](#cli-plugin) except place the `docker-lock` executable in
your `PATH`.
* To use `docker-lock`, replace any `docker` command such as `docker lock` with
the name of the executable, `docker-lock`, as in `docker-lock lock`.
* To verify that `docker-lock` was installed, run:
```bash
$ docker-lock lock --help
```

## Docker image
`docker-lock` can be run in a `docker` container, as below. If you leave off
the `${VERSION}` tag, you will use the latest, nightly build from the master branch.

> Note: If your host machine uses a credential helper such as `osxkeychain`,
> `wincred`, or `pass`, the credentials will not be available to the container even
> if you pass in your `docker` config.

### Linux / Mac
* Without your `docker` config:
```bash
$ docker run -v "${PWD}":/run safewaters/docker-lock:${VERSION} [commands]
```
* With your `docker` config:
```bash
$ docker run -v "${HOME}/.docker/config.json":/.docker/config.json:ro -v "${PWD}":/run safewaters/docker-lock:${VERSION} [commands]
```
### Windows
* Without your `docker` config:
```bash
$ docker run -v "%cd%":/run safewaters/docker-lock:${VERSION} [commands]
```
* With your `docker` config:
```bash
$ docker run -v "%USERPROFILE%\.docker\config.json":/.docker/config.json:ro -v "%cd%":/run safewaters/docker-lock:${VERSION} [commands]
```

# Use
## Registries
`docker-lock` supports public and private registries. If necessary, login to
docker before using `docker-lock`.

## How to specify configuration options
`docker-lock` supports options via cli flags or a configuration file,
`.docker-lock.yml`.
The root of this repo has an example,
[.docker-lock.yml.example](./.docker-lock.example.yml).

To see available options, run commands with `--help`. For instance:

```bash
$ docker lock --help
$ docker lock generate --help
$ docker lock verify --help
$ docker lock rewrite --help
$ docker lock version --help
```

> Note: You can mix and match cli flags to get the output that you want.

## Generate
### Commands for Dockerfiles, docker-compose files, and Kubernetes manifests
* `docker lock generate` will collect all default files (`Dockerfile`,
`docker-compose.yaml`, `docker-compose.yml`, `pod.yml`, `pod.yaml`,
`deployment.yml`, `deployment.yaml`, `job.yml`, and `job.yaml` in the default
base directory, the directory from which the command is run) and generate a Lockfile.

* `docker lock generate --lockfile-name=[file name]` will generate a Lockfile with the
file name as the output, instead of the default `docker-lock.json`.

* `docker lock generate --update-existing-digests` will generate a Lockfile,
querying for all digests, even those that are hardcoded in the files. Normally,
if a digest is hardcoded, it would be used in the Lockfile.

* `docker lock generate --ignore-missing-digests` will generate a Lockfile,
recording images for which a digest could not be found as not having a digest.
Normally, if a digest cannot be found, `docker-lock` would print an error.

* `docker lock generate --base-dir=[sub directory]` will collect all default
files in a sub directory and generate a Lockfile.

### Commands for Dockerfiles
* `docker lock generate --dockerfiles=[file1,file2,file3]` will collect all
files from a comma separated list ("file1,file2,file3") as well as default
docker-compose files and Kubernetes manifests and generate a Lockfile.

* `docker lock generate --exclude-all-dockerfiles` will generate a Lockfile,
excluding all Dockerfiles.

* `docker lock generate --dockerfile-recursive` will collect all default
Dockerfiles (`Dockerfile`) in subdirectories from the base directory as well
as default docker-compose files and Kubernetes manifests in the base directory
and generate a Lockfile.

* `docker lock generate --dockerfile-globs='[glob pattern]'` will collect all
Dockerfiles that match the glob pattern relative to the base directory as well
as default docker-compose files and Kubernetes manifests in the base directory
and generate a Lockfile. Use '**' to recursively search directories.
Remember to quote using single quotes so that the glob is not expanded
before `docker-lock` uses it.

### Commands for docker-compose files
* `docker lock generate --composefiles=[file1,file2,file3]` will collect all
files from a comma separated list ("file1,file2,file3") as well as default
Dockerfiles files and Kubernetes manifests and generate a Lockfile.

* `docker lock generate --exclude-all-composefiles` will generate a Lockfile,
excluding all docker-compose files.

* `docker lock generate --composefile-recursive` will collect all default
docker-compose files (`docker-compose.yaml`, `docker-compose.yml`) in
subdirectories from the base directory as well as default Dockerfiles
and Kubernetes manifests in the base directory and generate a Lockfile.

* `docker lock generate --composefile-globs='[glob pattern]'` will collect all
docker-compose files that match the glob pattern relative to the base directory as well
as default Dockerfiles and Kubernetes manifests in the base directory
and generate a Lockfile. Use '**' to recursively search directories.
Remember to quote using single quotes so that the glob is not expanded
before `docker-lock` uses it.

### Commands for Kubernetes manifests
* `docker lock generate --kubernetesfiles=[file1,file2,file3]` will collect all
files from a comma separated list ("file1,file2,file3") as well as default
Dockerfiles files and docker-compose files and generate a Lockfile.

* `docker lock generate --exclude-all-kubernetesfiles` will generate a Lockfile,
excluding all Kubernetes manifests.

* `docker lock generate --kubernetesfile-recursive` will collect all default
Kubernetes manifests (`pod.yaml`, `pod.yml`) in
subdirectories from the base directory as well as default Dockerfiles
and docker-compose files in the base directory and generate a Lockfile.

* `docker lock generate --kubernetesfile-globs='[glob pattern]'` will collect all
Kubernetes manifests that match the glob pattern relative to the base directory as well
as default Dockerfiles and docker-compose files in the base directory
and generate a Lockfile. Use '**' to recursively search directories.
Remember to quote using single quotes so that the glob is not expanded
before `docker-lock` uses it.

## Verify
* `docker lock verify` will take an existing Lockfile, with the default name,
`docker-lock.json`, generate a new Lockfile and report differences between
the new and existing Lockfiles.

* `docker lock verify --lockfile-name=[file name]` will use another file, instead
of the default `docker-lock.json`, as the Lockfile.

* `docker lock verify --exclude-tags` will check for differences between a newly
generated Lockfile and the existing Lockfile, ignoring if tags are different.

* `docker lock verify --ignore-missing-digests` will verify, but when generating
the new Lockfile to compare against, will assume that digests that cannot be
found are empty. Normally, if a digest could not be found, an error would be
reported.

* `docker lock verify --update-existing-digests` will verify, but when generating
the new Lockfile to compare against, will query for digests even if they are hardcoded.
Normally, the new Lockfile would use the hardcoded digests, instead of querying
for the most recent one.

## Rewrite
* `docker lock rewrite` will write the image names, tags, and digests
from the Lockfile into the referenced Dockerfiles, docker-compose files,
and Kubernetes manifests.

* `docker lock rewrite --lockfile-name=[file name]` will use another file, instead
of the default `docker-lock.json`, as the Lockfile.

* `docker lock rewrite --exclude-tags` will write image names and digests,
but not the tags, from the Lockfile into the referenced Dockerfiles,
docker-compose files, and Kubernetes manifests.

* `docker lock rewrite --tempdir=[directory]` will create a temporary directory in the `[directory]` and
write all files into it. Afterwards, the files are renamed to the appropriate
location and the temporary directory is deleted. Normally, this occurs in the
current directory. In general, this 2 step process happens to ensure that
either all rewrites succeed, or none of them do. There are also other rollback
measures in `docker-lock` to ensure this transaction happens and you are not
left with some files rewritten if a failure occurs.

# Suggested workflow
* Locally run `docker lock generate` to create a Lockfile, `docker-lock.json`,
and commit it.
* Continue developing normally, as if the Lockfile does not exist.
* When merging a code change/releasing, run `docker-lock` in a CI/CD
pipeline. Specifically:
    * In the pipeline, run `docker lock verify` to make sure that the
    Lockfile is up-to-date. If `docker lock verify` fails, the developer can
    locally rerun `docker lock generate` to update the Lockfile. This has
    the benefit that digest changes will be explicitly tracked in git.
    * Once the `docker lock verify` step in the pipeline passes, the pipeline
    should run `docker lock rewrite` so all files have correct digests
    hardcoded in them.
    * The pipeline should run tests that use the rewritten images.
    * If the tests pass, merge the code change/push the images to
    the registry, etc.

# Contributing
## Development environment
A development container based on `ubuntu:bionic` has been provided,
so ensure `docker` is installed and the `docker` daemon is running.

* Open the project in [VSCode](https://code.visualstudio.com/).
* Install VSCode's [Remote Development Extension - Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.vscode-remote-extensionpack).
* In the command palette (ctrl+shift+p on Windows/Linux,
command+shift+p on Mac), type "Reopen in Container".
* In the command palette type: "Go: Install/Update Tools" and select all.
* When all tools are finished installing, in the command palette type:
"Developer: Reload Window".
* The `docker` daemon is mapped from the host into the dev container,
so you can use `docker` and `docker-compose` commands from within the container
as if they were run on the host.

## Build from source
To build and install `docker-lock` in `docker`'s cli-plugins directory,
from the root of the project, run:

```bash
$ make install
```

## Code quality and correctness
To clean, format, lint, and run unit tests:
```bash
make
```

The CI pipeline will additionally run integration tests on pull requests.

You can run any step individually.

* To uninstall: `make clean`
* To install into `docker`'s cli-plugins directory: `make install`
* To format Go code: `make format`
* To lint all code: `make lint`
* To run unit tests: `make unittest`

To view the coverage report after running unit tests, open `coverage.html` in
your browser.

>Note: While there exists a target in the Makefile for integration tests, these
cannot run locally because they require credentials that are only available in
the CI pipeline.

# Tutorials
* [Tags Vs. Digests](./docs/tutorials/tags-vs-digests.md)
