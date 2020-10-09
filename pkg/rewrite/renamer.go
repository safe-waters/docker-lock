package rewrite

import (
	"os"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/rewrite/write"
)

// Renamer renames written paths to their original file paths.
type Renamer struct{}

// IRenamer provides an interface for Renamer's exported methods.
type IRenamer interface {
	RenameFiles(writtenPaths <-chan *write.WrittenPath) error
}

// RenameFiles renames written paths to their original file paths.
func (r *Renamer) RenameFiles(
	writtenPaths <-chan *write.WrittenPath,
) error {
	if writtenPaths == nil {
		return nil
	}

	var allWrittenPaths []*write.WrittenPath // nolint: prealloc

	// Ensure all files can be rewritten before attempting to rename
	for writtenPath := range writtenPaths {
		if writtenPath.Err != nil {
			return writtenPath.Err
		}

		allWrittenPaths = append(allWrittenPaths, writtenPath)
	}

	if len(allWrittenPaths) == 0 {
		return nil
	}

	errCh := make(chan error)

	done := make(chan struct{})
	defer close(done)

	var waitGroup sync.WaitGroup

	for _, writtenPath := range allWrittenPaths {
		waitGroup.Add(1)

		writtenPath := writtenPath

		go func() {
			defer waitGroup.Done()

			if err := os.Rename(
				writtenPath.Path, writtenPath.OriginalPath,
			); err != nil {
				select {
				case <-done:
				case errCh <- err:
				}

				return
			}
		}()
	}

	go func() {
		waitGroup.Wait()
		close(errCh)
	}()

	for err := range errCh {
		return err
	}

	return nil
}
