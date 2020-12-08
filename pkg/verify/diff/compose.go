package diff

import (
	"errors"

	"github.com/safe-waters/docker-lock/pkg/kind"
)

type composefileImageDifferentiator struct {
	kind                kind.Kind
	excludeTags         bool
	imageDifferentiator *imageDifferentiator
}

// NewComposefileDifferentiator returns an IImageDifferentiator for
// Composefiles.
func NewComposefileDifferentiator(excludeTags bool) IImageDifferentiator {
	return &composefileImageDifferentiator{
		kind:                kind.Composefile,
		excludeTags:         excludeTags,
		imageDifferentiator: &imageDifferentiator{},
	}
}

// DifferentiateImage reports differences between images in the fields
// "name", "tag", "digest", "dockerfile", and "service".
func (c *composefileImageDifferentiator) DifferentiateImage(
	existingImage map[string]interface{},
	newImage map[string]interface{},
) error {
	if existingImage == nil {
		return errors.New("'existingImage' cannot be nil")
	}

	if newImage == nil {
		return errors.New("'newImage' cannot be nil")
	}

	var diffFields = []string{"name", "tag", "digest", "dockerfile", "service"}

	if c.excludeTags {
		const tagIndex = 1

		diffFields = append(diffFields[:tagIndex], diffFields[tagIndex+1:]...)
	}

	return c.imageDifferentiator.differentiateImage(
		existingImage, newImage, diffFields,
	)
}

// Kind is a getter for the kind.
func (c *composefileImageDifferentiator) Kind() kind.Kind {
	return c.kind
}
