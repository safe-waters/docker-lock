package generate

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
)

// ImageParser contains ImageParsers for Dockerfiles and docker-compose files.
type ImageParser struct {
	DockerfileImageParser  parse.IDockerfileImageParser
	ComposefileImageParser parse.IComposefileImageParser
}

// IImageParser provides an interface for Parser's exported methods,
// which are used by Generator.
type IImageParser interface {
	ParseFiles(anyPaths <-chan *AnyPath, done <-chan struct{}) <-chan *AnyImage
}

// AnyImage contains any possible type of parser.
type AnyImage struct {
	DockerfileImage  *parse.DockerfileImage
	ComposefileImage *parse.ComposefileImage
	Err              error
}

// ParseFiles parses Dockerfiles and docker-compose files for Images.
func (i *ImageParser) ParseFiles(
	anyPaths <-chan *AnyPath,
	done <-chan struct{},
) <-chan *AnyImage {
	if ((i.DockerfileImageParser == nil ||
		reflect.ValueOf(i.DockerfileImageParser).IsNil()) &&
		(i.ComposefileImageParser == nil ||
			reflect.ValueOf(i.ComposefileImageParser).IsNil())) ||
		anyPaths == nil {
		return nil
	}

	anyImages := make(chan *AnyImage)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		dockerfilePaths := make(chan string)
		composefilePaths := make(chan string)

		var pathsWaitGroup sync.WaitGroup

		pathsWaitGroup.Add(1)

		go func() {
			defer pathsWaitGroup.Done()

			for anyPath := range anyPaths {
				if anyPath.Err != nil {
					select {
					case <-done:
					case anyImages <- &AnyImage{Err: anyPath.Err}:
					}

					return
				}

				switch {
				case anyPath.DockerfilePath != "":
					if i.DockerfileImageParser == nil ||
						reflect.ValueOf(i.DockerfileImageParser).IsNil() {
						select {
						case <-done:
						case anyImages <- &AnyImage{
							Err: fmt.Errorf(
								"dockerfile %s found, but its parser is nil",
								anyPath.DockerfilePath,
							),
						}:
						}

						return
					}

					select {
					case <-done:
						return
					case dockerfilePaths <- anyPath.DockerfilePath:
					}
				case anyPath.ComposefilePath != "":
					if i.ComposefileImageParser == nil ||
						reflect.ValueOf(i.ComposefileImageParser).IsNil() {
						select {
						case <-done:
						case anyImages <- &AnyImage{
							Err: fmt.Errorf(
								"composefile %s found, but its parser is nil",
								anyPath.ComposefilePath,
							),
						}:
						}

						return
					}

					select {
					case <-done:
						return
					case composefilePaths <- anyPath.ComposefilePath:
					}
				}
			}
		}()

		go func() {
			pathsWaitGroup.Wait()

			close(dockerfilePaths)
			close(composefilePaths)
		}()

		var dockerfileImages <-chan *parse.DockerfileImage

		var composefileImages <-chan *parse.ComposefileImage

		if i.DockerfileImageParser != nil &&
			!reflect.ValueOf(i.DockerfileImageParser).IsNil() {
			dockerfileImages = i.DockerfileImageParser.ParseFiles(
				dockerfilePaths, done,
			)
		}

		if i.ComposefileImageParser != nil &&
			!reflect.ValueOf(i.ComposefileImageParser).IsNil() {
			composefileImages = i.ComposefileImageParser.ParseFiles(
				composefilePaths, done,
			)
		}

		if dockerfileImages != nil {
			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				for dockerfileImage := range dockerfileImages {
					if dockerfileImage.Err != nil {
						select {
						case <-done:
						case anyImages <- &AnyImage{Err: dockerfileImage.Err}:
						}

						return
					}

					select {
					case <-done:
						return
					case anyImages <- &AnyImage{
						DockerfileImage: dockerfileImage,
					}:
					}
				}
			}()
		}

		if composefileImages != nil {
			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				for composefileImage := range composefileImages {
					if composefileImage.Err != nil {
						select {
						case <-done:
						case anyImages <- &AnyImage{Err: composefileImage.Err}:
						}

						return
					}

					select {
					case <-done:
						return
					case anyImages <- &AnyImage{
						ComposefileImage: composefileImage,
					}:
					}
				}
			}()
		}
	}()

	go func() {
		waitGroup.Wait()
		close(anyImages)
	}()

	return anyImages
}
