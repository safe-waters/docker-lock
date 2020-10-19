package verify

import (
	"encoding/json"
	"fmt"

	"github.com/safe-waters/docker-lock/pkg/generate"
)

// DifferentLockfileError reports differences between the existing Lockfile
// and one that is newly generated.
type DifferentLockfileError struct {
	ExistingLockfile *generate.Lockfile
	NewLockfile      *generate.Lockfile
}

// Error returns the different files, indented as JSON.
func (d *DifferentLockfileError) Error() string {
	existingPrettyLockfile, _ := d.jsonPrettyPrint(d.ExistingLockfile)
	newPrettyLockfile, _ := d.jsonPrettyPrint(d.NewLockfile)

	return fmt.Sprintf(
		"new:\n%s\nexisting:\n%s",
		newPrettyLockfile,
		existingPrettyLockfile,
	)
}

func (d *DifferentLockfileError) jsonPrettyPrint(
	lockfile *generate.Lockfile,
) (string, error) {
	byt, err := json.MarshalIndent(lockfile, "", "\t")
	if err != nil {
		return "", err
	}

	return string(byt), nil
}
