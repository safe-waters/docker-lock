// Package collect provides functionality to collect file paths for processing.
package collect

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// PathCollector gathers Dockerfile and docker-compose file paths.
type PathCollector struct {
	BaseDir      string
	DefaultPaths []string
	ManualPaths  []string
	Globs        []string
	Recursive    bool
}

// PathResult holds collected paths and errors that occurred
// while gathering them.
type PathResult struct {
	Path string
	Err  error
}

// NewPathCollector returns a PathCollector after validating its fields.
func NewPathCollector(
	baseDir string,
	defaultPaths []string,
	manualPaths []string,
	globs []string,
	recursive bool,
) (*PathCollector, error) {
	if recursive && len(defaultPaths) == 0 {
		return nil,
			errors.New("if recursive is true, defaultPaths must also be set")
	}

	return &PathCollector{
		BaseDir:      baseDir,
		DefaultPaths: defaultPaths,
		ManualPaths:  manualPaths,
		Globs:        globs,
		Recursive:    recursive,
	}, nil
}

// CollectPaths gathers file paths specified by PathCollector.
// It removes duplicates and ensures that the file paths are within
// a subdirectory of the base directory.
func (p *PathCollector) CollectPaths(done <-chan struct{}) <-chan *PathResult {
	pathResults := make(chan *PathResult)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		intermediatePathResults := make(chan *PathResult)
		intermediateDone := make(chan struct{})

		var intermediateWaitGroup sync.WaitGroup

		if len(p.ManualPaths) != 0 {
			intermediateWaitGroup.Add(1)

			go p.collectManualPaths(
				intermediatePathResults, intermediateDone,
				&intermediateWaitGroup,
			)
		}

		if len(p.Globs) != 0 {
			intermediateWaitGroup.Add(1)

			go p.collectGlobs(
				intermediatePathResults, intermediateDone,
				&intermediateWaitGroup,
			)
		}

		if p.Recursive {
			intermediateWaitGroup.Add(1)

			go p.collectRecursive(
				intermediatePathResults, intermediateDone,
				&intermediateWaitGroup,
			)
		}

		if len(p.ManualPaths) == 0 &&
			len(p.Globs) == 0 &&
			!p.Recursive &&
			len(p.DefaultPaths) != 0 {
			intermediateWaitGroup.Add(1)

			go p.collectDefaultPaths(
				intermediatePathResults, intermediateDone,
				&intermediateWaitGroup,
			)
		}

		go func() {
			intermediateWaitGroup.Wait()
			close(intermediatePathResults)
		}()

		seenPaths := map[string]struct{}{}

		for result := range intermediatePathResults {
			if result.Err != nil {
				close(intermediateDone)

				select {
				case <-done:
				case pathResults <- result:
				}

				return
			}

			if _, ok := seenPaths[result.Path]; !ok {
				seenPaths[result.Path] = struct{}{}

				select {
				case <-done:
				case pathResults <- result:
				}
			}
		}
	}()

	go func() {
		waitGroup.Wait()
		close(pathResults)
	}()

	return pathResults
}

func (p *PathCollector) collectManualPaths(
	pathResults chan<- *PathResult,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	for _, path := range p.ManualPaths {
		path = filepath.Join(p.BaseDir, path)

		if err := p.validatePath(path); err != nil {
			select {
			case <-done:
			case pathResults <- &PathResult{Err: err}:
			}

			return
		}

		select {
		case <-done:
			return
		case pathResults <- &PathResult{Path: path}:
		}
	}
}

func (p *PathCollector) collectDefaultPaths(
	pathResults chan<- *PathResult,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	for _, path := range p.DefaultPaths {
		path = filepath.Join(p.BaseDir, path)

		if err := p.validatePath(path); err != nil {
			select {
			case <-done:
			case pathResults <- &PathResult{Err: err}:
			}

			return
		}

		if err := p.fileExists(path); err == nil {
			select {
			case <-done:
				return
			case pathResults <- &PathResult{Path: path}:
			}
		}
	}
}

func (p *PathCollector) collectGlobs(
	pathResults chan<- *PathResult,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	for _, glob := range p.Globs {
		glob = filepath.Join(p.BaseDir, glob)

		paths, err := filepath.Glob(glob)
		if err != nil {
			select {
			case <-done:
			case pathResults <- &PathResult{Err: err}:
			}

			return
		}

		for _, path := range paths {
			if err := p.validatePath(path); err != nil {
				select {
				case <-done:
				case pathResults <- &PathResult{Err: err}:
				}

				return
			}

			select {
			case <-done:
				return
			case pathResults <- &PathResult{Path: path}:
			}
		}
	}
}

func (p *PathCollector) collectRecursive(
	pathResults chan<- *PathResult,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	defaultSet := map[string]struct{}{}

	for _, path := range p.DefaultPaths {
		defaultSet[path] = struct{}{}
	}

	if err := filepath.Walk(
		p.BaseDir, func(path string, info os.FileInfo, err error,
		) error {
			if err != nil {
				return err
			}

			if _, ok := defaultSet[filepath.Base(path)]; ok {
				if err := p.validatePath(path); err != nil {
					return err
				}

				select {
				case <-done:
				case pathResults <- &PathResult{Path: path}:
				}
			}

			return nil
		},
	); err != nil {
		select {
		case <-done:
		case pathResults <- &PathResult{Err: err}:
		}
	}
}

func (p *PathCollector) fileExists(path string) error {
	_, err := os.Stat(path)
	return err
}

func (p *PathCollector) validatePath(path string) error {
	if strings.HasPrefix(path, "..") {
		return fmt.Errorf("'%s' is outside the current working directory", path)
	}

	return nil
}
