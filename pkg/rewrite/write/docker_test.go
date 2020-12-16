package write_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/safe-waters/docker-lock/internal/testutils"
	"github.com/safe-waters/docker-lock/pkg/rewrite/write"
)

func TestDockerfileWriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name        string
		Contents    [][]byte
		Expected    [][]byte
		PathImages  map[string][]interface{}
		ExcludeTags bool
		ShouldFail  bool
	}{
		{
			Name: "Single Dockerfile",
			Contents: [][]byte{
				[]byte(`FROM busybox
COPY . .
FROM redis:latest
FROM golang:latest@sha256:12345
`),
			},
			PathImages: map[string][]interface{}{
				"Dockerfile": {
					map[string]interface{}{
						"name":   "busybox",
						"tag":    "latest",
						"digest": "busybox",
					},
					map[string]interface{}{
						"name":   "redis",
						"tag":    "latest",
						"digest": "redis",
					},
					map[string]interface{}{
						"name":   "golang",
						"tag":    "latest",
						"digest": "golang",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`FROM busybox:latest@sha256:busybox
COPY . .
FROM redis:latest@sha256:redis
FROM golang:latest@sha256:golang
`),
			},
		},
		{
			Name: "Comments and Newlines",
			Contents: [][]byte{
				[]byte(`FROM busybox


COPY . .

# my comment

RUN echo

# my next comment
RUN touch foo
`),
			},
			PathImages: map[string][]interface{}{
				"Dockerfile": {
					map[string]interface{}{
						"name":   "busybox",
						"tag":    "latest",
						"digest": "busybox",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`FROM busybox:latest@sha256:busybox


COPY . .

# my comment

RUN echo

# my next comment
RUN touch foo
`),
			},
		},
		{
			Name: "Multiline",
			Contents: [][]byte{
				[]byte(`FROM \
busybox
FROM redis \
AS \
prod

RUN apt-get update && \
    apt-get intall vim
`),
			},
			PathImages: map[string][]interface{}{
				"Dockerfile": {
					map[string]interface{}{
						"name":   "busybox",
						"tag":    "latest",
						"digest": "busybox",
					},
					map[string]interface{}{
						"name":   "redis",
						"tag":    "latest",
						"digest": "redis",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`FROM busybox:latest@sha256:busybox
FROM redis:latest@sha256:redis AS prod

RUN apt-get update && \
    apt-get intall vim
`),
			},
		},
		{
			Name: "Scratch",
			Contents: [][]byte{
				[]byte(`FROM scratch`),
			},
			PathImages: map[string][]interface{}{
				"Dockerfile": {
					map[string]interface{}{
						"name":   "scratch",
						"tag":    "",
						"digest": "",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`FROM scratch
`),
			},
		},
		{
			Name: "Multiple Dockerfiles",
			Contents: [][]byte{
				[]byte(`FROM busybox
FROM redis
FROM golang
`),
				[]byte(`FROM golang
FROM busybox
FROM redis
`),
			},
			PathImages: map[string][]interface{}{
				"Dockerfile-1": {
					map[string]interface{}{
						"name":   "busybox",
						"tag":    "latest",
						"digest": "busybox-1",
					},
					map[string]interface{}{
						"name":   "redis",
						"tag":    "latest",
						"digest": "redis-1",
					},
					map[string]interface{}{
						"name":   "golang",
						"tag":    "latest",
						"digest": "golang-1",
					},
				},
				"Dockerfile-2": {
					map[string]interface{}{
						"name":   "golang",
						"tag":    "latest",
						"digest": "golang-2",
					},
					map[string]interface{}{
						"name":   "busybox",
						"tag":    "latest",
						"digest": "busybox-2",
					},
					map[string]interface{}{
						"name":   "redis",
						"tag":    "latest",
						"digest": "redis-2",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`FROM busybox:latest@sha256:busybox-1
FROM redis:latest@sha256:redis-1
FROM golang:latest@sha256:golang-1
`),
				[]byte(`FROM golang:latest@sha256:golang-2
FROM busybox:latest@sha256:busybox-2
FROM redis:latest@sha256:redis-2
`),
			},
		},
		{
			Name: "Exclude Tags",
			Contents: [][]byte{
				[]byte(`FROM busybox
FROM redis
FROM golang
`),
			},
			ExcludeTags: true,
			PathImages: map[string][]interface{}{
				"Dockerfile": {
					map[string]interface{}{
						"name":   "busybox",
						"tag":    "latest",
						"digest": "busybox",
					},
					map[string]interface{}{
						"name":   "redis",
						"tag":    "latest",
						"digest": "redis",
					},
					map[string]interface{}{
						"name":   "golang",
						"tag":    "latest",
						"digest": "golang",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`FROM busybox@sha256:busybox
FROM redis@sha256:redis
FROM golang@sha256:golang
`),
			},
		},
		{
			Name: "Stages",
			Contents: [][]byte{
				[]byte(`FROM busybox AS base
FROM redis
FROM base
FROM golang
`),
			},
			PathImages: map[string][]interface{}{
				"Dockerfile": {
					map[string]interface{}{
						"name":   "busybox",
						"tag":    "latest",
						"digest": "busybox",
					},
					map[string]interface{}{
						"name":   "redis",
						"tag":    "latest",
						"digest": "redis",
					},
					map[string]interface{}{
						"name":   "golang",
						"tag":    "latest",
						"digest": "golang",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`FROM busybox:latest@sha256:busybox AS base
FROM redis:latest@sha256:redis
FROM base
FROM golang:latest@sha256:golang
`),
			},
		},
		{
			Name: "Platform",
			Contents: [][]byte{
				[]byte(`FROM --platform=$BUILDPLATFORM busybox \
AS base
FROM --platform=$BUILDPLATFORM redis
FROM --platform=$BUILDPLATFORM base AS anotherbase
`),
			},
			PathImages: map[string][]interface{}{
				"Dockerfile": {
					map[string]interface{}{
						"name":   "busybox",
						"tag":    "latest",
						"digest": "busybox",
					},
					map[string]interface{}{
						"name":   "redis",
						"tag":    "latest",
						"digest": "redis",
					},
				},
			},
			Expected: [][]byte{
				// nolint: lll
				[]byte(`FROM --platform=$BUILDPLATFORM busybox:latest@sha256:busybox AS base
FROM --platform=$BUILDPLATFORM redis:latest@sha256:redis
FROM --platform=$BUILDPLATFORM base AS anotherbase
`),
			},
		},
		{
			Name: "Fewer Images In Dockerfile",
			Contents: [][]byte{
				[]byte(`FROM busybox`),
			},
			PathImages: map[string][]interface{}{
				"Dockerfile": {
					map[string]interface{}{
						"name":   "busybox",
						"tag":    "latest",
						"digest": "busybox",
					},
					map[string]interface{}{
						"name":   "redis",
						"tag":    "latest",
						"digest": "redis",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "More Images In Dockerfile",
			Contents: [][]byte{
				[]byte(`FROM busybox
FROM redis
`),
			},
			PathImages: map[string][]interface{}{
				"Dockerfile": {
					map[string]interface{}{
						"name":   "busybox",
						"tag":    "latest",
						"digest": "busybox",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Only From",
			Contents: [][]byte{
				[]byte(`FROM`),
			},
			PathImages: map[string][]interface{}{
				"Dockerfile": {
					map[string]interface{}{
						"name":   "busybox",
						"tag":    "latest",
						"digest": "busybox",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Only Platform",
			Contents: [][]byte{
				[]byte(`FROM --platform=$BUILDTARGET`),
			},
			PathImages: map[string][]interface{}{
				"Dockerfile": {
					map[string]interface{}{
						"name":   "busybox",
						"tag":    "latest",
						"digest": "busybox",
					},
				},
			},
			ShouldFail: true,
		},
	}

	for _, test := range tests { // nolint: dupl
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := testutils.MakeTempDirInCurrentDir(t)
			defer os.RemoveAll(tempDir)

			var pathsToWrite []string

			tempPathImages := map[string][]interface{}{}

			for path, images := range test.PathImages {
				pathsToWrite = append(pathsToWrite, path)

				path = filepath.Join(tempDir, path)
				tempPathImages[path] = images
			}

			sort.Strings(pathsToWrite)

			testutils.WriteFilesToTempDir(
				t, tempDir, pathsToWrite, test.Contents,
			)

			writer := write.NewDockerfileWriter(test.ExcludeTags)

			done := make(chan struct{})
			defer close(done)

			writtenPathResults := writer.WriteFiles(
				tempPathImages, tempDir, done,
			)

			var got []string

			var err error

			for writtenPath := range writtenPathResults {
				if writtenPath.Err() != nil {
					err = writtenPath.Err()
				}
				got = append(got, writtenPath.NewPath())
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

			testutils.AssertWrittenFilesEqual(t, test.Expected, got)
		})
	}
}
