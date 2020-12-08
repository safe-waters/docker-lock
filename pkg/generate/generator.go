package generate

import (
	"encoding/json"
	"errors"
	"io"
	"reflect"
)

type generator struct {
	pathCollector      IPathCollector
	imageParser        IImageParser
	imageDigestUpdater IImageDigestUpdater
	imageFormatter     IImageFormatter
}

// NewGenerator returns an IGenerator after ensuring all arguments are non-nil.
func NewGenerator(
	pathCollector IPathCollector,
	imageParser IImageParser,
	imageDigestUpdater IImageDigestUpdater,
	imageFormatter IImageFormatter,
) (IGenerator, error) {
	if pathCollector == nil || reflect.ValueOf(pathCollector).IsNil() {
		return nil, errors.New("'pathCollector' may not be nil")
	}

	if imageParser == nil || reflect.ValueOf(imageParser).IsNil() {
		return nil, errors.New("'imageParser' may not be nil")
	}

	if imageDigestUpdater == nil ||
		reflect.ValueOf(imageDigestUpdater).IsNil() {
		return nil, errors.New("'imageDigestUpdater' may not be nil")
	}

	if imageFormatter == nil ||
		reflect.ValueOf(imageFormatter).IsNil() {
		return nil, errors.New("'imageFormatter' may not be nil")
	}

	return &generator{
		pathCollector:      pathCollector,
		imageParser:        imageParser,
		imageDigestUpdater: imageDigestUpdater,
		imageFormatter:     imageFormatter,
	}, nil
}

// GenerateLockfile creates a Lockfile and writes it to an io.Writer.
func (g *generator) GenerateLockfile(lockfileWriter io.Writer) error {
	if lockfileWriter == nil || reflect.ValueOf(lockfileWriter).IsNil() {
		return errors.New("'lockfileWriter' cannot be nil")
	}

	done := make(chan struct{})
	defer close(done)

	paths := g.pathCollector.CollectPaths(done)
	images := g.imageParser.ParseFiles(paths, done)
	images = g.imageDigestUpdater.UpdateDigests(images, done)

	formattedImages, err := g.imageFormatter.FormatImages(images, done)
	if err != nil {
		return err
	}

	if len(formattedImages) == 0 {
		return nil
	}

	byt, err := json.MarshalIndent(formattedImages, "", "\t")
	if err != nil {
		return err
	}

	_, err = lockfileWriter.Write(byt)

	return err
}
