// Package update provides functionality to update images with digests.
package update

import "github.com/safe-waters/docker-lock/pkg/generate/parse"

// IImageDigestUpdater provides an interface for ImageDigestUpdaters, which
// query registries for digests and update images with them.
type IImageDigestUpdater interface {
	UpdateDigests(
		images <-chan parse.IImage,
		done <-chan struct{},
	) <-chan parse.IImage
}
