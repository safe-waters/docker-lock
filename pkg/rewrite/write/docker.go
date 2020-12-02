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
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type dockerfileWriter struct {
	kind        kind.Kind
	excludeTags bool
	directory   string
}

// NewDockerfileWriter returns an IWriter for Dockerfiles.
func NewDockerfileWriter(excludeTags bool, directory string) IWriter {
	return &dockerfileWriter{
		kind:        kind.Dockerfile,
		excludeTags: excludeTags,
		directory:   directory,
	}
}

// Kind is a getter for the kind.
func (d *dockerfileWriter) Kind() kind.Kind {
	return d.kind
}

// WriteFiles writes new Dockerfiles given the paths of the original Dockerfiles
// and new images that should replace the exsting ones.
func (d *dockerfileWriter) WriteFiles( // nolint: dupl
	pathImages map[string][]interface{},
	done <-chan struct{},
) <-chan IWrittenPath {
	writtenPaths := make(chan IWrittenPath)

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
					case writtenPaths <- NewWrittenPath("", "", err):
					}

					return
				}

				select {
				case <-done:
					return
				case writtenPaths <- NewWrittenPath(path, writtenPath, nil):
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

func (d *dockerfileWriter) writeFile(
	path string,
	images []interface{},
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

	const maxNumFields = 3

	var (
		outputBuffer bytes.Buffer
		lastEndLine  int
		imageIndex   int
		stageNames   = map[string]bool{}
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

				image := images[imageIndex].(map[string]interface{})

				tag := image["tag"].(string)
				if d.excludeTags {
					tag = ""
				}

				replacementImageLine := parse.NewImage(
					kind.Dockerfile, image["name"].(string), tag,
					image["digest"].(string), nil, nil,
				).ImageLine()

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

	writtenFile, err := ioutil.TempFile(d.directory, tempPath)
	if err != nil {
		return "", err
	}
	defer writtenFile.Close()

	if _, err = writtenFile.Write(outputBuffer.Bytes()); err != nil {
		return "", err
	}

	return writtenFile.Name(), err
}

func (d *dockerfileWriter) formatASTLine(
	child *parser.Node, raw []string,
) string {
	line := []string{strings.ToUpper(child.Value)}
	if child.Flags != nil {
		line = append(line, child.Flags...)
	}

	line = append(line, raw...)

	return strings.Join(line, " ")
}
