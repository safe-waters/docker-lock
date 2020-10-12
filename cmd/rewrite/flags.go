package rewrite

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Flags are all possible flags to initialize a Rewriter.
type Flags struct {
	LockfileName string
	TempDir      string
	ExcludeTags  bool
}

// NewFlags returns Flags after validating its fields.
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
	if strings.Contains(lockfileName, string(filepath.Separator)) {
		return fmt.Errorf(
			"'%s' lockfile-name cannot contain slashes", lockfileName,
		)
	}

	return nil
}
