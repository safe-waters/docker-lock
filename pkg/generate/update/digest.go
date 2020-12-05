package update

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
)

type digestRequester struct{}

func NewDigestRequester() IDigestRequester {
	return &digestRequester{}
}

func (d *digestRequester) Digest(name string, tag string) (string, error) {
	digest, err := crane.Digest(fmt.Sprintf("%s:%s", name, tag))
	if err != nil {
		return "", err
	}

	return strings.TrimPrefix(digest, "sha256:"), nil
}
