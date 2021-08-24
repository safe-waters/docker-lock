module github.com/safe-waters/docker-lock

go 1.16

require (
	github.com/compose-spec/compose-go v0.0.0-20210722130045-6e1e1c2b26de
	github.com/google/go-containerregistry v0.5.1
	github.com/mattn/go-zglob v0.0.3
	github.com/moby/buildkit v0.8.3
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.8.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/client-go v0.21.1
)

replace github.com/compose-spec/compose-go => github.com/michaelperel/compose-go v0.0.0-20210824182527-54163a104a9a
