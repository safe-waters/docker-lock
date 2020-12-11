package preprocess

import (
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/kind"
)

type composefilePreprocessor struct {
	kind kind.Kind
}

type deduplicatedPath struct {
	val string
	err error
}

// NewComposefilePreprocessor returns an IPreprocessor for Composefiles.
func NewComposefilePreprocessor() IPreprocessor {
	return &composefilePreprocessor{
		kind: kind.Composefile,
	}
}

// Kind is a getter for the kind.
func (c *composefilePreprocessor) Kind() kind.Kind {
	return c.kind
}

// PreprocessLockfile removes Dockerfiles from the Lockfile if they are already
// referenced by Composefiles.
func (c *composefilePreprocessor) PreprocessLockfile(
	lockfile map[kind.Kind]map[string][]interface{},
) (map[kind.Kind]map[string][]interface{}, error) {
	if lockfile == nil {
		return nil, errors.New("'lockfile' cannot be nil")
	}

	if len(lockfile[kind.Composefile]) == 0 ||
		len(lockfile[kind.Dockerfile]) == 0 {
		return lockfile, nil
	}

	var (
		waitGroup                   sync.WaitGroup
		deduplicatedDockerfilePaths = make(chan *deduplicatedPath)
		done                        = make(chan struct{})
	)

	defer close(done)

	for _, images := range lockfile[kind.Composefile] {
		images := images

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			for _, image := range images {
				image, ok := image.(map[string]interface{})
				if !ok {
					select {
					case <-done:
					case deduplicatedDockerfilePaths <- &deduplicatedPath{
						err: errors.New("malformed image"),
					}:
					}

					return
				}

				if image["dockerfile"] != nil {
					dockerfilePath, ok := image["dockerfile"].(string)
					if !ok {
						select {
						case <-done:
						case deduplicatedDockerfilePaths <- &deduplicatedPath{
							err: errors.New("malformed 'dockerfile' in image"),
						}:
						}

						return
					}

					if filepath.IsAbs(dockerfilePath) {
						var err error

						dockerfilePath, err = c.convertAbsToRelPath(
							dockerfilePath,
						)
						if err != nil {
							select {
							case <-done:
							case deduplicatedDockerfilePaths <- &deduplicatedPath{ // nolint: lll
								err: err,
							}:
							}

							return
						}
					}

					select {
					case <-done:
						return
					case deduplicatedDockerfilePaths <- &deduplicatedPath{
						val: dockerfilePath,
					}:
					}
				}
			}
		}()
	}

	go func() {
		waitGroup.Wait()
		close(deduplicatedDockerfilePaths)
	}()

	dockerfilePathsCache := map[string]struct{}{}

	for deduplicatedDockerfilePath := range deduplicatedDockerfilePaths {
		if deduplicatedDockerfilePath.err != nil {
			return nil, deduplicatedDockerfilePath.err
		}

		dockerfilePathsCache[deduplicatedDockerfilePath.val] = struct{}{}
	}

	if len(dockerfilePathsCache) == 0 {
		return lockfile, nil
	}

	dockerfilePathImages := map[string][]interface{}{}

	for path, images := range lockfile[kind.Dockerfile] {
		if _, ok := dockerfilePathsCache[path]; !ok {
			dockerfilePathImages[path] = images
		}
	}

	lockfile[kind.Dockerfile] = dockerfilePathImages

	return lockfile, nil
}

func (c *composefilePreprocessor) convertAbsToRelPath(
	path string,
) (string, error) {
	currentWorkingDirectory, err := os.Getwd()
	if err != nil {
		return "", err
	}

	relativePath, err := filepath.Rel(
		currentWorkingDirectory, filepath.FromSlash(path),
	)
	if err != nil {
		return "", err
	}

	return filepath.ToSlash(relativePath), nil
}
