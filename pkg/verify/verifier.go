// Package verify provides functionality for verifying that an existing
// Lockfile is up-to-date.
package verify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/safe-waters/docker-lock/pkg/generate"
)

// Verifier verifies that the Lockfile is the same as one that would
// be generated if a new one were generated.
type Verifier struct {
	Generator generate.IGenerator
}

// IVerifier provides an interface for Verifiers's exported methods.
type IVerifier interface {
	VerifyLockfile(reader io.Reader) error
}

// NewVerifier returns a Verifier after validating its fields.
func NewVerifier(
	generator generate.IGenerator,
) (*Verifier, error) {
	if generator == nil || reflect.ValueOf(generator).IsNil() {
		return nil, errors.New("generator cannot be nil")
	}

	return &Verifier{Generator: generator}, nil
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

	if !reflect.DeepEqual(existingLockfile, newLockfile) {
		existingPrettyLockfile, err := jsonPrettyPrint(&existingLockfile)
		if err != nil {
			return err
		}

		newPrettyLockfile, err := jsonPrettyPrint(&newLockfile)
		if err != nil {
			return err
		}

		return fmt.Errorf(
			"got:\n%s\nwant:\n%s",
			newPrettyLockfile,
			existingPrettyLockfile,
		)
	}

	return nil
}

func jsonPrettyPrint(lockfile *generate.Lockfile) (string, error) {
	byt, err := json.MarshalIndent(lockfile, "", "\t")
	if err != nil {
		return "", err
	}

	return string(byt), nil
}
