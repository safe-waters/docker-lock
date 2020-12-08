package verify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/kind"
	"github.com/safe-waters/docker-lock/pkg/verify/diff"
)

type verifier struct {
	generator       generate.IGenerator
	differentiators map[kind.Kind]diff.IImageDifferentiator
}

// NewVerifier returns a Verifier after ensuring that generator is non nil,
// and there is at least one non nil differentiator.
func NewVerifier(
	generator generate.IGenerator,
	differentiators ...diff.IImageDifferentiator,
) (IVerifier, error) {
	if generator == nil || reflect.ValueOf(generator).IsNil() {
		return nil, errors.New("'generator' cannot be nil")
	}

	kindDifferentiator := map[kind.Kind]diff.IImageDifferentiator{}

	for _, differentiator := range differentiators {
		if differentiator != nil && !reflect.ValueOf(differentiator).IsNil() {
			kindDifferentiator[differentiator.Kind()] = differentiator
		}
	}

	if len(kindDifferentiator) == 0 {
		return nil, errors.New(
			"non nil 'differentiators' must be greater than 0",
		)
	}

	return &verifier{
		generator:       generator,
		differentiators: kindDifferentiator,
	}, nil
}

// VerifyLockfile reads an existing Lockfile and generates a new one
// for the specified paths. If it is different, the differences are
// returned as an error.
func (v *verifier) VerifyLockfile(lockfileReader io.Reader) error {
	if lockfileReader == nil || reflect.ValueOf(lockfileReader).IsNil() {
		return errors.New("'lockfileReader' cannot be nil")
	}

	existingLockfileByt, err := ioutil.ReadAll(lockfileReader)
	if err != nil {
		return err
	}

	var existingLockfile map[kind.Kind]map[string][]interface{}
	if err := json.Unmarshal(
		existingLockfileByt, &existingLockfile,
	); err != nil {
		return err
	}

	var newLockfileBytBuffer bytes.Buffer
	if err := v.generator.GenerateLockfile(&newLockfileBytBuffer); err != nil {
		return err
	}

	newLockfileByt := newLockfileBytBuffer.Bytes()

	var newLockfile map[kind.Kind]map[string][]interface{}
	if err := json.Unmarshal(newLockfileByt, &newLockfile); err != nil {
		return err
	}

	if len(existingLockfile) != len(newLockfile) {
		return &differentLockfileError{
			existingLockfile: existingLockfileByt,
			newLockfile:      newLockfileByt,
			err: fmt.Errorf(
				"existing has '%d' kind(s), but new has '%d'",
				len(existingLockfile), len(newLockfile),
			),
		}
	}

	var (
		waitGroup sync.WaitGroup
		errCh     = make(chan error)
		done      = make(chan struct{})
	)

	defer close(done)

	for kind := range existingLockfile {
		kind := kind

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			if _, ok := newLockfile[kind]; !ok {
				select {
				case <-done:
				case errCh <- fmt.Errorf(
					"existing kind '%s' does not exist in new", kind,
				):
				}

				return
			}

			if len(existingLockfile[kind]) != len(newLockfile[kind]) {
				select {
				case <-done:
				case errCh <- fmt.Errorf(
					"existing kind '%s' has '%d' paths, but new has '%d'",
					kind, len(existingLockfile[kind]), len(newLockfile[kind]),
				):
				}

				return
			}

			for path, existingImages := range existingLockfile[kind] {
				path := path
				existingImages := existingImages

				waitGroup.Add(1)

				go func() {
					defer waitGroup.Done()

					newImages, ok := newLockfile[kind][path]
					if !ok {
						select {
						case <-done:
						case errCh <- fmt.Errorf(
							"existing path '%s' does not exist in new", path,
						):
						}

						return
					}

					if len(existingImages) != len(newImages) {
						select {
						case <-done:
						case errCh <- fmt.Errorf(
							"existing path '%s' has '%d' images but "+
								"new has '%d'",
							path, len(existingImages), len(newImages),
						):
						}

						return
					}

					for i := range existingImages {
						i := i

						waitGroup.Add(1)

						go func() {
							defer waitGroup.Done()

							existingImage, ok := existingImages[i].(map[string]interface{}) // nolint: lll
							if !ok {
								select {
								case <-done:
								case errCh <- fmt.Errorf(
									"path '%s' has malformed existing "+
										"image '%v'",
									path, existingImages[i],
								):
								}

								return
							}

							newImage, ok := newImages[i].(map[string]interface{}) // nolint: lll
							if !ok {
								select {
								case <-done:
								case errCh <- fmt.Errorf(
									"path '%s' has malformed new image '%v'",
									path, existingImages[i],
								):
								}

								return
							}

							differentiator, ok := v.differentiators[kind]
							if !ok {
								select {
								case <-done:
								case errCh <- fmt.Errorf(
									"kind %s does not have a"+
										" differentiator defined",
									kind,
								):
								}

								return
							}

							if err := differentiator.DifferentiateImage(
								existingImage, newImage,
							); err != nil {
								select {
								case <-done:
								case errCh <- fmt.Errorf(
									"on path '%s', %s", path, err,
								):
								}

								return
							}
						}()
					}
				}()
			}
		}()
	}

	go func() {
		waitGroup.Wait()
		close(errCh)
	}()

	for err := range errCh {
		return &differentLockfileError{
			existingLockfile: existingLockfileByt,
			newLockfile:      newLockfileByt,
			err:              err,
		}
	}

	return nil
}
