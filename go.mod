module github.com/safe-waters/docker-lock

go 1.12

require (
	github.com/docker/docker-credential-helpers v0.6.3
	github.com/joho/godotenv v1.3.0
	github.com/kr/pretty v0.1.0 // indirect
	github.com/moby/buildkit v0.7.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.3.2
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.2.4
)

replace github.com/containerd/containerd => github.com/containerd/containerd v1.4.1-0.20201106004755-ac61e58cdd40

replace github.com/docker/docker => github.com/docker/docker v20.10.0-beta1.0.20201106221325-b5ea9abf258e+incompatible
