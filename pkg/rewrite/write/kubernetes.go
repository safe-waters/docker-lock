package write

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes/scheme"
)

type kubernetesfileWriter struct {
	kind        kind.Kind
	excludeTags bool
}

// NewKubernetesfileWriter returns an IWriter for Kubernetesfiles.
func NewKubernetesfileWriter(excludeTags bool) IWriter {
	return &kubernetesfileWriter{
		kind:        kind.Kubernetesfile,
		excludeTags: excludeTags,
	}
}

// Kind is a getter for the kind.
func (k *kubernetesfileWriter) Kind() kind.Kind {
	return k.kind
}

// WriteFiles writes new Kubernetesfiles given the paths of the
// original Kubernetesfiles and new images that should replace
// the exsting ones.
func (k *kubernetesfileWriter) WriteFiles( // nolint: dupl
	pathImages map[string][]interface{},
	outputDir string,
	done <-chan struct{},
) <-chan IWrittenPath {
	var (
		writtenPaths = make(chan IWrittenPath)
		waitGroup    sync.WaitGroup
	)

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for path, images := range pathImages {
			path := path
			images := images

			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				writtenPath, err := k.writeFile(path, images, outputDir)
				if err != nil {
					select {
					case <-done:
					case writtenPaths <- NewWrittenPath("", "", err):
					}

					return
				}

				select {
				case <-done:
					return
				case writtenPaths <- NewWrittenPath(path, writtenPath, nil):
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

func (k *kubernetesfileWriter) writeFile(
	path string,
	images []interface{},
	outputDir string,
) (string, error) {
	byt, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	_, _, err = scheme.Codecs.UniversalDeserializer().Decode(byt, nil, nil)
	if err != nil {
		return "", fmt.Errorf(
			"'%s' failed to parse with err: %v", path, err,
		)
	}

	var (
		encodedDocs   []interface{}
		imagePosition int
		dec           = yaml.NewDecoder(bytes.NewReader(byt))
	)

	for {
		var doc yaml.MapSlice

		if err = dec.Decode(&doc); err != nil {
			if err != io.EOF {
				return "", fmt.Errorf(
					"'%s' yaml decoder failed with err: %v", path, err,
				)
			}

			break
		}

		if err = k.encodeDoc(path, doc, images, &imagePosition); err != nil {
			return "", err
		}

		encodedDocs = append(encodedDocs, doc)
	}

	if imagePosition < len(images) {
		return "", fmt.Errorf(
			"fewer images exist in '%s' than asked to rewrite", path,
		)
	}

	replacer := strings.NewReplacer("/", "-", "\\", "-")
	outputPath := replacer.Replace(fmt.Sprintf("%s-*", path))

	writtenFile, err := ioutil.TempFile(outputDir, outputPath)
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

func (k *kubernetesfileWriter) encodeDoc(
	path string,
	doc interface{},
	images []interface{},
	imagePosition *int,
) error {
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
			if *imagePosition >= len(images) {
				return fmt.Errorf(
					"more images exist in '%s' than in the Lockfile", path,
				)
			}

			image, ok := images[*imagePosition].(map[string]interface{})
			if !ok {
				return errors.New("malformed image")
			}

			tag, ok := image["tag"].(string)
			if !ok {
				return errors.New("malformed 'tag' in image")
			}

			if k.excludeTags {
				tag = ""
			}

			name, ok := image["name"].(string)
			if !ok {
				return errors.New("malformed 'name' in image")
			}

			digest, ok := image["digest"].(string)
			if !ok {
				return errors.New("malformed 'digest' in image")
			}

			imageLine := parse.NewImage(
				k.kind, name, tag, digest, nil, nil,
			).ImageLine()
			doc[imageLineIndex].Value = imageLine

			*imagePosition++
		}

		for _, item := range doc {
			if err := k.encodeDoc(
				path, item.Value, images, imagePosition,
			); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, doc := range doc {
			if err := k.encodeDoc(
				path, doc, images, imagePosition,
			); err != nil {
				return err
			}
		}
	}

	return nil
}
