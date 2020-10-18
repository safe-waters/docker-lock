package write_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/rewrite/write"
)

func TestComposefileWriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name        string
		Contents    [][]byte
		Expected    [][]byte
		PathImages  map[string][]*parse.ComposefileImage
		ExcludeTags bool
		ShouldFail  bool
	}{
		{
			Name: "Dockerfile",
			Contents: [][]byte{
				[]byte(`
from busybox
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
			},
			PathImages: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						DockerfilePath: "Dockerfile",
						ServiceName:    "svc",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`
from busybox:latest@sha256:busybox
`,
				),
			},
		},
		{
			Name: "Composefile",
			Contents: [][]byte{
				[]byte(`
version: '3'

services:
  svc:
    image: busybox
`,
				),
			},
			PathImages: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						ServiceName: "svc",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`
version: '3'

services:
  svc:
    image: busybox:latest@sha256:busybox
`,
				),
			},
		},
		{
			Name: "Scratch",
			Contents: [][]byte{
				[]byte(`
version: '3'

services:
  svc:
    image: scratch
`,
				),
			},
			PathImages: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "scratch",
							Tag:    "",
							Digest: "",
						},
						ServiceName: "svc",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`
version: '3'

services:
  svc:
    image: scratch
`,
				),
			},
		},
		{
			Name: "Exclude Tags",
			Contents: [][]byte{
				[]byte(`
version: '3'

services:
  svc:
    image: busybox
`,
				),
			},
			PathImages: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						ServiceName: "svc",
					},
				},
			},
			ExcludeTags: true,
			Expected: [][]byte{
				[]byte(`
version: '3'

services:
  svc:
    image: busybox@sha256:busybox
`,
				),
			},
		},
		{
			Name: "Dockerfile And Composefile",
			Contents: [][]byte{
				[]byte(`
from golang
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox
  svc-docker:
    build: .
`,
				),
			},
			PathImages: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						ServiceName: "svc-compose",
					},
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang",
						},
						DockerfilePath: "Dockerfile",
						ServiceName:    "svc-docker",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`
from golang:latest@sha256:golang
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox:latest@sha256:busybox
  svc-docker:
    build: .
`,
				),
			},
		},
		{
			Name: "More Services In Composefile",
			Contents: [][]byte{
				[]byte(`
from golang
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox
  svc-docker:
    build: .
`,
				),
			},
			PathImages: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang",
						},
						DockerfilePath: "Dockerfile",
						ServiceName:    "svc-docker",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Fewer Services In Composefile",
			Contents: [][]byte{
				[]byte(`
from golang
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox
  svc-docker:
    build: .
`,
				),
			},
			PathImages: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						ServiceName: "svc-compose",
					},
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						ServiceName: "svc-unknown",
					},
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang",
						},
						DockerfilePath: "Dockerfile",
						ServiceName:    "svc-docker",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Multiple Services Same Dockerfile Different Images",
			Contents: [][]byte{
				[]byte(`
from golang
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox
  svc-docker:
    build: .
  svc-another-docker:
    build: .
`,
				),
			},
			PathImages: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						ServiceName: "svc-compose",
					},
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						DockerfilePath: "Dockerfile",
						ServiceName:    "svc-another-docker",
					},
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang",
						},
						DockerfilePath: "Dockerfile",
						ServiceName:    "svc-docker",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Multiple Composefiles Same Dockerfile Different Images",
			Contents: [][]byte{
				[]byte(`
from golang
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox
  svc-docker:
    build: .
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox
  svc-docker:
    build: .
`,
				),
			},
			PathImages: map[string][]*parse.ComposefileImage{
				"docker-compose-1.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						ServiceName: "svc-compose",
					},
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						DockerfilePath: "Dockerfile",
						ServiceName:    "svc-docker",
					},
				},
				"docker-compose-2.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						ServiceName: "svc-compose",
					},
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang",
						},
						DockerfilePath: "Dockerfile",
						ServiceName:    "svc-docker",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Multiple Services Same Dockerfile Same Images",
			Contents: [][]byte{
				[]byte(`
from golang
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox
  svc-docker:
    build: .
  svc-another-docker:
    build: .
`,
				),
			},
			PathImages: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						ServiceName: "svc-compose",
					},
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang",
						},
						DockerfilePath: "Dockerfile",
						ServiceName:    "svc-another-docker",
					},
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang",
						},
						DockerfilePath: "Dockerfile",
						ServiceName:    "svc-docker",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`
from golang:latest@sha256:golang
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox:latest@sha256:busybox
  svc-docker:
    build: .
  svc-another-docker:
    build: .
`,
				),
			},
		},
		{
			Name: "Multiple Composefiles Same Dockerfile Same Images",
			Contents: [][]byte{
				[]byte(`
from golang
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox
  svc-docker:
    build: .
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox
  svc-docker:
    build: .
`,
				),
			},
			PathImages: map[string][]*parse.ComposefileImage{
				"docker-compose-one.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						ServiceName: "svc-compose",
					},
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang",
						},
						DockerfilePath: "Dockerfile",
						ServiceName:    "svc-docker",
					},
				},
				"docker-compose-two.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						ServiceName: "svc-compose",
					},
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang",
						},
						DockerfilePath: "Dockerfile",
						ServiceName:    "svc-docker",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`
from golang:latest@sha256:golang
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox:latest@sha256:busybox
  svc-docker:
    build: .
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox:latest@sha256:busybox
  svc-docker:
    build: .
`,
				),
			},
		},
		{
			Name: "Multiple Composefiles Multiple Services",
			Contents: [][]byte{
				[]byte(`
from golang
`,
				),
				[]byte(`
from python
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox
  svc-docker:
    build: .
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: node
  svc-another-docker:
    build:
      context: .
      dockerfile: AnotherDockerfile
`,
				),
			},
			PathImages: map[string][]*parse.ComposefileImage{
				"docker-compose-1.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						ServiceName: "svc-compose",
					},
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang",
						},
						DockerfilePath: "Dockerfile-1",
						ServiceName:    "svc-docker",
					},
				},
				"docker-compose-2.yml": {
					{
						Image: &parse.Image{
							Name:   "node",
							Tag:    "latest",
							Digest: "node",
						},
						ServiceName: "svc-compose",
					},
					{
						Image: &parse.Image{
							Name:   "python",
							Tag:    "latest",
							Digest: "python",
						},
						DockerfilePath: "Dockerfile-2",
						ServiceName:    "svc-another-docker",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`
from golang:latest@sha256:golang
`,
				),
				[]byte(`
from python:latest@sha256:python
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox:latest@sha256:busybox
  svc-docker:
    build: .
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: node:latest@sha256:node
  svc-another-docker:
    build:
      context: .
      dockerfile: AnotherDockerfile
`,
				),
			},
		},
	}
	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := makeTempDirInCurrentDir(t)
			defer os.RemoveAll(tempDir)

			uniquePathsToWrite := map[string]struct{}{}

			tempPathImages := map[string][]*parse.ComposefileImage{}

			for composefilePath, images := range test.PathImages {
				for _, image := range images {
					if image.DockerfilePath != "" {
						uniquePathsToWrite[image.DockerfilePath] = struct{}{}
						image.DockerfilePath = filepath.Join(
							tempDir, image.DockerfilePath,
						)
					}
				}

				uniquePathsToWrite[composefilePath] = struct{}{}

				composefilePath = filepath.Join(tempDir, composefilePath)
				tempPathImages[composefilePath] = images
			}

			var pathsToWrite []string
			for path := range uniquePathsToWrite {
				pathsToWrite = append(pathsToWrite, path)
			}

			sort.Strings(pathsToWrite)

			writeFilesToTempDir(
				t, tempDir, pathsToWrite, test.Contents,
			)

			dockerfileWriter := &write.DockerfileWriter{
				Directory:   tempDir,
				ExcludeTags: test.ExcludeTags,
			}
			composefileWriter := &write.ComposefileWriter{
				DockerfileWriter: dockerfileWriter,
				Directory:        tempDir,
				ExcludeTags:      test.ExcludeTags,
			}

			done := make(chan struct{})
			writtenPathResults := composefileWriter.WriteFiles(
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
