package generate

import (
	"errors"

	"github.com/safe-waters/docker-lock/generate/collect"
)

// PathCollector contains PathCollectors for Dockerfiles
// and docker-compose files.
type PathCollector struct {
	DockerfileCollector  *collect.PathCollector
	ComposefileCollector *collect.PathCollector
}

// IPathCollector provides an interface for PathCollector's exported
// methods, which are used by Generator.
type IPathCollector interface {
	CollectPaths(done <-chan struct{}) (
		dockerfilePaths <-chan *collect.PathResult,
		composefilePaths <-chan *collect.PathResult,
	)
}

// CollectPaths collects Dockerfile and docker-compose file paths.
func (p *PathCollector) CollectPaths(
	done <-chan struct{},
) (
	dockerfilePaths <-chan *collect.PathResult,
	composefilePaths <-chan *collect.PathResult,
) {
	if p.DockerfileCollector != nil {
		dockerfilePaths = p.DockerfileCollector.CollectPaths(done)
	}

	if p.ComposefileCollector != nil {
		composefilePaths = p.ComposefileCollector.CollectPaths(done)
	}

	return
}

// DefaultPathCollector creates a Collector for Generator.
func DefaultPathCollector(flags *Flags) (IPathCollector, error) {
	if flags == nil {
		return nil, errors.New("flags cannot be nil")
	}

	if flags.DockerfileFlags == nil {
		return nil, errors.New("flags.DockerfileFlags cannot be nil")
	}

	if flags.ComposefileFlags == nil {
		return nil, errors.New("flags.ComposefileFlags cannot be nil")
	}

	var dockerfileCollector *collect.PathCollector

	var composefileCollector *collect.PathCollector

	var err error

	if !flags.DockerfileFlags.ExcludePaths {
		dockerfileCollector, err = collect.NewPathCollector(
			flags.FlagsWithSharedValues.BaseDir, []string{"Dockerfile"},
			flags.DockerfileFlags.ManualPaths, flags.DockerfileFlags.Globs,
			flags.DockerfileFlags.Recursive,
		)
		if err != nil {
			return nil, err
		}
	}

	if !flags.ComposefileFlags.ExcludePaths {
		composefileCollector, err = collect.NewPathCollector(
			flags.FlagsWithSharedValues.BaseDir,
			[]string{"docker-compose.yml", "docker-compose.yaml"},
			flags.ComposefileFlags.ManualPaths, flags.ComposefileFlags.Globs,
			flags.ComposefileFlags.Recursive,
		)
		if err != nil {
			return nil, err
		}
	}

	return &PathCollector{
		DockerfileCollector:  dockerfileCollector,
		ComposefileCollector: composefileCollector,
	}, nil
}
