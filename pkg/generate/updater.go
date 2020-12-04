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

		imagesToQuery := make(chan parse.IImage)
		imagesToQueryCache := map[string][]parse.IImage{}

		var imagesToQueryWaitGroup sync.WaitGroup

		imagesToQueryWaitGroup.Add(1)

		go func() {
			defer imagesToQueryWaitGroup.Done()

			for image := range images {
				if image.Err() != nil {
					select {
					case <-done:
					case updatedImages <- image:
					}

					return
				}

				key := image.Name() + image.Tag()
				if _, ok := imagesToQueryCache[key]; !ok {
					select {
					case <-done:
						return
					case imagesToQuery <- image:
					}
				}

				imagesToQueryCache[key] = append(imagesToQueryCache[key], image)
			}
		}()

		go func() {
			imagesToQueryWaitGroup.Wait()
			close(imagesToQuery)
		}()

		var allUpdatedImages []parse.IImage

		for updatedImage := range i.updater.UpdateDigests(
			imagesToQuery, done,
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

			for _, image := range imagesToQueryCache[key] {
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
