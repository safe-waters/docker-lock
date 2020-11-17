package write

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
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

func (k *KubernetesfileWriter) WriteFiles(
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

	for docPosition := 0; ; docPosition++ {
		var doc interface{}

		if err = dec.Decode(&doc); err != nil {
			if err != io.EOF {
				return "", err
			}

			break
		}

		k.encodeDoc(doc, images)
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
			// TODO: use position, and use exclude tags
			doc["image"] = convertImageToImageLine(
				images[0].Image, k.ExcludeTags,
			)
		}

		var keys []string

		for key := range doc {
			if k, ok := key.(string); ok {
				keys = append(keys, k)
			}
		}

		sort.Strings(keys)

		for _, key := range keys {
			k.encodeDoc(doc[key], images)
		}
	case []interface{}:
		for i := range doc {
			k.encodeDoc(doc[i], images)
		}
	}
}
