package generate

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func collectDockerfiles(cmd *cobra.Command) ([]string, error) {
	isDefaultDockerfile := func(fpath string) bool {
		return filepath.Base(fpath) == "Dockerfile"
	}
	dockerfiles, err := cmd.Flags().GetStringSlice("dockerfiles")
	if err != nil {
		return nil, err
	}
	dockerfileRecursive, err := cmd.Flags().GetBool("dockerfile-recursive")
	if err != nil {
		return nil, err
	}
	dockerfileRecursiveDirectory, err := cmd.Flags().GetString("dockerfile-recursive-directory")
	if err != nil {
		return nil, err
	}
	dockerfileGlobs, err := cmd.Flags().GetStringSlice("dockerfile-globs")
	if err != nil {
		return nil, err
	}
	return collectFiles(dockerfiles, dockerfileRecursive, dockerfileRecursiveDirectory, isDefaultDockerfile, dockerfileGlobs)
}

func collectComposefiles(cmd *cobra.Command) ([]string, error) {
	isDefaultComposefile := func(fpath string) bool {
		return filepath.Base(fpath) == "docker-compose.yml" || filepath.Base(fpath) == "docker-compose.yaml"
	}
	composefiles, err := cmd.Flags().GetStringSlice("compose-files")
	if err != nil {
		return nil, err
	}
	composefileRecursive, err := cmd.Flags().GetBool("compose-file-recursive")
	if err != nil {
		return nil, err
	}
	composefileRecursiveDirectory, err := cmd.Flags().GetString("compose-file-recursive-directory")
	if err != nil {
		return nil, err
	}
	composefileGlobs, err := cmd.Flags().GetStringSlice("compose-file-globs")
	if err != nil {
		return nil, err
	}
	return collectFiles(composefiles, composefileRecursive, composefileRecursiveDirectory, isDefaultComposefile, composefileGlobs)
}

func collectFiles(files []string, recursive bool, recursiveStartDir string, isDefaultName func(string) bool, globs []string) ([]string, error) {
	fileSet := make(map[string]bool)
	for _, fileName := range files {
		fileSet[fileName] = true
	}
	if recursive {
		err := filepath.Walk(recursiveStartDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if isDefaultName(filepath.Base(path)) {
				fileSet[path] = true
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
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
