// Package rewrite provides functionality to rewrite a Lockfile.
package rewrite

import (
	"io"

	"github.com/safe-waters/docker-lock/pkg/kind"
	"github.com/safe-waters/docker-lock/pkg/rewrite/write"
)

// IPreprocessor provides an interface for Preprocessors, which are responsible
// for modifying a Lockfile, if need be, before rewriting.
type IPreprocessor interface {
	PreprocessLockfile(
		lockfile map[kind.Kind]map[string][]interface{},
	) (map[kind.Kind]map[string][]interface{}, error)
}

// IWriter provides an interface for Writers, which are responsible for
// writing files from a Lockfile to temporary paths with images from the
// Lockfile.
type IWriter interface {
	WriteFiles(
		lockfile map[kind.Kind]map[string][]interface{},
		tempDir string,
		done <-chan struct{},
	) <-chan write.IWrittenPath
}

// IRewriter provides an interface for Rewriters, which are responsible for
// rewriting files referenced in a Lockfile with images from the Lockfile.
type IRewriter interface {
	RewriteLockfile(lockfileReader io.Reader, tempDir string) error
}

// IRenamer provides an interface for Renamers, which rename temporary files
// from IWriters to their original paths.
type IRenamer interface {
	RenameFiles(writtenPaths <-chan write.IWrittenPath) error
}
