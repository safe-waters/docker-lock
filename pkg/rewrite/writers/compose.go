package writers

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"gopkg.in/yaml.v2"
)

// ComposefileWriter contains information for writing new Composefiles.
type ComposefileWriter struct {
	DockerfileWriter *DockerfileWriter
	ExcludeTags      bool
	Directory        string
}

// IComposefileWriter provides an interface for ComposefileWriter's
// exported methods.
type IComposefileWriter interface {
	WriteFiles(
		pathImages map[string][]*parse.ComposefileImage,
		done <-chan struct{},
	) <-chan *WrittenPath
}

type filteredDockerfilePathImages struct {
	dockerfilePathImages map[string][]*parse.DockerfileImage
	err                  error
}

// WriteFiles writes new Composefiles and Dockerfiles referenced by the
// Composefiles given the paths of the original Composefiles
// and new images that should replace the exsting ones.
func (c *ComposefileWriter) WriteFiles(
	pathImages map[string][]*parse.ComposefileImage,
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

		if c.DockerfileWriter != nil {
			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				dockerfilePathImages, err := c.filterDockerfilePathImages(
					pathImages,
				)
				if err != nil {
					select {
					case <-done:
					case writtenPaths <- &WrittenPath{Err: err}:
					}

					return
				}

				if len(dockerfilePathImages) != 0 {
					dockerfileWrittenPaths := c.DockerfileWriter.WriteFiles(
						dockerfilePathImages, done,
					)

					for writtenPath := range dockerfileWrittenPaths {
						if writtenPath.Err != nil {
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
		}

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			composefileWrittenPaths := c.writeComposefiles(
				pathImages, done,
			)

			for writtenPath := range composefileWrittenPaths {
				if writtenPath.Err != nil {
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

func (c *ComposefileWriter) writeComposefiles(
	pathImages map[string][]*parse.ComposefileImage,
	done <-chan struct{},
) <-chan *WrittenPath {
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

				writtenPath, err := c.writeFile(path, images)
				if err != nil {
					select {
					case <-done:
					case writtenPaths <- &WrittenPath{Err: err}:
					}

					return
				}

				if writtenPath != "" {
					select {
					case <-done:
						return
					case writtenPaths <- &WrittenPath{
						OriginalPath: path,
						Path:         writtenPath,
					}:
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

func (c *ComposefileWriter) writeFile(
	path string,
	images []*parse.ComposefileImage,
) (string, error) {
	pathByt, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	serviceImageLines, err := c.filterComposefileServices(pathByt, images)
	if err != nil {
		return "", fmt.Errorf("in '%s', %s", path, err)
	}

	if len(serviceImageLines) == 0 {
		return "", nil
	}

	var serviceName string

	var numServicesWritten int

	inputBuffer := bytes.NewBuffer(pathByt)

	var outputBuffer bytes.Buffer

	scanner := bufio.NewScanner(inputBuffer)
	for scanner.Scan() {
		inputLine := scanner.Text()
		outputLine := inputLine
		possibleServiceName := strings.Trim(inputLine, " :")

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

	writtenFile, err := ioutil.TempFile(c.Directory, "")
	if err != nil {
		return "", err
	}
	defer writtenFile.Close()

	if _, err = writtenFile.Write(outputBuffer.Bytes()); err != nil {
		return "", err
	}

	return writtenFile.Name(), err
}

func (c *ComposefileWriter) filterComposefileServices(
	pathByt []byte,
	images []*parse.ComposefileImage,
) (map[string]string, error) {
	comp := compose{}
	if err := yaml.Unmarshal(pathByt, &comp); err != nil {
		return nil, err
	}

	uniqueServices := map[string]struct{}{}

	serviceImageLines := map[string]string{}

	for _, image := range images {
		if _, ok := comp.Services[image.ServiceName]; !ok {
			return nil, fmt.Errorf(
				"'%s' service does not exist", image.ServiceName,
			)
		}

		if image.DockerfilePath == "" {
			if _, ok := serviceImageLines[image.ServiceName]; ok {
				return nil, fmt.Errorf(
					"multiple images exist for the same service '%s'",
					image.ServiceName,
				)
			}

			serviceImageLines[image.ServiceName] = c.convertImageToImageLine(
				image.Image,
			)
		}

		uniqueServices[image.ServiceName] = struct{}{}
	}

	if len(comp.Services) != len(uniqueServices) {
		return nil, fmt.Errorf(
			"'%d' service(s) exist, yet asked to rewrite '%d'",
			len(comp.Services), len(uniqueServices),
		)
	}

	return serviceImageLines, nil
}

func (c *ComposefileWriter) filterDockerfilePathImages(
	pathImages map[string][]*parse.ComposefileImage,
) (
	map[string][]*parse.DockerfileImage,
	error,
) {
	filteredCh := make(chan *filteredDockerfilePathImages)
	done := make(chan struct{})

	defer close(done)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for _, allImages := range pathImages {
			allImages := allImages

			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				serviceDockerfileImages := map[string][]*parse.DockerfileImage{}

				for _, image := range allImages {
					if image.DockerfilePath != "" {
						dockerfilePath := image.DockerfilePath

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
						}

						serviceDockerfileImages[image.ServiceName] = append(
							serviceDockerfileImages[image.ServiceName],
							&parse.DockerfileImage{
								Image: image.Image,
								Path:  dockerfilePath,
							},
						)
					}
				}

				for _, images := range serviceDockerfileImages {
					dockerfilePathImages := map[string][]*parse.DockerfileImage{} // nolint: lll

					for _, image := range images {
						dockerfilePathImages[image.Path] = append(
							dockerfilePathImages[image.Path],
							&parse.DockerfileImage{
								Image: image.Image,
								Path:  image.Path,
							},
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

	dockerfilePathImages := map[string][]*parse.DockerfileImage{}

	for filtered := range filteredCh {
		if filtered.err != nil {
			return nil, filtered.err
		}

		for path, images := range filtered.dockerfilePathImages {
			if existingImages, ok := dockerfilePathImages[path]; ok {
				if !reflect.DeepEqual(existingImages, images) {
					return nil, fmt.Errorf(
						"multiple services reference the same Dockerfile '%s' with different images", // nolint: lll
						path,
					)
				}
			} else {
				dockerfilePathImages[path] = images
			}
		}
	}

	return dockerfilePathImages, nil
}

func (c *ComposefileWriter) convertAbsToRelPath(
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

func (c *ComposefileWriter) convertImageToImageLine(image *parse.Image) string {
	switch {
	case image.Tag == "" || c.ExcludeTags:
		return fmt.Sprintf(
			"%s@sha256:%s", image.Name, image.Digest,
		)
	default:
		return fmt.Sprintf(
			"%s:%s@sha256:%s", image.Name, image.Tag, image.Digest,
		)
	}
}
