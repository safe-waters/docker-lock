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
	Generator                    generate.IGenerator
	DockerfileDifferentiator     diff.IDockerfileDifferentiator
	ComposefileDifferentiator    diff.IComposefileDifferentiator
	KubernetesfileDifferentiator diff.IKubernetesfileDifferentiator
}

// IVerifier provides an interface for Verifiers's exported methods.
type IVerifier interface {
	VerifyLockfile(reader io.Reader) error
}

// NewVerifier returns a Verifier after validating its fields.
func NewVerifier(
	generator generate.IGenerator,
	dockerfileDifferentiator diff.IDockerfileDifferentiator,
	composefileDifferentiator diff.IComposefileDifferentiator,
	kubernetesfileDifferentiator diff.IKubernetesfileDifferentiator,
) (*Verifier, error) {
	if generator == nil || reflect.ValueOf(generator).IsNil() {
		return nil, errors.New("generator cannot be nil")
	}

	return &Verifier{
		Generator:                    generator,
		DockerfileDifferentiator:     dockerfileDifferentiator,
		ComposefileDifferentiator:    composefileDifferentiator,
		KubernetesfileDifferentiator: kubernetesfileDifferentiator,
	}, nil
}

// VerifyLockfile reads an existing Lockfile and generates a new one
// for the specified paths. If it is different, the differences are
// returned as an error.
func (v *Verifier) VerifyLockfile(reader io.Reader) error {
	if reader == nil || reflect.ValueOf(reader).IsNil() {
		return errors.New("reader cannot be nil")
	}

	if (v.DockerfileDifferentiator == nil ||
		reflect.ValueOf(v.DockerfileDifferentiator).IsNil()) &&
		(v.ComposefileDifferentiator == nil ||
			reflect.ValueOf(v.ComposefileDifferentiator).IsNil()) &&
		(v.KubernetesfileDifferentiator == nil ||
			reflect.ValueOf(v.KubernetesfileDifferentiator).IsNil()) {
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

	var kubernetesfileErrCh <-chan error

	if v.DockerfileDifferentiator != nil &&
		!reflect.ValueOf(v.DockerfileDifferentiator).IsNil() {
		dockerfileErrCh = v.DockerfileDifferentiator.Differentiate(
			existingLockfile.DockerfileImages, newLockfile.DockerfileImages,
			done,
		)
	}

	if v.ComposefileDifferentiator != nil &&
		!reflect.ValueOf(v.ComposefileDifferentiator).IsNil() {
		composefileErrCh = v.ComposefileDifferentiator.Differentiate(
			existingLockfile.ComposefileImages, newLockfile.ComposefileImages,
			done,
		)
	}

	if v.KubernetesfileDifferentiator != nil &&
		!reflect.ValueOf(v.KubernetesfileDifferentiator).IsNil() {
		kubernetesfileErrCh = v.KubernetesfileDifferentiator.Differentiate(
			existingLockfile.KubernetesfileImages,
			newLockfile.KubernetesfileImages,
			done,
		)
	}

	for {
		select {
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
		case _, ok := <-kubernetesfileErrCh:
			if !ok {
				kubernetesfileErrCh = nil
				break
			}

			return &DifferentLockfileError{
				ExistingLockfile: &existingLockfile,
				NewLockfile:      &newLockfile,
			}
		}

		if dockerfileErrCh == nil &&
			composefileErrCh == nil &&
			kubernetesfileErrCh == nil {
			return nil
		}
	}
}
