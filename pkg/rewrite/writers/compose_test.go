package writers_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/rewrite/writers"
)

func TestComposefileWriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name                string
		ComposefilePaths    []string
		ComposefileContents [][]byte
		DockerfilePaths     []string
		DockerfileContents  [][]byte
		Expected            [][]byte
		PathImages          map[string][]*parse.ComposefileImage
		ExcludeTags         bool
		ShouldFail          bool
	}{
		{
			Name:             "Dockerfile",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
			},
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
from busybox
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
			Name:             "Composefile",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
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
						Path:        "docker-compose.yml",
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
			Name:             "Exclude Tags",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
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
						Path:        "docker-compose.yml",
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
			Name:             "Dockerfile And Composefile",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
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
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
from golang
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
						Path:        "docker-compose.yml",
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
			Name:             "More Services In Composefile",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
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
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
from golang
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
			Name:             "Fewer Services In Composefile",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
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
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
from golang
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
						Path:        "docker-compose.yml",
						ServiceName: "svc-compose",
					},
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						Path:        "docker-compose.yml",
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
			Name:             "Multiple Services Same Dockerfile Different Images", // nolint: lll
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
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
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
from golang
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
						Path:        "docker-compose.yml",
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
			Name: "Multiple Composefiles Same Dockerfile Different Images", // nolint: lll
			ComposefilePaths: []string{
				"docker-compose-one.yml", "docker-compose-two.yml",
			},
			ComposefileContents: [][]byte{
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
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
from golang
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
						Path:        "docker-compose.yml",
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
				"docker-compose-two.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						Path:        "docker-compose.yml",
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
			Name:             "Multiple Services Same Dockerfile Same Images", // nolint: lll
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
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
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
from golang
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
						Path:        "docker-compose.yml",
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
			Name: "Multiple Composefiles Same Dockerfile Same Images", // nolint: lll
			ComposefilePaths: []string{
				"docker-compose-one.yml", "docker-compose-two.yml",
			}, // nolint: lll
			ComposefileContents: [][]byte{
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
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
from golang
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
						Path:        "docker-compose.yml",
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
						Path:        "docker-compose.yml",
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
			ComposefilePaths: []string{
				"docker-compose-one.yml", "docker-compose-two.yml",
			},
			ComposefileContents: [][]byte{
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
			DockerfilePaths: []string{"Dockerfile", "AnotherDockerfile"},
			DockerfileContents: [][]byte{
				[]byte(`
from golang
`,
				),
				[]byte(`
from python
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
						Path:        "docker-compose.yml",
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
							Name:   "node",
							Tag:    "latest",
							Digest: "node",
						},
						Path:        "docker-compose-two.yml",
						ServiceName: "svc-compose",
					},
					{
						Image: &parse.Image{
							Name:   "python",
							Tag:    "latest",
							Digest: "python",
						},
						DockerfilePath: "AnotherDockerfile",
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

			tempDir := generateUUID(t)
			makeDir(t, tempDir)
			defer os.RemoveAll(tempDir)

			tempDockerfilePaths := writeFilesToTempDir(
				t, tempDir, test.DockerfilePaths, test.DockerfileContents,
			)
			tempComposefilePaths := writeFilesToTempDir(
				t, tempDir, test.ComposefilePaths, test.ComposefileContents,
			)

			tempPaths := make(
				[]string,
				len(tempDockerfilePaths)+len(tempComposefilePaths),
			)

			var i int

			for _, tempPath := range tempDockerfilePaths {
				tempPaths[i] = tempPath
				i++
			}

			for _, tempPath := range tempComposefilePaths {
				tempPaths[i] = tempPath
				i++
			}

			tempDirPathImages := map[string][]*parse.ComposefileImage{}

			for path, images := range test.PathImages {
				tempDirPath := filepath.Join(tempDir, path)
				for _, image := range images {
					if image.DockerfilePath != "" {
						image.DockerfilePath = filepath.Join(
							tempDir, image.DockerfilePath,
						)
					}
					if image.Path != "" {
						image.Path = filepath.Join(tempDir, image.Path)
					}
				}
				tempDirPathImages[tempDirPath] = images
			}

			dockerfileWriter := &writers.DockerfileWriter{
				ExcludeTags: test.ExcludeTags, Directory: tempDir,
			}
			composefileWriter := &writers.ComposefileWriter{
				DockerfileWriter: dockerfileWriter,
				ExcludeTags:      test.ExcludeTags,
				Directory:        tempDir,
			}

			done := make(chan struct{})
			writtenFiles := composefileWriter.WriteFiles(
				tempDirPathImages, done,
			)

			var writtenPaths []*writers.WrittenPath

			var err error
			for rewrittenPath := range writtenFiles {
				if rewrittenPath.Err != nil {
					err = rewrittenPath.Err
				}
				writtenPaths = append(writtenPaths, rewrittenPath)
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

			for _, rewrittenPath := range writtenPaths {
				got, err := ioutil.ReadFile(rewrittenPath.Path)
				if err != nil {
					t.Fatal(err)
				}

				expectedIndex := -1

				for i, path := range tempPaths {
					if rewrittenPath.OriginalPath == path {
						expectedIndex = i
						break
					}
				}

				if expectedIndex == -1 {
					t.Fatalf(
						"rewrittenPath %s not found in %v",
						rewrittenPath.OriginalPath,
						tempPaths,
					)
				}

				assertWrittenPaths(
					t, test.Expected[expectedIndex], got,
				)
			}
		})
	}
}
