package generate

import (
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
)

// PathCollector contains PathCollectors for Dockerfiles
// and docker-compose files.
type PathCollector struct {
	DockerfileCollector  collect.IPathCollector
	ComposefileCollector collect.IPathCollector
}

// IPathCollector provides an interface for PathCollector's exported
// methods, which are used by Generator.
type IPathCollector interface {
	CollectPaths(done <-chan struct{}) <-chan *CollectedPath
}

// CollectedPath contains any possible type of path.
type CollectedPath struct {
	Type FileType
	Path string
	Err  error
}

// CollectPaths collects paths to be parsed.
func (p *PathCollector) CollectPaths(
	done <-chan struct{},
) <-chan *CollectedPath {
	if (p.DockerfileCollector == nil ||
		reflect.ValueOf(p.DockerfileCollector).IsNil()) &&
		(p.ComposefileCollector == nil ||
			reflect.ValueOf(p.ComposefileCollector).IsNil()) {
		return nil
	}

	collectedPaths := make(chan *CollectedPath)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		if p.DockerfileCollector != nil &&
			!reflect.ValueOf(p.DockerfileCollector).IsNil() {
			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				dockerfilePathResults := p.DockerfileCollector.CollectPaths(
					done,
				)
				for dockerfilePathResult := range dockerfilePathResults {
					if dockerfilePathResult.Err != nil {
						select {
						case <-done:
						case collectedPaths <- &CollectedPath{
							Type: Dockerfile,
							Err:  dockerfilePathResult.Err,
						}:
						}

						return
					}

					select {
					case <-done:
						return
					case collectedPaths <- &CollectedPath{
						Type: Dockerfile,
						Path: dockerfilePathResult.Path,
					}:
					}
				}
			}()
		}

		if p.ComposefileCollector != nil &&
			!reflect.ValueOf(p.ComposefileCollector).IsNil() {
			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				composefilePathResults := p.ComposefileCollector.CollectPaths(
					done,
				)
				for composefilePathResult := range composefilePathResults {
					if composefilePathResult.Err != nil {
						select {
						case <-done:
						case collectedPaths <- &CollectedPath{
							Type: Composefile,
							Err:  composefilePathResult.Err,
						}:
						}

						return
					}

					select {
					case <-done:
						return
					case collectedPaths <- &CollectedPath{
						Type: Composefile,
						Path: composefilePathResult.Path,
					}:
					}
				}
			}()
		}
	}()

	go func() {
		waitGroup.Wait()
		close(collectedPaths)
	}()

	return collectedPaths
}
