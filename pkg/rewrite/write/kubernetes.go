package write

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/deprecated/scheme"
)

type KubernetesfileWriter struct {
	ExcludeTags bool
	Directory   string
}

type IKubernetesfileWriter interface {
	WriteFiles(
		pathImages map[string][]*parse.KubernetesfileImage,
		done <-chan struct{},
	) <-chan *WrittenPath
}

func (k *KubernetesfileWriter) WriteFiles( // nolint: dupl
	pathImages map[string][]*parse.KubernetesfileImage,
	done <-chan struct{},
) <-chan *WrittenPath {
	if len(pathImages) == 0 {
		return nil
	}

	writtenPaths := make(chan *WrittenPath)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for path, images := range pathImages {
			path := path
			images := images

			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				writtenPath, err := k.writeFile(path, images)
				if err != nil {
					select {
					case <-done:
					case writtenPaths <- &WrittenPath{Err: err}:
					}

					return
				}

				select {
				case <-done:
					return
				case writtenPaths <- &WrittenPath{
					OriginalPath: path,
					Path:         writtenPath,
				}:
				}
			}()
		}
	}()

	go func() {
		waitGroup.Wait()
		close(writtenPaths)
	}()

	return writtenPaths
}

func (k *KubernetesfileWriter) writeFile(
	path string,
	images []*parse.KubernetesfileImage,
) (string, error) {
	byt, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	_, _, err = scheme.Codecs.UniversalDeserializer().Decode(byt, nil, nil)
	if err != nil {
		return "", err
	}

	dec := yaml.NewDecoder(bytes.NewReader(byt))

	var encodedDocs []interface{}

	var imagePosition int

	for {
		var doc yaml.MapSlice

		if err = dec.Decode(&doc); err != nil {
			if err != io.EOF {
				return "", err
			}

			break
		}

		k.encodeDoc(doc, images, &imagePosition)
		encodedDocs = append(encodedDocs, doc)
	}

	replacer := strings.NewReplacer("/", "-", "\\", "-")
	tempPath := replacer.Replace(fmt.Sprintf("%s-*", path))

	writtenFile, err := ioutil.TempFile(k.Directory, tempPath)
	if err != nil {
		return "", err
	}
	defer writtenFile.Close()

	enc := yaml.NewEncoder(writtenFile)

	for _, encodedDoc := range encodedDocs {
		if err := enc.Encode(encodedDoc); err != nil {
			return "", err
		}
	}

	return writtenFile.Name(), nil
}

func (k *KubernetesfileWriter) encodeDoc(
	doc interface{},
	images []*parse.KubernetesfileImage,
	imagePosition *int,
) {
	switch doc := doc.(type) {
	case yaml.MapSlice:
		nameIndex := -1
		imageLineIndex := -1

		for i, item := range doc {
			key, _ := item.Key.(string)

			switch key {
			case "name":
				nameIndex = i
			case "image":
				imageLineIndex = i
			}
		}

		if nameIndex != -1 && imageLineIndex != -1 {
			doc[imageLineIndex].Value = convertImageToImageLine(
				images[*imagePosition].Image, k.ExcludeTags,
			)

			*imagePosition++
		}

		for _, item := range doc {
			k.encodeDoc(item.Value, images, imagePosition)
		}
	case []interface{}:
		for _, doc := range doc {
			k.encodeDoc(doc, images, imagePosition)
		}
	}
}
