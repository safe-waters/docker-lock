package generate

import (
	"errors"
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/generate/update"
)

// ImageDigestUpdater contains an ImageDigestUpdater for all Images.
type ImageDigestUpdater struct {
	ImageDigestUpdater   update.IImageDigestUpdater
	IgnoreMissingDigests bool
}

// IImageDigestUpdater provides an interface for ImageDigestUpdater's exported
// methods, which are used by Generator.
type IImageDigestUpdater interface {
	UpdateDigests(
		anyImages <-chan *AnyImage, done <-chan struct{},
	) <-chan *AnyImage
}

// NewImageDigestUpdater returns an ImageDigestUpdater after validating its
// fields.
func NewImageDigestUpdater(
	imageDigestUpdater update.IImageDigestUpdater,
	ignoreMissingDigests bool,
) (*ImageDigestUpdater, error) {
	if imageDigestUpdater == nil ||
		reflect.ValueOf(imageDigestUpdater).IsNil() {
		return nil, errors.New("imageDigestUpdater cannot be nil")
	}

	return &ImageDigestUpdater{
		ImageDigestUpdater:   imageDigestUpdater,
		IgnoreMissingDigests: ignoreMissingDigests,
	}, nil
}

// UpdateDigests updates digests for DockerfileImages and ComposefileImages.
func (i *ImageDigestUpdater) UpdateDigests(
	anyImages <-chan *AnyImage,
	done <-chan struct{},
) <-chan *AnyImage {
	if anyImages == nil {
		return nil
	}

	updatedAnyImages := make(chan *AnyImage)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		imagesWithoutDigests := make(chan *parse.Image)
		digestsToUpdate := map[parse.Image][]*AnyImage{}

		var imagesWithoutDigestsWaitGroup sync.WaitGroup

		imagesWithoutDigestsWaitGroup.Add(1)

		go func() {
			defer imagesWithoutDigestsWaitGroup.Done()

			for anyImage := range anyImages {
				if anyImage.Err != nil {
					select {
					case <-done:
					case updatedAnyImages <- anyImage:
					}

					return
				}

				switch {
				case anyImage.DockerfileImage != nil:
					if anyImage.DockerfileImage.Image.Digest != "" {
						select {
						case <-done:
							return
						case updatedAnyImages <- anyImage:
						}

						continue
					}

					if _, ok := digestsToUpdate[*anyImage.DockerfileImage.Image]; !ok { // nolint: lll
						select {
						case <-done:
							return
						case imagesWithoutDigests <- anyImage.DockerfileImage.Image: // nolint: lll
						}
					}

					digestsToUpdate[*anyImage.DockerfileImage.Image] = append(
						digestsToUpdate[*anyImage.DockerfileImage.Image],
						anyImage,
					)
				case anyImage.ComposefileImage != nil:
					if anyImage.ComposefileImage.Image.Digest != "" {
						select {
						case <-done:
							return
						case updatedAnyImages <- anyImage:
						}

						continue
					}

					if _, ok := digestsToUpdate[*anyImage.ComposefileImage.Image]; !ok { // nolint: lll
						select {
						case <-done:
							return
						case imagesWithoutDigests <- anyImage.ComposefileImage.Image: // nolint: lll
						}
					}

					digestsToUpdate[*anyImage.ComposefileImage.Image] = append(
						digestsToUpdate[*anyImage.ComposefileImage.Image],
						anyImage,
					)
				}
			}
		}()

		go func() {
			imagesWithoutDigestsWaitGroup.Wait()
			close(imagesWithoutDigests)
		}()

		var allUpdatedImages []*parse.Image

		updatedImages := i.ImageDigestUpdater.UpdateDigests(
			imagesWithoutDigests, done,
		)

		for updatedImage := range updatedImages {
			if updatedImage.Err != nil && !i.IgnoreMissingDigests {
				select {
				case <-done:
				case updatedAnyImages <- &AnyImage{Err: updatedImage.Err}:
				}

				return
			}

			allUpdatedImages = append(allUpdatedImages, updatedImage.Image)
		}

		for _, updatedImage := range allUpdatedImages {
			key := parse.Image{Name: updatedImage.Name, Tag: updatedImage.Tag}

			for _, anyImage := range digestsToUpdate[key] {
				switch {
				case anyImage.DockerfileImage != nil:
					anyImage.DockerfileImage.Digest = updatedImage.Digest
				case anyImage.ComposefileImage != nil:
					anyImage.ComposefileImage.Digest = updatedImage.Digest
				}

				select {
				case <-done:
					return
				case updatedAnyImages <- anyImage:
				}
			}
		}
	}()

	go func() {
		waitGroup.Wait()
		close(updatedAnyImages)
	}()

	return updatedAnyImages
}
