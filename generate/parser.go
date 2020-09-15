package generate

import (
	"github.com/safe-waters/docker-lock/generate/collect"
	"github.com/safe-waters/docker-lock/generate/parse"
)

// ImageParser contains ImageParsers for Dockerfiles and docker-compose files.
type ImageParser struct {
	DockerfileImageParser  *parse.DockerfileImageParser
	ComposefileImageParser *parse.ComposefileImageParser
}

// IImageParser provides an interface for Parser's exported methods,
// which are used by Generator.
type IImageParser interface {
	ParseFiles(
		dockerfilePaths <-chan *collect.PathResult,
		composefilePaths <-chan *collect.PathResult,
		done <-chan struct{},
	) (
		dockerfileImages <-chan *parse.DockerfileImage,
		composefileImages <-chan *parse.ComposefileImage,
	)
}

// ParseFiles parses Dockerfiles and docker-compose files for Images.
func (i *ImageParser) ParseFiles(
	dockerfilePaths <-chan *collect.PathResult,
	composefilePaths <-chan *collect.PathResult,
	done <-chan struct{},
) (
	dockerfileImages <-chan *parse.DockerfileImage,
	composefileImages <-chan *parse.ComposefileImage,
) {
	if i.DockerfileImageParser != nil {
		dockerfileImages = i.DockerfileImageParser.ParseFiles(
			dockerfilePaths, done,
		)
	}

	if i.ComposefileImageParser != nil {
		composefileImages = i.ComposefileImageParser.ParseFiles(
			composefilePaths, done,
		)
	}

	return
}

// DefaultImageParser create an ImageParser for Generator.
func DefaultImageParser(flags *Flags) IImageParser {
	return &ImageParser{
		DockerfileImageParser:  &parse.DockerfileImageParser{},
		ComposefileImageParser: &parse.ComposefileImageParser{},
	}
}
