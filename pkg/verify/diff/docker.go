package diff

import (
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type dockerfileImageDifferentiator struct {
	kind                kind.Kind
	excludeTags         bool
	imageDifferentiator *imageDifferentiator
}

// NewDockerfileDifferentiator returns an IImageDifferentiator for
// Dockerfiles.
func NewDockerfileDifferentiator(excludeTags bool) IImageDifferentiator {
	return &dockerfileImageDifferentiator{
		kind:                kind.Dockerfile,
		excludeTags:         excludeTags,
		imageDifferentiator: &imageDifferentiator{},
	}
}

// DifferentiateImage reports differences between images in the fields
// "name", "tag", and "digest".
func (d *dockerfileImageDifferentiator) DifferentiateImage(
	existingImage map[string]interface{},
	newImage map[string]interface{},
) error {
	var diffFields = []string{"name", "tag", "digest"}

	if d.excludeTags {
		const tagIndex = 1

		diffFields = append(diffFields[:tagIndex], diffFields[tagIndex+1:]...)
	}

	return d.imageDifferentiator.differentiateImage(
		existingImage, newImage, diffFields,
	)
}

// Kind is a getter for the kind.
func (d *dockerfileImageDifferentiator) Kind() kind.Kind {
	return d.kind
}
