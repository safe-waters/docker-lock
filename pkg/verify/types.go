// Package verify provides functionality for verifying that an existing
// Lockfile is up-to-date.
package verify

import "io"

// IVerifier provides an interface for Verifiers, which are responsible
// for verifying that a newly generated Lockfile equals the existing Lockfile.
type IVerifier interface {
	VerifyLockfile(lockfileReader io.Reader) error
}
