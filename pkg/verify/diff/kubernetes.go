package diff

import (
	"errors"

	"github.com/safe-waters/docker-lock/pkg/kind"
)

type kubernetesfileImageDifferentiator struct {
	kind                kind.Kind
	excludeTags         bool
	imageDifferentiator *imageDifferentiator
}

// NewKubernetesfileDifferentiator returns an IImageDifferentiator for
// Kubernetesfiles.
func NewKubernetesfileDifferentiator(excludeTags bool) IImageDifferentiator {
	return &kubernetesfileImageDifferentiator{
		kind:                kind.Kubernetesfile,
		excludeTags:         excludeTags,
		imageDifferentiator: &imageDifferentiator{},
	}
}

// DifferentiateImage reports differences between images in the fields
// "name", "tag", "digest", and "container".
func (k *kubernetesfileImageDifferentiator) DifferentiateImage(
	existingImage map[string]interface{},
	newImage map[string]interface{},
) error {
	if existingImage == nil {
		return errors.New("'existingImage' cannot be nil")
	}

	if newImage == nil {
		return errors.New("'newImage' cannot be nil")
	}

	var diffFields = []string{"name", "tag", "digest", "container"}

	if k.excludeTags {
		const tagIndex = 1

		diffFields = append(diffFields[:tagIndex], diffFields[tagIndex+1:]...)
	}

	return k.imageDifferentiator.differentiateImage(
		existingImage, newImage, diffFields,
	)
}

// Kind is a getter for the kind.
func (k *kubernetesfileImageDifferentiator) Kind() kind.Kind {
	return k.kind
}
