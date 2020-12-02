// Package format provides functionality to format images for a Lockfile.
package format

import (
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

// IImageFormatter provides an interface for ImageFormatters, which
// ensure images are properly formatted for a Lockfile.
type IImageFormatter interface {
	Kind() kind.Kind
	FormatImages(images <-chan parse.IImage) (map[string][]interface{}, error)
}
