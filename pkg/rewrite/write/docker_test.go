package write_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/rewrite/write"
)

func TestDockerfileWriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name        string
		Contents    [][]byte
		Expected    [][]byte
		PathImages  map[string][]*parse.DockerfileImage
		ExcludeTags bool
		ShouldFail  bool
	}{
		{
			Name: "Single Dockerfile",
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
			Name: "Scratch",
			Contents: [][]byte{
				[]byte(`
FROM scratch
`),
			},
			PathImages: map[string][]*parse.DockerfileImage{
				"Dockerfile": {
					{
						Image: &parse.Image{
							Name:   "scratch",
							Tag:    "",
							Digest: "",
						},
					},
				},
			},
			Expected: [][]byte{
				[]byte(`
FROM scratch
`),
			},
		},
		{
			Name: "Multiple Dockerfiles",
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
				"Dockerfile-1": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox-1",
						},
					},
					{
						Image: &parse.Image{
							Name:   "redis",
							Tag:    "latest",
							Digest: "redis-1",
						},
					},
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang-1",
						},
					},
				},
				"Dockerfile-2": {
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang-2",
						},
					},
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox-2",
						},
					},
					{
						Image: &parse.Image{
							Name:   "redis",
							Tag:    "latest",
							Digest: "redis-2",
						},
					},
				},
			},
			Expected: [][]byte{
				[]byte(`
FROM busybox:latest@sha256:busybox-1
FROM redis:latest@sha256:redis-1
FROM golang:latest@sha256:golang-1
`),
				[]byte(`
FROM golang:latest@sha256:golang-2
FROM busybox:latest@sha256:busybox-2
FROM redis:latest@sha256:redis-2
`),
			},
		},
		{
			Name: "Exclude Tags",
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
			Name: "Stages",
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
			Name: "Fewer Images In Dockerfile",
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
			Name: "More Images In Dockerfile",
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

			tempDir := makeTempDirInCurrentDir(t)
			defer os.RemoveAll(tempDir)

			var pathsToWrite []string

			tempPathImages := map[string][]*parse.DockerfileImage{}

			for path, images := range test.PathImages {
				pathsToWrite = append(pathsToWrite, path)

				path = filepath.Join(tempDir, path)
				tempPathImages[path] = images
			}

			sort.Strings(pathsToWrite)

			writeFilesToTempDir(
				t, tempDir, pathsToWrite, test.Contents,
			)

			writer := &write.DockerfileWriter{
				Directory:   tempDir,
				ExcludeTags: test.ExcludeTags,
			}
			done := make(chan struct{})
			writtenPathResults := writer.WriteFiles(
				tempPathImages, done,
			)

			var got []string

			var err error

			for writtenPath := range writtenPathResults {
				if writtenPath.Err != nil {
					err = writtenPath.Err
				}
				got = append(got, writtenPath.Path)
			}

			if test.ShouldFail {
				if err == nil {
					t.Fatal("expected error but did not get one")
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}

			sort.Strings(got)

			assertWrittenFiles(t, test.Expected, got)
		})
	}
}
