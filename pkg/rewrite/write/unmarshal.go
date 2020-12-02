package write

type compose struct {
	Services map[string]*service `yaml:"services"`
}

type service struct {
	Image string      `yaml:"image"`
	Build interface{} `yaml:"build"`
}
