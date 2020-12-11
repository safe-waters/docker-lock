package format

import (
	"errors"
	"path/filepath"
	"sort"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type dockerfileImageFormatter struct {
	kind kind.Kind
}

type formattedDockerfileImage struct {
	Name     string `json:"name"`
	Tag      string `json:"tag"`
	Digest   string `json:"digest"`
	position int
}

// NewDockerfileImageFormatter returns an IImageFormatter for Dockerfiles.
func NewDockerfileImageFormatter() IImageFormatter {
	return &dockerfileImageFormatter{kind: kind.Dockerfile}
}

// Kind is a getter for the kind.
func (d *dockerfileImageFormatter) Kind() kind.Kind {
	return d.kind
}

// FormatImages returns a map with a key of filepath and a slice of images
// formatted for a Lockfile.
func (d *dockerfileImageFormatter) FormatImages(
	images <-chan parse.IImage,
) (map[string][]interface{}, error) {
	if images == nil {
		return nil, errors.New("'images' cannot be nil")
	}

	formattedImages := map[string][]interface{}{}

	for image := range images {
		if image.Err() != nil {
			return nil, image.Err()
		}

		metadata := image.Metadata()
		if metadata == nil {
			return nil, errors.New("'metadata' cannot be nil")
		}

		path, ok := metadata["path"].(string)
		if !ok {
			return nil, errors.New("malformed 'path' in dockerfile image")
		}

		path = filepath.ToSlash(path)

		position, ok := metadata["position"].(int)
		if !ok {
			return nil, errors.New("malformed 'position' in dockerfile image")
		}

		formattedImage := &formattedDockerfileImage{
			Name:     image.Name(),
			Tag:      image.Tag(),
			Digest:   image.Digest(),
			position: position,
		}

		formattedImages[path] = append(formattedImages[path], formattedImage)
	}

	var waitGroup sync.WaitGroup

	for _, images := range formattedImages {
		images := images

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			sort.Slice(images, func(i int, j int) bool {
				image1 := images[i].(*formattedDockerfileImage)
				image2 := images[j].(*formattedDockerfileImage)

				return image1.position < image2.position
			})
		}()
	}

	waitGroup.Wait()

	return formattedImages, nil
}
