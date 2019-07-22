package generate

import (
	"encoding/json"
	"fmt"
	"github.com/michaelperel/docker-lock/registry"
	"io/ioutil"
	"os"
	"sort"
	"sync"
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

type Lockfile struct {
	Generator *Generator
	Images    []Image
}

type imageResult struct {
	image Image
	err   error
}

func NewGenerator(flags *Flags) (*Generator, error) {
	dockerfiles, err := findDockerfiles(flags)
	if err != nil {
		return nil, err
	}
	composefiles, err := findComposefiles(flags)
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
	if err != nil {
		return nil, err
	}
	lockfile := Lockfile{Generator: g, Images: images}
	lockfileBytes, err := json.MarshalIndent(lockfile, "", "\t")
	if err != nil {
		return nil, err
	}
	return lockfileBytes, nil
}

func (g *Generator) getImages(wrapperManager *registry.WrapperManager) ([]Image, error) {
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
	var images []Image
	for i := 0; i < numImages; i++ {
		result := <-imageResults
		if result.err != nil {
			return nil, result.err
		}
		images = append(images, result.image)
	}
	sort.Slice(images, func(i, j int) bool {
		if images[i].Name != images[j].Name {
			return images[i].Name < images[j].Name
		}
		if images[i].Tag != images[j].Tag {
			return images[i].Tag < images[j].Tag
		}
		if images[i].Digest != images[j].Digest {
			return images[i].Digest < images[j].Digest
		}
		return true
	})
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
		imageResults <- imageResult{image: Image{Name: name, Tag: tag, Digest: digest}, err: nil}
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
			imageResults <- imageResult{image: Image{}, err: err}
			return
		}
		imageResults <- imageResult{image: Image{Name: name, Tag: tag, Digest: digest}, err: nil}
		return
	}
	// ubuntu@sha256:9b1702dcfe32c873a770a32cfd306dd7fc1c4fd134adfb783db68defc8894b3c
	if tagSeparator == -1 && digestSeparator != -1 {
		name := line[:digestSeparator]
		digest := line[digestSeparator+1+len("sha256:"):]
		imageResults <- imageResult{image: Image{Name: name, Digest: digest}, err: nil}
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
			imageResults <- imageResult{image: Image{}, err: err}
			return
		}
		imageResults <- imageResult{image: Image{Name: name, Tag: tag, Digest: digest}, err: nil}
		return
	}
}
