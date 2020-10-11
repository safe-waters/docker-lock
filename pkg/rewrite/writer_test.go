package rewrite_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/rewrite"
	"github.com/safe-waters/docker-lock/pkg/rewrite/write"
)

func TestWriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name          string
		AnyPathImages *rewrite.AnyPathImages
		Contents      [][]byte
		Expected      [][]byte
		ShouldFail    bool
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
							ServiceName: "svc-compose",
						},
					},
				},
			},
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

			tempDir := makeTempDirInCurrentDir(t)
			defer os.RemoveAll(tempDir)

			uniquePathsToWrite := map[string]struct{}{}

			tempAnyPaths := &rewrite.AnyPathImages{
				DockerfilePathImages:  map[string][]*parse.DockerfileImage{},
				ComposefilePathImages: map[string][]*parse.ComposefileImage{},
			}

			for composefilePath, images := range test.AnyPathImages.ComposefilePathImages { // nolint: lll
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
				tempAnyPaths.ComposefilePathImages[composefilePath] = images
			}

			for dockerfilePath, images := range test.AnyPathImages.DockerfilePathImages { // nolint: lll
				uniquePathsToWrite[dockerfilePath] = struct{}{}

				dockerfilePath = filepath.Join(tempDir, dockerfilePath)
				tempAnyPaths.DockerfilePathImages[dockerfilePath] = images
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
			writtenPathResults := writer.WriteFiles(tempAnyPaths, done)

			var got []string

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
