package rewrite

import (
	"errors"
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/kind"
	"github.com/safe-waters/docker-lock/pkg/rewrite/write"
)

type writer struct {
	writers map[kind.Kind]write.IWriter
}

// NewWriter creates an IWriter from IWriters for different
// kinds of files. At least one writer must be non nil, otherwise there
// would be no way to write files.
func NewWriter(writers ...write.IWriter) (IWriter, error) {
	kindWriter := map[kind.Kind]write.IWriter{}

	for _, writer := range writers {
		if writer != nil && !reflect.ValueOf(writer).IsNil() {
			kindWriter[writer.Kind()] = writer
		}
	}

	if len(kindWriter) == 0 {
		return nil, errors.New("non nil 'writers' must be greater than 0")
	}

	return &writer{writers: kindWriter}, nil
}

// WriteFiles writes files with images from a Lockfile.
func (w *writer) WriteFiles(
	lockfile map[kind.Kind]map[string][]interface{},
	tempDir string,
	done <-chan struct{},
) <-chan write.IWrittenPath {
	if lockfile == nil {
		return nil
	}

	var (
		waitGroup    sync.WaitGroup
		writtenPaths = make(chan write.IWrittenPath)
	)

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for kind, writer := range w.writers {
			kind := kind
			writer := writer

			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				for writtenPath := range writer.WriteFiles(
					lockfile[kind], tempDir, done,
				) {
					select {
					case <-done:
						return
					case writtenPaths <- writtenPath:
					}

					if writtenPath.Err() != nil {
						return
					}
				}
			}()
		}
	}()

	go func() {
		waitGroup.Wait()
		close(writtenPaths)
	}()

	return writtenPaths
}
