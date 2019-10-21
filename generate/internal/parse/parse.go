package parse

import "errors"

type Compose struct {
	Services map[string]Service `yaml:"services"`
}

type Service struct {
	ImageName    string        `yaml:"image"`
	BuildWrapper *BuildWrapper `yaml:"build"`
}

type Verbose struct {
	Context    string   `yaml:"context"`
	Dockerfile string   `yaml:"dockerfile"`
	Args       []string `yaml:"args"`
}

type Simple string

type BuildWrapper struct {
	Build interface{}
}

func (b *BuildWrapper) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*b = BuildWrapper{}
	var v Verbose
	if err := unmarshal(&v); err == nil {
		b.Build = v
		return nil
	}
	var s Simple
	if err := unmarshal(&s); err == nil {
		b.Build = s
		return nil
	}
	return errors.New("Unable to parse service.")
}
