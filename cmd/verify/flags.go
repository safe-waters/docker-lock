package verify

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Flags holds all command line options for Dockerfiles, Composefiles,
// and Kubernetesfiles.
type Flags struct {
	LockfileName          string
	IgnoreMissingDigests  bool
	UpdateExistingDigests bool
	ExcludeTags           bool
}

// NewFlags returns Flags for Dockerfiles, Composefiles, and Kubernetesfiles,
// after validating their fields.
//
// lockfileName may not contain slashes.
func NewFlags(
	lockfileName string,
	ignoreMissingDigests bool,
	updateExistingDigests bool,
	excludeTags bool,
) (*Flags, error) {
	if err := validateLockfileName(lockfileName); err != nil {
		return nil, err
	}

	return &Flags{
		LockfileName:          lockfileName,
		IgnoreMissingDigests:  ignoreMissingDigests,
		UpdateExistingDigests: updateExistingDigests,
		ExcludeTags:           excludeTags,
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
