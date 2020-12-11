package generate

import (
	"errors"
	"reflect"
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
	if updater == nil || reflect.ValueOf(updater).IsNil() {
		return nil, errors.New("'updater' cannot be nil")
	}

	return &imageDigestUpdater{updater: updater}, nil
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
					if metadata == nil {
						metadata = map[string]interface{}{}
					}

					if _, ok := metadata["__updateKey"]; ok {
						select {
						case <-done:
						case updatedImages <- parse.NewImage(
							image.Kind(), "", "", "", nil,
							errors.New(
								"image metadata key '__updateKey' is reserved",
							),
						):
						}

						return
					}

					metadata["__updateKey"] = key

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
			metadata := updatedImage.Metadata()
			if metadata == nil {
				select {
				case <-done:
				case updatedImages <- parse.NewImage(
					updatedImage.Kind(), "", "", "", nil,
					errors.New("image 'metadata' cannot be nil"),
				):
				}

				return
			}

			key, _ := metadata["__updateKey"].(string)
			if key == "" {
				select {
				case <-done:
				case updatedImages <- parse.NewImage(
					updatedImage.Kind(), "", "", "", nil,
					errors.New("missing '__updateKey' in image"),
				):
				}

				return
			}

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
