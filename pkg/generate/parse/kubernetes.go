package parse

import (
	"bytes"
	"io"
	"io/ioutil"
	"sort"
	"sync"

	"gopkg.in/yaml.v2"
)

type KubernetesfileImageParser struct{}

type KubernetesfileImage struct {
	*Image
	ContainerName string
	ImagePosition int    `json:"-"`
	DocPosition   int    `json:"-"`
	Path          string `json:"-"`
	Err           error  `json:"-"`
}

type IKubernetesfileImageParser interface {
	ParseFiles(
		paths <-chan string,
		done <-chan struct{},
	) <-chan *KubernetesfileImage
}

func (k *KubernetesfileImageParser) ParseFiles(
	paths <-chan string,
	done <-chan struct{},
) <-chan *KubernetesfileImage {
	if paths == nil {
		return nil
	}

	kubernetesfileImages := make(chan *KubernetesfileImage)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for path := range paths {
			waitGroup.Add(1)

			go k.parseFile(
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

func (k *KubernetesfileImageParser) parseFile(
	path string,
	kubernetesfileImages chan<- *KubernetesfileImage,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	byt, err := ioutil.ReadFile(path)
	if err != nil {
		select {
		case <-done:
		case kubernetesfileImages <- &KubernetesfileImage{Err: err}:
		}

		return
	}

	dec := yaml.NewDecoder(bytes.NewReader(byt))

	for docPosition := 0; ; docPosition++ {
		var doc interface{}

		if err := dec.Decode(&doc); err != nil {
			if err != io.EOF {
				select {
				case <-done:
				case kubernetesfileImages <- &KubernetesfileImage{Err: err}:
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

func (k *KubernetesfileImageParser) parseDoc(
	path string,
	doc interface{},
	kubernetesfileImages chan<- *KubernetesfileImage,
	docPosition int,
	done <-chan struct{},
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	var imagePosition int

	parseDocRecursive(
		path, doc, kubernetesfileImages, docPosition, &imagePosition, done,
	)
}

func parseDocRecursive(
	path string,
	doc interface{},
	kubernetesfileImages chan<- *KubernetesfileImage,
	docPosition int,
	imagePosition *int,
	done <-chan struct{},
) {
	switch doc := doc.(type) {
	case map[interface{}]interface{}:
		var containerName string

		var imageLine string

		if possibleContainerName, ok := doc["name"]; ok {
			containerName, _ = possibleContainerName.(string)
		}

		if possibleImageLine, ok := doc["image"]; ok {
			imageLine, _ = possibleImageLine.(string)
		}

		if containerName != "" && imageLine != "" {
			image := convertImageLineToImage(imageLine)

			select {
			case <-done:
			case kubernetesfileImages <- &KubernetesfileImage{
				Image:         image,
				ContainerName: containerName,
				Path:          path,
				ImagePosition: *imagePosition,
				DocPosition:   docPosition,
			}:
			}

			*imagePosition++
		}

		var keys []string

		for k := range doc {
			if k, ok := k.(string); ok {
				keys = append(keys, k)
			}
		}

		sort.Strings(keys)

		for _, k := range keys {
			parseDocRecursive(
				path, doc[k], kubernetesfileImages,
				docPosition, imagePosition, done,
			)
		}
	case []interface{}:
		for i := range doc {
			parseDocRecursive(
				path, doc[i], kubernetesfileImages,
				docPosition, imagePosition, done,
			)
		}
	}
}
