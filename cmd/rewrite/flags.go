package rewrite

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Flags holds all command line options for Dockerfiles, Composefiles,
// and Kubernetesfiles.
type Flags struct {
	LockfileName string
	TempDir      string
	ExcludeTags  bool
}

// NewFlags returns Flags after validating its fields.
// lockfileName may not contain slashes.
func NewFlags(
	lockfileName string,
	tempDir string,
	excludeTags bool,
) (*Flags, error) {
	if err := validateLockfileName(lockfileName); err != nil {
		return nil, err
	}

	return &Flags{
		LockfileName: lockfileName,
		TempDir:      tempDir,
		ExcludeTags:  excludeTags,
	}, nil
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
