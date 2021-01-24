package collect

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/kind"
)

type pathCollector struct {
	kind            kind.Kind
	baseDir         string
	defaultPathVals []string
	manualPathVals  []string
	globVals        []string
	recursive       bool
}

// NewPathCollector returns an IPathCollector after validating its fields. If
// recursive is true, defaultPathVals must be defined so the collector knows
// which files to collect as it recurs.
func NewPathCollector(
	kind kind.Kind,
	baseDir string,
	defaultPathVals []string,
	manualPathVals []string,
	globVals []string,
	recursive bool,
) (IPathCollector, error) {
	if recursive && len(defaultPathVals) == 0 {
		return nil,
			errors.New(
				"if 'recursive' is true, 'defaultPathVals' must also be set",
			)
	}

	baseDir = filepath.Join(".", baseDir)
	if err := isSubPath(baseDir); err != nil {
		return nil, err
	}

	return &pathCollector{
		kind:            kind,
		baseDir:         baseDir,
		defaultPathVals: defaultPathVals,
		manualPathVals:  manualPathVals,
		globVals:        globVals,
		recursive:       recursive,
	}, nil
}

// Kind is a getter for the kind.
func (p *pathCollector) Kind() kind.Kind {
	return p.kind
}

// CollectPaths gathers specified file paths if they are within the base
// directory or a subdirectory of the base directory. Paths are deduplicated.
func (p *pathCollector) CollectPaths(done <-chan struct{}) <-chan IPath {
	var (
		waitGroup sync.WaitGroup
		paths     = make(chan IPath)
	)

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		var (
			intermediateWaitGroup sync.WaitGroup
			intermediatePaths     = make(chan IPath)
		)

		if len(p.manualPathVals) != 0 {
			intermediateWaitGroup.Add(1)

			go p.collectManualPaths(
				intermediatePaths, done, &intermediateWaitGroup,
			)
		}

		if len(p.globVals) != 0 {
			intermediateWaitGroup.Add(1)

			go p.collectGlobs(
				intermediatePaths, done, &intermediateWaitGroup,
			)
		}

		if p.recursive {
			intermediateWaitGroup.Add(1)

			go p.collectRecursive(
				intermediatePaths, done, &intermediateWaitGroup,
			)
		}

		if len(p.manualPathVals) == 0 &&
			len(p.globVals) == 0 &&
			!p.recursive &&
			len(p.defaultPathVals) != 0 {
			intermediateWaitGroup.Add(1)

			go p.collectDefaultPaths(
				intermediatePaths, done, &intermediateWaitGroup,
			)
		}

		go func() {
			intermediateWaitGroup.Wait()
			close(intermediatePaths)
		}()

		seenPathVals := map[string]struct{}{}

		for result := range intermediatePaths {
			if result.Err() != nil {
				select {
				case <-done:
				case paths <- result:
				}

				return
			}

			if _, ok := seenPathVals[result.Val()]; !ok {
				seenPathVals[result.Val()] = struct{}{}

				select {
				case <-done:
				case paths <- result:
				}
			}
		}
	}()

	go func() {
		waitGroup.Wait()
		close(paths)
	}()

	return paths
}

func (p *pathCollector) collectManualPaths(
	paths chan<- IPath,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	for _, val := range p.manualPathVals {
		val = filepath.Join(p.baseDir, val)

		if err := validatePath(val); err != nil {
			select {
			case <-done:
			case paths <- NewPath(p.kind, "", err):
			}

			return
		}

		select {
		case <-done:
			return
		case paths <- NewPath(p.kind, val, nil):
		}
	}
}

func (p *pathCollector) collectDefaultPaths(
	paths chan<- IPath,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	for _, val := range p.defaultPathVals {
		val = filepath.Join(p.baseDir, val)

		if err := isSubPath(val); err != nil {
			select {
			case <-done:
			case paths <- NewPath(p.kind, "", err):
			}

			return
		}

		if err := validatePath(val); err == nil {
			select {
			case <-done:
				return
			case paths <- NewPath(p.kind, val, nil):
			}
		}
	}
}

func (p *pathCollector) collectGlobs(
	pathResults chan<- IPath,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	for _, val := range p.globVals {
		val = filepath.Join(p.baseDir, val)

		vals, err := filepath.Glob(val)
		if err != nil {
			select {
			case <-done:
			case pathResults <- NewPath(p.kind, "", err):
			}

			return
		}

		for _, val := range vals {
			if err := validatePath(val); err != nil {
				select {
				case <-done:
				case pathResults <- NewPath(p.kind, "", err):
				}

				return
			}

			select {
			case <-done:
				return
			case pathResults <- NewPath(p.kind, val, nil):
			}
		}
	}
}

func (p *pathCollector) collectRecursive(
	paths chan<- IPath,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	defaultSet := map[string]struct{}{}

	for _, val := range p.defaultPathVals {
		defaultSet[val] = struct{}{}
	}

	if err := filepath.Walk(
		p.baseDir, func(val string, info os.FileInfo, err error,
		) error {
			if err != nil {
				return err
			}

			if _, ok := defaultSet[filepath.Base(val)]; ok {
				if err := validatePath(val); err != nil {
					return err
				}

				select {
				case <-done:
				case paths <- NewPath(p.kind, val, nil):
				}
			}

			return nil
		},
	); err != nil {
		select {
		case <-done:
		case paths <- NewPath(p.kind, "", err):
		}
	}
}

func validatePath(val string) error {
	if err := isSubPath(val); err != nil {
		return err
	}

	fileInfo, err := os.Stat(val)
	if err != nil {
		return fmt.Errorf("'%s' encountered an error '%v'", err, val)
	}

	if mode := fileInfo.Mode(); mode.IsDir() {
		return fmt.Errorf(
			"'%s' was collected but is a directory rather than a file", val,
		)
	}

	return nil
}

func isSubPath(val string) error {
	if strings.HasPrefix(val, "..") {
		return fmt.Errorf("'%s' is outside the current working directory", val)
	}

	return nil
}
