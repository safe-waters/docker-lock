package generate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/michaelperel/docker-lock/registry"
	"github.com/spf13/cobra"
)

type Generator struct {
	Dockerfiles  []string
	Composefiles []string
	outfile      string
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

type dockerfileImageResponse struct {
	images map[string][]DockerfileImage
	err    error
}

type composefileImageResponse struct {
	images map[string][]ComposefileImage
	err    error
}

func (i Image) String() string {
	pretty, _ := json.MarshalIndent(i, "", "\t")
	return string(pretty)
}

func NewGenerator(cmd *cobra.Command) (*Generator, error) {
	dockerfiles, err := collectDockerfiles(cmd)
	if err != nil {
		return nil, err
	}
	composefiles, err := collectComposefiles(cmd)
	if err != nil {
		return nil, err
	}
	if len(dockerfiles) == 0 && len(composefiles) == 0 {
		fi, err := os.Stat("Dockerfile")
		if err == nil {
			if mode := fi.Mode(); mode.IsRegular() {
				dockerfiles = []string{"Dockerfile"}
			}
		}
		for _, defaultComposefile := range []string{"docker-compose.yml", "docker-compose.yaml"} {
			fi, err := os.Stat(defaultComposefile)
			if err == nil {
				if mode := fi.Mode(); mode.IsRegular() {
					composefiles = append(composefiles, defaultComposefile)
				}
			}
		}
	}
	outfile, err := cmd.Flags().GetString("outfile")
	if err != nil {
		return nil, err
	}
	return &Generator{Dockerfiles: dockerfiles, Composefiles: composefiles, outfile: outfile}, nil
}

func (g *Generator) GenerateLockfile(wrapperManager *registry.WrapperManager) error {
	lockfileBytes, err := g.GenerateLockfileBytes(wrapperManager)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(g.outfile, lockfileBytes, 0644)
}

func (g *Generator) GenerateLockfileBytes(wrapperManager *registry.WrapperManager) ([]byte, error) {
	dResponseCh := make(chan dockerfileImageResponse)
	go g.getDockerfileImages(wrapperManager, dResponseCh)
	cResponseCh := make(chan composefileImageResponse)
	go g.getComposefileImages(wrapperManager, cResponseCh)
	var dImages map[string][]DockerfileImage
	var cImages map[string][]ComposefileImage
	var numResponses int
	for {
		select {
		case resp := <-dResponseCh:
			numResponses++
			if resp.err != nil {
				return nil, resp.err
			}
			dImages = make(map[string][]DockerfileImage)
			for fileName := range resp.images {
				dImages[filepath.ToSlash(fileName)] = resp.images[fileName]
			}
		case resp := <-cResponseCh:
			numResponses++
			if resp.err != nil {
				return nil, resp.err
			}
			cImages = make(map[string][]ComposefileImage)
			for fileName := range resp.images {
				for i := range resp.images[fileName] {
					resp.images[fileName][i].Dockerfile = filepath.ToSlash(resp.images[fileName][i].Dockerfile)
				}
				cImages[filepath.ToSlash(fileName)] = resp.images[fileName]
			}
		}
		if numResponses == 2 {
			break
		}
	}
	lockfile := Lockfile{DockerfileImages: dImages, ComposefileImages: cImages}
	lockfileBytes, err := json.MarshalIndent(lockfile, "", "\t")
	if err != nil {
		return nil, err
	}
	return lockfileBytes, nil
}

func (g *Generator) getDockerfileImages(wrapperManager *registry.WrapperManager, response chan<- dockerfileImageResponse) {
	parsedImageLines := make(chan parsedImageLine)
	var parseWg sync.WaitGroup
	for _, fileName := range g.Dockerfiles {
		parseWg.Add(1)
		go parseDockerfile(fileName, nil, "", "", parsedImageLines, &parseWg)
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
			response <- dockerfileImageResponse{err: pil.err}
			return
		}
		allPils[pil.line] = append(allPils[pil.line], pil)
		if !pilReqs[pil.line] {
			pilReqs[pil.line] = true
			numRequests++
			go g.getImage(pil, wrapperManager, imageResponses)
		}
	}
	batchedResponses := []imageResponse{}
	for i := 0; i < numRequests; i++ {
		resp := <-imageResponses
		if resp.err != nil {
			response <- dockerfileImageResponse{err: resp.err}
			return
		}
		batchedResponses = append(batchedResponses, resp)
	}
	close(imageResponses)
	images := make(map[string][]DockerfileImage)
	for _, resp := range batchedResponses {
		for _, pil := range allPils[resp.line] {
			dImage := DockerfileImage{Image: resp.image, position: pil.position}
			images[pil.dockerfileName] = append(images[pil.dockerfileName], dImage)
		}
	}
	for _, imageSlice := range images {
		sort.Slice(imageSlice, func(i, j int) bool {
			return imageSlice[i].position < imageSlice[j].position
		})
	}
	response <- dockerfileImageResponse{images: images}
}

func (g *Generator) getComposefileImages(wrapperManager *registry.WrapperManager, response chan<- composefileImageResponse) {
	parsedImageLines := make(chan parsedImageLine)
	var parseWg sync.WaitGroup
	for _, fileName := range g.Composefiles {
		parseWg.Add(1)
		go parseComposefile(fileName, parsedImageLines, &parseWg)
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
			response <- composefileImageResponse{err: pil.err}
			return
		}
		allPils[pil.line] = append(allPils[pil.line], pil)
		if !pilReqs[pil.line] {
			pilReqs[pil.line] = true
			numRequests++
			go g.getImage(pil, wrapperManager, imageResponses)
		}
	}
	batchedResponses := []imageResponse{}
	for i := 0; i < numRequests; i++ {
		resp := <-imageResponses
		if resp.err != nil {
			response <- composefileImageResponse{err: resp.err}
			return
		}
		batchedResponses = append(batchedResponses, resp)
	}
	close(imageResponses)
	images := make(map[string][]ComposefileImage)
	for _, resp := range batchedResponses {
		for _, pil := range allPils[resp.line] {
			cImage := ComposefileImage{Image: resp.image,
				ServiceName: pil.serviceName,
				Dockerfile:  pil.dockerfileName,
				position:    pil.position}
			images[pil.composefileName] = append(images[pil.composefileName], cImage)
		}
	}
	for _, imageSlice := range images {
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
	response <- composefileImageResponse{images: images}
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
