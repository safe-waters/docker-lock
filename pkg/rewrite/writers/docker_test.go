package writers_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/rewrite/writers"
)

func TestDockerfileWriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name        string
		Paths       []string
		Contents    [][]byte
		Expected    [][]byte
		PathImages  map[string][]*parse.DockerfileImage
		ExcludeTags bool
		ShouldFail  bool
	}{
		{
			Name:  "Single Dockerfile",
			Paths: []string{"Dockerfile"},
			Contents: [][]byte{
				[]byte(`
from busybox
COPY . .
FROM redis:latest
# comment
FROM golang:latest@sha256:12345
`),
			},
			PathImages: map[string][]*parse.DockerfileImage{
				"Dockerfile": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
					},
					{
						Image: &parse.Image{
							Name:   "redis",
							Tag:    "latest",
							Digest: "redis",
						},
					},
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang",
						},
					},
				},
			},
			Expected: [][]byte{
				[]byte(`
from busybox:latest@sha256:busybox
COPY . .
FROM redis:latest@sha256:redis
# comment
FROM golang:latest@sha256:golang
`),
			},
		},
		{
			Name:  "Multiple Dockerfiles",
			Paths: []string{"Dockerfile-one", "Dockerfile-two"},
			Contents: [][]byte{
				[]byte(`
FROM busybox
FROM redis
FROM golang
`),
				[]byte(`
FROM golang
FROM busybox
FROM redis
`),
			},
			PathImages: map[string][]*parse.DockerfileImage{
				"Dockerfile-one": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox-one",
						},
					},
					{
						Image: &parse.Image{
							Name:   "redis",
							Tag:    "latest",
							Digest: "redis-one",
						},
					},
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang-one",
						},
					},
				},
				"Dockerfile-two": {
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang-two",
						},
					},
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox-two",
						},
					},
					{
						Image: &parse.Image{
							Name:   "redis",
							Tag:    "latest",
							Digest: "redis-two",
						},
					},
				},
			},
			Expected: [][]byte{
				[]byte(`
FROM busybox:latest@sha256:busybox-one
FROM redis:latest@sha256:redis-one
FROM golang:latest@sha256:golang-one
`),
				[]byte(`
FROM golang:latest@sha256:golang-two
FROM busybox:latest@sha256:busybox-two
FROM redis:latest@sha256:redis-two
`),
			},
		},
		{
			Name:  "Exclude Tags",
			Paths: []string{"Dockerfile"},
			Contents: [][]byte{
				[]byte(`
FROM busybox
FROM redis
FROM golang
`),
			},
			ExcludeTags: true,
			PathImages: map[string][]*parse.DockerfileImage{
				"Dockerfile": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
					},
					{
						Image: &parse.Image{
							Name:   "redis",
							Tag:    "latest",
							Digest: "redis",
						},
					},
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang",
						},
					},
				},
			},
			Expected: [][]byte{
				[]byte(`
FROM busybox@sha256:busybox
FROM redis@sha256:redis
FROM golang@sha256:golang
`),
			},
		},
		{
			Name:  "Stages",
			Paths: []string{"Dockerfile"},
			Contents: [][]byte{
				[]byte(`
FROM busybox AS base
FROM redis
FROM base
FROM golang
`),
			},
			PathImages: map[string][]*parse.DockerfileImage{
				"Dockerfile": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
					},
					{
						Image: &parse.Image{
							Name:   "redis",
							Tag:    "latest",
							Digest: "redis",
						},
					},
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang",
						},
					},
				},
			},
			Expected: [][]byte{
				[]byte(`
FROM busybox:latest@sha256:busybox AS base
FROM redis:latest@sha256:redis
FROM base
FROM golang:latest@sha256:golang
`),
			},
		},
		{
			Name:  "Fewer Images In Dockerfile",
			Paths: []string{"Dockerfile"},
			Contents: [][]byte{
				[]byte(`
FROM busybox
`),
			},
			PathImages: map[string][]*parse.DockerfileImage{
				"Dockerfile": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
					},
					{
						Image: &parse.Image{
							Name:   "redis",
							Tag:    "latest",
							Digest: "redis",
						},
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name:  "More Images In Dockerfile",
			Paths: []string{"Dockerfile"},
			Contents: [][]byte{
				[]byte(`
FROM busybox
FROM redis
`),
			},
			PathImages: map[string][]*parse.DockerfileImage{
				"Dockerfile": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
					},
				},
			},
			ShouldFail: true,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := generateUUID(t)
			makeDir(t, tempDir)
			defer os.RemoveAll(tempDir)

			tempPaths := writeFilesToTempDir(
				t, tempDir, test.Paths, test.Contents,
			)

			tempDirPathImages := map[string][]*parse.DockerfileImage{}

			for path, images := range test.PathImages {
				tempDirPath := filepath.Join(tempDir, path)
				tempDirPathImages[tempDirPath] = images
			}

			writer := &writers.DockerfileWriter{
				ExcludeTags: test.ExcludeTags, Directory: tempDir,
			}

			done := make(chan struct{})
			writtenFiles := writer.WriteFiles(tempDirPathImages, done)

			for writtenPath := range writtenFiles {
				if test.ShouldFail {
					if writtenPath.Err == nil {
						t.Fatal("expected error but did not get one")
					}

					return
				}

				if writtenPath.Err != nil {
					t.Fatal(writtenPath.Err)
				}

				got, err := ioutil.ReadFile(writtenPath.Path)
				if err != nil {
					t.Fatal(err)
				}

				var expectedIndex int

				var writtenPathFound bool
				for _, path := range tempPaths {
					if writtenPath.OriginalPath == path {
						writtenPathFound = true
						break
					}
					expectedIndex++
				}

				if !writtenPathFound {
					t.Fatalf(
						"rewrittenPath %s not found in %v",
						writtenPath.OriginalPath,
						tempPaths,
					)
				}

				assertWrittenPaths(t, test.Expected[expectedIndex], got)
			}
		})
	}
}
