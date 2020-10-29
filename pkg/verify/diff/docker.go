// Package diff provides functionality to diff a Lockfile.
package diff

import (
	"fmt"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
)

// IDockerfileDifferentiator provides an interface for diffing Dockerfiles.
type IDockerfileDifferentiator interface {
	Differentiate(
		existingPathImages map[string][]*parse.DockerfileImage,
		newPathImages map[string][]*parse.DockerfileImage,
		done <-chan struct{},
	) <-chan error
}

// DockerfileDifferentiator provides methods for diffing Dockerfile Path Images.
type DockerfileDifferentiator struct {
	ExcludeTags bool
}

// Differentiate diffs Dockerfile Path Images.
func (d *DockerfileDifferentiator) Differentiate(
	existingPathImages map[string][]*parse.DockerfileImage,
	newPathImages map[string][]*parse.DockerfileImage,
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

						existingImage := parse.DockerfileImage{
							Image: &parse.Image{
								Name:   existingImages[i].Name,
								Tag:    existingImages[i].Tag,
								Digest: existingImages[i].Digest,
							},
						}

						newImage := parse.DockerfileImage{
							Image: &parse.Image{
								Name:   newImages[i].Name,
								Tag:    newImages[i].Tag,
								Digest: newImages[i].Digest,
							},
						}

						if d.ExcludeTags {
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
