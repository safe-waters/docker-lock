// Package write provides functionality to write files with image digests.
package write

import "github.com/safe-waters/docker-lock/pkg/kind"

// IWriter provides an interface for Writers, which are responsible for
// writing files with information from a Lockfile to paths in outputDir.
type IWriter interface {
	Kind() kind.Kind
	WriteFiles(
		pathImages map[string][]interface{},
		outputDir string,
		done <-chan struct{},
	) <-chan IWrittenPath
}

// IWrittenPath provides an interface for WrittenPaths, which contain the
// original path from a Lockfile and a new path written by an IWriter.
type IWrittenPath interface {
	OriginalPath() string
	SetOriginalPath(originalPath string)
	NewPath() string
	SetNewPath(newPath string)
	Err() error
	SetErr(err error)
}
