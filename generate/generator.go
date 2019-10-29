package generate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/michaelperel/docker-lock/registry"
	"github.com/spf13/cobra"
)

type Generator struct {
	Dockerfiles            []string
	Composefiles           []string
	DockerfileEnvBuildArgs bool
	outPath                string
}

type Image struct {
	Name   string `json:"name"`
	Tag    string `json:"tag"`
	Digest string `json:"digest"`
}

type DockerfileImage struct {
	Image
	position int
}

type ComposefileImage struct {
	Image
	ServiceName string `json:"serviceName"`
	Dockerfile  string `json:"dockerfile"`
	position    int
}

type Lockfile struct {
	DockerfileImages  map[string][]DockerfileImage  `json:"dockerfiles"`
	ComposefileImages map[string][]ComposefileImage `json:"composefiles"`
}

type imageResponse struct {
	image Image
	line  string
	err   error
}

func (i Image) Prettify() string {
	pretty, _ := json.MarshalIndent(i, "", "\t")
	return string(pretty)
}

func NewGenerator(cmd *cobra.Command) (*Generator, error) {
	var (
		wg                        sync.WaitGroup
		dockerfiles, composefiles []string
		dErr, cErr                error
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		dockerfiles, dErr = collectDockerfiles(cmd)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		composefiles, cErr = collectComposefiles(cmd)
	}()
	wg.Wait()
	if dErr != nil {
		return nil, dErr
	}
	if cErr != nil {
		return nil, cErr
	}
	baseDir, err := cmd.Flags().GetString("base-dir")
	if err != nil {
		return nil, err
	}
	if len(dockerfiles) == 0 && len(composefiles) == 0 {
		dockerfiles, composefiles = collectDefaultFiles(baseDir)
	}
	outPath, err := cmd.Flags().GetString("outpath")
	if err != nil {
		return nil, err
	}
	dockerfileEnvBuildArgs, err := cmd.Flags().GetBool("dockerfile-env-build-args")
	if err != nil {
		return nil, err
	}
	return &Generator{Dockerfiles: dockerfiles,
		Composefiles:           composefiles,
		DockerfileEnvBuildArgs: dockerfileEnvBuildArgs,
		outPath:                outPath}, nil
}

func (g *Generator) GenerateLockfile(wrapperManager *registry.WrapperManager) error {
	lockfileBytes, err := g.GenerateLockfileBytes(wrapperManager)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(g.outPath, lockfileBytes, 0644)
}

