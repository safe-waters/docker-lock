package rewriter

type Compose struct {
	Services map[string]Service `yaml:"services"`
}

type Service struct {
	Image string      `yaml:"image"`
	Build interface{} `yaml:"build"`
}
