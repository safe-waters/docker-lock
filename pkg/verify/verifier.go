// Package verify provides functionality for verifying that an existing
// Lockfile is up-to-date.
package verify

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"

	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/verify/diff"
)

// Verifier verifies that the Lockfile is the same as one that would
// be generated if a new one were generated.
type Verifier struct {
	Generator           generate.IGenerator
	DockerfileVerifier  diff.IDockerfileDifferentiator
	ComposefileVerifier diff.IComposefileDifferentiator
}

// IVerifier provides an interface for Verifiers's exported methods.
type IVerifier interface {
	VerifyLockfile(reader io.Reader) error
}

// NewVerifier returns a Verifier after validating its fields.
func NewVerifier(
	generator generate.IGenerator,
	dockerfileVerifier diff.IDockerfileDifferentiator,
	composefileVerifier diff.IComposefileDifferentiator,
) (*Verifier, error) {
	if generator == nil || reflect.ValueOf(generator).IsNil() {
		return nil, errors.New("generator cannot be nil")
	}

	return &Verifier{
		Generator:           generator,
		DockerfileVerifier:  dockerfileVerifier,
		ComposefileVerifier: composefileVerifier,
	}, nil
}

// VerifyLockfile reads an existing Lockfile and generates a new one
// for the specified paths. If it is different, the differences are
// returned as an error.
func (v *Verifier) VerifyLockfile(reader io.Reader) error {
	if reader == nil || reflect.ValueOf(reader).IsNil() {
		return errors.New("reader cannot be nil")
	}

	if (v.DockerfileVerifier == nil ||
		reflect.ValueOf(v.DockerfileVerifier).IsNil()) &&
		(v.ComposefileVerifier == nil ||
			reflect.ValueOf(v.ComposefileVerifier).IsNil()) {
		return nil
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

	done := make(chan struct{})
	defer close(done)

	var dockerfileErrCh <-chan error

	var composefileErrCh <-chan error

	if v.DockerfileVerifier != nil &&
		!reflect.ValueOf(v.DockerfileVerifier).IsNil() {
		dockerfileErrCh = v.DockerfileVerifier.Differentiate(
			existingLockfile.DockerfileImages, newLockfile.DockerfileImages,
			done,
		)
	}

	if v.ComposefileVerifier != nil &&
		!reflect.ValueOf(v.ComposefileVerifier).IsNil() {
		composefileErrCh = v.ComposefileVerifier.Differentiate(
			existingLockfile.ComposefileImages, newLockfile.ComposefileImages,
			done,
		)
	}

	for {
		select {
		case <-done:
			return nil
		case _, ok := <-dockerfileErrCh:
			if !ok {
				dockerfileErrCh = nil
				break
			}

			return &DifferentLockfileError{
				ExistingLockfile: &existingLockfile,
				NewLockfile:      &newLockfile,
			}
		case _, ok := <-composefileErrCh:
			if !ok {
				composefileErrCh = nil
				break
			}

			return &DifferentLockfileError{
				ExistingLockfile: &existingLockfile,
				NewLockfile:      &newLockfile,
			}
		}

		if dockerfileErrCh == nil && composefileErrCh == nil {
			return nil
		}
	}
}
