package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FlagsWithSharedValues represents flags whose values
// are the same for Dockerfiles, Composefiles and Kubernetesfiles.
type FlagsWithSharedValues struct {
	BaseDir               string
	LockfileName          string
	IgnoreMissingDigests  bool
	UpdateExistingDigests bool
}

// FlagsWithSharedNames represents flags whose values
// differ for Dockerfiles, Composefiles, and Kubernetesfiles.
type FlagsWithSharedNames struct {
	ManualPaths  []string
	Globs        []string
	Recursive    bool
	ExcludePaths bool
}

// Flags holds all command line options for Dockerfiles, Composefiles,
// and Kubernetesfiles.
type Flags struct {
	FlagsWithSharedValues *FlagsWithSharedValues
	DockerfileFlags       *FlagsWithSharedNames
	ComposefileFlags      *FlagsWithSharedNames
	KubernetesfileFlags   *FlagsWithSharedNames
}

// NewFlagsWithSharedValues returns Flags that are shared among Dockerfiles,
// Composefiles, and Kubernetesfiles, after validating its fields.
//
// baseDir must be the current working directory or a sub directory.
// Absolute paths are not supported.
//
// lockfileName may not contain slashes.
func NewFlagsWithSharedValues(
	baseDir string,
	lockfileName string,
	ignoreMissingDigests bool,
	updateExistingDigests bool,
) (*FlagsWithSharedValues, error) {
	if baseDir != "" {
		if err := validateBaseDirectory(baseDir); err != nil {
			return nil, err
		}
	}

	if lockfileName != "" {
		if err := validateLockfileName(lockfileName); err != nil {
			return nil, err
		}
	}

	return &FlagsWithSharedValues{
		BaseDir:               baseDir,
		LockfileName:          lockfileName,
		IgnoreMissingDigests:  ignoreMissingDigests,
		UpdateExistingDigests: updateExistingDigests,
	}, nil
}

// NewFlagsWithSharedNames returns Flags whose values differ
// between Dockerfiles, Composefiles, and Kubernetesfiles, after
// validating its fields.
//
// baseDir must be the current working directory or a sub directory.
//
// manualPaths and globs must be in the current working directory or
// in a sub directory.
//
// Absolute paths are not supported.
func NewFlagsWithSharedNames(
	baseDir string,
	manualPaths []string,
	globs []string,
	recursive bool,
	excludePaths bool,
) (*FlagsWithSharedNames, error) {
	if baseDir != "" {
		if err := validateBaseDirectory(baseDir); err != nil {
			return nil, err
		}
	}

	if len(manualPaths) != 0 {
		if err := validateManualPaths(baseDir, manualPaths); err != nil {
			return nil, err
		}
	}

	if len(globs) != 0 {
		if err := validateGlobs(globs); err != nil {
			return nil, err
		}
	}

	return &FlagsWithSharedNames{
		ManualPaths:  manualPaths,
		Globs:        globs,
		Recursive:    recursive,
		ExcludePaths: excludePaths,
	}, nil
}

// NewFlags returns Flags for Dockerfiles, Composefiles, and Kubernetesfiles,
// subject to the validation logic in NewFlagsWithSharedNames and
// NewFlagsWithSharedValues.
func NewFlags(
	baseDir string,
	lockfileName string,
	ignoreMissingDigests bool,
	updateExistingDigests bool,
	dockerfilePaths []string,
	composefilePaths []string,
	kubernetesfilePaths []string,
	dockerfileGlobs []string,
	composefileGlobs []string,
	kubernetesfileGlobs []string,
	dockerfileRecursive bool,
	composefileRecursive bool,
	kubernetesfileRecursive bool,
	dockerfileExcludeAll bool,
	composefileExcludeAll bool,
	kubernetesfileExcludeAll bool,
) (*Flags, error) {
	sharedFlags, err := NewFlagsWithSharedValues(
		baseDir, lockfileName, ignoreMissingDigests, updateExistingDigests,
	)
	if err != nil {
		return nil, err
	}

	dockerfileFlags, err := NewFlagsWithSharedNames(
		baseDir, dockerfilePaths, dockerfileGlobs,
		dockerfileRecursive, dockerfileExcludeAll,
	)
	if err != nil {
		return nil, err
	}

	composefileFlags, err := NewFlagsWithSharedNames(
		baseDir, composefilePaths, composefileGlobs,
		composefileRecursive, composefileExcludeAll,
	)
	if err != nil {
		return nil, err
	}

	kubernetesfileFlags, err := NewFlagsWithSharedNames(
		baseDir, kubernetesfilePaths, kubernetesfileGlobs,
		kubernetesfileRecursive, kubernetesfileExcludeAll,
	)
	if err != nil {
		return nil, err
	}

	return &Flags{
		FlagsWithSharedValues: sharedFlags,
		DockerfileFlags:       dockerfileFlags,
		ComposefileFlags:      composefileFlags,
		KubernetesfileFlags:   kubernetesfileFlags,
	}, nil
}

func validateBaseDirectory(baseDir string) error {
	if filepath.IsAbs(baseDir) {
		return fmt.Errorf(
			"'%s' base-dir does not support absolute paths", baseDir,
		)
	}

	if strings.HasPrefix(filepath.Join(".", baseDir), "..") {
		return fmt.Errorf(
			"'%s' base-dir is outside the current working directory", baseDir,
		)
	}

	fileInfo, err := os.Stat(baseDir)
	if err != nil {
		return err
	}

	if mode := fileInfo.Mode(); !mode.IsDir() {
		return fmt.Errorf(
			"'%s' base-dir is not sub directory "+
				"of the current working directory",
			baseDir,
		)
	}

	return nil
}

func validateLockfileName(lockfileName string) error {
	if filepath.IsAbs(lockfileName) {
		return fmt.Errorf(
			"'%s' lockfile-name does not support absolute paths", lockfileName,
		)
	}

	lockfileName = filepath.Join(".", lockfileName)

	if strings.ContainsAny(lockfileName, `/\`) {
		return fmt.Errorf(
			"'%s' lockfile-name cannot contain slashes", lockfileName,
		)
	}

	return nil
}

func validateManualPaths(baseDir string, manualPaths []string) error {
	for _, path := range manualPaths {
		if filepath.IsAbs(path) {
			return fmt.Errorf(
				"'%s' input paths do not support absolute paths", path,
			)
		}

		path = filepath.Join(baseDir, path)

		if strings.HasPrefix(path, "..") {
			return fmt.Errorf(
				"'%s' is outside the current working directory", path,
			)
		}
	}

	return nil
}

func validateGlobs(globs []string) error {
	for _, glob := range globs {
		if filepath.IsAbs(glob) {
			return fmt.Errorf("'%s' globs do not support absolute paths", glob)
		}
	}

	return nil
}
