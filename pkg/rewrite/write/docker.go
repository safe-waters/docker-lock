package write

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type dockerfileWriter struct {
	kind        kind.Kind
	excludeTags bool
}

// NewDockerfileWriter returns an IWriter for Dockerfiles.
func NewDockerfileWriter(excludeTags bool) IWriter {
	return &dockerfileWriter{
		kind:        kind.Dockerfile,
		excludeTags: excludeTags,
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
	outputDir string,
	done <-chan struct{},
) <-chan IWrittenPath {
	var (
		writtenPaths = make(chan IWrittenPath)
		waitGroup    sync.WaitGroup
	)

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for path, images := range pathImages {
			path := path
			images := images

			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				writtenPath, err := d.writeFile(path, images, outputDir)
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
	outputDir string,
) (string, error) {
	pathByt, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	if _, err = parser.Parse(bytes.NewBuffer(pathByt)); err != nil {
		return "", fmt.Errorf(
			"'%s' failed to parse with err: %v", path, err,
		)
	}

	var (
		scanner      = bufio.NewScanner(bytes.NewBuffer(pathByt))
		stageNames   = map[string]bool{}
		imageIndex   int
		outputBuffer bytes.Buffer
		outputLine   string
	)

	const instructionIndex = 0 // for instance, FROM is an instruction

	for scanner.Scan() {
		outputLine = fmt.Sprintf("%s%s", outputLine, scanner.Text())
		fields := strings.Fields(outputLine)

		if len(fields) > 1 &&
			strings.ToLower(fields[instructionIndex]) == "from" {
			if fields[len(fields)-1] == "\\" {
				fields = fields[:len(fields)-1]
				outputLine = fmt.Sprintf("%s ", strings.Join(fields, " "))

				continue
			}

			// FROM instructions may take the form:
			// FROM <image>
			// FROM --platform <image>
			// FROM <image> AS <stage>
			// FROM --platform <image> AS <stage>
			// FROM <stage> AS <another stage>
			// FROM --platform <stage> AS <another stage>
			var (
				imageLineIndex = 1
				stageIndex     = 3
				maxNumFields   = 4
			)

			if strings.HasPrefix(fields[1], "--") {
				imageLineIndex++
				stageIndex++
				maxNumFields++
			}

			if len(fields) > imageLineIndex {
				imageLine := fields[imageLineIndex]

				if !stageNames[imageLine] {
					if imageIndex >= len(images) {
						return "", fmt.Errorf(
							"more images exist in '%s' than in the Lockfile",
							path,
						)
					}

					image, ok := images[imageIndex].(map[string]interface{})
					if !ok {
						return "", errors.New("malformed image")
					}

					tag, ok := image["tag"].(string)
					if !ok {
						return "", errors.New("malformed 'tag' in image")
					}

					if d.excludeTags {
						tag = ""
					}

					name, ok := image["name"].(string)
					if !ok {
						return "", errors.New("malformed 'name' in image")
					}

					digest, ok := image["digest"].(string)
					if !ok {
						return "", errors.New("malformed 'digest' in image")
					}

					replacementImageLine := parse.NewImage(
						kind.Dockerfile, name, tag, digest, nil, nil,
					).ImageLine()

					fields[imageLineIndex] = replacementImageLine
					imageIndex++
				}

				// Ensure stage is added to the stage name set:
				// FROM <image> AS <stage>

				// Ensure another stage is added to the stage name set:
				// FROM <stage> AS <another stage>
				if len(fields) == maxNumFields {
					stageNames[fields[stageIndex]] = true
				}
			}

			outputLine = strings.Join(fields, " ")
		}

		outputBuffer.WriteString(fmt.Sprintf("%s\n", outputLine))

		outputLine = ""
	}

	if imageIndex < len(images) {
		return "", fmt.Errorf(
			"fewer images exist in '%s' than asked to rewrite", path,
		)
	}

	replacer := strings.NewReplacer("/", "-", "\\", "-")
	outputPath := replacer.Replace(fmt.Sprintf("%s-*", path))

	writtenFile, err := ioutil.TempFile(outputDir, outputPath)
	if err != nil {
		return "", err
	}
	defer writtenFile.Close()

	if _, err = writtenFile.Write(outputBuffer.Bytes()); err != nil {
		return "", err
	}

	return writtenFile.Name(), err
}
