package generate

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type imageParser struct {
	parsers map[kind.Kind]parse.IImageParser
}

// NewImageParser creates an IImageParser from IImageParsers for different
// kinds of images. At least one parser must be non nil, otherwise there would
// no way to parse images.
func NewImageParser(parsers ...parse.IImageParser) (IImageParser, error) {
	kindParser := map[kind.Kind]parse.IImageParser{}

	for _, parser := range parsers {
		if parser != nil && !reflect.ValueOf(parser).IsNil() {
			kindParser[parser.Kind()] = parser
		}
	}

	if len(kindParser) == 0 {
		return nil, errors.New("non nil 'parsers' must be greater than 0")
	}

	return &imageParser{parsers: kindParser}, nil
}

// ParseFiles parses all images from paths.
func (i *imageParser) ParseFiles(
	paths <-chan collect.IPath,
	done <-chan struct{},
) <-chan parse.IImage {
	if paths == nil {
		return nil
	}

	var (
		waitGroup sync.WaitGroup
		images    = make(chan parse.IImage)
	)

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		var (
			kindPathsWaitGroup sync.WaitGroup
			kindPaths          = map[kind.Kind]chan collect.IPath{}
		)

		for kind := range i.parsers {
			kindPaths[kind] = make(chan collect.IPath)
		}

		for path := range paths {
			path := path

			kindPathsWaitGroup.Add(1)

			go func() {
				defer kindPathsWaitGroup.Done()

				if path.Err() != nil {
					select {
					case <-done:
					case images <- parse.NewImage(
						path.Kind(), "", "", "", nil, path.Err(),
					):
					}

					return
				}

				if _, ok := kindPaths[path.Kind()]; !ok {
					select {
					case <-done:
					case images <- parse.NewImage(
						path.Kind(), "", "", "", nil,
						fmt.Errorf(
							"kind '%s' does not have a parser defined",
							path.Kind(),
						),
					):
					}

					return
				}

				select {
				case <-done:
				case kindPaths[path.Kind()] <- path:
				}
			}()
		}

		go func() {
			kindPathsWaitGroup.Wait()

			for _, paths := range kindPaths {
				close(paths)
			}
		}()

		for kind, paths := range kindPaths {
			kind := kind
			paths := paths

			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				for image := range i.parsers[kind].ParseFiles(paths, done) {
					select {
					case <-done:
						return
					case images <- image:
					}

					if image.Err() != nil {
						return
					}
				}
			}()
		}
	}()

	go func() {
		waitGroup.Wait()
		close(images)
	}()

	return images
}
