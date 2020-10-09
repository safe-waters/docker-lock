package rewrite_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/rewrite"
	"github.com/safe-waters/docker-lock/pkg/rewrite/write"
)

func TestWriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name                string
		AnyPathImages       *rewrite.AnyPathImages
		ComposefileContents [][]byte
		DockerfileContents  [][]byte
		Expected            [][]byte
		ShouldFail          bool
	}{
		{
			Name: "Dockerfile And Composefile",
			AnyPathImages: &rewrite.AnyPathImages{
				DockerfilePathImages: map[string][]*parse.DockerfileImage{
					"Dockerfile": {
						{
							Image: &parse.Image{
								Name:   "golang",
								Tag:    "latest",
								Digest: "golang",
							},
							Path: "Dockerfile",
						},
					},
				},
				ComposefilePathImages: map[string][]*parse.ComposefileImage{
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
					},
				},
			},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox
`,
				),
			},
			DockerfileContents: [][]byte{
				[]byte(`
from golang
`,
				),
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

			var dockerfilePaths []string
			var composefilePaths []string

			// TODO: include dockerfilepaths in composefileimages
			for path := range test.AnyPathImages.DockerfilePathImages {
				dockerfilePaths = append(dockerfilePaths, path)
			}

			for path := range test.AnyPathImages.ComposefilePathImages {
				composefilePaths = append(composefilePaths, path)
			}

			tempDockerfilePaths := writeFilesToTempDir(
				t, tempDir, dockerfilePaths, test.DockerfileContents,
			)
			tempComposefilePaths := writeFilesToTempDir(
				t, tempDir, composefilePaths, test.ComposefileContents,
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

			tempAnyPaths := &rewrite.AnyPathImages{
				DockerfilePathImages:  map[string][]*parse.DockerfileImage{},
				ComposefilePathImages: map[string][]*parse.ComposefileImage{},
			}

			for path, images := range test.AnyPathImages.ComposefilePathImages {
				for _, image := range images {
					if image.DockerfilePath != "" {
						image.DockerfilePath = filepath.Join(
							tempDir, image.DockerfilePath,
						)
					}
					image.Path = filepath.Join(tempDir, image.Path)
				}
				tempPath := filepath.Join(tempDir, path)
				tempAnyPaths.ComposefilePathImages[tempPath] = images
			}

			for path, images := range test.AnyPathImages.DockerfilePathImages {
				for _, image := range images {
					image.Path = filepath.Join(tempDir, image.Path)
				}
				tempPath := filepath.Join(tempDir, path)
				tempAnyPaths.DockerfilePathImages[tempPath] = images
			}

			dockerfileWriter := &write.DockerfileWriter{
				Directory: tempDir,
			}
			composefileWriter := &write.ComposefileWriter{
				DockerfileWriter: dockerfileWriter,
				Directory:        tempDir,
			}

			writer, err := rewrite.NewWriter(
				dockerfileWriter, composefileWriter,
			)
			if err != nil {
				t.Fatal(err)
			}

			done := make(chan struct{})
			resultPaths := writer.WriteFiles(tempAnyPaths, done)

			var writtenPaths []*write.WrittenPath

			for rewrittenPath := range resultPaths {
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
