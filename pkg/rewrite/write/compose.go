package write

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/docker/cli/cli/compose/loader"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
	"gopkg.in/yaml.v2"
)

type composefileWriter struct {
	kind             kind.Kind
	dockerfileWriter IWriter
	excludeTags      bool
	directory        string
}

type filteredDockerfilePathImages struct {
	dockerfilePathImages map[string][]interface{}
	err                  error
}

// NewComposefileWriter returns an IWriter for Composefiles. dockerfileWriter
// cannot be nil as it handles writing Dockerfiles referenced by Composefiles.
func NewComposefileWriter(
	dockerfileWriter IWriter,
	excludeTags bool,
	directory string,
) (IWriter, error) {
	if dockerfileWriter == nil || reflect.ValueOf(dockerfileWriter).IsNil() {
		return nil, errors.New("dockerfileWriter cannot be nil")
	}

	return &composefileWriter{
		kind:             kind.Composefile,
		dockerfileWriter: dockerfileWriter,
		excludeTags:      excludeTags,
		directory:        directory,
	}, nil
}

// Kind is a getter for the kind.
func (c *composefileWriter) Kind() kind.Kind {
	return c.kind
}

// WriteFiles writes new Composefiles and Dockerfiles referenced by the
// Composefiles given the paths of the original Composefiles
// and new images that should replace the exsting ones.
func (c *composefileWriter) WriteFiles(
	pathImages map[string][]interface{},
	done <-chan struct{},
) <-chan IWrittenPath {
	var (
		waitGroup    sync.WaitGroup
		writtenPaths = make(chan IWrittenPath)
	)

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			dockerfilePathImages, err := c.filterDockerfilePathImages(
				pathImages,
			)
			if err != nil {
				select {
				case <-done:
				case writtenPaths <- NewWrittenPath("", "", err):
				}

				return
			}

			if len(dockerfilePathImages) != 0 {
				for writtenPath := range c.dockerfileWriter.WriteFiles(
					dockerfilePathImages, done,
				) {
					if writtenPath.Err() != nil {
						select {
						case <-done:
						case writtenPaths <- writtenPath:
						}

						return
					}

					select {
					case <-done:
						return
					case writtenPaths <- writtenPath:
					}
				}
			}
		}()

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			for writtenPath := range c.writeComposefiles(
				pathImages, done,
			) {
				if writtenPath.Err() != nil {
					select {
					case <-done:
					case writtenPaths <- writtenPath:
					}

					return
				}

				select {
				case <-done:
					return
				case writtenPaths <- writtenPath:
				}
			}
		}()
	}()

	go func() {
		waitGroup.Wait()
		close(writtenPaths)
	}()

	return writtenPaths
}

