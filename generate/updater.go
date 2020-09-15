package generate

import (
	"errors"

	"github.com/safe-waters/docker-lock/generate/parse"
	"github.com/safe-waters/docker-lock/generate/update"
	"github.com/safe-waters/docker-lock/registry"
)

// ImageDigestUpdater contains ImageDigestUpdaters for
// DockerfileImages and ComposefileImages.
type ImageDigestUpdater struct {
	DockerImageUpdater  *update.DockerfileImageDigestUpdater
	ComposeImageUpdater *update.ComposefileImageDigestUpdater
}

// IImageDigestUpdater provides an interface for ImageDigestUpdater's exported
// methods, which are used by Generator.
type IImageDigestUpdater interface {
	UpdateDigests(
		dockerfileImages <-chan *parse.DockerfileImage,
		composefileImages <-chan *parse.ComposefileImage,
		done <-chan struct{},
	) (
		updatedDockerfileImages <-chan *parse.DockerfileImage,
		updatedComposefileImages <-chan *parse.ComposefileImage,
	)
}

// UpdateDigests updates digests for DockerfileImages and ComposefileImages.
func (i *ImageDigestUpdater) UpdateDigests(
	dockerfileImages <-chan *parse.DockerfileImage,
	composefileImages <-chan *parse.ComposefileImage,
	done <-chan struct{},
) (
	updatedDockerfileImages <-chan *parse.DockerfileImage,
	updatedComposefileImages <-chan *parse.ComposefileImage,
) {
	if i.DockerImageUpdater != nil {
		updatedDockerfileImages = i.DockerImageUpdater.UpdateDigests(
			dockerfileImages, done,
		)
	}

	if i.ComposeImageUpdater != nil {
		updatedComposefileImages = i.ComposeImageUpdater.UpdateDigests(
			composefileImages, done,
		)
	}

	return
}

// DefaultImageDigestUpdater creates an ImageDigestUpdater for Generator.
func DefaultImageDigestUpdater(
	wrapperManager *registry.WrapperManager,
) (IImageDigestUpdater, error) {
	if wrapperManager == nil {
		return nil, errors.New("wrapperManager cannot be nil")
	}

	queryExecutor, err := update.NewQueryExecutor(wrapperManager)
	if err != nil {
		return nil, err
	}

	dockerfileImageUpdater := &update.DockerfileImageDigestUpdater{
		QueryExecutor: queryExecutor,
	}

	composefileImageUpdater := &update.ComposefileImageDigestUpdater{
		QueryExecutor: queryExecutor,
	}

	return &ImageDigestUpdater{
		DockerImageUpdater:  dockerfileImageUpdater,
		ComposeImageUpdater: composefileImageUpdater,
	}, nil
}
