// Package update provides functionality to update images with digests.
package update

import (
	"errors"
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/generate/parse"
)

// DockerfileImageDigestUpdater replaces image digests
// with the most recent ones from their registries.
type DockerfileImageDigestUpdater struct {
	QueryExecutor IQueryExecutor
}

// NewDockerfileImageDigestUpdater returns a DockerfileImageDigestUpdater
// after validating its fields.
func NewDockerfileImageDigestUpdater(
	queryExecutor IQueryExecutor,
) (*DockerfileImageDigestUpdater, error) {
	if queryExecutor == nil || reflect.ValueOf(queryExecutor).IsNil() {
		return nil, errors.New("queryExecutor cannot be nil")
	}

	return &DockerfileImageDigestUpdater{QueryExecutor: queryExecutor}, nil
}

// UpdateDigests queries registries for digests of images that do not
// already specify their digests. It updates images with those
// digests.
func (d *DockerfileImageDigestUpdater) UpdateDigests(
	dockerfileImages <-chan *parse.DockerfileImage,
	done <-chan struct{},
) <-chan *parse.DockerfileImage {
	updatedDockerfileImages := make(chan *parse.DockerfileImage)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for dockerfileImage := range dockerfileImages {
			dockerfileImage := dockerfileImage

			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				if dockerfileImage.Err != nil {
					select {
					case <-done:
					case updatedDockerfileImages <- dockerfileImage:
					}

					return
				}

				if dockerfileImage.Digest != "" {
					select {
					case <-done:
					case updatedDockerfileImages <- dockerfileImage:
					}

					return
				}

				queryResult := d.QueryExecutor.QueryRegistry(
					dockerfileImage.Image,
				)

				if queryResult.Err != nil {
					select {
					case <-done:
					case updatedDockerfileImages <- &parse.DockerfileImage{
						Err: queryResult.Err,
					}:
					}

					return
				}

				select {
				case <-done:
				case updatedDockerfileImages <- &parse.DockerfileImage{
					Image:    queryResult.Image,
					Position: dockerfileImage.Position,
					Path:     dockerfileImage.Path,
				}:
				}
			}()
		}
	}()

	go func() {
		waitGroup.Wait()
		close(updatedDockerfileImages)
	}()

	return updatedDockerfileImages
}
