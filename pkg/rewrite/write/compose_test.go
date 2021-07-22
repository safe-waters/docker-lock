package write_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/safe-waters/docker-lock/internal/testutils"
	"github.com/safe-waters/docker-lock/pkg/rewrite/write"
)

func TestComposefileWriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name        string
		Contents    [][]byte
		Expected    [][]byte
		PathImages  map[string][]interface{}
		EnvVars     map[string]string
		ExcludeTags bool
		ShouldFail  bool
	}{
		{
			Name: "Dockerfile",
			Contents: [][]byte{
				[]byte(`FROM busybox
`),
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
			},
			PathImages: map[string][]interface{}{
				"docker-compose.yml": {
					map[string]interface{}{
						"name":       "busybox",
						"tag":        "latest",
						"digest":     "busybox",
						"dockerfile": "Dockerfile",
						"service":    "svc",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`FROM busybox:latest@sha256:busybox
`),
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
			PathImages: map[string][]interface{}{
				"docker-compose.yml": {
					map[string]interface{}{
						"name":    "busybox",
						"tag":     "latest",
						"digest":  "busybox",
						"service": "svc",
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
			Name: "Composefile With Required Env Vars",
			Contents: [][]byte{
				[]byte(`
version: '3'

services:
  svc:
    image: busybox
    container_name: "${SOME_VAR?error message}"
`,
				),
			},
			PathImages: map[string][]interface{}{
				"docker-compose.yml": {
					map[string]interface{}{
						"name":    "busybox",
						"tag":     "latest",
						"digest":  "busybox",
						"service": "svc",
					},
				},
			},
			EnvVars: map[string]string{"SOME_VAR": "value"},
			Expected: [][]byte{
				[]byte(`
version: '3'

services:
  svc:
    image: busybox:latest@sha256:busybox
    container_name: "${SOME_VAR?error message}"
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
			PathImages: map[string][]interface{}{
				"docker-compose.yml": {
					map[string]interface{}{
						"name":    "scratch",
						"tag":     "",
						"digest":  "",
						"service": "svc",
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
			PathImages: map[string][]interface{}{
				"docker-compose.yml": {
					map[string]interface{}{
						"name":    "busybox",
						"tag":     "latest",
						"digest":  "busybox",
						"service": "svc",
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
				[]byte(`FROM golang
`),
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
			PathImages: map[string][]interface{}{
				"docker-compose.yml": {
					map[string]interface{}{
						"name":    "busybox",
						"tag":     "latest",
						"digest":  "busybox",
						"service": "svc-compose",
					},
					map[string]interface{}{
						"name":       "golang",
						"tag":        "latest",
						"digest":     "golang",
						"dockerfile": "Dockerfile",
						"service":    "svc-docker",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`FROM golang:latest@sha256:golang
`),
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
			Name: "More Services Without Images In Composefile",
			Contents: [][]byte{
				[]byte(`FROM golang
`),
				[]byte(`FROM golang
`),

				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox
  svc-docker:
    build:
      context: .
      dockerfile: Dockerfile-1
  svc-docker-context:
    build:
      context: .
      dockerfile: Dockerfile-2
  svc-extra:
    build:
      args:
        a: b
`,
				),
			},
			PathImages: map[string][]interface{}{
				"docker-compose.yml": {
					map[string]interface{}{
						"name":    "busybox",
						"tag":     "latest",
						"digest":  "busybox",
						"service": "svc-compose",
					},
					map[string]interface{}{
						"name":       "golang",
						"tag":        "latest",
						"digest":     "golang",
						"dockerfile": "Dockerfile-1",
						"service":    "svc-docker",
					},
					map[string]interface{}{
						"name":       "golang",
						"tag":        "latest",
						"digest":     "golang",
						"dockerfile": "Dockerfile-2",
						"service":    "svc-docker-context",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`FROM golang:latest@sha256:golang
`),
				[]byte(`FROM golang:latest@sha256:golang
`),

				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox:latest@sha256:busybox
  svc-docker:
    build:
      context: .
      dockerfile: Dockerfile-1
  svc-docker-context:
    build:
      context: .
      dockerfile: Dockerfile-2
  svc-extra:
    build:
      args:
        a: b
`,
				),
			},
		},
		{
			Name: "More Services With Images In Composefile",
			Contents: [][]byte{
				[]byte(`FROM golang
`),
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
			PathImages: map[string][]interface{}{
				"docker-compose.yml": {
					map[string]interface{}{
						"name":       "golang",
						"tag":        "latest",
						"digest":     "golang",
						"dockerfile": "Dockerfile",
						"service":    "svc-docker",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Fewer Services In Composefile",
			Contents: [][]byte{
				[]byte(`FROM golang
`),
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
			PathImages: map[string][]interface{}{
				"docker-compose.yml": {
					map[string]interface{}{
						"name":    "busybox",
						"tag":     "latest",
						"digest":  "busybox",
						"service": "svc-compose",
					},
					map[string]interface{}{
						"name":    "busybox",
						"tag":     "latest",
						"digest":  "busybox",
						"service": "svc-unknown",
					},
					map[string]interface{}{
						"name":       "golang",
						"tag":        "latest",
						"digest":     "golang",
						"dockerfile": "Dockerfile",
						"service":    "svc-docker",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Multiple Services Same Dockerfile Different Images",
			Contents: [][]byte{
				[]byte(`FROM golang
`),
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
			PathImages: map[string][]interface{}{
				"docker-compose.yml": {
					map[string]interface{}{
						"name":    "busybox",
						"tag":     "latest",
						"digest":  "busybox",
						"service": "svc-compose",
					},
					map[string]interface{}{
						"name":       "busybox",
						"tag":        "latest",
						"digest":     "busybox",
						"dockerfile": "Dockerfile",
						"service":    "svc-another-docker",
					},
					map[string]interface{}{
						"name":       "golang",
						"tag":        "latest",
						"digest":     "golang",
						"dockerfile": "Dockerfile",
						"service":    "svc-docker",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Multiple Composefiles Same Dockerfile Different Images",
			Contents: [][]byte{
				[]byte(`FROM golang
`),
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
			PathImages: map[string][]interface{}{
				"docker-compose-1.yml": {
					map[string]interface{}{
						"name":    "busybox",
						"tag":     "latest",
						"digest":  "busybox",
						"service": "svc-compose",
					},
					map[string]interface{}{
						"name":       "busybox",
						"tag":        "latest",
						"digest":     "busybox",
						"dockerfile": "Dockerfile",
						"service":    "svc-docker",
					},
				},
				"docker-compose-2.yml": {
					map[string]interface{}{
						"name":    "busybox",
						"tag":     "latest",
						"digest":  "busybox",
						"service": "svc-compose",
					},
					map[string]interface{}{
						"name":       "golang",
						"tag":        "latest",
						"digest":     "golang",
						"dockerfile": "Dockerfile",
						"service":    "svc-docker",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Multiple Services Same Dockerfile Same Images",
			Contents: [][]byte{
				[]byte(`FROM golang
`),
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
			PathImages: map[string][]interface{}{
				"docker-compose.yml": {
					map[string]interface{}{
						"name":    "busybox",
						"tag":     "latest",
						"digest":  "busybox",
						"service": "svc-compose",
					},
					map[string]interface{}{
						"name":       "golang",
						"tag":        "latest",
						"digest":     "golang",
						"dockerfile": "Dockerfile",
						"service":    "svc-another-docker",
					},
					map[string]interface{}{
						"name":       "golang",
						"tag":        "latest",
						"digest":     "golang",
						"dockerfile": "Dockerfile",
						"service":    "svc-docker",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`FROM golang:latest@sha256:golang
`),
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
				[]byte(`FROM golang
`),
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
			PathImages: map[string][]interface{}{
				"docker-compose-one.yml": {
					map[string]interface{}{
						"name":    "busybox",
						"tag":     "latest",
						"digest":  "busybox",
						"service": "svc-compose",
					},
					map[string]interface{}{
						"name":       "golang",
						"tag":        "latest",
						"digest":     "golang",
						"dockerfile": "Dockerfile",
						"service":    "svc-docker",
					},
				},
				"docker-compose-two.yml": {
					map[string]interface{}{
						"name":    "busybox",
						"tag":     "latest",
						"digest":  "busybox",
						"service": "svc-compose",
					},
					map[string]interface{}{
						"name":       "golang",
						"tag":        "latest",
						"digest":     "golang",
						"dockerfile": "Dockerfile",
						"service":    "svc-docker",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`FROM golang:latest@sha256:golang
`),
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
				[]byte(`FROM golang
`),
				[]byte(`FROM python
`),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox
  svc-docker:
    build:
        context: .
        dockerfile: Dockerfile-1
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
      dockerfile: Dockerfile-2
`,
				),
			},
			PathImages: map[string][]interface{}{
				"docker-compose-1.yml": {
					map[string]interface{}{
						"name":    "busybox",
						"tag":     "latest",
						"digest":  "busybox",
						"service": "svc-compose",
					},
					map[string]interface{}{
						"name":       "golang",
						"tag":        "latest",
						"digest":     "golang",
						"dockerfile": "Dockerfile-1",
						"service":    "svc-docker",
					},
				},
				"docker-compose-2.yml": {
					map[string]interface{}{
						"name":    "node",
						"tag":     "latest",
						"digest":  "node",
						"service": "svc-compose",
					},
					map[string]interface{}{
						"name":       "python",
						"tag":        "latest",
						"digest":     "python",
						"dockerfile": "Dockerfile-2",
						"service":    "svc-another-docker",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`FROM golang:latest@sha256:golang
`),
				[]byte(`FROM python:latest@sha256:python
`),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox:latest@sha256:busybox
  svc-docker:
    build:
        context: .
        dockerfile: Dockerfile-1
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
      dockerfile: Dockerfile-2
`,
				),
			},
		},
	}
	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			for k, v := range test.EnvVars {
				os.Setenv(k, v)
			}

			tempDir := testutils.MakeTempDirInCurrentDir(t)
			defer os.RemoveAll(tempDir)

			uniquePathsToWrite := map[string]struct{}{}

			tempPathImages := map[string][]interface{}{}

			for composefilePath, images := range test.PathImages {
				for _, image := range images {
					image := image.(map[string]interface{})
					if image["dockerfile"] != nil {
						dockerfilePath := image["dockerfile"].(string)
						uniquePathsToWrite[dockerfilePath] = struct{}{}
						image["dockerfile"] = filepath.Join(
							tempDir, dockerfilePath,
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

			testutils.WriteFilesToTempDir(
				t, tempDir, pathsToWrite, test.Contents,
			)

			dockerfileWriter := write.NewDockerfileWriter(test.ExcludeTags)

			composefileWriter, err := write.NewComposefileWriter(
				dockerfileWriter, test.ExcludeTags,
			)
			if err != nil {
				t.Fatal(err)
			}

			done := make(chan struct{})
			defer close(done)

			writtenPathResults := composefileWriter.WriteFiles(
				tempPathImages, tempDir, done,
			)

			var got []string

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
