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
	dockerfileRecursive, err := cmd.Flags().GetBool("dockerfile-recursive")
	dockerfileRecursiveDirectory, err := cmd.Flags().GetString("dockerfile-recursive-directory")
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
	composefileRecursive, err := cmd.Flags().GetBool("compose-file-recursive")
	composefileRecursiveDirectory, err := cmd.Flags().GetString("compose-file-recursive-directory")
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
