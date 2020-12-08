package generate

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/format"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type imageFormatter struct {
	formatters map[kind.Kind]format.IImageFormatter
}

type formattedResult struct {
	kind            kind.Kind
	formattedImages map[string][]interface{}
	err             error
}

// NewImageFormatter creates an IImageFormatter from IImageFormatters for
// different kinds of images. At least one formatter must be non nil, otherwise
// there would be no way to format images.
func NewImageFormatter(
	formatters ...format.IImageFormatter,
) (IImageFormatter, error) {
	kindFormatter := map[kind.Kind]format.IImageFormatter{}

	for _, formatter := range formatters {
		if formatter != nil && !reflect.ValueOf(formatter).IsNil() {
			kindFormatter[formatter.Kind()] = formatter
		}
	}

	if len(kindFormatter) == 0 {
		return nil, errors.New("non nil 'formatters' must be greater than 0")
	}

	return &imageFormatter{formatters: kindFormatter}, nil
}

// FormatImages formats all images for a Lockfile.
func (i *imageFormatter) FormatImages(
	images <-chan parse.IImage,
	done <-chan struct{},
) (map[kind.Kind]map[string][]interface{}, error) {
	if images == nil {
		return nil, errors.New("'images' cannot be nil")
	}

	var (
		kindImagesWaitGroup sync.WaitGroup
		kindImages          = map[kind.Kind]chan parse.IImage{}
	)

	for kind := range i.formatters {
		kindImages[kind] = make(chan parse.IImage)
	}

	for image := range images {
		if image.Err() != nil {
			return nil, image.Err()
		}

		if _, ok := i.formatters[image.Kind()]; !ok {
			return nil, fmt.Errorf(
				"kind '%s' does not have a formatter defined", image.Kind(),
			)
		}

		image := image

		kindImagesWaitGroup.Add(1)

		go func() {
			defer kindImagesWaitGroup.Done()

			select {
			case <-done:
				return
			case kindImages[image.Kind()] <- image:
			}
		}()
	}

	go func() {
		kindImagesWaitGroup.Wait()

		for _, images := range kindImages {
			close(images)
		}
	}()

	var (
		waitGroup        sync.WaitGroup
		formattedResults = make(chan *formattedResult)
	)

	for kind, images := range kindImages {
		kind := kind
		images := images

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			formattedImages, err := i.formatters[kind].FormatImages(images)
			if err != nil {
				select {
				case <-done:
				case formattedResults <- &formattedResult{err: err}:
				}

				return
			}

			if len(formattedImages) > 0 {
				select {
				case <-done:
				case formattedResults <- &formattedResult{
					kind: kind, formattedImages: formattedImages,
				}:
				}
			}
		}()
	}

	go func() {
		waitGroup.Wait()
		close(formattedResults)
	}()

	formattedKindImages := map[kind.Kind]map[string][]interface{}{}

	for formattedResult := range formattedResults {
		if formattedResult.err != nil {
			return nil, formattedResult.err
		}

		kind := formattedResult.kind
		formattedKindImages[kind] = formattedResult.formattedImages
	}

	return formattedKindImages, nil
}
