package parse

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/types"
	"github.com/docker/cli/opts"
	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type composefileImageParser struct {
	kind                  kind.Kind
	dockerfileImageParser IDockerfileImageParser
}

// NewComposefileImageParser returns an IImageParser for Composefiles.
// dockerfileImageParser cannot be nil as it is responsible for parsing
// Dockerfiles referenced by Composefiles.
func NewComposefileImageParser(
	dockerfileImageParser IDockerfileImageParser,
) (IComposefileImageParser, error) {
	if dockerfileImageParser == nil ||
		reflect.ValueOf(dockerfileImageParser).IsNil() {
		return nil, errors.New("dockerfileImageParser cannot be nil")
	}

	return &composefileImageParser{
		kind:                  kind.Composefile,
		dockerfileImageParser: dockerfileImageParser,
	}, nil
}

// Kind is a getter for the kind.
func (c *composefileImageParser) Kind() kind.Kind {
	return c.kind
}

// ParseFiles parses IImages from Composefiles.
func (c *composefileImageParser) ParseFiles(
	paths <-chan collect.IPath,
	done <-chan struct{},
) <-chan IImage {
	composefileImages := make(chan IImage)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for path := range paths {
			waitGroup.Add(1)

			go c.ParseFile(
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

// ParseFile parses IImages from a Composefile.
func (c *composefileImageParser) ParseFile(
	path collect.IPath,
	composefileImages chan<- IImage,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	if path.Err() != nil {
		select {
		case <-done:
		case composefileImages <- NewImage(c.kind, "", "", "", nil, path.Err()):
		}

		return
	}

	byt, err := ioutil.ReadFile(path.Val())
	if err != nil {
		select {
		case <-done:
		case composefileImages <- NewImage(c.kind, "", "", "", nil, err):
		}

		return
	}

	composefileData, err := loader.ParseYAML(byt)
	if err != nil {
		select {
		case <-done:
		case composefileImages <- NewImage(c.kind, "", "", "", nil, err):
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
		filepath.Join(filepath.Dir(path.Val()), ".env"),
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
					Filename: path.Val(),
				},
			},
			// replaces env vars with $ in file
			Environment: envVars,
		},
	)
	if err != nil {
		select {
		case <-done:
		case composefileImages <- NewImage(c.kind, "", "", "", nil, err):
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

func (c *composefileImageParser) parseService(
	serviceConfig types.ServiceConfig,
	path collect.IPath,
	envVars map[string]string,
	composefileImages chan<- IImage,
	waitGroup *sync.WaitGroup,
	done <-chan struct{},
) {
	defer waitGroup.Done()

	if serviceConfig.Build.Context == "" {
		image := NewImage(c.kind, "", "", "", map[string]interface{}{
			"serviceName":     serviceConfig.Name,
			"servicePosition": 0,
			"path":            path.Val(),
		}, nil)

		image.SetNameTagDigestFromImageLine(serviceConfig.Image)

		select {
		case <-done:
		case composefileImages <- image:
		}

		return
	}

	dockerfileImages := make(chan IImage)

	var dockerfileImageWaitGroup sync.WaitGroup

	dockerfileImageWaitGroup.Add(1)

	go func() {
		defer dockerfileImageWaitGroup.Done()

		context := serviceConfig.Build.Context
		if !filepath.IsAbs(context) {
			context = filepath.Join(filepath.Dir(path.Val()), context)
		}

		dockerfile := serviceConfig.Build.Dockerfile
		if dockerfile == "" {
			dockerfile = "Dockerfile"
		}

		dockerfilePath := collect.NewPath(
			c.kind, filepath.Join(context, dockerfile), nil,
		)

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

		go c.dockerfileImageParser.ParseFile(
			dockerfilePath, buildArgs, dockerfileImages,
			done, &dockerfileImageWaitGroup,
		)
	}()

	go func() {
		dockerfileImageWaitGroup.Wait()
		close(dockerfileImages)
	}()

	for dockerfileImage := range dockerfileImages {
		dockerfileImage.SetKind(c.kind)

		if dockerfileImage.Err() != nil {
			select {
			case <-done:
			case composefileImages <- dockerfileImage:
			}

			return
		}

		dockerfileImage.SetMetadata(map[string]interface{}{
			"dockerfilePath":  dockerfileImage.Metadata()["path"],
			"servicePosition": dockerfileImage.Metadata()["position"],
			"serviceName":     serviceConfig.Name,
			"path":            path.Val(),
		})

		select {
		case <-done:
			return
		case composefileImages <- dockerfileImage:
		}
	}
}