func (c *composefileWriter) writeComposefiles(
	pathImages map[string][]interface{},
	done <-chan struct{},
) <-chan IWrittenPath {
	var (
		waitGroup    sync.WaitGroup
		writtenPaths = make(chan IWrittenPath)
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

				writtenPath, err := c.writeFile(path, images)
				if err != nil {
					select {
					case <-done:
					case writtenPaths <- NewWrittenPath("", "", err):
					}

					return
				}

				if writtenPath != "" {
					select {
					case <-done:
						return
					case writtenPaths <- NewWrittenPath(path, writtenPath, nil):
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

func (c *composefileWriter) writeFile(
	path string,
	images []interface{},
) (string, error) {
	pathByt, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	if _, err = loader.ParseYAML(pathByt); err != nil {
		return "", err
	}

	serviceImageLines, err := c.filterComposefileServices(pathByt, images)
	if err != nil {
		return "", fmt.Errorf("in '%s', %s", path, err)
	}

	if len(serviceImageLines) == 0 {
		return "", nil
	}

	var (
		serviceName        string
		numServicesWritten int
		outputBuffer       bytes.Buffer
		scanner            = bufio.NewScanner(bytes.NewBuffer(pathByt))
	)

	for scanner.Scan() {
		var (
			inputLine           = scanner.Text()
			outputLine          = inputLine
			possibleServiceName = strings.Trim(inputLine, " :")
		)

		switch {
		case serviceImageLines[possibleServiceName] != "":
			serviceName = possibleServiceName
		case serviceName != "" &&
			strings.HasPrefix(strings.TrimLeft(inputLine, " "), "image:"):
			imageIndex := strings.Index(inputLine, "image:")

			outputLine = fmt.Sprintf(
				"%s %s", inputLine[:imageIndex+len("image:")],
				serviceImageLines[serviceName],
			)

			serviceName = ""

			numServicesWritten++
		}

		outputBuffer.WriteString(fmt.Sprintf("%s\n", outputLine))
	}

	if numServicesWritten != len(serviceImageLines) {
		return "", fmt.Errorf(
			"in '%s' '%d' images rewritten, but expected to rewrite '%d'",
			path, numServicesWritten, len(serviceImageLines),
		)
	}

	replacer := strings.NewReplacer("/", "-", "\\", "-")
	tempPath := replacer.Replace(fmt.Sprintf("%s-*", path))

	writtenFile, err := ioutil.TempFile(c.directory, tempPath)
	if err != nil {
		return "", err
	}
	defer writtenFile.Close()

	if _, err = writtenFile.Write(outputBuffer.Bytes()); err != nil {
		return "", err
	}

	return writtenFile.Name(), err
}

func (c *composefileWriter) filterComposefileServices(
	pathByt []byte,
	images []interface{},
) (map[string]string, error) {
	comp := compose{}
	if err := yaml.Unmarshal(pathByt, &comp); err != nil {
		return nil, err
	}

	var (
		uniqueServices    = map[string]struct{}{}
		serviceImageLines = map[string]string{}
	)

	for _, image := range images {
		image, ok := image.(map[string]interface{})
		if !ok {
			return nil, errors.New("malformed image")
		}

		serviceName, ok := image["service"].(string)
		if !ok {
			return nil, errors.New("malformed 'service' in image")
		}

		if _, ok := comp.Services[serviceName]; !ok {
			return nil, fmt.Errorf(
				"'%s' service does not exist", serviceName,
			)
		}

		if image["dockerfile"] == nil {
			if _, ok := serviceImageLines[serviceName]; ok {
				return nil, fmt.Errorf(
					"multiple images exist for the same service '%s'",
					serviceName,
				)
			}

			tag, ok := image["tag"].(string)
			if !ok {
				return nil, errors.New("malformed 'tag' in image")
			}

			if c.excludeTags {
				tag = ""
			}

			name, ok := image["name"].(string)
			if !ok {
				return nil, errors.New("malformed 'name' in image")
			}

			digest, ok := image["digest"].(string)
			if !ok {
				return nil, errors.New("malformed 'digest' in image")
			}

			imageLine := parse.NewImage(
				c.kind, name, tag, digest, nil, nil,
			).ImageLine()
			serviceImageLines[serviceName] = imageLine
		}

		uniqueServices[serviceName] = struct{}{}
	}

	if len(comp.Services) != len(uniqueServices) {
		return nil, fmt.Errorf(
			"'%d' service(s) exist, yet asked to rewrite '%d'",
			len(comp.Services), len(uniqueServices),
		)
	}

	return serviceImageLines, nil
}

func (c *composefileWriter) filterDockerfilePathImages(
	pathImages map[string][]interface{},
) (map[string][]interface{}, error) {
	var (
		filteredCh = make(chan *filteredDockerfilePathImages)
		done       = make(chan struct{})
		waitGroup  sync.WaitGroup
	)

	defer close(done)

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for _, images := range pathImages {
			images := images

			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				serviceDockerfileImages := map[string][]interface{}{}

				for _, image := range images {
					image, ok := image.(map[string]interface{})
					if !ok {
						select {
						case <-done:
						case filteredCh <- &filteredDockerfilePathImages{
							err: errors.New("malformed image"),
						}:
						}

						return
					}

					if image["dockerfile"] != nil {
						dockerfilePath, ok := image["dockerfile"].(string)
						if !ok {
							select {
							case <-done:
							case filteredCh <- &filteredDockerfilePathImages{
								err: errors.New(
									"malformed 'dockerfile' in image",
								),
							}:
							}

							return
						}

						if filepath.IsAbs(dockerfilePath) {
							var err error

							dockerfilePath, err = c.convertAbsToRelPath(
								dockerfilePath,
							)
							if err != nil {
								select {
								case <-done:
								case filteredCh <- &filteredDockerfilePathImages{ // nolint: lll
									err: err,
								}:
								}

								return
							}

							image["dockerfile"] = dockerfilePath
						}

						serviceName, ok := image["service"].(string)
						if !ok {
							select {
							case <-done:
							case filteredCh <- &filteredDockerfilePathImages{
								err: errors.New("malformed 'service' in image"),
							}:
							}

							return
						}

						serviceDockerfileImages[serviceName] = append(
							serviceDockerfileImages[serviceName], image,
						)
					}
				}

				for _, images := range serviceDockerfileImages {
					dockerfilePathImages := map[string][]interface{}{}

					for _, image := range images {
						image, ok := image.(map[string]interface{})
						if !ok {
							select {
							case <-done:
							case filteredCh <- &filteredDockerfilePathImages{
								err: errors.New("malformed image"),
							}:
							}

							return
						}

						dockerfilePath, ok := image["dockerfile"].(string)
						if !ok {
							select {
							case <-done:
							case filteredCh <- &filteredDockerfilePathImages{
								err: errors.New(
									"malformed 'dockerfile' in image",
								),
							}:
							}

							return
						}

						dockerfilePathImages[dockerfilePath] = append(
							dockerfilePathImages[dockerfilePath], image,
						)
					}

					select {
					case <-done:
						return
					case filteredCh <- &filteredDockerfilePathImages{
						dockerfilePathImages: dockerfilePathImages,
					}:
					}
				}
			}()
		}
	}()

	go func() {
		waitGroup.Wait()
		close(filteredCh)
	}()

	dockerfilePathImages := map[string][]interface{}{}

	for filtered := range filteredCh {
		if filtered.err != nil {
			return nil, filtered.err
		}

		for path, images := range filtered.dockerfilePathImages {
			if existingImages, ok := dockerfilePathImages[path]; ok {
				if len(existingImages) != len(images) {
					return nil, fmt.Errorf(
						"multiple services reference the same Dockerfile"+
							"'%s' with different images",
						path,
					)
				}

				for i := 0; i < len(existingImages); i++ {
					existingImage, ok := existingImages[i].(map[string]interface{}) // nolint: lll
					if !ok {
						return nil, errors.New("malformed image")
					}

					image, ok := images[i].(map[string]interface{})
					if !ok {
						return nil, errors.New("malformed image")
					}

					if existingImage["name"] != image["name"] ||
						existingImage["tag"] != image["tag"] ||
						existingImage["digest"] != image["digest"] {
						return nil, fmt.Errorf(
							"multiple services reference the same Dockerfile"+
								" '%s' with different images",
							path,
						)
					}
				}
			} else {
				dockerfilePathImages[path] = images
			}
		}
	}

	return dockerfilePathImages, nil
}

func (c *composefileWriter) convertAbsToRelPath(
	path string,
) (string, error) {
	currentWorkingDirectory, err := os.Getwd()
	if err != nil {
		return "", err
	}

	relativePath, err := filepath.Rel(
		currentWorkingDirectory, filepath.FromSlash(path),
	)
	if err != nil {
		return "", err
	}

	return filepath.ToSlash(relativePath), nil
}
