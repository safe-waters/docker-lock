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
		Name                        string
		ComposefileContents         [][]byte
		DockerfileContents          [][]byte
		ExpectedDockerfileContents  [][]byte
		ExpectedComposefileContents [][]byte
		LockfileContents            []byte
		ShouldFail                  bool
	}{
		{
			Name: "Composefile Overrides Dockerfile",
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
			},
			DockerfileContents: [][]byte{
				[]byte(`
from golang
`,
				),
			},
			LockfileContents: []byte(`
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
			ExpectedComposefileContents: [][]byte{
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
			},
			ExpectedDockerfileContents: [][]byte{
				[]byte(`
from golang:latest@sha256:golang
`,
				),
			},
		},
		{
			Name: "Duplicate Services Same Dockerfile Images",
			ComposefileContents: [][]byte{
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
			DockerfileContents: [][]byte{
				[]byte(`
from golang
`,
				),
			},
			LockfileContents: []byte(`
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
			ExpectedComposefileContents: [][]byte{
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
			ExpectedDockerfileContents: [][]byte{
				[]byte(`
from golang:latest@sha256:golang
`,
				),
			},
		},
		{
			Name: "Different Composefiles Same Dockerfile Images",
			ComposefileContents: [][]byte{
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
			DockerfileContents: [][]byte{
				[]byte(`
from golang
`,
				),
			},
			LockfileContents: []byte(`
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
			ExpectedComposefileContents: [][]byte{
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
			ExpectedDockerfileContents: [][]byte{
				[]byte(`
from golang:latest@sha256:golang
`,
				),
			},
		},
		{
			Name: "Duplicate Services Different Dockerfile Images",
			ComposefileContents: [][]byte{
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
			DockerfileContents: [][]byte{
				[]byte(`
from golang
`,
				),
			},
			LockfileContents: []byte(`
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
			ShouldFail: true,
		},
		{
			Name: "Different Composefiles Different Dockerfile Images",
			ComposefileContents: [][]byte{
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
			DockerfileContents: [][]byte{
				[]byte(`
from golang
`,
				),
			},
			LockfileContents: []byte(`
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
				test.LockfileContents, &lockfile,
			); err != nil {
				t.Fatal(err)
			}

			uniqueDockerfilePaths := map[string]struct{}{}

			var composefilePaths []string

			composefileImagesWithTempDir := map[string][]*parse.ComposefileImage{} // nolint: lll

			for composefilePath, images := range lockfile.ComposefileImages {
				for _, image := range images {
					if image.DockerfilePath != "" {
						uniqueDockerfilePaths[image.DockerfilePath] = struct{}{}
						image.DockerfilePath = filepath.Join(
							tempDir, image.DockerfilePath,
						)
					}
				}
				composefilePaths = append(composefilePaths, composefilePath)

				composefilePath = filepath.Join(tempDir, composefilePath)
				composefileImagesWithTempDir[composefilePath] = images
			}

			dockerfileImagesWithTempDir := map[string][]*parse.DockerfileImage{}
			for dockerfilePath, images := range lockfile.DockerfileImages {
				uniqueDockerfilePaths[dockerfilePath] = struct{}{}

				dockerfilePath = filepath.Join(tempDir, dockerfilePath)
				dockerfileImagesWithTempDir[dockerfilePath] = images
			}

			var dockerfilePaths []string
			for dockerfilePath := range uniqueDockerfilePaths {
				dockerfilePaths = append(dockerfilePaths, dockerfilePath)
			}

			sort.Strings(dockerfilePaths)
			sort.Strings(composefilePaths)

			tempDockerfilePaths := writeFilesToTempDir(
				t, tempDir, dockerfilePaths, test.DockerfileContents,
			)
			tempComposefilePaths := writeFilesToTempDir(
				t, tempDir, composefilePaths, test.ComposefileContents,
			)

			flags := &cmd_rewrite.Flags{TempDir: tempDir}

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

			assertRewrittenFiles(
				t, test.ExpectedDockerfileContents, tempDockerfilePaths,
			)
			assertRewrittenFiles(
				t, test.ExpectedComposefileContents, tempComposefilePaths,
			)
		})
	}
}
