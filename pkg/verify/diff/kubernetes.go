package diff

import (
	"fmt"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
)

// IKubernetesfileDifferentiator provides an interface for diffing
// Kubernetesfiles.
type IKubernetesfileDifferentiator interface {
	Differentiate(
		existingPathImages map[string][]*parse.KubernetesfileImage,
		newPathImages map[string][]*parse.KubernetesfileImage,
		done <-chan struct{},
	) <-chan error
}

// KubernetesfileDifferentiator provides methods for diffing Kubernetes
// Path Images.
type KubernetesfileDifferentiator struct {
	ExcludeTags bool
}

// Differentiate diffs Kubernetesfile Path Images.
func (k *KubernetesfileDifferentiator) Differentiate(
	existingPathImages map[string][]*parse.KubernetesfileImage,
	newPathImages map[string][]*parse.KubernetesfileImage,
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

						existingImage := parse.KubernetesfileImage{
							Image: &parse.Image{
								Name:   existingImages[i].Name,
								Tag:    existingImages[i].Tag,
								Digest: existingImages[i].Digest,
							},
							ContainerName: existingImages[i].ContainerName,
						}

						newImage := parse.KubernetesfileImage{
							Image: &parse.Image{
								Name:   newImages[i].Name,
								Tag:    newImages[i].Tag,
								Digest: newImages[i].Digest,
							},
							ContainerName: newImages[i].ContainerName,
						}

						if k.ExcludeTags {
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

						if existingImage.ContainerName != newImage.ContainerName { // nolint: lll
							select {
							case errCh <- fmt.Errorf(
								"on path %s existing ContainerName %s differs "+
									"from the new ContainerName %s",
								path, existingImage.ContainerName,
								newImage.ContainerName,
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
