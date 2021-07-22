package rewrite

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"github.com/safe-waters/docker-lock/pkg/kind"
)

type rewriter struct {
	preprocessor IPreprocessor
	writer       IWriter
	renamer      IRenamer
}

// NewRewriter returns an IRewriter after ensuring writer and renamer
// are non nil.
func NewRewriter(
	preprocessor IPreprocessor,
	writer IWriter,
	renamer IRenamer,
) (IRewriter, error) {
	if writer == nil || reflect.ValueOf(writer).IsNil() {
		return nil, errors.New("'writer' cannot be nil")
	}

	if renamer == nil || reflect.ValueOf(renamer).IsNil() {
		return nil, errors.New("'renamer' cannot be nil")
	}

	return &rewriter{
		writer:       writer,
		renamer:      renamer,
		preprocessor: preprocessor,
	}, nil
}

// RewriteLockfile rewrites files referenced by a Lockfile with images from
// the Lockfile. Rewriting is a two step process. First, all of the files
// are written to temporary paths in a subdirectory of tempDir (which will be
// created if it does not exist). Next, all temporary files are renamed to
// their original names.
func (r *rewriter) RewriteLockfile(
	lockfileReader io.Reader,
	tempDir string,
) error {
	if lockfileReader == nil || reflect.ValueOf(lockfileReader).IsNil() {
		return errors.New("'lockfileReader' cannot be nil")
	}

	tempDir = filepath.Join(tempDir, "temp-rewrite")
	if err := os.MkdirAll(tempDir, 0700); err != nil { // nolint: gomnd
		return err
	}

	defer os.RemoveAll(tempDir)

	var lockfile map[kind.Kind]map[string][]interface{}
	if err := json.NewDecoder(lockfileReader).Decode(&lockfile); err != nil {
		return err
	}

	if r.preprocessor != nil && !reflect.ValueOf(r.preprocessor).IsNil() {
		var err error

		lockfile, err = r.preprocessor.PreprocessLockfile(lockfile)
		if err != nil {
			return err
		}
	}

	done := make(chan struct{})
	defer close(done)

	writtenPaths := r.writer.WriteFiles(lockfile, tempDir, done)

	return r.renamer.RenameFiles(writtenPaths)
}
