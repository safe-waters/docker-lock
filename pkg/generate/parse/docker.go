// Package parse provides functionality to parse images from collected files.
package parse

import (
	"os"
	"strings"
	"sync"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

// DockerfileImageParser extracts image values from Dockerfiles.
type DockerfileImageParser struct{}

// DockerfileImage annotates an image with data about the Dockerfile
// from which it was parsed.
type DockerfileImage struct {
	*Image
	Position int    `json:"-"`
	Path     string `json:"-"`
	Err      error  `json:"-"`
}

// IDockerfileImageParser provides an interface for DockerfileImageParser's
// exported methods.
type IDockerfileImageParser interface {
	ParseFiles(
		paths <-chan string,
		done <-chan struct{},
	) <-chan *DockerfileImage
}

// ParseFiles reads a Dockerfile to parse all images in FROM instructions.
func (d *DockerfileImageParser) ParseFiles(
	paths <-chan string,
	done <-chan struct{},
) <-chan *DockerfileImage {
	if paths == nil {
		return nil
	}

	dockerfileImages := make(chan *DockerfileImage)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for path := range paths {
			waitGroup.Add(1)

			go d.parseFile(
				path, nil, dockerfileImages, done, &waitGroup,
			)
		}
	}()

	go func() {
		waitGroup.Wait()
		close(dockerfileImages)
	}()

	return dockerfileImages
}

func (d *DockerfileImageParser) parseFile(
	path string,
	buildArgs map[string]string,
	dockerfileImages chan<- *DockerfileImage,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	f, err := os.Open(path)
	if err != nil {
		select {
		case <-done:
		case dockerfileImages <- &DockerfileImage{Err: err}:
		}

		return
	}
	defer f.Close()

	res, err := parser.Parse(f)
	if err != nil {
		select {
		case <-done:
		case dockerfileImages <- &DockerfileImage{Err: err}:
		}

		return
	}

	position := 0                     // order of image in Dockerfile
	stages := map[string]bool{}       // FROM <image line> as <stage>
	globalArgs := map[string]string{} // ARGs before the first FROM
	globalContext := true             // true if before first FROM

	for _, child := range res.AST.Children {
		switch child.Value {
		case "arg":
			var raw []string
			for n := child.Next; n != nil; n = n.Next {
				raw = append(raw, n.Value)
			}

			rawStr := strings.Join(raw, " ")

			if globalContext {
				if strings.Contains(rawStr, "=") {
					// ARG VAR=VAL
					varVal := strings.SplitN(rawStr, "=", 2)

					const varIndex = 0

					const valIndex = 1

					strippedVar := d.stripQuotes(varVal[varIndex])
					strippedVal := d.stripQuotes(varVal[valIndex])

					globalArgs[strippedVar] = strippedVal
				} else {
					// ARG VAR1
					strippedVar := d.stripQuotes(rawStr)

					globalArgs[strippedVar] = ""
				}
			}
		case "from":
			var raw []string
			for n := child.Next; n != nil; n = n.Next {
				raw = append(raw, n.Value)
			}

			globalContext = false

			imageLine := raw[0]

			if !stages[imageLine] {
				imageLine = expandField(imageLine, globalArgs, buildArgs)

				image := convertImageLineToImage(imageLine)

				select {
				case <-done:
					return
				case dockerfileImages <- &DockerfileImage{
					Image: image, Position: position, Path: path,
				}:
					position++
				}
			}

			// <image> AS <stage>
			// <stage> AS <another stage>
			const maxNumFields = 3
			if len(raw) == maxNumFields {
				const stageIndex = 2

				stage := raw[stageIndex]
				stages[stage] = true
			}
		}
	}
}

func (d *DockerfileImageParser) stripQuotes(s string) string {
	// Valid in a Dockerfile - any number of quotes if quote is on either side.
	// ARG "IMAGE"="busybox"
	// ARG "IMAGE"""""="busybox"""""""""""""
	if s[0] == '"' && s[len(s)-1] == '"' {
		s = strings.TrimRight(strings.TrimLeft(s, "\""), "\"")
	}

	return s
}

func convertImageLineToImage(imageLine string) *Image {
	tagSeparator := -1
	digestSeparator := -1

loop:
	for i, c := range imageLine {
		switch c {
		case ':':
			tagSeparator = i
		case '/':
			// reset tagSeparator
			// for instance, 'localhost:5000/my-image'
			tagSeparator = -1
		case '@':
			digestSeparator = i
			break loop
		}
	}

	var name, tag, digest string

	switch {
	case tagSeparator != -1 && digestSeparator != -1:
		// ubuntu:18.04@sha256:9b1702...
		name = imageLine[:tagSeparator]
		tag = imageLine[tagSeparator+1 : digestSeparator]
		digest = imageLine[digestSeparator+1+len("sha256:"):]
	case tagSeparator != -1 && digestSeparator == -1:
		// ubuntu:18.04
		name = imageLine[:tagSeparator]
		tag = imageLine[tagSeparator+1:]
	case tagSeparator == -1 && digestSeparator != -1:
		// ubuntu@sha256:9b1702...
		name = imageLine[:digestSeparator]
		digest = imageLine[digestSeparator+1+len("sha256:"):]
	default:
		// ubuntu
		name = imageLine
		if name != "scratch" {
			tag = "latest"
		}
	}

	return &Image{Name: name, Tag: tag, Digest: digest}
}

func expandField(
	field string,
	globalArgs map[string]string,
	buildArgs map[string]string,
) string {
	return os.Expand(field, func(arg string) string {
		globalVal, ok := globalArgs[arg]
		if !ok {
			return ""
		}

		buildVal, ok := buildArgs[arg]
		if !ok {
			return globalVal
		}

		return buildVal
	})
}
