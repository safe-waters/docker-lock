package generate

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/michaelperel/docker-lock/registry"
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

type imageResult struct {
	image           Image
	dockerfileName  string
	composefileName string
	position        int
	serviceName     string
	err             error
}

func (i Image) String() string {
	pretty, _ := json.MarshalIndent(i, "", "\t")
	return string(pretty)
}

func NewGenerator(flags *Flags) (*Generator, error) {
	dockerfiles, err := collectDockerfiles(flags)
	if err != nil {
		return nil, err
	}
	composefiles, err := collectComposefiles(flags)
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
	return &Generator{Dockerfiles: dockerfiles, Composefiles: composefiles, outfile: flags.Outfile}, nil
}

func (g *Generator) GenerateLockfile(wrapperManager *registry.WrapperManager) error {
	lockfileBytes, err := g.GenerateLockfileBytes(wrapperManager)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(g.outfile, lockfileBytes, 0644)
}

func (g *Generator) GenerateLockfileBytes(wrapperManager *registry.WrapperManager) ([]byte, error) {
	dImages, err := g.getDockerfileImages(wrapperManager)
	if err != nil {
		return nil, err
	}
	dSlashImages := make(map[string][]DockerfileImage)
	for fileName := range dImages {
		dSlashImages[filepath.ToSlash(fileName)] = dImages[fileName]
	}
	cImages, err := g.getComposefileImages(wrapperManager)
	if err != nil {
		return nil, err
	}
	cSlashImages := make(map[string][]ComposefileImage)
	for fileName := range cImages {
		for i := range cImages[fileName] {
			cImages[fileName][i].Dockerfile = filepath.ToSlash(cImages[fileName][i].Dockerfile)
		}
		cSlashImages[filepath.ToSlash(fileName)] = cImages[fileName]
	}
	lockfile := Lockfile{DockerfileImages: dSlashImages, ComposefileImages: cSlashImages}
	lockfileBytes, err := json.MarshalIndent(lockfile, "", "\t")
	if err != nil {
		return nil, err
	}
	return lockfileBytes, nil
}

func (g *Generator) getDockerfileImages(wrapperManager *registry.WrapperManager) (map[string][]DockerfileImage, error) {
	parsedImageLines := make(chan parsedImageLine)
	var wg sync.WaitGroup
	for _, fileName := range g.Dockerfiles {
		wg.Add(1)
		go parseDockerfile(fileName, nil, "", "", parsedImageLines, &wg)
	}
	go func() {
		wg.Wait()
		close(parsedImageLines)
	}()
	imageResults := make(chan imageResult)
	var numImages int
	for parsedImageLine := range parsedImageLines {
		if parsedImageLine.err != nil {
			return nil, parsedImageLine.err
		}
		numImages++
		go g.getImage(parsedImageLine, wrapperManager, imageResults)
	}
	images := make(map[string][]DockerfileImage)
	for i := 0; i < numImages; i++ {
		result := <-imageResults
		if result.err != nil {
			return nil, result.err
		}
		dImage := DockerfileImage{Image: result.image, position: result.position}
		_, ok := images[result.dockerfileName]
		if ok {
			images[result.dockerfileName] = append(images[result.dockerfileName], dImage)
		} else {
			images[result.dockerfileName] = []DockerfileImage{dImage}
		}
	}
	for _, imageSlice := range images {
		sort.Slice(imageSlice, func(i, j int) bool {
			return imageSlice[i].position < imageSlice[j].position
		})
	}
	return images, nil
}

