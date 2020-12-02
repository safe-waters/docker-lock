// Package kind marks a type of resource.
package kind

// Kind marks a type of resource.
type Kind string

const (
	Dockerfile     Kind = "dockerfiles"
	Composefile    Kind = "composefiles"
	Kubernetesfile Kind = "kubernetesfiles"
)
