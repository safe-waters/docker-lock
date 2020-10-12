package verify

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Flags are all possible flags to initialize a Verifier.
type Flags struct {
	LockfileName string
	ConfigPath   string
	EnvPath      string
}

// NewFlags returns Flags after validating its fields.
func NewFlags(
	lockfileName string,
	configPath string,
	envPath string,
) (*Flags, error) {
	if err := validateLockfileName(lockfileName); err != nil {
		return nil, err
	}

	return &Flags{
		LockfileName: lockfileName,
		ConfigPath:   configPath,
		EnvPath:      envPath,
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
