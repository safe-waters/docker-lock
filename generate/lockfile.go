package generate

import (
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"reflect"
	"sort"
	"sync"

	"github.com/safe-waters/docker-lock/generate/parse"
)

// Lockfile represents the canonical 'docker-lock.json'. It provides
// the capability to write its contents in JSON format.
type Lockfile struct {
	DockerfileImages  map[string][]*parse.DockerfileImage  `json:"dockerfiles,omitempty"`  // nolint: lll
	ComposefileImages map[string][]*parse.ComposefileImage `json:"composefiles,omitempty"` // nolint: lll
}

// NewLockfile sorts DockerfileImages and Composefile images and
// returns a Lockfile.
func NewLockfile(
	dockerfileImages <-chan *parse.DockerfileImage,
	composefileImages <-chan *parse.ComposefileImage,
	done <-chan struct{},
) (*Lockfile, error) {
	if dockerfileImages == nil && composefileImages == nil {
		return &Lockfile{}, nil
	}

	var dockerfileImagesWithPath map[string][]*parse.DockerfileImage

	var composefileImagesWithPath map[string][]*parse.ComposefileImage

	errCh := make(chan error)

	var waitGroup sync.WaitGroup

	if dockerfileImages != nil {
		dockerfileImagesWithPath = map[string][]*parse.DockerfileImage{}

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			for dockerfileImage := range dockerfileImages {
				if dockerfileImage.Err != nil {
					select {
					case <-done:
					case errCh <- dockerfileImage.Err:
					}

					return
				}

				dockerfileImage.Path = filepath.ToSlash(dockerfileImage.Path)

				dockerfileImagesWithPath[dockerfileImage.Path] = append(
					dockerfileImagesWithPath[dockerfileImage.Path],
					dockerfileImage,
				)
			}
		}()
	}

	if composefileImages != nil {
		composefileImagesWithPath = map[string][]*parse.ComposefileImage{}

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			for composefileImage := range composefileImages {
				if composefileImage.Err != nil {
					select {
					case <-done:
					case errCh <- composefileImage.Err:
					}

					return
				}

				composefileImage.Path = filepath.ToSlash(composefileImage.Path)
				composefileImage.DockerfilePath = filepath.ToSlash(
					composefileImage.DockerfilePath,
				)

				composefileImagesWithPath[composefileImage.Path] = append(
					composefileImagesWithPath[composefileImage.Path],
					composefileImage,
				)
			}
		}()
	}

	go func() {
		waitGroup.Wait()
		close(errCh)
	}()

	for err := range errCh {
		return nil, err
	}

	lockfile := &Lockfile{
		DockerfileImages:  dockerfileImagesWithPath,
		ComposefileImages: composefileImagesWithPath,
	}
	lockfile.sortImages()

	return lockfile, nil
}

// Write writes the Lockfile in JSON format to an io.Writer.
func (l *Lockfile) Write(writer io.Writer) error {
	if writer == nil || reflect.ValueOf(writer).IsNil() {
		return errors.New("writer cannot be nil")
	}

	lByt, err := json.MarshalIndent(l, "", "\t")
	if err != nil {
		return err
	}

	if _, err := writer.Write(lByt); err != nil {
		return err
	}

	return nil
}

func (l *Lockfile) sortImages() {
	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go l.sortDockerfileImages(&waitGroup)

	waitGroup.Add(1)

	go l.sortComposefileImages(&waitGroup)

	waitGroup.Wait()
}

func (l *Lockfile) sortDockerfileImages(waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	for _, images := range l.DockerfileImages {
		waitGroup.Add(1)

		go func(images []*parse.DockerfileImage) {
			defer waitGroup.Done()

			sort.Slice(images, func(i, j int) bool {
				return images[i].Position < images[j].Position
			})
		}(images)
	}
}

func (l *Lockfile) sortComposefileImages(waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	for _, images := range l.ComposefileImages {
		waitGroup.Add(1)

		go func(images []*parse.ComposefileImage) {
			defer waitGroup.Done()

			sort.Slice(images, func(i, j int) bool {
				switch {
				case images[i].ServiceName != images[j].ServiceName:
					return images[i].ServiceName < images[j].ServiceName
				case images[i].DockerfilePath != images[j].DockerfilePath:
					return images[i].DockerfilePath < images[j].DockerfilePath
				default:
					return images[i].Position < images[j].Position
				}
			})
		}(images)
	}
}
