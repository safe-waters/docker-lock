package parse

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/kind"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes/scheme"
)

type kubernetesfileImageParser struct {
	kind kind.Kind
}

// NewComposefileImageParser returns an IImageParser for Kubernetesfiles.
func NewKubernetesfileImageParser() IKubernetesfileImageParser {
	return &kubernetesfileImageParser{
		kind: kind.Kubernetesfile,
	}
}

// Kind is a getter for the kind.
func (k *kubernetesfileImageParser) Kind() kind.Kind {
	return k.kind
}

// ParseFiles parses IImages from Kubernetesfiles.
func (k *kubernetesfileImageParser) ParseFiles(
	paths <-chan collect.IPath,
	done <-chan struct{},
) <-chan IImage {
	if paths == nil {
		return nil
	}

	var (
		waitGroup            sync.WaitGroup
		kubernetesfileImages = make(chan IImage)
	)

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for path := range paths {
			waitGroup.Add(1)

			go k.ParseFile(
				path, kubernetesfileImages, done, &waitGroup,
			)
		}
	}()

	go func() {
		waitGroup.Wait()
		close(kubernetesfileImages)
	}()

	return kubernetesfileImages
}

// ParseFile parses IImages from a Kubernetesfile.
func (k *kubernetesfileImageParser) ParseFile(
	path collect.IPath,
	kubernetesfileImages chan<- IImage,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	if path == nil || reflect.ValueOf(path).IsNil() ||
		kubernetesfileImages == nil {
		return
	}

	if path.Err() != nil {
		select {
		case <-done:
		case kubernetesfileImages <- NewImage(
			k.kind, "", "", "", nil, path.Err(),
		):
		}

		return
	}

	byt, err := ioutil.ReadFile(path.Val())
	if err != nil {
		select {
		case <-done:
		case kubernetesfileImages <- NewImage(k.kind, "", "", "", nil, err):
		}

		return
	}

	_, _, err = scheme.Codecs.UniversalDeserializer().Decode(byt, nil, nil)
	if err != nil {
		select {
		case <-done:
		case kubernetesfileImages <- NewImage(
			k.kind, "", "", "", nil, fmt.Errorf(
				"'%s' failed to parse with err: %v", path.Val(), err,
			),
		):
		}

		return
	}

	dec := yaml.NewDecoder(bytes.NewReader(byt))

	for docPosition := 0; ; docPosition++ {
		var doc yaml.MapSlice

		if err := dec.Decode(&doc); err != nil {
			if err != io.EOF {
				select {
				case <-done:
				case kubernetesfileImages <- NewImage(
					k.kind, "", "", "", nil, fmt.Errorf(
						"'%s' yaml decoder failed with err: %v", path.Val(),
						err,
					),
				):
				}

				return
			}

			break
		}

		waitGroup.Add(1)

		go k.parseDoc(
			path, doc, kubernetesfileImages, docPosition, done, waitGroup,
		)
	}
}

func (k *kubernetesfileImageParser) parseDoc(
	path collect.IPath,
	doc interface{},
	kubernetesfileImages chan<- IImage,
	docPosition int,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	var imagePosition int

	k.parseDocRecursive(
		path, doc, kubernetesfileImages, docPosition, &imagePosition, done,
	)
}

func (k *kubernetesfileImageParser) parseDocRecursive(
	path collect.IPath,
	doc interface{},
	kubernetesfileImages chan<- IImage,
	docPosition int,
	imagePosition *int,
	done <-chan struct{},
) {
	switch doc := doc.(type) {
	case yaml.MapSlice:
		var name, imageLine string

		for _, item := range doc {
			key, _ := item.Key.(string)
			val, _ := item.Value.(string)

			switch key {
			case "name":
				name = val
			case "image":
				imageLine = val
			}
		}

		if name != "" && imageLine != "" {
			image := NewImage(k.kind, "", "", "", map[string]interface{}{
				"containerName": name,
				"path":          path.Val(),
				"imagePosition": *imagePosition,
				"docPosition":   docPosition,
			}, nil)
			image.SetNameTagDigestFromImageLine(imageLine)

			select {
			case <-done:
			case kubernetesfileImages <- image:
			}

			*imagePosition++
		}

		for _, item := range doc {
			k.parseDocRecursive(
				path, item.Value, kubernetesfileImages,
				docPosition, imagePosition, done,
			)
		}
	case []interface{}:
		for _, doc := range doc {
			k.parseDocRecursive(
				path, doc, kubernetesfileImages,
				docPosition, imagePosition, done,
			)
		}
	}
}
