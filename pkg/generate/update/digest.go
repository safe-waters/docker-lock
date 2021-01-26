package update

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
)

type digestRequester struct{}

// NewDigestRequester returns a digest requester based on the library "crane".
func NewDigestRequester() IDigestRequester {
	return &digestRequester{}
}

// Digest queries a registry for a sha256 digest given a name and tag.
func (d *digestRequester) Digest(name string, tag string) (string, error) {
	if name == "" {
		return "", errors.New("image 'name' cannot be empty")
	}

	if tag == "" {
		return "", errors.New("image 'tag' cannot be empty")
	}

	nameTag := fmt.Sprintf("%s:%s", name, tag)

	digest, err := crane.Digest(nameTag)
	if err != nil {
		return "", fmt.Errorf(
			"failed to find digest for '%s' with err: %v", nameTag, err,
		)
	}

	return strings.TrimPrefix(digest, "sha256:"), nil
}
