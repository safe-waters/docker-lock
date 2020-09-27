package parse

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
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

	composeYmlByt, err := ioutil.ReadFile(path)
	if err != nil {
		select {
		case <-done:
		case composefileImages <- &ComposefileImage{Err: err}:
		}

		return
	}

	comp := compose{}
	if err := yaml.Unmarshal(composeYmlByt, &comp); err != nil {
		err = fmt.Errorf("from '%s': %v", path, err)

		select {
		case <-done:
		case composefileImages <- &ComposefileImage{Err: err}:
		}

		return
	}

	for svcName, svc := range comp.Services {
		waitGroup.Add(1)

		go c.parseService(
			svcName, svc, path, composefileImages, waitGroup, done,
		)
	}
}

func (c *ComposefileImageParser) parseService(
	svcName string,
	svc *service,
	path string,
	composefileImages chan<- *ComposefileImage,
	waitGroup *sync.WaitGroup,
	done <-chan struct{},
) {
	defer waitGroup.Done()

	if svc.BuildWrapper == nil {
		imageLine := os.ExpandEnv(svc.ImageName)
		image := convertImageLineToImage(imageLine)

		select {
		case <-done:
		case composefileImages <- &ComposefileImage{
			Image:       image,
			ServiceName: svcName,
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

		switch build := svc.BuildWrapper.Build.(type) {
		case simple:
			context := filepath.FromSlash(
				filepath.ToSlash(os.ExpandEnv(string(build))),
			)
			if !filepath.IsAbs(context) {
				context = filepath.Join(filepath.Dir(path), context)
			}

			dockerfilePath := filepath.Join(context, "Dockerfile")

			dockerfileImageWaitGroup.Add(1)

			go c.DockerfileImageParser.parseFile(
				dockerfilePath, nil, dockerfileImages,
				done, &dockerfileImageWaitGroup,
			)
		case verbose:
			context := filepath.FromSlash(
				filepath.ToSlash(os.ExpandEnv(build.Context)),
			)
			if !filepath.IsAbs(context) {
				context = filepath.Join(filepath.Dir(path), context)
			}

			dockerfilePath := filepath.FromSlash(
				filepath.ToSlash(os.ExpandEnv(build.DockerfilePath)),
			)
			if dockerfilePath == "" {
				dockerfilePath = "Dockerfile"
			}

			dockerfilePath = filepath.Join(context, dockerfilePath)

			buildArgs := c.parseBuildArgs(build)

			dockerfileImageWaitGroup.Add(1)

			go c.DockerfileImageParser.parseFile(
				dockerfilePath, buildArgs, dockerfileImages,
				done, &dockerfileImageWaitGroup,
			)
		}
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
			ServiceName:    svcName,
			Path:           path,
		}:
		}
	}
}

func (c *ComposefileImageParser) parseBuildArgs(
	build verbose,
) map[string]string {
	buildArgs := map[string]string{}

	if build.ArgsWrapper == nil {
		return buildArgs
	}

	switch args := build.ArgsWrapper.Args.(type) {
	case argsMap:
		for k, v := range args {
			arg := os.ExpandEnv(k)
			val := os.ExpandEnv(v)
			buildArgs[arg] = val
		}
	case argsSlice:
		for _, argValStr := range args {
			argValSl := strings.SplitN(argValStr, "=", 2)
			arg := os.ExpandEnv(argValSl[0])

			const argOnlyLen = 1

			switch len(argValSl) {
			case argOnlyLen:
				buildArgs[arg] = os.Getenv(arg)
			default:
				val := os.ExpandEnv(argValSl[1])
				buildArgs[arg] = val
			}
		}
	}

	return buildArgs
}
