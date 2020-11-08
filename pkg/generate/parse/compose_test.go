package parse_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
)

const composefileImageParserTestDir = "composefileParser-tests"

func TestComposefileImageParser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name                 string
		EnvironmentVariables map[string]string
		DotEnvContents       [][]byte
		ComposefilePaths     []string
		ComposefileContents  [][]byte
		DockerfilePaths      []string
		DockerfileContents   [][]byte
		Expected             []*parse.ComposefileImage
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
			Expected: []*parse.ComposefileImage{
				{
					Image: &parse.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					Path:        "docker-compose.yml",
					ServiceName: "svc",
				},
			},
		},
		{
			Name:             "Scratch",
			ComposefilePaths: []string{"docker-compose.yml"},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: scratch
`),
			},
			Expected: []*parse.ComposefileImage{
				{
					Image: &parse.Image{
						Name: "scratch",
						Tag:  "",
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
			Expected: []*parse.ComposefileImage{
				{
					Image: &parse.Image{
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
			Expected: []*parse.ComposefileImage{
				{
					Image: &parse.Image{
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
			Expected: []*parse.ComposefileImage{
				{
					Image: &parse.Image{
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
			Expected: []*parse.ComposefileImage{
				{
					Image: &parse.Image{
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
			Name:             "Dot Env",
			ComposefilePaths: []string{"docker-compose.yml"},
			DotEnvContents: [][]byte{
				[]byte(`
DOT_ENV_IMAGE=busybox
`),
			},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: ${DOT_ENV_IMAGE}
`),
			},
			Expected: []*parse.ComposefileImage{
				{
					Image: &parse.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					Path:        "docker-compose.yml",
					ServiceName: "svc",
				},
			},
		},
		{
			Name:             "Os Env Overrides Dot Env",
			ComposefilePaths: []string{"docker-compose.yml"},
			EnvironmentVariables: map[string]string{
				"OS_ENV_OVERRIDES_DOT_ENV_IMAGE": "busybox",
			},
			DotEnvContents: [][]byte{
				[]byte(`
OS_ENV_OVERRIDES_DOT_ENV_IMAGE=ubuntu
`),
			},
			ComposefileContents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: ${OS_ENV_OVERRIDES_DOT_ENV_IMAGE}
`),
			},
			Expected: []*parse.ComposefileImage{
				{
					Image: &parse.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					Path:        "docker-compose.yml",
					ServiceName: "svc",
				},
			},
		},
		{
			Name: "Dot Env Args Env List",
			DotEnvContents: [][]byte{
				[]byte(`
DOT_ENV_ARGS_ENV_LIST_IMAGE=busybox
`),
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
      - DOT_ENV_ARGS_ENV_LIST_IMAGE
`),
			},
			DockerfilePaths: []string{"Dockerfile"},
			DockerfileContents: [][]byte{[]byte(`
ARG DOT_ENV_ARGS_ENV_LIST_IMAGE
FROM ${DOT_ENV_ARGS_ENV_LIST_IMAGE}
`)},
			Expected: []*parse.ComposefileImage{
				{
					Image: &parse.Image{
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
			Expected: []*parse.ComposefileImage{
				{
					Image: &parse.Image{
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
			Expected: []*parse.ComposefileImage{
				{
					Image: &parse.Image{
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
			Expected: []*parse.ComposefileImage{
				{
					Image: &parse.Image{
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
			Expected: []*parse.ComposefileImage{
				{
					Image: &parse.Image{
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
			Expected: []*parse.ComposefileImage{
				{
					Image: &parse.Image{
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
			Expected: []*parse.ComposefileImage{
				{
					Image: &parse.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					DockerfilePath: filepath.Join("one", "Dockerfile"),
					Path:           "docker-compose-one.yml",
					ServiceName:    "svc-one",
				},
				{
					Image: &parse.Image{
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
			Expected: []*parse.ComposefileImage{
				{
					Image: &parse.Image{
						Name: "busybox",
						Tag:  "latest",
					},
					DockerfilePath: filepath.Join("one", "Dockerfile"),
					Path:           "docker-compose.yml",
					ServiceName:    "svc-one",
				},
				{
					Image: &parse.Image{
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

			tempDir := makeTempDir(t, composefileImageParserTestDir)
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
			if len(test.DotEnvContents) != 0 {
				dotEnvFiles := make([]string, len(test.DotEnvContents))

				for i := range test.DotEnvContents {
					dotEnvFiles[i] = ".env"
				}

				_ = writeFilesToTempDir(
					t, tempDir, dotEnvFiles, test.DotEnvContents,
				)
			}

			_ = writeFilesToTempDir(
				t, tempDir, test.DockerfilePaths, test.DockerfileContents,
			)
			pathsToParse := writeFilesToTempDir(
				t, tempDir, test.ComposefilePaths, test.ComposefileContents,
			)

			pathsToParseCh := make(chan string, len(pathsToParse))
			for _, path := range pathsToParse {
				pathsToParseCh <- path
			}
			close(pathsToParseCh)

			done := make(chan struct{})

			composefileParser, err := parse.NewComposefileImageParser(
				&parse.DockerfileImageParser{},
			)
			if err != nil {
				t.Fatal(err)
			}

			composefileImages := composefileParser.ParseFiles(
				pathsToParseCh, done,
			)

			var got []*parse.ComposefileImage

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
			sortComposefileImageParserResults(t, got)

			assertComposefileImagesEqual(t, test.Expected, got)
		})
	}
}
