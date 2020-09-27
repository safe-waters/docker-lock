package generate

import (
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"reflect"
	"sort"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
)

// Lockfile represents the canonical 'docker-lock.json'. It provides
// the capability to write its contents in JSON format.
type Lockfile struct {
	DockerfileImages  map[string][]*parse.DockerfileImage  `json:"dockerfiles,omitempty"`  // nolint: lll
	ComposefileImages map[string][]*parse.ComposefileImage `json:"composefiles,omitempty"` // nolint: lll
}

// NewLockfile sorts DockerfileImages and Composefile images and
// returns a Lockfile.
func NewLockfile(anyImages <-chan *AnyImage) (*Lockfile, error) {
	if anyImages == nil {
		return &Lockfile{}, nil
	}

	var dockerfileImages map[string][]*parse.DockerfileImage

	var composefileImages map[string][]*parse.ComposefileImage

	for anyImage := range anyImages {
		if anyImage.Err != nil {
			return nil, anyImage.Err
		}

		switch {
		case anyImage.DockerfileImage != nil:
			if dockerfileImages == nil {
				dockerfileImages = map[string][]*parse.DockerfileImage{}
			}

			anyImage.DockerfileImage.Path = filepath.ToSlash(
				anyImage.DockerfileImage.Path,
			)

			dockerfileImages[anyImage.DockerfileImage.Path] = append(
				dockerfileImages[anyImage.DockerfileImage.Path],
				anyImage.DockerfileImage,
			)
		case anyImage.ComposefileImage != nil:
			if composefileImages == nil {
				composefileImages = map[string][]*parse.ComposefileImage{}
			}

			anyImage.ComposefileImage.Path = filepath.ToSlash(
				anyImage.ComposefileImage.Path,
			)
			anyImage.ComposefileImage.DockerfilePath = filepath.ToSlash(
				anyImage.ComposefileImage.DockerfilePath,
			)

			composefileImages[anyImage.ComposefileImage.Path] = append(
				composefileImages[anyImage.ComposefileImage.Path],
				anyImage.ComposefileImage,
			)
		}
	}

	lockfile := &Lockfile{
		DockerfileImages:  dockerfileImages,
		ComposefileImages: composefileImages,
	}

	lockfile.sortImages()

	return lockfile, nil
}

// Write writes the Lockfile in JSON format to an io.Writer.
func (l *Lockfile) Write(writer io.Writer) error {
	if writer == nil || reflect.ValueOf(writer).IsNil() {
		return errors.New("writer cannot be nil")
	}

	lockfileByt, err := json.MarshalIndent(l, "", "\t")
	if err != nil {
		return err
	}

	if _, err := writer.Write(lockfileByt); err != nil {
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
		images := images

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			sort.Slice(images, func(i, j int) bool {
				return images[i].Position < images[j].Position
			})
		}()
	}
}

func (l *Lockfile) sortComposefileImages(waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	for _, images := range l.ComposefileImages {
		images := images

		waitGroup.Add(1)

		go func() {
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
		}()
	}
}
