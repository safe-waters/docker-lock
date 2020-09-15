// Package generate provides functionality to generate a Lockfile.
package generate

import (
	"errors"
	"io"
	"reflect"
)

// Generator creates a Lockfile.
type Generator struct {
	PathCollector      IPathCollector
	ImageParser        IImageParser
	ImageDigestUpdater IImageDigestUpdater
}

// IGenerator provides an interface for Generator's exported
// methods, which are used by docker-lock's cli as well as Verifier.
type IGenerator interface {
	GenerateLockfile(writer io.Writer) error
}

// NewGenerator returns a Generator after validating its fields.
func NewGenerator(
	pathCollector IPathCollector,
	imageParser IImageParser,
	imageDigestUpdater IImageDigestUpdater,
) (*Generator, error) {
	if pathCollector == nil || reflect.ValueOf(pathCollector).IsNil() {
		return nil, errors.New("pathCollector may not be nil")
	}

	if imageParser == nil || reflect.ValueOf(imageParser).IsNil() {
		return nil, errors.New("imageParser may not be nil")
	}

	if imageDigestUpdater == nil ||
		reflect.ValueOf(imageDigestUpdater).IsNil() {
		return nil, errors.New("imageDigestUpdater may not be nil")
	}

	return &Generator{
		PathCollector:      pathCollector,
		ImageParser:        imageParser,
		ImageDigestUpdater: imageDigestUpdater,
	}, nil
}

// GenerateLockfile creates a Lockfile and writes it to an io.Writer.
func (g *Generator) GenerateLockfile(writer io.Writer) error {
	if writer == nil || reflect.ValueOf(writer).IsNil() {
		return errors.New("writer cannot be nil")
	}

	done := make(chan struct{})

	dockerfilePaths, composefilePaths := g.PathCollector.CollectPaths(done)

	dockerfileImages, composefileImages := g.ImageParser.ParseFiles(
		dockerfilePaths, composefilePaths, done,
	)

	updatedDockerfileImages, updatedComposefileImages := g.ImageDigestUpdater.UpdateDigests( // nolint: lll
		dockerfileImages, composefileImages, done,
	)

	lockfile, err := NewLockfile(
		updatedDockerfileImages, updatedComposefileImages, done,
	)
	if err != nil {
		close(done)
		return err
	}

	return lockfile.Write(writer)
}
