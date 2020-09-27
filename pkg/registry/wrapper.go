// Package registry provides functionality to get digests from registries.
package registry

// Wrapper defines an interface that registry wrappers implement.
type Wrapper interface {
	// Digest returns the digest from a repo and ref. For instance,
	// the repo and ref for dockerlocktestaccount.azurecr.io/busybox:latest
	// would be busybox and latest, respectively.
	Digest(repo string, ref string) (string, error)

	// Prefix returns the registry prefix and is used by the wrapper manager
	// to select which registry to use. For instance, the prefix for
	// 'dockerlocktestaccount.azurecr.io/busybox' would be
	// 'dockerlocktestaccount.azurecr.io/'.
	Prefix() string
}
