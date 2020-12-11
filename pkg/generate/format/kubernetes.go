package format

import (
	"errors"
	"path/filepath"
	"sort"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type kubernetesfileImageFormatter struct {
	kind kind.Kind
}

type formattedKubernetesfileImage struct {
	Name          string `json:"name"`
	Tag           string `json:"tag"`
	Digest        string `json:"digest"`
	ContainerName string `json:"container"`
	imagePosition int
	docPosition   int
}

// NewKubernetesfileImageFormatter returns an IImageFormatter for
// Kubernetesfiles.
func NewKubernetesfileImageFormatter() IImageFormatter {
	return &kubernetesfileImageFormatter{kind: kind.Kubernetesfile}
}

// Kind is a getter for the kind.
func (k *kubernetesfileImageFormatter) Kind() kind.Kind {
	return k.kind
}

// FormatImages returns a map with a key of filepath and a slice of images
// formatted for a Lockfile.
func (k *kubernetesfileImageFormatter) FormatImages(
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
			return nil, errors.New("malformed 'path' in kubernetesfile image")
		}

		path = filepath.ToSlash(path)

		containerName, ok := metadata["containerName"].(string)
		if !ok {
			return nil, errors.New(
				"malformed 'containerName' in kubernetesfile image",
			)
		}

		imagePosition, ok := metadata["imagePosition"].(int)
		if !ok {
			return nil, errors.New(
				"malformed 'imagePosition' in kubernetesfile image",
			)
		}

		docPosition, ok := metadata["docPosition"].(int)
		if !ok {
			return nil, errors.New(
				"malformed 'docPosition' in kubernetesfile image",
			)
		}

		formattedImage := &formattedKubernetesfileImage{
			Name:          image.Name(),
			Tag:           image.Tag(),
			Digest:        image.Digest(),
			ContainerName: containerName,
			imagePosition: imagePosition,
			docPosition:   docPosition,
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
				image1 := images[i].(*formattedKubernetesfileImage)
				image2 := images[j].(*formattedKubernetesfileImage)

				switch {
				case image1.docPosition != image2.docPosition:
					return image1.docPosition < image2.docPosition
				default:
					return image1.imagePosition < image2.imagePosition
				}
			})
		}()
	}

	waitGroup.Wait()

	return formattedImages, nil
}
