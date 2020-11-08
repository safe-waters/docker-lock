package rewrite_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"

	cmd_rewrite "github.com/safe-waters/docker-lock/cmd/rewrite"
)

func TestRewriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name       string
		Contents   [][]byte
		Expected   [][]byte
		ShouldFail bool
	}{
		{
			Name: "Composefile Overrides Dockerfile",
			Contents: [][]byte{
				[]byte(`FROM golang
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
				[]byte(`
{
	"dockerfiles": {
		"Dockerfile": [
			{
				"name": "not_used",
				"tag": "latest",
				"digest": "not_used"
			}
		]
	},
	"composefiles": {
		"docker-compose.yml": [
			{
				"name": "golang",
				"tag": "latest",
				"digest": "golang",
				"dockerfile": "Dockerfile",
				"service": "svc"
			}
		]
	}
}
`,
				),
			},
			Expected: [][]byte{
				[]byte(`FROM golang:latest@sha256:golang
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
		},
		{
			Name: "Duplicate Services Same Dockerfile Images",
			Contents: [][]byte{
				[]byte(`FROM golang
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
  another-svc:
    build: .
`,
				),
				[]byte(`
{
	"composefiles": {
		"docker-compose.yml": [
			{
				"name": "golang",
				"tag": "latest",
				"digest": "golang",
				"dockerfile": "Dockerfile",
				"service": "another-svc"
			},
			{
				"name": "golang",
				"tag": "latest",
				"digest": "golang",
				"dockerfile": "Dockerfile",
				"service": "svc"
			}
		]
	}
}
`,
				),
			},
			Expected: [][]byte{
				[]byte(`FROM golang:latest@sha256:golang
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
  another-svc:
    build: .
`,
				),
			},
		},
		{
			Name: "Different Composefiles Same Dockerfile Images",
			Contents: [][]byte{
				[]byte(`FROM golang
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
				[]byte(`
{
	"composefiles": {
		"docker-compose-1.yml": [
			{
				"name": "golang",
				"tag": "latest",
				"digest": "golang",
				"dockerfile": "Dockerfile",
				"service": "svc"
			}
		],
		"docker-compose-2.yml": [
			{
				"name": "golang",
				"tag": "latest",
				"digest": "golang",
				"dockerfile": "Dockerfile",
				"service": "svc"
			}
		]
	}
}
`,
				),
			},
			Expected: [][]byte{
				[]byte(`FROM golang:latest@sha256:golang
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
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
		},
		{
			Name: "Duplicate Services Different Dockerfile Images",
			Contents: [][]byte{
				[]byte(`FROM golang
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
  another-svc:
    build: .
`,
				),
				[]byte(`
{
	"composefiles": {
		"docker-compose.yml": [
			{
				"name": "golang",
				"tag": "latest",
				"digest": "golang",
				"dockerfile": "Dockerfile",
				"service": "another-svc"
			},
			{
				"name": "notgolang",
				"tag": "latest",
				"digest": "notgolang",
				"dockerfile": "Dockerfile",
				"service": "svc"
			}
		]
	}
}
`,
				),
			},
			ShouldFail: true,
		},
		{
			Name: "Different Composefiles Different Dockerfile Images",
			Contents: [][]byte{
				[]byte(`FROM golang
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
				[]byte(`
{
	"composefiles": {
		"docker-compose-one.yml": [
			{
				"name": "golang",
				"tag": "latest",
				"digest": "golang",
				"dockerfile": "Dockerfile",
				"service": "svc"
			}
		],
		"docker-compose-two.yml": [
			{
				"name": "notgolang",
				"tag": "latest",
				"digest": "notgolang",
				"dockerfile": "Dockerfile",
				"service": "svc"
			}
		]
	}
}
`,
				),
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

			var lockfile generate.Lockfile
			if err := json.Unmarshal(
				test.Contents[len(test.Contents)-1], &lockfile,
			); err != nil {
				t.Fatal(err)
			}

			uniquePathsToWrite := map[string]struct{}{}

			composefileImagesWithTempDir := map[string][]*parse.ComposefileImage{} // nolint: lll

			for composefilePath, images := range lockfile.ComposefileImages {
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
				composefileImagesWithTempDir[composefilePath] = images
			}

			dockerfileImagesWithTempDir := map[string][]*parse.DockerfileImage{}

			for dockerfilePath, images := range lockfile.DockerfileImages {
				uniquePathsToWrite[dockerfilePath] = struct{}{}

				dockerfilePath = filepath.Join(tempDir, dockerfilePath)
				dockerfileImagesWithTempDir[dockerfilePath] = images
			}

			var pathsToWrite []string
			for path := range uniquePathsToWrite {
				pathsToWrite = append(pathsToWrite, path)
			}

			sort.Strings(pathsToWrite)

			got := writeFilesToTempDir(
				t, tempDir, pathsToWrite, test.Contents[:len(test.Contents)-1],
			)

			flags, err := cmd_rewrite.NewFlags("", tempDir, false)
			if err != nil {
				t.Fatal(err)
			}

			rewriter, err := cmd_rewrite.SetupRewriter(flags)
			if err != nil {
				t.Fatal(err)
			}

			lockfileWithTempDir := &generate.Lockfile{
				DockerfileImages:  dockerfileImagesWithTempDir,
				ComposefileImages: composefileImagesWithTempDir,
			}

			lockfileByt, err := json.Marshal(lockfileWithTempDir)
			if err != nil {
				t.Fatal(err)
			}
			reader := bytes.NewReader(lockfileByt)

			err = rewriter.RewriteLockfile(reader)

			if test.ShouldFail {
				if err == nil {
					t.Fatal("expected error but did not get one")
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assertWrittenFiles(t, test.Expected, got)
		})
	}
}
