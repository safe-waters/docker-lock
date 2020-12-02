// Package diff provides functionality to diff an image.
package diff

import "github.com/safe-waters/docker-lock/pkg/kind"

// IImageDifferentiator provides an interface for ImageDifferentiators, which
// are responsible for reporting the difference between an image in the existing
// Lockfile and an image in the newly generated Lockfile.
type IImageDifferentiator interface {
	DifferentiateImage(
		existingImage map[string]interface{},
		newImage map[string]interface{},
	) error
	Kind() kind.Kind
}
