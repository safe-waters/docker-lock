package generate

import (
	"errors"
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
)

type pathCollector struct {
	collectors []collect.IPathCollector
}

// NewPathCollector creates an IPathCollector from IPathCollectors for different
// kinds of paths. At least one collector must be non nil, otherwise there
// would be no paths to collect.
func NewPathCollector(
	collectors ...collect.IPathCollector,
) (IPathCollector, error) {
	var nonNilCollectors []collect.IPathCollector

	for _, collector := range collectors {
		if collector != nil && !reflect.ValueOf(collector).IsNil() {
			nonNilCollectors = append(nonNilCollectors, collector)
		}
	}

	if len(nonNilCollectors) == 0 {
		return nil, errors.New("non nil 'collectors' must be greater than 0")
	}

	return &pathCollector{collectors: nonNilCollectors}, nil
}

// CollectPaths collects all paths to be parsed.
func (p *pathCollector) CollectPaths(
	done <-chan struct{},
) <-chan collect.IPath {
	var (
		waitGroup sync.WaitGroup
		paths     = make(chan collect.IPath)
	)

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for _, collector := range p.collectors {
			collector := collector

			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				collectedPaths := collector.CollectPaths(done)
				for path := range collectedPaths {
					select {
					case <-done:
						return
					case paths <- path:
					}

					if path.Err() != nil {
						return
					}
				}
			}()
		}
	}()

	go func() {
		waitGroup.Wait()
		close(paths)
	}()

	return paths
}
