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
	Name     string `json:"name"`
	Tag      string `json:"tag"`
	Digest   string `json:"digest"`
	position int
}

type Lockfile struct {
	Generator *Generator
	Images    map[string][]Image
}

type imageResult struct {
	image    Image
	fileName string
	err      error
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
	images, err := g.getImages(wrapperManager)
	slashImages := make(map[string][]Image)
	for fileName := range images {
		slashImages[filepath.ToSlash(fileName)] = images[fileName]
	}
	if err != nil {
		return nil, err
	}
	slashGen := *g
	slashGen.Dockerfiles = make([]string, len(g.Dockerfiles))
	slashGen.Composefiles = make([]string, len(g.Composefiles))
	for i := range g.Dockerfiles {
		slashGen.Dockerfiles[i] = filepath.ToSlash(g.Dockerfiles[i])
	}
	for i := range g.Composefiles {
		slashGen.Composefiles[i] = filepath.ToSlash(g.Composefiles[i])
	}
	lockfile := Lockfile{Generator: &slashGen, Images: slashImages}
	lockfileBytes, err := json.MarshalIndent(lockfile, "", "\t")
	if err != nil {
		return nil, err
	}
	return lockfileBytes, nil
}

func (g *Generator) getImages(wrapperManager *registry.WrapperManager) (map[string][]Image, error) {
	parsedImageLines := make(chan parsedImageLine)
	var dwg sync.WaitGroup
	for _, fileName := range g.Dockerfiles {
		dwg.Add(1)
		go parseDockerfile(fileName, nil, parsedImageLines, &dwg)
	}
	var cwg sync.WaitGroup
	for _, fileName := range g.Composefiles {
		cwg.Add(1)
		go parseComposefile(fileName, parsedImageLines, &cwg)
	}
	go func() {
		dwg.Wait()
		cwg.Wait()
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
	images := make(map[string][]Image)
	for i := 0; i < numImages; i++ {
		result := <-imageResults
		if result.err != nil {
			return nil, result.err
		}
		_, ok := images[result.fileName]
		if ok {
			images[result.fileName] = append(images[result.fileName], result.image)
		} else {
			images[result.fileName] = []Image{result.image}
		}
	}
	for _, imageSlice := range images {
		sort.Slice(imageSlice, func(i, j int) bool {
			return imageSlice[i].position < imageSlice[j].position
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
		imageResults <- imageResult{image: Image{Name: name, Tag: tag, Digest: digest, position: imLine.position},
			fileName: imLine.fileName}
		return
	}
	// ubuntu:18.04
	if tagSeparator != -1 && digestSeparator == -1 {
		name := line[:tagSeparator]
		tag := line[tagSeparator+1:]
		wrapper := wrapperManager.GetWrapper(name)
		digest, err := wrapper.GetDigest(name, tag)
		if err != nil {
			err := fmt.Errorf("%s. From line: '%s'. From file: '%s'.", err, line, imLine.fileName)
			imageResults <- imageResult{err: err}
			return
		}
		imageResults <- imageResult{image: Image{Name: name, Tag: tag, Digest: digest, position: imLine.position},
			fileName: imLine.fileName}
		return
	}
	// ubuntu@sha256:9b1702dcfe32c873a770a32cfd306dd7fc1c4fd134adfb783db68defc8894b3c
	if tagSeparator == -1 && digestSeparator != -1 {
		name := line[:digestSeparator]
		digest := line[digestSeparator+1+len("sha256:"):]
		imageResults <- imageResult{image: Image{Name: name, Digest: digest, position: imLine.position},
			fileName: imLine.fileName}
		return
	}
	// ubuntu
	if tagSeparator == -1 && digestSeparator == -1 {
		name := line
		tag := "latest"
		wrapper := wrapperManager.GetWrapper(name)
		digest, err := wrapper.GetDigest(name, tag)
		if err != nil {
			err := fmt.Errorf("%s. From line: '%s'. From file: '%s'.", err, line, imLine.fileName)
			imageResults <- imageResult{err: err}
			return
		}
		imageResults <- imageResult{image: Image{Name: name, Tag: tag, Digest: digest, position: imLine.position},
			fileName: imLine.fileName}
		return
	}
}
