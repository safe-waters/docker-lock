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
	if images == nil {
		return nil
	}

	var (
		waitGroup     sync.WaitGroup
		updatedImages = make(chan parse.IImage)
	)

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		var (
			imagesToQueryWaitGroup sync.WaitGroup
			imagesToQuery          = make(chan parse.IImage)
			imageLineCache         = map[string][]parse.IImage{}
		)

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

				key := image.ImageLine()
				if _, ok := imageLineCache[key]; !ok {
					metadata := image.Metadata()
					metadata["key"] = key

					select {
					case <-done:
						return
					case imagesToQuery <- parse.NewImage(
						image.Kind(), image.Name(), image.Tag(),
						image.Digest(), metadata, image.Err(),
					):
					}
				}

				imageLineCache[key] = append(imageLineCache[key], image)
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
			key := updatedImage.Metadata()["key"].(string)

			for _, image := range imageLineCache[key] {
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
