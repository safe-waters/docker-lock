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

	if v.ExcludeTags {
		if len(existingLockfile.DockerfileImages) != 0 {
			existingLockfile.DockerfileImages = v.filterDockerfileImageTags(
				existingLockfile.DockerfileImages,
			)
		}

		if len(existingLockfile.ComposefileImages) != 0 {
			existingLockfile.ComposefileImages = v.filterComposefileImageTags(
				existingLockfile.ComposefileImages,
			)
		}

		if len(newLockfile.DockerfileImages) != 0 {
			newLockfile.DockerfileImages = v.filterDockerfileImageTags(
				newLockfile.DockerfileImages,
			)
		}

		if len(newLockfile.ComposefileImages) != 0 {
			newLockfile.ComposefileImages = v.filterComposefileImageTags(
				newLockfile.ComposefileImages,
			)
		}
	}

	if !reflect.DeepEqual(existingLockfile, newLockfile) {
		return &DifferentLockfileError{
			ExistingLockfile: &existingLockfile,
			NewLockfile:      &newLockfile,
		}
	}

	return nil
}

func (*Verifier) filterDockerfileImageTags(
	pathImages map[string][]*parse.DockerfileImage,
) map[string][]*parse.DockerfileImage {
	filteredPathImages := map[string][]*parse.DockerfileImage{}

	for path, images := range pathImages {
		filteredImages := make([]*parse.DockerfileImage, len(images))

		for i, image := range images {
			filteredImage := &parse.Image{
				Name:   image.Name,
				Digest: image.Digest,
			}

			filteredDockerfileImage := &parse.DockerfileImage{
				Image: filteredImage,
			}

			filteredImages[i] = filteredDockerfileImage
		}

		filteredPathImages[path] = filteredImages
	}

	return filteredPathImages
}

func (*Verifier) filterComposefileImageTags(
	pathImages map[string][]*parse.ComposefileImage,
) map[string][]*parse.ComposefileImage {
	filteredPathImages := map[string][]*parse.ComposefileImage{}

	for path, images := range pathImages {
		filteredImages := make([]*parse.ComposefileImage, len(images))

		for i, image := range images {
			filteredImage := &parse.Image{
				Name:   image.Name,
				Digest: image.Digest,
			}

			filteredComposefileImage := &parse.ComposefileImage{
				Image:          filteredImage,
				ServiceName:    image.ServiceName,
				DockerfilePath: image.DockerfilePath,
			}

			filteredImages[i] = filteredComposefileImage
		}

		filteredPathImages[path] = filteredImages
	}

	return filteredPathImages
}
