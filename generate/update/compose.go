package update

import (
	"errors"
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/generate/parse"
)

// ComposefileImageDigestUpdater replaces image digests with the most
// recent ones from their registries.
type ComposefileImageDigestUpdater struct {
	QueryExecutor IQueryExecutor
}

// NewComposefileImageDigestUpdater returns a ComposefileImageDigestUpdater
// after validating its fields.
func NewComposefileImageDigestUpdater(
	queryExecutor IQueryExecutor,
) (*ComposefileImageDigestUpdater, error) {
	if queryExecutor == nil || reflect.ValueOf(queryExecutor).IsNil() {
		return nil, errors.New("queryExecutor cannot be nil")
	}

	return &ComposefileImageDigestUpdater{QueryExecutor: queryExecutor}, nil
}

// UpdateDigests queries registries for digests of images that do not
// already specify their digests. It updates images with those
// digests.
func (c *ComposefileImageDigestUpdater) UpdateDigests(
	composefileImages <-chan *parse.ComposefileImage,
	done <-chan struct{},
) <-chan *parse.ComposefileImage {
	updatedComposefileImages := make(chan *parse.ComposefileImage)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for composefileImage := range composefileImages {
			composefileImage := composefileImage

			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				if composefileImage.Err != nil {
					select {
					case <-done:
					case updatedComposefileImages <- composefileImage:
					}

					return
				}

				if composefileImage.Digest != "" {
					select {
					case <-done:
					case updatedComposefileImages <- composefileImage:
					}

					return
				}

				queryResult := c.QueryExecutor.QueryRegistry(
					composefileImage.Image,
				)

				if queryResult.Err != nil {
					select {
					case <-done:
					case updatedComposefileImages <- &parse.ComposefileImage{
						Err: queryResult.Err,
					}:
					}

					return
				}

				select {
				case <-done:
				case updatedComposefileImages <- &parse.ComposefileImage{
					Image:          queryResult.Image,
					DockerfilePath: composefileImage.DockerfilePath,
					Position:       composefileImage.Position,
					ServiceName:    composefileImage.ServiceName,
					Path:           composefileImage.Path,
				}:
				}
			}()
		}
	}()

	go func() {
		waitGroup.Wait()
		close(updatedComposefileImages)
	}()

	return updatedComposefileImages
}
