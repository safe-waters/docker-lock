package generate

import (
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
)

// PathCollector contains PathCollectors for all files.
type PathCollector struct {
	DockerfileCollector     collect.IPathCollector
	ComposefileCollector    collect.IPathCollector
	KubernetesfileCollector collect.IPathCollector
}

// IPathCollector provides an interface for PathCollector's exported
// methods, which are used by Generator.
type IPathCollector interface {
	CollectPaths(done <-chan struct{}) <-chan *AnyPath
}

// AnyPath contains any possible type of path.
type AnyPath struct {
	DockerfilePath     string
	ComposefilePath    string
	KubernetesfilePath string
	Err                error
}

// CollectPaths collects paths to be parsed.
func (p *PathCollector) CollectPaths(done <-chan struct{}) <-chan *AnyPath {
	if (p.DockerfileCollector == nil ||
		reflect.ValueOf(p.DockerfileCollector).IsNil()) &&
		(p.ComposefileCollector == nil ||
			reflect.ValueOf(p.ComposefileCollector).IsNil()) &&
		(p.KubernetesfileCollector == nil ||
			reflect.ValueOf(p.KubernetesfileCollector).IsNil()) {
		return nil
	}

	anyPaths := make(chan *AnyPath)

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
						case anyPaths <- &AnyPath{
							Err: dockerfilePathResult.Err,
						}:
						}

						return
					}

					select {
					case <-done:
						return
					case anyPaths <- &AnyPath{
						DockerfilePath: dockerfilePathResult.Path,
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
						case anyPaths <- &AnyPath{
							Err: composefilePathResult.Err,
						}:
						}

						return
					}

					select {
					case <-done:
						return
					case anyPaths <- &AnyPath{
						ComposefilePath: composefilePathResult.Path,
					}:
					}
				}
			}()
		}

		if p.KubernetesfileCollector != nil &&
			!reflect.ValueOf(p.KubernetesfileCollector).IsNil() {
			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				kubernetesfilePathResults := p.KubernetesfileCollector.CollectPaths( // nolint: lll
					done,
				)
				for kubernetesfilePathResult := range kubernetesfilePathResults { // nolint: lll
					if kubernetesfilePathResult.Err != nil {
						select {
						case <-done:
						case anyPaths <- &AnyPath{
							Err: kubernetesfilePathResult.Err,
						}:
						}

						return
					}

					select {
					case <-done:
						return
					case anyPaths <- &AnyPath{
						KubernetesfilePath: kubernetesfilePathResult.Path,
					}:
					}
				}
			}()
		}
	}()

	go func() {
		waitGroup.Wait()
		close(anyPaths)
	}()

	return anyPaths
}
