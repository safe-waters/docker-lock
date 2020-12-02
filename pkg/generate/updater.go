package generate

import (
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/generate/update"
)

type imageDigestUpdater struct {
	updater update.IImageDigestUpdater
}

// NewImageDigestUpdater creates an IImageDigestUpdater from an
// IImageDigestUpdater.
func NewImageDigestUpdater(
	updater update.IImageDigestUpdater,
) (IImageDigestUpdater, error) {
	return &imageDigestUpdater{
		updater: updater,
	}, nil
}

// UpdateDigests updates images with the most recent digests from registries.
func (i *imageDigestUpdater) UpdateDigests(
	images <-chan parse.IImage,
	done <-chan struct{},
) <-chan parse.IImage {
	updatedImages := make(chan parse.IImage)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		imagesWithoutDigests := make(chan parse.IImage)
		digestsToUpdate := map[string][]parse.IImage{}

		var imagesWithoutDigestsWaitGroup sync.WaitGroup

		imagesWithoutDigestsWaitGroup.Add(1)

		go func() {
			defer imagesWithoutDigestsWaitGroup.Done()

			for image := range images {
				if image.Err() != nil {
					select {
					case <-done:
					case updatedImages <- image:
					}

					return
				}

				if image.Digest() == "" {
					key := image.Name() + image.Tag()
					if _, ok := digestsToUpdate[key]; !ok {
						select {
						case <-done:
							return
						case imagesWithoutDigests <- image:
						}
					}

					digestsToUpdate[key] = append(digestsToUpdate[key], image)
				} else {
					select {
					case <-done:
						return
					case updatedImages <- image:
					}
				}
			}
		}()

		go func() {
			imagesWithoutDigestsWaitGroup.Wait()
			close(imagesWithoutDigests)
		}()

		var allUpdatedImages []parse.IImage

		for updatedImage := range i.updater.UpdateDigests(
			imagesWithoutDigests, done,
		) {
			if updatedImage.Err() != nil {
				select {
				case <-done:
				case updatedImages <- updatedImage:
				}

				return
			}

			allUpdatedImages = append(allUpdatedImages, updatedImage)
		}

		for _, updatedImage := range allUpdatedImages {
			key := updatedImage.Name() + updatedImage.Tag()

			for _, image := range digestsToUpdate[key] {
				image.SetDigest(updatedImage.Digest())

				select {
				case <-done:
					return
				case updatedImages <- image:
				}
			}
		}
	}()

	go func() {
		waitGroup.Wait()
		close(updatedImages)
	}()

	return updatedImages
}