func (g *Generator) GenerateLockfileBytes(wrapperManager *registry.WrapperManager) ([]byte, error) {
	parsedImageLines := make(chan parsedImageLine)
	var parseWg sync.WaitGroup
	for _, fileName := range g.Dockerfiles {
		parseWg.Add(1)
		go func(fileName string) {
			defer parseWg.Done()
			var buildArgs = make(map[string]string)
			if g.DockerfileEnvBuildArgs {
				for _, e := range os.Environ() {
					pair := strings.SplitN(e, "=", 2)
					buildArgs[pair[0]] = pair[1]
				}
			}
			parseDockerfile(fileName, buildArgs, parsedImageLines)
		}(fileName)
	}
	for _, fileName := range g.Composefiles {
		parseWg.Add(1)
		go func(fileName string) {
			defer parseWg.Done()
			parseComposefile(fileName, parsedImageLines)
		}(fileName)
	}
	go func() {
		parseWg.Wait()
		close(parsedImageLines)
	}()
	pilReqs := map[string]bool{}
	allPils := map[string][]parsedImageLine{}
	imageResponses := make(chan imageResponse)
	var numRequests int
	for pil := range parsedImageLines {
		if pil.err != nil {
			return nil, pil.err
		}
		allPils[pil.line] = append(allPils[pil.line], pil)
		if !pilReqs[pil.line] {
			pilReqs[pil.line] = true
			numRequests++
			go g.getImage(pil, wrapperManager, imageResponses)
		}
	}
	dImages := make(map[string][]DockerfileImage)
	cImages := make(map[string][]ComposefileImage)
	for i := 0; i < numRequests; i++ {
		resp := <-imageResponses
		if resp.err != nil {
			return nil, resp.err
		}
		for _, pil := range allPils[resp.line] {
			if pil.composefileName == "" {
				dImage := DockerfileImage{Image: resp.image, position: pil.position}
				dKey := filepath.ToSlash(pil.dockerfileName)
				dImages[dKey] = append(dImages[dKey], dImage)
			} else {
				cImage := ComposefileImage{Image: resp.image,
					ServiceName: pil.serviceName,
					Dockerfile:  filepath.ToSlash(pil.dockerfileName),
					position:    pil.position}
				cKey := filepath.ToSlash(pil.composefileName)
				cImages[cKey] = append(cImages[cKey], cImage)
			}
		}
	}
	close(imageResponses)
	var sortWg sync.WaitGroup
	sortWg.Add(1)
	go func() {
		defer sortWg.Done()
		for _, imageSlice := range dImages {
			sort.Slice(imageSlice, func(i, j int) bool {
				return imageSlice[i].position < imageSlice[j].position
			})
		}
	}()
	sortWg.Add(1)
	go func() {
		defer sortWg.Done()
		for _, imageSlice := range cImages {
			sort.Slice(imageSlice, func(i, j int) bool {
				if imageSlice[i].ServiceName != imageSlice[j].ServiceName {
					return imageSlice[i].ServiceName < imageSlice[j].ServiceName
				} else if imageSlice[i].Dockerfile != imageSlice[i].Dockerfile {
					return imageSlice[i].Dockerfile < imageSlice[j].Dockerfile
				} else {
					return imageSlice[i].position < imageSlice[j].position
				}
			})
		}
	}()
	sortWg.Wait()
	lockfile := Lockfile{DockerfileImages: dImages, ComposefileImages: cImages}
	lockfileBytes, err := json.MarshalIndent(lockfile, "", "\t")
	if err != nil {
		return nil, err
	}
	return lockfileBytes, nil
}

func (g *Generator) getImage(imLine parsedImageLine, wrapperManager *registry.WrapperManager, response chan<- imageResponse) {
	line := imLine.line
	tagSeparator := -1
	digestSeparator := -1
	for i, c := range line {
		if c == ':' {
			tagSeparator = i
		}
		if c == '@' {
			digestSeparator = i
			break
		}
	}
	var name, tag, digest string
	// 4 valid cases
	if tagSeparator != -1 && digestSeparator != -1 {
		// ubuntu:18.04@sha256:9b1702dcfe32c873a770a32cfd306dd7fc1c4fd134adfb783db68defc8894b3c
		name = line[:tagSeparator]
		tag = line[tagSeparator+1 : digestSeparator]
		digest = line[digestSeparator+1+len("sha256:"):]
	} else if tagSeparator != -1 && digestSeparator == -1 {
		// ubuntu:18.04
		name = line[:tagSeparator]
		tag = line[tagSeparator+1:]
	} else if tagSeparator == -1 && digestSeparator != -1 {
		// ubuntu@sha256:9b1702dcfe32c873a770a32cfd306dd7fc1c4fd134adfb783db68defc8894b3c
		name = line[:digestSeparator]
		digest = line[digestSeparator+1+len("sha256:"):]
	} else {
		// ubuntu
		name = line
		tag = "latest"
	}
	if digest == "" {
		wrapper := wrapperManager.GetWrapper(name)
		var err error
		digest, err = wrapper.GetDigest(name, tag)
		if err != nil {
			extraErrInfo := fmt.Sprintf("%s. From line: '%s'.", err, line)
			if imLine.dockerfileName != "" {
				extraErrInfo += fmt.Sprintf(" From dockerfile: '%s'.", imLine.dockerfileName)
			}
			if imLine.composefileName != "" {
				extraErrInfo += fmt.Sprintf(" From service: '%s' in compose-file: '%s'.", imLine.serviceName, imLine.composefileName)
			}
			response <- imageResponse{err: errors.New(extraErrInfo)}
			return
		}
	}
	response <- imageResponse{image: Image{Name: name, Tag: tag, Digest: digest}, line: line}
}
