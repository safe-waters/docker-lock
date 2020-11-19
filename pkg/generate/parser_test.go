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
		Name                   string
		DockerfilePaths        []string
		ComposefilePaths       []string
		KubernetesfilePaths    []string
		ComposefileContents    [][]byte
		DockerfileContents     [][]byte
		KubernetesfileContents [][]byte
		Expected               []*generate.AnyImage
	}{
		{
			Name:                "Dockerfiles, Composefiles, And Kubernetesfiles", // nolint: lll
			DockerfilePaths:     []string{"Dockerfile"},
			ComposefilePaths:    []string{"docker-compose.yml"},
			KubernetesfilePaths: []string{"pod.yml"},
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
			KubernetesfileContents: [][]byte{
				[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
    app: test
spec:
  containers:
  - name: redis
    image: redis
    ports:
    - containerPort: 80
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
				{
					KubernetesfileImage: &parse.KubernetesfileImage{
						Image: &parse.Image{
							Name: "redis",
							Tag:  "latest",
						},
						ContainerName: "redis",
						Path:          "pod.yml",
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
			kubernetesfilePaths := writeFilesToTempDir(
				t, tempDir, test.KubernetesfilePaths,
				test.KubernetesfileContents,
			)

			anyPaths := make(
				chan *generate.AnyPath,
				len(dockerfilePaths)+
					len(composefilePaths)+
					len(kubernetesfilePaths),
			)
			for _, path := range dockerfilePaths {
				anyPaths <- &generate.AnyPath{DockerfilePath: path}
			}
			for _, path := range composefilePaths {
				anyPaths <- &generate.AnyPath{ComposefilePath: path}
			}
			for _, path := range kubernetesfilePaths {
				anyPaths <- &generate.AnyPath{KubernetesfilePath: path}
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

			kubernetesfileImageParser := &parse.KubernetesfileImageParser{}

			imageParser := &generate.ImageParser{
				DockerfileImageParser:     dockerfileImageParser,
				ComposefileImageParser:    composefileImageParser,
				KubernetesfileImageParser: kubernetesfileImageParser,
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
				case anyImage.KubernetesfileImage != nil:
					anyImage.KubernetesfileImage.Path = filepath.Join(
						tempDir, anyImage.KubernetesfileImage.Path,
					)
				}
			}

			sortedExpected := sortAnyImages(t, test.Expected)
			sortedGot := sortAnyImages(t, got)

			assertAnyImagesEqual(t, sortedExpected, sortedGot)
		})
	}
}
