package parse

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/compose-spec/compose-go/cli"
	"github.com/compose-spec/compose-go/types"
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
		return nil, errors.New("'dockerfileImageParser' cannot be nil")
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
	if paths == nil {
		return nil
	}

	var (
		waitGroup         sync.WaitGroup
		composefileImages = make(chan IImage)
	)

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

	if path == nil || reflect.ValueOf(path).IsNil() ||
		composefileImages == nil {
		return
	}

	if path.Err() != nil {
		select {
		case <-done:
		case composefileImages <- NewImage(c.kind, "", "", "", nil, path.Err()):
		}

		return
	}
	//
	//
	//
	// TODO: use cli.NewProjectOptions...

	o := &cli.ProjectOptions{
		WorkingDir:  filepath.Dir(path.Val()),
		ConfigPaths: []string{path.Val()},
		Environment: map[string]string{},
	}

	err := cli.WithDotEnv(o)
	if err != nil {
		select {
		case <-done:
		case composefileImages <- NewImage(c.kind, "", "", "", nil, err):
		}

		return
	}

	err = cli.WithOsEnv(o)
	if err != nil {
		select {
		case <-done:
		case composefileImages <- NewImage(c.kind, "", "", "", nil, err):
		}

		return
	}

	_, e := cli.NewProjectOptions([]string{}, cli.WithDotEnv, cli.WithOsEnv, cli.WithInterpolation(false))
	if e != nil {
		panic(e)
	}

	p, err := cli.ProjectFromOptions(o)
	if err != nil {
		select {
		case <-done:
		case composefileImages <- NewImage(c.kind, "", "", "", nil, err):
		}

		return
	}
	//
	//
	//
	for _, serviceConfig := range p.Services {
		waitGroup.Add(1)

		go c.parseService(
			serviceConfig, path, composefileImages, waitGroup, done,
		)
	}
}

func (c *composefileImageParser) parseService(
	serviceConfig types.ServiceConfig,
	path collect.IPath,
	composefileImages chan<- IImage,
	waitGroup *sync.WaitGroup,
	done <-chan struct{},
) {
	defer waitGroup.Done()

	l, _ := json.MarshalIndent(serviceConfig, "", "\t")
	fmt.Println("THIS IS STRING", string(l))

	if serviceConfig.Build == nil {
		fmt.Println("SERVICE CONFIG BUILD NIL")
		if serviceConfig.Image == "" {
			return
		}

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
	fmt.Println("SERVICE CONFIG NOT NIL")

	var (
		dockerfileImageWaitGroup sync.WaitGroup
		dockerfileImages         = make(chan IImage)
	)

	dockerfileImageWaitGroup.Add(1)

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	fmt.Println("THIS IS WD", wd)
	relPath := strings.TrimPrefix(serviceConfig.Build.Dockerfile, wd+"/")
	fmt.Println("THIS IS RELPATH", relPath)
	go func() {
		defer dockerfileImageWaitGroup.Done()

		dockerfilePath := collect.NewPath(
			c.kind, relPath, nil,
		)

		buildArgs := map[string]string{}

		for arg, val := range serviceConfig.Build.Args {
			buildArgs[arg] = *val
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

		dockerfileImageMetadata := dockerfileImage.Metadata()
		if dockerfileImageMetadata == nil {
			select {
			case <-done:
			case composefileImages <- NewImage(
				dockerfileImage.Kind(), "", "", "", nil,
				errors.New("'metadata' cannot be nil"),
			):
			}

			return
		}

		dockerfileImage.SetMetadata(map[string]interface{}{
			"dockerfilePath":  dockerfileImageMetadata["path"],
			"servicePosition": dockerfileImageMetadata["position"],
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
