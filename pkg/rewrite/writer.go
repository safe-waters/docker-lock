package rewrite

import (
	"errors"
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/rewrite/write"
)

// Writer is used to write files with their image digests.
type Writer struct {
	DockerfileWriter  write.IDockerfileWriter
	ComposefileWriter write.IComposefileWriter
}

// AnyPathImages contains any possible type of path and associated images.
type AnyPathImages struct {
	DockerfilePathImages  map[string][]*parse.DockerfileImage
	ComposefilePathImages map[string][]*parse.ComposefileImage
}

// IWriter provides an interface for Writer's exported methods.
type IWriter interface {
	WriteFiles(
		anyPathImages *AnyPathImages,
		done <-chan struct{},
	) <-chan *write.WrittenPath
}

// NewWriter returns a Writer after validating its fields.
func NewWriter(
	dockerfileWriter write.IDockerfileWriter,
	composefileWriter write.IComposefileWriter,
) (*Writer, error) {
	if (dockerfileWriter == nil ||
		reflect.ValueOf(dockerfileWriter).IsNil()) &&
		(composefileWriter == nil ||
			reflect.ValueOf(composefileWriter).IsNil()) {
		return nil, errors.New("at least one writer must not be nil")
	}

	return &Writer{
		DockerfileWriter:  dockerfileWriter,
		ComposefileWriter: composefileWriter,
	}, nil
}

// WriteFiles writes files with their image digests.
func (w *Writer) WriteFiles(
	anyPathImages *AnyPathImages,
	done <-chan struct{},
) <-chan *write.WrittenPath {
	if anyPathImages == nil {
		return nil
	}

	writtenPaths := make(chan *write.WrittenPath)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		if w.DockerfileWriter != nil &&
			!reflect.ValueOf(w.DockerfileWriter).IsNil() &&
			len(anyPathImages.DockerfilePathImages) != 0 {
			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				writtenPathsFromDockerfiles := w.DockerfileWriter.WriteFiles(
					anyPathImages.DockerfilePathImages, done,
				)

				for writtenPath := range writtenPathsFromDockerfiles {
					select {
					case <-done:
						return
					case writtenPaths <- writtenPath:
					}

					if writtenPath.Err != nil {
						return
					}
				}
			}()
		}

		if w.ComposefileWriter != nil &&
			!reflect.ValueOf(w.ComposefileWriter).IsNil() &&
			len(anyPathImages.ComposefilePathImages) != 0 {
			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				writtenPathsFromComposefiles := w.ComposefileWriter.WriteFiles(
					anyPathImages.ComposefilePathImages, done,
				)

				for writtenPath := range writtenPathsFromComposefiles {
					select {
					case <-done:
						return
					case writtenPaths <- writtenPath:
					}

					if writtenPath.Err != nil {
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
