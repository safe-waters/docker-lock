package diff

import (
	"fmt"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
)

// IComposefileDifferentiator provides an interface for diffing Composefiles.
type IComposefileDifferentiator interface {
	Differentiate(
		existingPathImages map[string][]*parse.ComposefileImage,
		newPathImages map[string][]*parse.ComposefileImage,
		done <-chan struct{},
	) <-chan error
}

// ComposefileDifferentiator provides methods for diffing Composefile Path
// Images.
type ComposefileDifferentiator struct {
	ExcludeTags bool
}

// Differentiate diffs Composefile Path Images.
func (c *ComposefileDifferentiator) Differentiate(
	existingPathImages map[string][]*parse.ComposefileImage,
	newPathImages map[string][]*parse.ComposefileImage,
	done <-chan struct{},
) <-chan error {
	errCh := make(chan error)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		if len(existingPathImages) != len(newPathImages) {
			select {
			case errCh <- fmt.Errorf(
				"existing has %d paths, but new has %d",
				len(existingPathImages), len(newPathImages),
			):
			case <-done:
			}

			return
		}

		for path, existingImages := range existingPathImages {
			path := path
			existingImages := existingImages

			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				newImages, ok := newPathImages[path]
				if !ok {
					select {
					case errCh <- fmt.Errorf(
						"existing path %s does not exist in new", path,
					):
					case <-done:
					}

					return
				}

				if len(existingImages) != len(newImages) {
					select {
					case errCh <- fmt.Errorf(
						"existing path %s has %d images but new has %d",
						path, len(existingImages), len(newImages),
					):
					case <-done:
					}

					return
				}

				for i := range existingImages {
					i := i

					waitGroup.Add(1)

					go func() {
						defer waitGroup.Done()

						if existingImages[i] == nil ||
							newImages[i] == nil ||
							existingImages[i].Image == nil ||
							newImages[i].Image == nil {
							select {
							case errCh <- fmt.Errorf("images cannot be nil"):
							case <-done:
							}

							return
						}

						existingImage := parse.ComposefileImage{
							Image: &parse.Image{
								Name:   existingImages[i].Name,
								Tag:    existingImages[i].Tag,
								Digest: existingImages[i].Digest,
							},
							DockerfilePath: existingImages[i].DockerfilePath,
							ServiceName:    existingImages[i].ServiceName,
						}

						newImage := parse.ComposefileImage{
							Image: &parse.Image{
								Name:   newImages[i].Name,
								Tag:    newImages[i].Tag,
								Digest: newImages[i].Digest,
							},
							DockerfilePath: newImages[i].DockerfilePath,
							ServiceName:    newImages[i].ServiceName,
						}

						if c.ExcludeTags {
							existingImage.Tag = ""
							newImage.Tag = ""
						}

						if *existingImage.Image != *newImage.Image {
							select {
							case errCh <- fmt.Errorf(
								"on path %s existing image %v differs "+
									"from the new image %v",
								path, *existingImage.Image, *newImage.Image,
							):
							case <-done:
							}

							return
						}

						if existingImage.ServiceName != newImage.ServiceName {
							select {
							case errCh <- fmt.Errorf(
								"on path %s existing ServiceName %s differs "+
									"from the new ServiceName %s",
								path, existingImage.ServiceName,
								newImage.ServiceName,
							):
							case <-done:
							}

							return
						}

						if existingImage.DockerfilePath != newImage.DockerfilePath { // nolint: lll
							select {
							case errCh <- fmt.Errorf(
								"on path %s existing DockerfilePath %s "+
									"differs from the new DockerfilePath %s",
								path, existingImage.DockerfilePath,
								newImage.DockerfilePath,
							):
							case <-done:
							}

							return
						}
					}()
				}
			}()
		}
	}()

	go func() {
		waitGroup.Wait()
		close(errCh)
	}()

	return errCh
}
