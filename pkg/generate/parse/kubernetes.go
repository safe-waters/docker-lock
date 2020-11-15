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
	Position      int    `json:"-"`
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

	for {
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

		parsedImages := k.parseDoc(doc)
		for _, image := range parsedImages {
			select {
			case <-done:
				return
			case kubernetesfileImages <- &KubernetesfileImage{
				Image:    image.Image,
				Position: image.Position,
				Path:     path,
			}:
			}
		}
	}
}

func (k *KubernetesfileImageParser) parseDoc(
	doc interface{},
) []*KubernetesfileImage {
	var k8sImages []*KubernetesfileImage

	var position int

	parseDocRecursive(doc, &k8sImages, &position)

	return k8sImages
}

func parseDocRecursive(
	k8sYAML interface{},
	k8sImages *[]*KubernetesfileImage,
	position *int,
) {
	switch k8sYAML := k8sYAML.(type) {
	case map[interface{}]interface{}:
		var containerName string

		var imageName string

		if name, ok := k8sYAML["name"]; ok {
			containerName, _ = name.(string)
		}

		if image, ok := k8sYAML["image"]; ok {
			imageName, _ = image.(string)
		}

		if containerName != "" && imageName != "" {
			k8sImage := &KubernetesfileImage{
				Image:         &Image{Name: imageName},
				ContainerName: containerName,
				Position:      *position,
			}
			*k8sImages = append(*k8sImages, k8sImage)

			*position++
		}

		var keys []string

		for k := range k8sYAML {
			if k, ok := k.(string); ok {
				keys = append(keys, k)
			}
		}

		sort.Strings(keys)

		for _, k := range keys {
			parseDocRecursive(k8sYAML[k], k8sImages, position)
		}
	case []interface{}:
		for _, v := range k8sYAML {
			parseDocRecursive(v, k8sImages, position)
		}
	}
}
