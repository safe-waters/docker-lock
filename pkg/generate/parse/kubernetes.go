package parse

import (
	"bytes"
	"io"
	"io/ioutil"
	"sync"

	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes/scheme"
)

// KubernetesfileImageParser extracts image values from Kubernetesfiles.
type KubernetesfileImageParser struct{}

// KubernetesfileImage annotates an image with data about the
// Kubernetesfile from which it was parsed.
type KubernetesfileImage struct {
	*Image
	ContainerName string `json:"container"`
	ImagePosition int    `json:"-"`
	DocPosition   int    `json:"-"`
	Path          string `json:"-"`
	Err           error  `json:"-"`
}

// IKubernetesfileImageParser provides an interface for
// KubernetesfileImageParser's exported methods.
type IKubernetesfileImageParser interface {
	ParseFiles(
		paths <-chan string,
		done <-chan struct{},
	) <-chan *KubernetesfileImage
}

// ParseFiles reads Kubernetesfiles to parse all images.
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

	_, _, err = scheme.Codecs.UniversalDeserializer().Decode(byt, nil, nil)
	if err != nil {
		select {
		case <-done:
		case kubernetesfileImages <- &KubernetesfileImage{Err: err}:
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

	k.parseDocRecursive(
		path, doc, kubernetesfileImages, docPosition, &imagePosition, done,
	)
}

func (k *KubernetesfileImageParser) parseDocRecursive(
	path string,
	doc interface{},
	kubernetesfileImages chan<- *KubernetesfileImage,
	docPosition int,
	imagePosition *int,
	done <-chan struct{},
) {
	switch doc := doc.(type) {
	case yaml.MapSlice:
		var name string

		var imageLine string

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
			image := convertImageLineToImage(imageLine)

			select {
			case <-done:
			case kubernetesfileImages <- &KubernetesfileImage{
				Image:         image,
				ContainerName: name,
				Path:          path,
				ImagePosition: *imagePosition,
				DocPosition:   docPosition,
			}:
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
