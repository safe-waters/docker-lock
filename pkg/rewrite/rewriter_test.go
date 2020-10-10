package rewrite_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
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
			Name: "In Memory",
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
		// {
		// 	Name: "Composefile Overrides Dockerfile",
		// 	LockfilePath: filepath.Join(
		// 		"testdata", "override_dockerfile", "docker-lock.json",
		// 	),
		// },
		// {
		// 	Name: "Duplicate Services Same Dockerfile Images",
		// 	LockfilePath: filepath.Join(
		// 		"testdata", "duplicate_svc_same_images", "docker-lock.json",
		// 	),
		// },
		// {
		// 	Name: "Different Composefiles Same Dockerfile Images",
		// 	LockfilePath: filepath.Join(
		// 		"testdata", "duplicate_files_same_images", "docker-lock.json",
		// 	),
		// },
		// {
		// 	Name: "Duplicate Services Different Dockerfile Images",
		// 	LockfilePath: filepath.Join(
		// 		"testdata", "duplicate_svc_diff_images", "docker-lock.json",
		// 	),
		// 	ShouldFail: true,
		// },
		// {
		// 	Name: "Different Composefiles Different Dockerfile Images",
		// 	LockfilePath: filepath.Join(
		// 		"testdata", "duplicate_files_diff_images", "docker-lock.json",
		// 	),
		// 	ShouldFail: true,
		// },
	}

	for _, test := range tests {
		test := test
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := generateUUID(t)
			makeDir(t, tempDir)
			defer os.RemoveAll(tempDir)

			var lockfile generate.Lockfile
			if err := json.Unmarshal(test.LockfileContents, &lockfile); err != nil {
				t.Fatal(err)
			}

			uniqueDockerfilePaths := map[string]struct{}{}

			var composefilePaths []string

			composefileImagesWithTempDir := map[string][]*parse.ComposefileImage{}

			for composefilePath, images := range lockfile.ComposefileImages {
				for _, image := range images {
					if image.DockerfilePath != "" {
						uniqueDockerfilePaths[image.DockerfilePath] = struct{}{}
						image.DockerfilePath = filepath.Join(tempDir, image.DockerfilePath)
					}

					image.Path = filepath.Join(tempDir, image.Path)
				}
				composefilePaths = append(composefilePaths, composefilePath)

				composefilePath = filepath.Join(tempDir, composefilePath)
				composefileImagesWithTempDir[composefilePath] = images
			}

			dockerfileImagesWithTempDir := map[string][]*parse.DockerfileImage{}
			for dockerfilePath, images := range lockfile.DockerfileImages {
				for _, image := range images {
					image.Path = filepath.Join(tempDir, image.Path)
				}
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

			lockfileBytes, err := json.Marshal(lockfileWithTempDir)
			if err != nil {
				t.Fatal(err)
			}
			reader := bytes.NewReader(lockfileBytes)

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

			for i, rewrittenDockerfilePath := range tempDockerfilePaths {
				rewrittenBytes, err := ioutil.ReadFile(rewrittenDockerfilePath)
				if err != nil {
					t.Fatal(err)
				}

				if !bytes.Equal(test.ExpectedDockerfileContents[i], rewrittenBytes) {
					t.Fatalf(
						"expected:\n%s\ngot:\n%s",
						string(test.ExpectedDockerfileContents[i]),
						string(rewrittenBytes),
					)
				}
			}

			for i, rewrittenComposefilePath := range tempComposefilePaths {
				rewrittenBytes, err := ioutil.ReadFile(rewrittenComposefilePath)
				if err != nil {
					t.Fatal(err)
				}

				if !bytes.Equal(test.ExpectedComposefileContents[i], rewrittenBytes) {
					t.Fatalf(
						"expected:\n%s\ngot:\n%s",
						string(test.ExpectedComposefileContents[i]),
						string(rewrittenBytes),
					)
				}
			}
		})
	}
}
