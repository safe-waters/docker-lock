package parse

import (
	"fmt"
	"sync"

	"github.com/safe-waters/docker-lock/pkg/kind"
)

type image struct {
	kind          kind.Kind
	name          string
	tag           string
	digest        string
	metadataMutex *sync.Mutex
	metadata      map[string]interface{}
	err           error
}

// NewImage is an image of a specific kind, such as a
// Dockerfile, Composefile, or Kubernetesfile.
// The differences in these images can be captured in the metadata.
func NewImage(
	kind kind.Kind,
	name string,
	tag string,
	digest string,
	metadata map[string]interface{},
	err error,
) IImage {
	return &image{
		kind:          kind,
		name:          name,
		tag:           tag,
		digest:        digest,
		metadataMutex: &sync.Mutex{},
		metadata:      metadata,
		err:           err,
	}
}

// SetKind is a setter for the kind.
func (i *image) SetKind(kind kind.Kind) {
	i.kind = kind
}

// Kind is a getter for the kind.
func (i *image) Kind() kind.Kind {
	return i.kind
}

// SetName is a setter for the name.
func (i *image) SetName(name string) {
	i.name = name
}

// Name is a getter for the name.
func (i *image) Name() string {
	return i.name
}

// SetTag is a setter for the tag.
func (i *image) SetTag(tag string) {
	i.tag = tag
}

// Tag is a getter for the tag.
func (i *image) Tag() string {
	return i.tag
}

// SetDigest is a setter for the digest.
func (i *image) SetDigest(digest string) {
	i.digest = digest
}

// Digest is a getter for the digest.
func (i *image) Digest() string {
	return i.digest
}

// SetMetdata is a setter for the metadata.
func (i *image) SetMetadata(metadata map[string]interface{}) {
	i.metadataMutex.Lock()
	defer i.metadataMutex.Unlock()

	i.metadata = i.deepCopyMetadata(metadata)
}

// Metadata is a getter for the metadata.
func (i *image) Metadata() map[string]interface{} {
	i.metadataMutex.Lock()
	defer i.metadataMutex.Unlock()

	metadata := i.deepCopyMetadata(i.metadata)

	return metadata
}

// ImageLine is a getter for the image line. If the image
// has a tag, this will be of the format "name:tag". If the image
// has a digest, this will be of the format "name@sha256:digest". If the
// image has a tag and a digest, this will be of the format
// "name:tag@sha256:digest".
func (i *image) ImageLine() string {
	imageLine := i.Name()

	if i.Tag() != "" {
		imageLine = fmt.Sprintf("%s:%s", imageLine, i.Tag())
	}

	if i.Digest() != "" {
		imageLine = fmt.Sprintf("%s@sha256:%s", imageLine, i.Digest())
	}

	return imageLine
}

// SetNameTagDigestFromImageLine is a setter for the name, tag, and digest from
// an image line. It accepts inputs of the format "name", "name:tag",
// "name@sha256:digest", and "name:tag@sha256:digest".
func (i *image) SetNameTagDigestFromImageLine(imageLine string) {
	var (
		tagSeparator    = -1
		digestSeparator = -1
	)

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

	i.SetName(name)
	i.SetTag(tag)
	i.SetDigest(digest)
}

// SetErr is a setter for the error.
func (i *image) SetErr(err error) {
	i.err = err
}

// Err is a getter for the error.
func (i *image) Err() error {
	return i.err
}

func (i *image) deepCopyMetadata(
	metadata map[string]interface{},
) map[string]interface{} {
	copy := make(map[string]interface{})

	for k, v := range metadata {
		vm, ok := v.(map[string]interface{})
		if ok {
			copy[k] = i.deepCopyMetadata(vm)
		} else {
			copy[k] = v
		}
	}

	return copy
}
