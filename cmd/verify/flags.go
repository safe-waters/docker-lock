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
	lockfileName = filepath.Join(".", lockfileName)
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
	if strings.Contains(lockfileName, string(filepath.Separator)) {
		return fmt.Errorf(
			"'%s' lockfile-name cannot contain slashes", lockfileName,
		)
	}

	return nil
}
