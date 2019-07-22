# About
`docker-lock` is a [cli-plugin](https://github.com/docker/cli/issues/1534) that generates lockfiles (think `package-lock.json` or `Pipfile.lock`) for docker and docker-compose to ensure reproducible builds.

Consider a project with a Dockerfile:
```
# Dockerfile
FROM python:3.6
...
```
Running `docker lock generate` would produce a lockfile, `docker-lock.json`, with an images section as follows:
```
// docker-lock.json
{
...
 "Images": [
   {
     "name": "python",
     "tag": "3.6",
     "digest": "25a189a536ae4d7c77dd5d0929da73057b85555d6b6f8a66bfbcc1a7a7de094b"
   }
 ]
...
}
```
The sha256 digest is recorded alongside the name and tag information. Since docker images are mutable, `python:3.6` maintainers could push an updated image with the same name and tag, which could break downstream applications.

Running `docker lock verify` compares the lockfile with digest information on the registry. If the digests have changed, verification will fail and print to the console.

## Why not specify the digest explicitly, instead of using `docker-lock`?
Dockerfiles can be written with the digest to avoid headaches from mutable images. For instance:
```
# Dockerfile
FROM python:3.6@sha256:25a189a536ae4d7c77dd5d0929da73057b85555d6b6f8a66bfbcc1a7a7de094b
```
Specifying the digest has a few downsides:
* The application will no longer benefit from updates (security updates, performance updates, etc.).
* The Dockerfile is considerably less readable.
* It is unclear whether the digest is specified as a precaution against future changes or if the application is incompatible with future changes.
* In projects with many Dockerfiles (think microservices), manually specifying the digests and keeping track of updates can be unwieldly.

By using `docker-lock`, developers get all of the benefits of specifying the digest without any of the downsides.

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

# Usage
***
The following commands assume that you installed the cli-plugin. If using `docker-lock` as a standalone tool, replace the following `docker lock ...` commands with `docker-lock lock ...`.
***
* `docker lock generate` generates a lockfile.
* `docker lock verify` verifies that the lockfile digests are the same as the ones in the registry.

By default, `docker-lock` looks for `Dockerfile`, `docker-compose.yml`, `docker-compose.yaml`, and `.env` files in the directory from which the command was run. However, `docker-lock` supports many different options via cli flags such as searching for Dockerfiles/docker-compose files via globs or recursive patterns.

To see the available options:
* `docker lock generate --help`
* `docker lock verify --help`

Where sensical, `docker lock generate` records flags in the output lockfile, so `docker lock verify` can be run without flags. However, there are a few flags that exist globally, such as `-o` to specify a different path/name for the lockfile.

# Coming soon
* `docker lock rewrite` to rewrite Dockerfiles and docker-compose files to include the digest (useful for CI/CD).
