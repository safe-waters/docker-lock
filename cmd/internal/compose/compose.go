package compose

type Compose struct {
	Services map[string]Service `yaml:"services"`
}

type Service struct {
	ImageName string `yaml:"image"`
}
