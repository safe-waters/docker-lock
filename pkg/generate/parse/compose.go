package parse

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/types"
	"github.com/docker/cli/opts"
)

// ComposefileImageParser extracts image values from docker-compose files
// and Dockerfiles referenced by those docker-compose files.
type ComposefileImageParser struct {
	DockerfileImageParser *DockerfileImageParser
}

// IComposefileImageParser provides an interface for ComposefileImageParser's
// exported methods.
type IComposefileImageParser interface {
	ParseFiles(
		paths <-chan string,
		done <-chan struct{},
	) <-chan *ComposefileImage
}

// ComposefileImage annotates an image with data about the docker-compose file
// and/or the Dockerfile from which it was parsed.
type ComposefileImage struct {
	*Image
	DockerfilePath string `json:"dockerfile,omitempty"`
	Position       int    `json:"-"`
	ServiceName    string `json:"service"`
	Path           string `json:"-"`
	Err            error  `json:"-"`
}

// NewComposefileImageParser returns a ComposefileImageParser after validating
// its fields.
func NewComposefileImageParser(
	dockerfileImageParser *DockerfileImageParser,
) (*ComposefileImageParser, error) {
	if dockerfileImageParser == nil {
		return nil, errors.New("dockerfileImageParser cannot be nil")
	}

	return &ComposefileImageParser{
		DockerfileImageParser: dockerfileImageParser,
	}, nil
}

// ParseFiles reads docker-compose YAML to parse all images
// referenced services.
func (c *ComposefileImageParser) ParseFiles(
	paths <-chan string,
	done <-chan struct{},
) <-chan *ComposefileImage {
	if paths == nil {
		return nil
	}

	composefileImages := make(chan *ComposefileImage)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for path := range paths {
			waitGroup.Add(1)

			go c.parseFile(
				path, composefileImages, done, &waitGroup,
			)
		}
	}()

	go func() {
		waitGroup.Wait()
		close(composefileImages)
	}()

	return composefileImages
}

func (c *ComposefileImageParser) parseFile(
	path string,
	composefileImages chan<- *ComposefileImage,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	byt, err := ioutil.ReadFile(path)
	if err != nil {
		select {
		case <-done:
		case composefileImages <- &ComposefileImage{Err: err}:
		}

		return
	}

	composefileData, err := loader.ParseYAML(byt)
	if err != nil {
		select {
		case <-done:
		case composefileImages <- &ComposefileImage{Err: err}:
		}

		return
	}

	envVars := map[string]string{}

	for _, envVarStr := range os.Environ() {
		envVarVal := strings.SplitN(envVarStr, "=", 2)
		envVars[envVarVal[0]] = envVarVal[1]
	}

	var envFileVars []string

	if envFileVars, err = opts.ParseEnvFile(
		filepath.Join(filepath.Dir(path), ".env"),
	); err == nil {
		for _, envVarStr := range envFileVars {
			envVarVal := strings.SplitN(envVarStr, "=", 2)
			if _, ok := envVars[envVarVal[0]]; !ok {
				envVars[envVarVal[0]] = envVarVal[1]
			}
		}
	}

	loadedComposefile, err := loader.Load(
		types.ConfigDetails{
			ConfigFiles: []types.ConfigFile{
				{
					Config:   composefileData,
					Filename: path,
				},
			},
			// replaces env vars with $ in file
			Environment: envVars,
		},
	)
	if err != nil {
		select {
		case <-done:
		case composefileImages <- &ComposefileImage{Err: err}:
		}

		return
	}

	for _, serviceConfig := range loadedComposefile.Services {
		waitGroup.Add(1)

		go c.parseService(
			serviceConfig, path, envVars, composefileImages, waitGroup, done,
		)
	}
}

func (c *ComposefileImageParser) parseService(
	serviceConfig types.ServiceConfig,
	path string,
	envVars map[string]string,
	composefileImages chan<- *ComposefileImage,
	waitGroup *sync.WaitGroup,
	done <-chan struct{},
) {
	defer waitGroup.Done()

	if serviceConfig.Build.Context == "" {
		image := convertImageLineToImage(serviceConfig.Image)

		select {
		case <-done:
		case composefileImages <- &ComposefileImage{
			Image:       image,
			ServiceName: serviceConfig.Name,
			Path:        path,
		}:
		}

		return
	}

	dockerfileImages := make(chan *DockerfileImage)

	var dockerfileImageWaitGroup sync.WaitGroup

	dockerfileImageWaitGroup.Add(1)

	go func() {
		defer dockerfileImageWaitGroup.Done()

		context := serviceConfig.Build.Context
		if !filepath.IsAbs(context) {
			context = filepath.Join(filepath.Dir(path), context)
		}

		dockerfile := serviceConfig.Build.Dockerfile
		if dockerfile == "" {
			dockerfile = "Dockerfile"
		}

		dockerfilePath := filepath.Join(context, dockerfile)

		buildArgs := map[string]string{}

		for arg, val := range serviceConfig.Build.Args {
			if val == nil {
				// For the case:
				//	args:
				//	  - MYENVVAR
				// where MYENVVAR does not have $ in front
				buildArgs[arg] = envVars[arg]
			} else {
				buildArgs[arg] = *val
			}
		}

		dockerfileImageWaitGroup.Add(1)

		go c.DockerfileImageParser.parseFile(
			dockerfilePath, buildArgs, dockerfileImages,
			done, &dockerfileImageWaitGroup,
		)
	}()

	go func() {
		dockerfileImageWaitGroup.Wait()
		close(dockerfileImages)
	}()

	for dockerfileImage := range dockerfileImages {
		if dockerfileImage.Err != nil {
			select {
			case <-done:
			case composefileImages <- &ComposefileImage{
				Err: dockerfileImage.Err,
			}:
			}

			return
		}

		select {
		case <-done:
			return
		case composefileImages <- &ComposefileImage{
			Image:          dockerfileImage.Image,
			DockerfilePath: dockerfileImage.Path,
			Position:       dockerfileImage.Position,
			ServiceName:    serviceConfig.Name,
			Path:           path,
		}:
		}
	}
}