func (g *Generator) getComposefileImages(wrapperManager *registry.WrapperManager) (map[string][]ComposefileImage, error) {
	parsedImageLines := make(chan parsedImageLine)
	var wg sync.WaitGroup
	for _, fileName := range g.Composefiles {
		wg.Add(1)
		go parseComposefile(fileName, parsedImageLines, &wg)
	}
	go func() {
		wg.Wait()
		close(parsedImageLines)
	}()
	imageResults := make(chan imageResult)
	var numImages int
	for parsedImageLine := range parsedImageLines {
		if parsedImageLine.err != nil {
			return nil, parsedImageLine.err
		}
		numImages++
		go g.getImage(parsedImageLine, wrapperManager, imageResults)
	}
	images := make(map[string][]ComposefileImage)
	for i := 0; i < numImages; i++ {
		result := <-imageResults
		if result.err != nil {
			return nil, result.err
		}
		cImage := ComposefileImage{Image: result.image,
			ServiceName: result.serviceName,
			Dockerfile:  result.dockerfileName,
			position:    result.position}
		_, ok := images[result.composefileName]
		if ok {
			images[result.composefileName] = append(images[result.composefileName], cImage)
		} else {
			images[result.composefileName] = []ComposefileImage{cImage}
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
	return images, nil
}

func (g *Generator) getImage(imLine parsedImageLine, wrapperManager *registry.WrapperManager, imageResults chan<- imageResult) {
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
	// 4 valid cases
	// ubuntu:18.04@sha256:9b1702dcfe32c873a770a32cfd306dd7fc1c4fd134adfb783db68defc8894b3c
	if tagSeparator != -1 && digestSeparator != -1 {
		name := line[:tagSeparator]
		tag := line[tagSeparator+1 : digestSeparator]
		digest := line[digestSeparator+1+len("sha256:"):]
		imageResults <- imageResult{image: Image{Name: name, Tag: tag, Digest: digest},
			position:        imLine.position,
			serviceName:     imLine.serviceName,
			dockerfileName:  imLine.dockerfileName,
			composefileName: imLine.composefileName}
		return
	}
	// ubuntu:18.04
	if tagSeparator != -1 && digestSeparator == -1 {
		name := line[:tagSeparator]
		tag := line[tagSeparator+1:]
		wrapper := wrapperManager.GetWrapper(name)
		digest, err := wrapper.GetDigest(name, tag)
		if err != nil {
			err := fmt.Errorf("%s. From line: '%s'. From dockerfile: '%s'. From composefile: '%s'. From service: '%s'.",
				err,
				line,
				imLine.dockerfileName,
				imLine.composefileName,
				imLine.serviceName)
			imageResults <- imageResult{err: err}
			return
		}
		imageResults <- imageResult{image: Image{Name: name, Tag: tag, Digest: digest},
			position:        imLine.position,
			serviceName:     imLine.serviceName,
			composefileName: imLine.composefileName,
			dockerfileName:  imLine.dockerfileName}
		return
	}
	// ubuntu@sha256:9b1702dcfe32c873a770a32cfd306dd7fc1c4fd134adfb783db68defc8894b3c
	if tagSeparator == -1 && digestSeparator != -1 {
		name := line[:digestSeparator]
		digest := line[digestSeparator+1+len("sha256:"):]
		imageResults <- imageResult{image: Image{Name: name, Digest: digest},
			position:        imLine.position,
			serviceName:     imLine.serviceName,
			composefileName: imLine.composefileName,
			dockerfileName:  imLine.dockerfileName}
		return
	}
	// ubuntu
	if tagSeparator == -1 && digestSeparator == -1 {
		name := line
		tag := "latest"
		wrapper := wrapperManager.GetWrapper(name)
		digest, err := wrapper.GetDigest(name, tag)
		if err != nil {
			err := fmt.Errorf("%s. From line: '%s'. From dockerfile: '%s'. From composefile: '%s'. From service: '%s'.",
				err,
				line,
				imLine.dockerfileName,
				imLine.composefileName,
				imLine.serviceName)
			imageResults <- imageResult{err: err}
			return
		}
		imageResults <- imageResult{image: Image{Name: name, Tag: tag, Digest: digest},
			position:        imLine.position,
			serviceName:     imLine.serviceName,
			dockerfileName:  imLine.dockerfileName,
			composefileName: imLine.composefileName}
		return
	}
}
