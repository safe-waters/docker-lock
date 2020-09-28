// Package writers provides functionality to write files with image digests.
package writers

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
)

// DockerfileWriter contains information for writing new Dockerfiles.
type DockerfileWriter struct {
	ExcludeTags bool
	Directory   string
}

// WrittenPath contains information linking a newly written file and its
// original.
type WrittenPath struct {
	OriginalPath string
	Path         string
	Err          error
}

// IDockerfileWriter provides an interface for DockerfileWriter's exported
// methods.
type IDockerfileWriter interface {
	WriteFiles(
		pathImages map[string][]*parse.DockerfileImage,
		done <-chan struct{},
	) <-chan *WrittenPath
}

// WriteFiles writes new Dockerfiles given the paths of the original Dockerfiles
// and new images that should replace the exsting ones.
func (d *DockerfileWriter) WriteFiles(
	pathImages map[string][]*parse.DockerfileImage,
	done <-chan struct{},
) <-chan *WrittenPath {
	if len(pathImages) == 0 {
		return nil
	}

	writtenPaths := make(chan *WrittenPath)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for path, images := range pathImages {
			path := path
			images := images

			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				writtenPath, err := d.writeFile(path, images)
				if err != nil {
					select {
					case <-done:
					case writtenPaths <- &WrittenPath{Err: err}:
					}

					return
				}

				select {
				case <-done:
					return
				case writtenPaths <- &WrittenPath{
					OriginalPath: path,
					Path:         writtenPath,
				}:
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

func (d *DockerfileWriter) writeFile(
	path string,
	images []*parse.DockerfileImage,
) (string, error) {
	dockerfile, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer dockerfile.Close()

	scanner := bufio.NewScanner(dockerfile)

	stageNames := map[string]bool{}

	var imageIndex int

	var outputBuffer bytes.Buffer

	for scanner.Scan() {
		inputLine := scanner.Text()
		outputLine := inputLine
		fields := strings.Fields(inputLine)

		const instructionIndex = 0 // for instance, FROM is an instruction
		if len(fields) > 0 &&
			strings.ToLower(fields[instructionIndex]) == "from" {
			// FROM instructions may take the form:
			// FROM <image>
			// FROM <image> AS <stage>
			// FROM <stage> AS <another stage>
			// Only replace the image, never the stage.
			const imageLineIndex = 1

			imageLine := fields[imageLineIndex]

			if !stageNames[imageLine] {
				if imageIndex >= len(images) {
					return "", fmt.Errorf(
						"more images exist in '%s' than in the Lockfile",
						path,
					)
				}

				replacementImageLine := d.convertImageToImageLine(
					images[imageIndex].Image,
				)
				fields[imageLineIndex] = replacementImageLine
				imageIndex++
			}
			// Ensure stage is added to the stage name set:
			// FROM <image> AS <stage>

			// Ensure another stage is added to the stage name set:
			// FROM <stage> AS <another stage>
			const maxNumFields = 4
			if len(fields) == maxNumFields {
				const stageIndex = 3

				stageNames[fields[stageIndex]] = true
			}

			outputLine = strings.Join(fields, " ")
		}

		outputBuffer.WriteString(fmt.Sprintf("%s\n", outputLine))
	}

	if imageIndex < len(images) {
		return "", fmt.Errorf(
			"fewer images exist in '%s' than asked to rewrite", path,
		)
	}

	writtenFile, err := ioutil.TempFile(d.Directory, "")
	if err != nil {
		return "", err
	}
	defer writtenFile.Close()

	if _, err = writtenFile.Write(outputBuffer.Bytes()); err != nil {
		return "", err
	}

	return writtenFile.Name(), err
}

func (d *DockerfileWriter) convertImageToImageLine(image *parse.Image) string {
	switch {
	case image.Tag == "" || d.ExcludeTags:
		return fmt.Sprintf(
			"%s@sha256:%s", image.Name, image.Digest,
		)
	default:
		return fmt.Sprintf(
			"%s:%s@sha256:%s", image.Name, image.Tag, image.Digest,
		)
	}
}
