package update

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
)

type imageDigestUpdater struct {
	digestRequester       IDigestRequester
	ignoreMissingDigests  bool
	updateExistingDigests bool
}

// NewImageDigestUpdater returns an IImageDigestUpdater after validating its
// fields. digestRequester cannot be nil as it is responsible for querying
// registries for digests.
func NewImageDigestUpdater(
	digestRequester IDigestRequester,
	ignoreMissingDigests bool,
	updateExistingDigests bool,
) (IImageDigestUpdater, error) {
	if digestRequester == nil || reflect.ValueOf(digestRequester).IsNil() {
		return nil, errors.New("'digestRequester' cannot be nil")
	}

	return &imageDigestUpdater{
		digestRequester:       digestRequester,
		ignoreMissingDigests:  ignoreMissingDigests,
		updateExistingDigests: updateExistingDigests,
	}, nil
}

// UpdateDigests queries registries for digests of images that do not
// already specify their digests. It updates images with those
// digests.
func (i *imageDigestUpdater) UpdateDigests(
	images <-chan parse.IImage,
	done <-chan struct{},
) <-chan parse.IImage {
	if images == nil {
		return nil
	}

	var (
		waitGroup     sync.WaitGroup
		doOnce        sync.Once
		updatedImages = make(chan parse.IImage)
	)

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for image := range images {
			image := image

			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				if image.Err() != nil {
					select {
					case <-done:
					case updatedImages <- image:
					}

					return
				}

				if image.Digest() != "" &&
					i.updateExistingDigests &&
					image.Tag() == "" {
					doOnce.Do(func() {
						fmt.Println(
							"warning: image(s) with a digest but no tag " +
								"found, will use existing digest",
						)
					})

					select {
					case <-done:
					case updatedImages <- image:
					}

					return
				}

				if (image.Digest() != "" && !i.updateExistingDigests) ||
					image.Tag() == "" {
					select {
					case <-done:
					case updatedImages <- image:
					}

					return
				}

				digest, err := i.digestRequester.Digest(
					image.Name(), image.Tag(),
				)
				if err != nil && !i.ignoreMissingDigests {
					errMsg := fmt.Errorf(
						"failed to update image with err: %v", err,
					)

					metadata := image.Metadata()
					if path, ok := metadata["path"]; ok {
						errMsg = fmt.Errorf("on '%s': %v", path, errMsg)
					}

					select {
					case <-done:
					case updatedImages <- parse.NewImage(
						image.Kind(), "", "", "", nil, errMsg,
					):
					}

					return
				}

				select {
				case <-done:
					return
				case updatedImages <- parse.NewImage(
					image.Kind(), image.Name(), image.Tag(),
					digest, image.Metadata(), nil,
				):
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
