package writers

// compose represents the services section of a docker-compose file.
type compose struct {
	Services map[string]*service `yaml:"services"`
}

// service represents a service in the service section of a docker-compose file.
type service struct {
	Image string      `yaml:"image"`
	Build interface{} `yaml:"build"`
}
