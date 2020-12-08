package rewrite

import (
	"errors"
	"os"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/rewrite/write"
)

type renamer struct{}

// NewRenamer returns an IRenamer.
func NewRenamer() IRenamer {
	return &renamer{}
}

// RenameFiles renames new paths in IWrittenPaths to their original paths.
func (r *renamer) RenameFiles(
	writtenPaths <-chan write.IWrittenPath,
) error {
	if writtenPaths == nil {
		return errors.New("'writtenPaths' cannot be nil")
	}

	var allWrittenPaths []write.IWrittenPath // nolint: prealloc

	// Ensure all files can be rewritten before attempting to rename
	for writtenPath := range writtenPaths {
		if writtenPath.Err() != nil {
			return writtenPath.Err()
		}

		allWrittenPaths = append(allWrittenPaths, writtenPath)
	}

	if len(allWrittenPaths) == 0 {
		return nil
	}

	var (
		waitGroup sync.WaitGroup
		errCh     = make(chan error)
		done      = make(chan struct{})
	)

	defer close(done)

	for _, writtenPath := range allWrittenPaths {
		waitGroup.Add(1)

		writtenPath := writtenPath

		go func() {
			defer waitGroup.Done()

			if err := os.Rename(
				writtenPath.NewPath(), writtenPath.OriginalPath(),
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
