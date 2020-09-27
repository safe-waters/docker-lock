// Package rewrite provides functionality to rewrite a Lockfile.
package rewrite

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
)

// Rewriter rewrites files referenced by a Lockfile with their image digests.
type Rewriter struct {
	Writer  IWriter
	Renamer IRenamer
}

type deduplicatedPath struct {
	path string
	err  error
}

// NewRewriter returns a Rewriter after validating its fields.
func NewRewriter(writer IWriter, renamer IRenamer) (*Rewriter, error) {
	if writer == nil || reflect.ValueOf(writer).IsNil() {
		return nil, errors.New("writer cannot be nil")
	}

	if renamer == nil || reflect.ValueOf(renamer).IsNil() {
		return nil, errors.New("renamer cannot be nil")
	}

	return &Rewriter{
		Writer:  writer,
		Renamer: renamer,
	}, nil
}

// RewriteLockfile rewrites files referenced by a Lockfile with their image
// digests.
func (r *Rewriter) RewriteLockfile(reader io.Reader) error {
	if reader == nil || reflect.ValueOf(reader).IsNil() {
		return errors.New("reader cannot be nil")
	}

	var lockfile generate.Lockfile
	if err := json.NewDecoder(reader).Decode(&lockfile); err != nil {
		return err
	}

	if len(lockfile.DockerfileImages) == 0 &&
		len(lockfile.ComposefileImages) == 0 {
		return nil
	}

	anyPathImages := &AnyPathImages{
		DockerfilePathImages:  lockfile.DockerfileImages,
		ComposefilePathImages: lockfile.ComposefileImages,
	}

	anyPathImages, err := r.deduplicateAnyPathImages(anyPathImages)
	if err != nil {
		return err
	}

	done := make(chan struct{})
	defer close(done)

	writtenPaths := r.Writer.WriteFiles(anyPathImages, done)

	return r.Renamer.RenameFiles(writtenPaths)
}

func (r *Rewriter) deduplicateAnyPathImages(
	anyPathImages *AnyPathImages,
) (*AnyPathImages, error) {
	if len(anyPathImages.ComposefilePathImages) == 0 ||
		len(anyPathImages.DockerfilePathImages) == 0 {
		return anyPathImages, nil
	}

	var waitGroup sync.WaitGroup

	done := make(chan struct{})
	defer close(done)

	deduplicatedDockerfilePaths := make(chan *deduplicatedPath)

	for _, images := range anyPathImages.ComposefilePathImages {
		images := images

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			for _, image := range images {
				if image.DockerfilePath != "" {
					dockerfilePath := image.DockerfilePath

					if filepath.IsAbs(dockerfilePath) {
						var err error

						dockerfilePath, err = r.convertAbsToRelPath(
							dockerfilePath,
						)
						if err != nil {
							select {
							case <-done:
							case deduplicatedDockerfilePaths <- &deduplicatedPath{ // nolint: lll
								err: err,
							}:
							}

							return
						}

						select {
						case <-done:
							return
						case deduplicatedDockerfilePaths <- &deduplicatedPath{
							path: dockerfilePath,
						}:
						}
					}
				}
			}
		}()
	}

	go func() {
		waitGroup.Wait()
		close(deduplicatedDockerfilePaths)
	}()

	dockerfilePathsCache := map[string]struct{}{}

	for deduplicatedDockerfilePath := range deduplicatedDockerfilePaths {
		if deduplicatedDockerfilePath.err != nil {
			return nil, deduplicatedDockerfilePath.err
		}

		dockerfilePathsCache[deduplicatedDockerfilePath.path] = struct{}{}
	}

	if len(dockerfilePathsCache) == 0 {
		return anyPathImages, nil
	}

	dockerfilePathImages := map[string][]*parse.DockerfileImage{}

	for path, images := range anyPathImages.DockerfilePathImages {
		if _, ok := dockerfilePathsCache[path]; !ok {
			dockerfilePathImages[path] = images
		}
	}

	return &AnyPathImages{
		DockerfilePathImages:  dockerfilePathImages,
		ComposefilePathImages: anyPathImages.ComposefilePathImages,
	}, nil
}

func (r *Rewriter) convertAbsToRelPath(path string) (string, error) {
	currentWorkingDirectory, err := os.Getwd()
	if err != nil {
		return "", err
	}

	relativePath, err := filepath.Rel(
		currentWorkingDirectory, filepath.FromSlash(path),
	)
	if err != nil {
		return "", err
	}

	return filepath.ToSlash(relativePath), nil
}
