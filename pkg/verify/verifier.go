// Package verify provides functionality for verifying that an existing
// Lockfile is up-to-date.
package verify

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
)

// Verifier verifies that the Lockfile is the same as one that would
// be generated if a new one were generated.
type Verifier struct {
	Generator   generate.IGenerator
	ExcludeTags bool
}

// IVerifier provides an interface for Verifiers's exported methods.
type IVerifier interface {
	VerifyLockfile(reader io.Reader) error
}

// NewVerifier returns a Verifier after validating its fields.
func NewVerifier(
	generator generate.IGenerator,
	excludeTags bool,
) (*Verifier, error) {
	if generator == nil || reflect.ValueOf(generator).IsNil() {
		return nil, errors.New("generator cannot be nil")
	}

	return &Verifier{Generator: generator, ExcludeTags: excludeTags}, nil
}

// VerifyLockfile reads an existing Lockfile and generates a new one
// for the specified paths. If it is different, the differences are
// returned as an error.
func (v *Verifier) VerifyLockfile(reader io.Reader) error {
	if reader == nil || reflect.ValueOf(reader).IsNil() {
		return errors.New("reader cannot be nil")
	}

	var existingLockfile generate.Lockfile
	if err := json.NewDecoder(reader).Decode(&existingLockfile); err != nil {
		return err
	}

	var newLockfileByt bytes.Buffer
	if err := v.Generator.GenerateLockfile(&newLockfileByt); err != nil {
		return err
	}

	var newLockfile generate.Lockfile
	if err := json.Unmarshal(newLockfileByt.Bytes(), &newLockfile); err != nil {
		return err
	}

	var waitGroup sync.WaitGroup

	errSignal := make(chan struct{})
	done := make(chan struct{})

	defer close(done)

	waitGroup.Add(1)

	go v.verifyDockerfileImages(
		existingLockfile.DockerfileImages, newLockfile.DockerfileImages,
		errSignal, done, &waitGroup,
	)

	waitGroup.Add(1)

	go v.verifyComposefileImages(
		existingLockfile.ComposefileImages, newLockfile.ComposefileImages,
		errSignal, done, &waitGroup,
	)

	go func() {
		waitGroup.Wait()
		close(errSignal)
	}()

	for range errSignal {
		return &DifferentLockfileError{
			ExistingLockfile: &existingLockfile,
			NewLockfile:      &newLockfile,
		}
	}

	return nil
}

func (v *Verifier) verifyDockerfileImages(
	existingPathImages map[string][]*parse.DockerfileImage,
	newPathImages map[string][]*parse.DockerfileImage,
	errSignal chan<- struct{},
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	if len(existingPathImages) != len(newPathImages) {
		select {
		case errSignal <- struct{}{}:
		case <-done:
		}

		return
	}

	for path, existingImages := range existingPathImages {
		path := path
		existingImages := existingImages

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			newImages, ok := newPathImages[path]
			if !ok {
				select {
				case errSignal <- struct{}{}:
				case <-done:
				}

				return
			}

			if len(existingImages) != len(newImages) {
				select {
				case errSignal <- struct{}{}:
				case <-done:
				}

				return
			}

			for i, existingImage := range existingImages {
				i := i
				existingImage := existingImage

				waitGroup.Add(1)

				go func() {
					defer waitGroup.Done()

					if existingImage == nil ||
						newImages[i] == nil ||
						existingImage.Image == nil ||
						newImages[i].Image == nil {
						select {
						case errSignal <- struct{}{}:
						case <-done:
						}

						return
					}

					if v.ExcludeTags {
						existingImage.Image.Tag = ""
						newImages[i].Image.Tag = ""
					}

					if *existingImage.Image != *newImages[i].Image {
						select {
						case errSignal <- struct{}{}:
						case <-done:
						}

						return
					}
				}()
			}
		}()
	}
}

func (v *Verifier) verifyComposefileImages(
	existingPathImages map[string][]*parse.ComposefileImage,
	newPathImages map[string][]*parse.ComposefileImage,
	errSignal chan<- struct{},
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	if len(existingPathImages) != len(newPathImages) {
		select {
		case errSignal <- struct{}{}:
		case <-done:
		}

		return
	}

	for path, existingImages := range existingPathImages {
		path := path
		existingImages := existingImages

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			newImages, ok := newPathImages[path]
			if !ok {
				select {
				case errSignal <- struct{}{}:
				case <-done:
				}

				return
			}

			if len(existingImages) != len(newImages) {
				select {
				case errSignal <- struct{}{}:
				case <-done:
				}

				return
			}

			for i, existingImage := range existingImages {
				i := i
				existingImage := existingImage

				waitGroup.Add(1)

				go func() {
					defer waitGroup.Done()

					if existingImage == nil ||
						newImages[i] == nil ||
						existingImage.Image == nil ||
						newImages[i].Image == nil {
						select {
						case errSignal <- struct{}{}:
						case <-done:
						}

						return
					}

					if v.ExcludeTags {
						existingImage.Image.Tag = ""
						newImages[i].Image.Tag = ""
					}

					if *existingImage.Image != *newImages[i].Image ||
						existingImage.ServiceName != newImages[i].ServiceName ||
						existingImage.DockerfilePath != newImages[i].DockerfilePath { // nolint: lll
						select {
						case errSignal <- struct{}{}:
						case <-done:
						}

						return
					}
				}()
			}
		}()
	}
}
