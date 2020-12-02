// Package collect provides functionality to collect file paths for processing.
package collect

import "github.com/safe-waters/docker-lock/pkg/kind"

// IPathCollector provides an interface for PathCollectors,
// which collect paths to be processed downstream.
type IPathCollector interface {
	Kind() kind.Kind
	CollectPaths(done <-chan struct{}) <-chan IPath
}

// IPath provides an interface for Paths collected by IPathCollectors.
type IPath interface {
	Kind() kind.Kind
	SetKind(kind kind.Kind)
	Val() string
	SetVal(val string)
	Err() error
	SetErr(err error)
}
