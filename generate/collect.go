package generate

import (
	"os"
	"path/filepath"
)

func collectDockerfiles(flags *Flags) ([]string, error) {
	isDefaultDockerfile := func(fpath string) bool {
		return filepath.Base(fpath) == "Dockerfile"
	}
	return collectFiles(flags.Dockerfiles, flags.Recursive, flags.RecursiveDir, isDefaultDockerfile, flags.Globs)
}

func collectComposefiles(flags *Flags) ([]string, error) {
	isDefaultComposefile := func(fpath string) bool {
		return filepath.Base(fpath) == "docker-compose.yml" || filepath.Base(fpath) == "docker-compose.yaml"
	}
	return collectFiles(flags.Composefiles, flags.ComposeRecursive, flags.ComposeRecursiveDir, isDefaultComposefile, flags.ComposeGlobs)
}

func collectFiles(files []string, recursive bool, recursiveStartDir string, isDefaultName func(string) bool, globs []string) ([]string, error) {
	fileSet := make(map[string]bool)
	for _, fileName := range files {
		fileSet[fileName] = true
	}
	if recursive {
		filepath.Walk(recursiveStartDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if isDefaultName(filepath.Base(path)) {
				fileSet[path] = true
			}
			return nil
		})
	}
	for _, pattern := range globs {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			fileSet[match] = true
		}
	}
	collectedFiles := make([]string, len(fileSet))
	i := 0
	for file := range fileSet {
		collectedFiles[i] = file
		i++
	}
	return collectedFiles, nil
}
