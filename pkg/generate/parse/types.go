// Package parse provides functionality to parse images from collected files.
package parse

import (
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

// IImage provides an interface for Images.
type IImage interface {
	SetKind(kind kind.Kind)
	Kind() kind.Kind
	SetName(name string)
	Name() string
	SetTag(tag string)
	Tag() string
	SetDigest(digest string)
	Digest() string
	SetMetadata(metadata map[string]interface{})
	Metadata() map[string]interface{}
	ImageLine() string
	SetNameTagDigestFromImageLine(imageLine string)
	SetErr(err error)
	Err() error
}

// IImageParser provides an interface for ImageParsers, which are responsible
// for reading files and extracting Images from them.
type IImageParser interface {
	Kind() kind.Kind
	ParseFiles(
		paths <-chan collect.IPath,
		done <-chan struct{},
	) <-chan IImage
}

// IDockerfileImageParser is an IImageParser for Dockerfiles.
type IDockerfileImageParser interface {
	IImageParser
	ParseFile(
		path collect.IPath,
		buildArgs map[string]string,
		dockerfileImages chan<- IImage,
		done <-chan struct{},
		waitGroup *sync.WaitGroup,
	)
}

// IComposefileImageParser is an IImageParser for Composefiles.
type IComposefileImageParser interface {
	IImageParser
	ParseFile(
		path collect.IPath,
		composefileImages chan<- IImage,
		done <-chan struct{},
		waitGroup *sync.WaitGroup,
	)
}

// IKubernetesfileImageParser is an IImageParser for Kubernetesfiles.
type IKubernetesfileImageParser interface {
	IImageParser
	ParseFile(
		path collect.IPath,
		kubernetesfileImages chan<- IImage,
		done <-chan struct{},
		waitGroup *sync.WaitGroup,
	)
}
