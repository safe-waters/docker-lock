// Package write provides functionality to write files with image digests.
package write

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
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
func (d *DockerfileWriter) WriteFiles( // nolint: dupl
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
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	loadedDockerfile, err := parser.Parse(f)
	if err != nil {
		return "", err
	}

	stageNames := map[string]bool{}

	var imageIndex int

	const maxNumFields = 3

	var (
		outputBuffer bytes.Buffer
		lastEndLine  int
	)

	for _, child := range loadedDockerfile.AST.Children {
		outputLine := child.Original

		if child.Value == "from" {
			var raw []string
			for n := child.Next; n != nil; n = n.Next {
				raw = append(raw, n.Value)
			}

			if len(raw) == 0 {
				return "", fmt.Errorf(
					"invalid from instruction in Dockerfile '%s'", path,
				)
			}

			if !stageNames[raw[0]] {
				if imageIndex >= len(images) {
					return "", fmt.Errorf(
						"more images exist in '%s' than in the Lockfile",
						path,
					)
				}

				replacementImageLine := convertImageToImageLine(
					images[imageIndex].Image, d.ExcludeTags,
				)

				raw[0] = replacementImageLine
				imageIndex++
			}
			// Ensure stage is added to the stage name set:
			// FROM <image> AS <stage>

			// Ensure another stage is added to the stage name set:
			// FROM <stage> AS <another stage>
			if len(raw) == maxNumFields {
				const stageIndex = maxNumFields - 1

				stageNames[raw[stageIndex]] = true
			}

			outputLine = d.formatASTLine(child, raw)
		}

		expectedLineNo := lastEndLine + len(child.PrevComment) + 1
		if expectedLineNo != child.StartLine {
			newlines := strings.Repeat("\n", child.StartLine-expectedLineNo)
			outputBuffer.WriteString(newlines)
		}

		lastEndLine = child.EndLine

		for _, comment := range child.PrevComment {
			fmt.Fprintf(&outputBuffer, "# %s\n", comment)
		}

		outputBuffer.WriteString(fmt.Sprintf("%s\n", outputLine))
	}

	if imageIndex < len(images) {
		return "", fmt.Errorf(
			"fewer images exist in '%s' than asked to rewrite", path,
		)
	}

	replacer := strings.NewReplacer("/", "-", "\\", "-")
	tempPath := replacer.Replace(fmt.Sprintf("%s-*", path))

	writtenFile, err := ioutil.TempFile(d.Directory, tempPath)
	if err != nil {
		return "", err
	}
	defer writtenFile.Close()

	if _, err = writtenFile.Write(outputBuffer.Bytes()); err != nil {
		return "", err
	}

	return writtenFile.Name(), err
}

func (d *DockerfileWriter) formatASTLine(
	child *parser.Node, raw []string,
) string {
	line := []string{strings.ToUpper(child.Value)}
	if child.Flags != nil {
		line = append(line, child.Flags...)
	}

	line = append(line, raw...)

	return strings.Join(line, " ")
}

func convertImageToImageLine(image *parse.Image, excludeTags bool) string {
	imageLine := image.Name

	if image.Tag != "" && !excludeTags {
		imageLine = fmt.Sprintf("%s:%s", imageLine, image.Tag)
	}

	if image.Digest != "" {
		imageLine = fmt.Sprintf("%s@sha256:%s", imageLine, image.Digest)
	}

	return imageLine
}
