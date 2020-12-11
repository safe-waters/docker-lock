package format

import (
	"errors"
	"path/filepath"
	"sort"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type composefileImageFormatter struct {
	kind kind.Kind
}

type formattedComposefileImage struct {
	Name            string `json:"name"`
	Tag             string `json:"tag"`
	Digest          string `json:"digest"`
	DockerfilePath  string `json:"dockerfile,omitempty"`
	ServiceName     string `json:"service"`
	servicePosition int
}

// NewComposefileImageFormatter returns an IImageFormatter for Composefiles.
func NewComposefileImageFormatter() IImageFormatter {
	return &composefileImageFormatter{kind: kind.Composefile}
}

// Kind is a getter for the kind.
func (c *composefileImageFormatter) Kind() kind.Kind {
	return c.kind
}

// FormatImages returns a map with a key of filepath and a slice of images
// formatted for a Lockfile.
func (c *composefileImageFormatter) FormatImages(
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
			return nil, errors.New(
				"malformed 'path' in composefile image metadata",
			)
		}

		path = filepath.ToSlash(path)

		dockerfilePath, _ := metadata["dockerfilePath"].(string)
		dockerfilePath = filepath.ToSlash(dockerfilePath)

		serviceName, ok := metadata["serviceName"].(string)
		if !ok {
			return nil, errors.New(
				"malformed 'serviceName' in composefile image metadata",
			)
		}

		servicePosition, ok := metadata["servicePosition"].(int)
		if !ok {
			return nil, errors.New(
				"malformed 'servicePosition' in composefile image metadata",
			)
		}

		formattedImage := &formattedComposefileImage{
			Name:            image.Name(),
			Tag:             image.Tag(),
			Digest:          image.Digest(),
			DockerfilePath:  dockerfilePath,
			ServiceName:     serviceName,
			servicePosition: servicePosition,
		}

		formattedImages[path] = append(formattedImages[path], formattedImage)
	}

	var waitGroup sync.WaitGroup

	for _, images := range formattedImages {
		images := images

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			sort.Slice(images, func(i, j int) bool {
				image1 := images[i].(*formattedComposefileImage)
				image2 := images[j].(*formattedComposefileImage)

				switch {
				case image1.ServiceName != image2.ServiceName:
					return image1.ServiceName < image2.ServiceName
				case image1.DockerfilePath != image2.DockerfilePath:
					return image1.DockerfilePath < image2.DockerfilePath
				default:
					return image1.servicePosition < image2.servicePosition
				}
			})
		}()
	}

	waitGroup.Wait()

	return formattedImages, nil
}
