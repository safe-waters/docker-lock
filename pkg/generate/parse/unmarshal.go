package parse

import (
	"errors"
)

// compose contains the services in a docker-compose file.
type compose struct {
	Services map[string]*service `yaml:"services"`
}

// service represents the service section of a docker-compose file.
type service struct {
	ImageName    string        `yaml:"image"`
	BuildWrapper *buildWrapper `yaml:"build"`
}

// buildWrapper describes the "build" section of a service. It is used
// when unmarshalling to either contain simple or verbose build sections.
type buildWrapper struct {
	Build interface{}
}

// simple represents a "build" section without build keys. For instance,
// build: dirWithDockerfile
type simple string

// verbose represents a "build" section with build keys specified. For instance,
// build:
//     context: ./dirWithDockerfile
//     dockerfile: Dockerfile
type verbose struct {
	Context        string       `yaml:"context"`
	DockerfilePath string       `yaml:"dockerfile"`
	ArgsWrapper    *argsWrapper `yaml:"args"`
}

// argsWrapper describes the "args" section of a build section. It can contain
// a slice of strings or a map.
type argsWrapper struct {
	Args interface{}
}

// argsSlice can be build args as keys that reference environment vars or
// keys and values.
type argsSlice []string

// argsMap are build args as keys and values.
type argsMap map[string]string

// UnmarshalYAML unmarshals the "build" section of a service into either
// a simple or verbose build.
func (b *buildWrapper) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if unmarshal == nil {
		return errors.New("unmarshal cannot be nil")
	}

	*b = buildWrapper{}

	var v verbose
	if err := unmarshal(&v); err == nil {
		b.Build = v
		return nil
	}

	var s simple
	if err := unmarshal(&s); err == nil {
		b.Build = s
		return nil
	}

	return errors.New("unable to unmarshal service")
}

// UnmarshalYAML unmarshals the "args" section of a verbose service. Args can
// be either slices or maps.
func (a *argsWrapper) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if unmarshal == nil {
		return errors.New("unmarshal cannot be nil")
	}

	*a = argsWrapper{}

	var as argsSlice
	if err := unmarshal(&as); err == nil {
		a.Args = as
		return nil
	}

	var am argsMap
	if err := unmarshal(&am); err == nil {
		a.Args = am
		return nil
	}

	return errors.New("unable to unmarshal build args")
}
