// Package generate provides functionality to generate a Lockfile.
package generate

import (
	"io"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

// IGenerator provides an interface for Generators, which are responsible
// for creating Lockfiles.
type IGenerator interface {
	GenerateLockfile(lockfileWriter io.Writer) error
}

// IPathCollector provides an interface for PathCollectors, which are
// responsible for collecting paths.
type IPathCollector interface {
	CollectPaths(done <-chan struct{}) <-chan collect.IPath
}

// IImageParser provides an interface for ImageParsers, which are responsible
// for parsing images from paths.
type IImageParser interface {
	ParseFiles(
		paths <-chan collect.IPath,
		done <-chan struct{},
	) <-chan parse.IImage
}

// IImageDigestUpdater provides an interface for ImageDigestUpdaters, which
// are responsible for querying registries for digests and updating images
// with them.
type IImageDigestUpdater interface {
	UpdateDigests(
		images <-chan parse.IImage,
		done <-chan struct{},
	) <-chan parse.IImage
}

// IImageFormatter provides an interface for ImageFormatters, which
// are responsible for formatting images for a Lockfile.
type IImageFormatter interface {
	FormatImages(
		images <-chan parse.IImage,
		done <-chan struct{},
	) (map[kind.Kind]map[string][]interface{}, error)
}
