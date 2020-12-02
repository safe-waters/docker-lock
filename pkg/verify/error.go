package verify

import (
	"fmt"
)

type differentLockfileError struct {
	existingLockfile []byte
	newLockfile      []byte
	err              error
}

func (d *differentLockfileError) Error() string {
	return fmt.Sprintf(
		"existing:\n%s\nnew:\n%s\nmsg: %s",
		string(d.existingLockfile),
		string(d.newLockfile),
		d.err,
	)
}
