// Package update provides functionality to update images with digests.
package update

import (
	"errors"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/generate/registry"
)

// ImageDigestUpdater uses a WrapperManager to update Images with their most
// recent digests from their registries.
type ImageDigestUpdater struct {
	WrapperManager *registry.WrapperManager
}

// IImageDigestUpdater provides an interface for ImageDigestUpdater's
// exported methods.
type IImageDigestUpdater interface {
	UpdateDigests(
		images <-chan *parse.Image,
		done <-chan struct{},
	) <-chan *UpdatedImage
}

// UpdatedImage contains an Image with its updated digest.
type UpdatedImage struct {
	Image *parse.Image
	Err   error
}

// NewImageDigestUpdater returns an ImageDigestUpdater after validating its
// fields.
func NewImageDigestUpdater(
	wrapperManager *registry.WrapperManager,
) (*ImageDigestUpdater, error) {
	if wrapperManager == nil {
		return nil, errors.New("wrapperManager cannot be nil")
	}

	return &ImageDigestUpdater{WrapperManager: wrapperManager}, nil
}

// UpdateDigests queries registries for digests of images that do not
// already specify their digests. It updates images with those
// digests.
func (i *ImageDigestUpdater) UpdateDigests(
	images <-chan *parse.Image,
	done <-chan struct{},
) <-chan *UpdatedImage {
	if images == nil {
		return nil
	}

	updatedImages := make(chan *UpdatedImage)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for image := range images {
			image := image

			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				if image.Digest != "" {
					select {
					case <-done:
					case updatedImages <- &UpdatedImage{Image: image}:
					}

					return
				}

				wrapper := i.WrapperManager.Wrapper(image.Name)

				digest, err := wrapper.Digest(image.Name, image.Tag)
				if err != nil {
					select {
					case <-done:
					case updatedImages <- &UpdatedImage{Image: image, Err: err}:
					}

					return
				}

				select {
				case <-done:
					return
				case updatedImages <- &UpdatedImage{
					Image: &parse.Image{
						Name:   image.Name,
						Tag:    image.Tag,
						Digest: digest,
					},
				}:
				}
			}()
		}
	}()

	go func() {
		waitGroup.Wait()
		close(updatedImages)
	}()

	return updatedImages
}
