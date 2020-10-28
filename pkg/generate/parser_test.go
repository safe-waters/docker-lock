package generate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
)

func TestImageParser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name                string
		DockerfilePaths     []string
		ComposefilePaths    []string
		ComposefileContents [][]byte
		DockerfileContents  [][]byte
		Expected            []*generate.AnyImage
	}{
		{
			Name:             "Dockerfiles And Composefiles",
			DockerfilePaths:  []string{"Dockerfile"},
			ComposefilePaths: []string{"docker-compose.yml"},
			DockerfileContents: [][]byte{
				[]byte(`
FROM ubuntu:bionic
FROM busybox
`),
			},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: busybox
  anothersvc:
    image: golang
`),
			},
			Expected: []*generate.AnyImage{
				{
					DockerfileImage: &parse.DockerfileImage{
						Image:    &parse.Image{Name: "ubuntu", Tag: "bionic"},
						Position: 0,
						Path:     "Dockerfile",
					},
				},
				{
					DockerfileImage: &parse.DockerfileImage{
						Image:    &parse.Image{Name: "busybox", Tag: "latest"},
						Position: 1,
						Path:     "Dockerfile",
					},
				},
				{
					ComposefileImage: &parse.ComposefileImage{
						Image: &parse.Image{
							Name: "busybox",
							Tag:  "latest",
						},
						Position:    0,
						Path:        "docker-compose.yml",
						ServiceName: "svc",
					},
				},
				{
					ComposefileImage: &parse.ComposefileImage{
						Image: &parse.Image{
							Name: "golang",
							Tag:  "latest",
						},
						Position:    0,
						Path:        "docker-compose.yml",
						ServiceName: "anothersvc",
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := makeTempDir(t, "")
			defer os.RemoveAll(tempDir)

			dockerfilePaths := writeFilesToTempDir(
				t, tempDir, test.DockerfilePaths, test.DockerfileContents,
			)
			composefilePaths := writeFilesToTempDir(
				t, tempDir, test.ComposefilePaths, test.ComposefileContents,
			)

			anyPaths := make(
				chan *generate.AnyPath,
				len(dockerfilePaths)+len(composefilePaths),
			)
			for _, path := range dockerfilePaths {
				anyPaths <- &generate.AnyPath{DockerfilePath: path}
			}
			for _, path := range composefilePaths {
				anyPaths <- &generate.AnyPath{ComposefilePath: path}
			}
			close(anyPaths)

			done := make(chan struct{})

			dockerfileImageParser := &parse.DockerfileImageParser{}

			composefileImageParser, err := parse.NewComposefileImageParser(
				dockerfileImageParser,
			)
			if err != nil {
				t.Fatal(err)
			}

			imageParser := &generate.ImageParser{
				DockerfileImageParser:  dockerfileImageParser,
				ComposefileImageParser: composefileImageParser,
			}
			anyImages := imageParser.ParseFiles(anyPaths, done)

			var got []*generate.AnyImage

			for anyImage := range anyImages {
				if anyImage.Err != nil {
					close(done)
					t.Fatal(anyImage.Err)
				}
				got = append(got, anyImage)
			}

			for _, anyImage := range test.Expected {
				switch {
				case anyImage.DockerfileImage != nil:
					anyImage.DockerfileImage.Path = filepath.Join(
						tempDir, anyImage.DockerfileImage.Path,
					)

				case anyImage.ComposefileImage != nil:
					anyImage.ComposefileImage.Path = filepath.Join(
						tempDir, anyImage.ComposefileImage.Path,
					)
				}
			}

			sortedExpected := sortAnyImages(t, test.Expected)
			sortedGot := sortAnyImages(t, got)

			assertAnyImagesEqual(t, sortedExpected, sortedGot)
		})
	}
}
