package parse

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

type dockerfileImageParser struct {
	kind kind.Kind
}

// NewDockerfileImageParser returns an IImageParser for Dockerfiles.
func NewDockerfileImageParser() IDockerfileImageParser {
	return &dockerfileImageParser{
		kind: kind.Dockerfile,
	}
}

// Kind is a getter for the kind.
func (d *dockerfileImageParser) Kind() kind.Kind {
	return d.kind
}

// ParseFiles parses IImages from Dockerfiles.
func (d *dockerfileImageParser) ParseFiles(
	paths <-chan collect.IPath,
	done <-chan struct{},
) <-chan IImage {
	if paths == nil {
		return nil
	}

	var (
		waitGroup        sync.WaitGroup
		dockerfileImages = make(chan IImage)
	)

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for path := range paths {
			waitGroup.Add(1)

			go d.ParseFile(
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

// ParseFile parses IImages from a Dockerfile.
func (d *dockerfileImageParser) ParseFile(
	path collect.IPath,
	buildArgs map[string]string,
	dockerfileImages chan<- IImage,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	if path == nil || reflect.ValueOf(path).IsNil() || dockerfileImages == nil {
		return
	}

	if path.Err() != nil {
		select {
		case <-done:
		case dockerfileImages <- NewImage(d.kind, "", "", "", nil, path.Err()):
		}

		return
	}

	dockerfile, err := os.Open(path.Val())
	if err != nil {
		select {
		case <-done:
		case dockerfileImages <- NewImage(d.kind, "", "", "", nil, err):
		}

		return
	}
	defer dockerfile.Close()

	loadedDockerfile, err := parser.Parse(dockerfile)
	if err != nil {
		select {
		case <-done:
		case dockerfileImages <- NewImage(
			d.kind, "", "", "", nil, fmt.Errorf(
				"'%s' failed to parse with err: %v", path.Val(), err,
			),
		):
		}

		return
	}

	var (
		position      int                   // order of image in Dockerfile
		stages        = map[string]bool{}   // FROM <image line> as <stage>
		globalArgs    = map[string]string{} // ARGs before the first FROM
		globalContext = true                // true if before first FROM
	)

	for _, child := range loadedDockerfile.AST.Children {
		switch child.Value {
		case "arg":
			var raw []string

			for n := child.Next; n != nil; n = n.Next {
				raw = append(raw, n.Value)
			}

			if len(raw) == 0 {
				err := fmt.Errorf(
					"invalid arg instruction in Dockerfile '%s'", path,
				)

				select {
				case <-done:
				case dockerfileImages <- NewImage(d.kind, "", "", "", nil, err):
				}

				return
			}

			if globalContext {
				if strings.Contains(raw[0], "=") {
					// ARG VAR=VAL
					const (
						argValLen = 2
						varIndex  = 0
						valIndex  = 1
					)

					varVal := strings.SplitN(raw[0], "=", argValLen)

					strippedVar := d.stripQuotes(varVal[varIndex])
					strippedVal := d.stripQuotes(varVal[valIndex])

					globalArgs[strippedVar] = strippedVal
				} else {
					// ARG VAR1
					strippedVar := d.stripQuotes(raw[0])

					globalArgs[strippedVar] = ""
				}
			}
		case "from":
			var raw []string

			for n := child.Next; n != nil; n = n.Next {
				raw = append(raw, n.Value)
			}

			if len(raw) == 0 {
				err := fmt.Errorf(
					"invalid FROM instruction in Dockerfile '%s'", path,
				)

				select {
				case <-done:
				case dockerfileImages <- NewImage(d.kind, "", "", "", nil, err):
				}

				return
			}

			globalContext = false

			if !stages[raw[0]] {
				image := NewImage(d.kind, "", "", "", map[string]interface{}{
					"position": position,
					"path":     path.Val(),
				}, nil)
				imageLine := d.expandField(raw[0], globalArgs, buildArgs)

				image.SetNameTagDigestFromImageLine(imageLine)

				select {
				case <-done:
					return
				case dockerfileImages <- image:
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

func (d *dockerfileImageParser) stripQuotes(s string) string {
	// Valid in a Dockerfile - any number of quotes if quote is on either side.
	// ARG "IMAGE"="busybox"
	// ARG "IMAGE"""""="busybox"""""""""""""
	if s[0] == '"' && s[len(s)-1] == '"' {
		s = strings.TrimRight(strings.TrimLeft(s, "\""), "\"")
	}

	return s
}

func (d *dockerfileImageParser) expandField(
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
