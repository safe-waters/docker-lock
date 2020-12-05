// Package update provides functionality to update images with digests.
package update

import "github.com/safe-waters/docker-lock/pkg/generate/parse"

// IImageDigestUpdater provides an interface for ImageDigestUpdaters, which
// update images with their digests.
type IImageDigestUpdater interface {
	UpdateDigests(
		images <-chan parse.IImage,
		done <-chan struct{},
	) <-chan parse.IImage
}

// IDigestRequester provides an interface for DigestRequesters, which are
// responsible for querying the proper registry for a digest given an image's
// name and tag.
type IDigestRequester interface {
	Digest(name string, tag string) (string, error)
}
