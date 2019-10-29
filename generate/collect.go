package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"
)

type collectedFileResult struct {
	path string
	err  error
}

func collectDockerfiles(cmd *cobra.Command) ([]string, error) {
	isDefaultDockerfile := func(fpath string) bool {
		return filepath.Base(fpath) == "Dockerfile"
	}
	baseDir, err := cmd.Flags().GetString("base-dir")
	if err != nil {
		return nil, err
	}
	dockerfiles, err := cmd.Flags().GetStringSlice("dockerfiles")
	if err != nil {
		return nil, err
	}
	dockerfileRecursive, err := cmd.Flags().GetBool("dockerfile-recursive")
	if err != nil {
		return nil, err
	}
	dockerfileGlobs, err := cmd.Flags().GetStringSlice("dockerfile-globs")
	if err != nil {
		return nil, err
	}
	return collectFiles(baseDir, dockerfiles, dockerfileRecursive, isDefaultDockerfile, dockerfileGlobs)
}

func collectComposefiles(cmd *cobra.Command) ([]string, error) {
	isDefaultComposefile := func(fpath string) bool {
		return filepath.Base(fpath) == "docker-compose.yml" || filepath.Base(fpath) == "docker-compose.yaml"
	}
	baseDir, err := cmd.Flags().GetString("base-dir")
	if err != nil {
		return nil, err
	}
	composefiles, err := cmd.Flags().GetStringSlice("compose-files")
	if err != nil {
		return nil, err
	}
	composefileRecursive, err := cmd.Flags().GetBool("compose-file-recursive")
	if err != nil {
		return nil, err
	}
	composefileGlobs, err := cmd.Flags().GetStringSlice("compose-file-globs")
	if err != nil {
		return nil, err
	}
	return collectFiles(baseDir, composefiles, composefileRecursive, isDefaultComposefile, composefileGlobs)
}

func collectFiles(baseDir string, files []string, recursive bool, isDefaultName func(string) bool, globs []string) ([]string, error) {
	var wg sync.WaitGroup
	fileCh := make(chan collectedFileResult)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, path := range files {
			fileCh <- collectedFileResult{path: path}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if recursive {
			err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
					return err
				}
				if info.Mode().IsRegular() && isDefaultName(filepath.Base(path)) {
					fileCh <- collectedFileResult{path: path}
				}
				return nil
			})
			if err != nil {
				fileCh <- collectedFileResult{err: err}
			}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, pattern := range globs {
			pattern = filepath.Join(baseDir, pattern)
			paths, err := filepath.Glob(pattern)
			if err != nil {
				fileCh <- collectedFileResult{err: err}
				break
			}
			for _, path := range paths {
				fileCh <- collectedFileResult{path: path}
			}
		}
	}()
	go func() {
		wg.Wait()
		close(fileCh)
	}()
	uniqueFiles := make([]string, 0)
	fileSet := make(map[string]bool)
	for res := range fileCh {
		if res.err != nil {
			return nil, res.err
		}
		if !fileSet[res.path] {
			uniqueFiles = append(uniqueFiles, res.path)
		}
		fileSet[res.path] = true
	}
	return uniqueFiles, nil
}

func collectDefaultFiles(baseDir string) ([]string, []string) {
	var dockerfiles []string
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defaultDockerfile := filepath.Join(baseDir, "Dockerfile")
		fi, err := os.Stat(defaultDockerfile)
		if err == nil {
			if mode := fi.Mode(); mode.IsRegular() {
				dockerfiles = []string{defaultDockerfile}
			}
		}
	}()
	var composefiles []string
	wg.Add(1)
	go func() {
		defer wg.Done()
		defaultComposefiles := []string{
			filepath.Join(baseDir, "docker-compose.yml"),
			filepath.Join(baseDir, "docker-compose.yaml"),
		}
		for _, defaultComposefile := range defaultComposefiles {
			fi, err := os.Stat(defaultComposefile)
			if err == nil {
				if mode := fi.Mode(); mode.IsRegular() {
					composefiles = append(composefiles, defaultComposefile)
				}
			}
		}
	}()
	wg.Wait()
	return dockerfiles, composefiles
}
