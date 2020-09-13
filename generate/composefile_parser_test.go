package generate_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/safe-waters/docker-lock/generate"
)

const composefileParserTestDir = "composefileParser-tests"

func TestComposefileParser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name                 string
		EnvironmentVariables map[string]string
		ComposefilePaths     []string
		ComposefileContents  [][]byte
		DockerfilePaths      []string
		DockerfileContents   [][]byte
		Expected             []*generate.ComposefileImage
	}{
		{
			Name:             "Image",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: busybox
`),
			},
			Expected: []*generate.ComposefileImage{
				{
					Image: &generate.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					Path:        "docker-compose.yml",
					ServiceName: "svc",
				},
			},
		},
		{
			Name:             "Build",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build: ./build
`),
			},
			DockerfilePaths:    []string{filepath.Join("build", "Dockerfile")},
			DockerfileContents: [][]byte{[]byte(`FROM busybox`)},
			Expected: []*generate.ComposefileImage{
				{
					Image: &generate.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					DockerfilePath: filepath.Join("build", "Dockerfile"),
					Path:           "docker-compose.yml",
					ServiceName:    "svc",
				},
			},
		},
		{
			Name:             "Context",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build:
      context: ./dockerfile
`),
			},
			DockerfilePaths: []string{
				filepath.Join("dockerfile", "Dockerfile"),
			},
			DockerfileContents: [][]byte{[]byte(`FROM busybox`)},
			Expected: []*generate.ComposefileImage{
				{
					Image: &generate.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					DockerfilePath: filepath.Join("dockerfile", "Dockerfile"),
					Path:           "docker-compose.yml",
					ServiceName:    "svc",
				},
			},
		},
		{
			Name:             "Context Dockerfile",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build:
      context: ./dockerfile
      dockerfile: Dockerfile
`),
			},
			DockerfilePaths: []string{
				filepath.Join("dockerfile", "Dockerfile"),
			},
			DockerfileContents: [][]byte{[]byte(`FROM busybox`)},
			Expected: []*generate.ComposefileImage{
				{
					Image: &generate.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					DockerfilePath: filepath.Join("dockerfile", "Dockerfile"),
					Path:           "docker-compose.yml",
					ServiceName:    "svc",
				},
			},
		},
		{
			Name: "Env",
			EnvironmentVariables: map[string]string{
				"ENV_CONTEXT": "dockerfile",
			},
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build:
      context: ./${ENV_CONTEXT}
`),
			},
			DockerfilePaths: []string{
				filepath.Join("dockerfile", "Dockerfile"),
			},
			DockerfileContents: [][]byte{[]byte(`FROM busybox`)},
			Expected: []*generate.ComposefileImage{
				{
					Image: &generate.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					DockerfilePath: filepath.Join("dockerfile", "Dockerfile"),
					Path:           "docker-compose.yml",
					ServiceName:    "svc",
				},
			},
		},
		{
			Name: "Args Env List",
			EnvironmentVariables: map[string]string{
				"ARGS_ENV_LIST_IMAGE": "busybox",
			},
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build:
      context: .
      args:
      - ARGS_ENV_LIST_IMAGE
`),
			},
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{[]byte(`
ARG ARGS_ENV_LIST_IMAGE
FROM ${ARGS_ENV_LIST_IMAGE}
`)},
			Expected: []*generate.ComposefileImage{
				{
					Image: &generate.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					DockerfilePath: "Dockerfile",
					Path:           "docker-compose.yml",
					ServiceName:    "svc",
				},
			},
		},
		{
			Name:             "Args Key Val List",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build:
      context: .
      args:
      - IMAGE=busybox
`),
			},
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{[]byte(`
ARG IMAGE
FROM ${IMAGE}
`)},
			Expected: []*generate.ComposefileImage{
				{
					Image: &generate.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					DockerfilePath: "Dockerfile",
					Path:           "docker-compose.yml",
					ServiceName:    "svc",
				},
			},
		},
		{
			Name:             "Args Key Val Map",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build:
      context: .
      args:
        IMAGE: busybox
`),
			},
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{[]byte(`
ARG IMAGE
FROM ${IMAGE}
`)},
			Expected: []*generate.ComposefileImage{
				{
					Image: &generate.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					DockerfilePath: "Dockerfile",
					Path:           "docker-compose.yml",
					ServiceName:    "svc",
				},
			},
		},
		{
			Name:             "Args Override",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build:
      context: .
      args:
        IMAGE: busybox
`),
			},
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{[]byte(`
ARG IMAGE=ubuntu
FROM ${IMAGE}
`)},
			Expected: []*generate.ComposefileImage{
				{
					Image: &generate.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					DockerfilePath: "Dockerfile",
					Path:           "docker-compose.yml",
					ServiceName:    "svc",
				},
			},
		},
		{
			Name:             "Args No Arg",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: unused
    build:
      context: .
`),
			},
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{[]byte(`
ARG IMAGE=busybox
FROM ${IMAGE}
`)},
			Expected: []*generate.ComposefileImage{
				{
					Image: &generate.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					DockerfilePath: "Dockerfile",
					Path:           "docker-compose.yml",
					ServiceName:    "svc",
				},
			},
		},
		{
			Name: "Multiple Files",
			ComposefilePaths: []string{
				"docker-compose-one.yml", "docker-compose-two.yml",
			},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc-one:
    image: unused
    build: ./one
`),
				[]byte(`
version: '3'
services:
  svc-two:
    image: unused
    build: ./two
`),
			},
			DockerfilePaths: []string{
				filepath.Join("one", "Dockerfile"),
				filepath.Join("two", "Dockerfile"),
			},
			DockerfileContents: [][]byte{
				[]byte(`FROM busybox`), []byte(`FROM busybox`),
			},
			Expected: []*generate.ComposefileImage{
				{
					Image: &generate.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					DockerfilePath: filepath.Join("one", "Dockerfile"),
					Path:           "docker-compose-one.yml",
					ServiceName:    "svc-one",
				},
				{
					Image: &generate.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					DockerfilePath: filepath.Join("two", "Dockerfile"),
					Path:           "docker-compose-two.yml",
					ServiceName:    "svc-two",
				},
			},
		},
		{
			Name:             "Multiple Services",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc-one:
    image: unused
    build: ./one
  svc-two:
    image: unused
    build: ./two
`),
			},
			DockerfilePaths: []string{
				filepath.Join("one", "Dockerfile"),
				filepath.Join("two", "Dockerfile"),
			},
			DockerfileContents: [][]byte{
				[]byte(`FROM busybox`), []byte(`FROM busybox`),
			},
			Expected: []*generate.ComposefileImage{
				{
					Image: &generate.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					DockerfilePath: filepath.Join("one", "Dockerfile"),
					Path:           "docker-compose.yml",
					ServiceName:    "svc-one",
				},
				{
					Image: &generate.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					DockerfilePath: filepath.Join("two", "Dockerfile"),
					Path:           "docker-compose.yml",
					ServiceName:    "svc-two",
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := makeTempDir(t, composefileParserTestDir)
			defer os.RemoveAll(tempDir)

			for k, v := range test.EnvironmentVariables {
				os.Setenv(k, v)
			}

			makeParentDirsInTempDirFromFilePaths(
				t, tempDir, test.DockerfilePaths,
			)
			makeParentDirsInTempDirFromFilePaths(
				t, tempDir, test.ComposefilePaths,
			)

			_ = writeFilesToTempDir(
				t, tempDir, test.DockerfilePaths, test.DockerfileContents,
			)
			pathsToParse := writeFilesToTempDir(
				t, tempDir, test.ComposefilePaths, test.ComposefileContents,
			)

			pathsToParseCh := make(chan *generate.PathResult, len(pathsToParse))
			for _, path := range pathsToParse {
				pathsToParseCh <- &generate.PathResult{Path: path}
			}
			close(pathsToParseCh)

			done := make(chan struct{})
			composefileParser := &generate.ComposefileParser{}
			composefileImages := composefileParser.ParseFiles(
				pathsToParseCh, done,
			)

			var got []*generate.ComposefileImage

			for composefileImage := range composefileImages {
				if composefileImage.Err != nil {
					close(done)
					t.Fatal(composefileImage.Err)
				}
				got = append(got, composefileImage)
			}

			for _, composefileImage := range test.Expected {
				composefileImage.Path = filepath.Join(
					tempDir, composefileImage.Path,
				)

				if composefileImage.DockerfilePath != "" {
					composefileImage.DockerfilePath = filepath.Join(
						tempDir, composefileImage.DockerfilePath,
					)
				}
			}
			sortComposefileParserResults(t, got)

			assertComposefileImagesEqual(t, test.Expected, got)
		})
	}
}

func sortComposefileParserResults(
	t *testing.T,
	results []*generate.ComposefileImage,
) {
	t.Helper()

	sort.Slice(results, func(i, j int) bool {
		switch {
		case results[i].Path != results[j].Path:
			return results[i].Path < results[j].Path
		case results[i].ServiceName != results[j].ServiceName:
			return results[i].ServiceName < results[j].ServiceName
		case results[i].DockerfilePath != results[j].DockerfilePath:
			return results[i].DockerfilePath < results[j].DockerfilePath
		default:
			return results[i].Position < results[j].Position
		}
	})
}
