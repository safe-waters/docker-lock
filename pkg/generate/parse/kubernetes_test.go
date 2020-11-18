package parse_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
)

const kubernetesfileImageParserTestDir = "kubernetesfileParser-tests"

func TestKubernetesfileImageParser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name                   string
		KubernetesfilePaths    []string
		KubernetesfileContents [][]byte
		Expected               []*parse.KubernetesfileImage
		ShouldFail             bool
	}{
		{
			Name:                "Image Position",
			KubernetesfilePaths: []string{"pod.yaml"},
			KubernetesfileContents: [][]byte{
				[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
    app: test
spec:
  containers:
  - name: busybox
    image: busybox
    ports:
    - containerPort: 80
  - name: golang
    image: golang
    ports:
    - containerPort: 88
`),
			},
			Expected: []*parse.KubernetesfileImage{
				{
					Image:         &parse.Image{Name: "busybox", Tag: "latest"},
					ImagePosition: 0,
					ContainerName: "busybox",
					Path:          "pod.yaml",
				},
				{
					Image:         &parse.Image{Name: "golang", Tag: "latest"},
					ImagePosition: 1,
					ContainerName: "golang",
					Path:          "pod.yaml",
				},
			},
		},
		// 		{
		// 			Name:            "Digest",
		// 			DockerfilePaths: []string{"Dockerfile"},
		// 			DockerfileContents: [][]byte{
		// 				[]byte(`
		// FROM ubuntu@sha256:bae015c28bc7
		// `),
		// 			},
		// 			Expected: []*parse.DockerfileImage{
		// 				{
		// 					Image: &parse.Image{
		// 						Name:   "ubuntu",
		// 						Digest: "bae015c28bc7",
		// 					},
		// 					Position: 0,
		// 					Path:     "Dockerfile",
		// 				},
		// 			},
		// 		},
		// 		{
		// 			Name:            "Flag",
		// 			DockerfilePaths: []string{"Dockerfile"},
		// 			DockerfileContents: [][]byte{
		// 				[]byte(`
		// FROM --platform=$BUILDPLATFORM ubuntu@sha256:bae015c28bc7
		// `),
		// 			},
		// 			Expected: []*parse.DockerfileImage{
		// 				{
		// 					Image: &parse.Image{
		// 						Name:   "ubuntu",
		// 						Digest: "bae015c28bc7",
		// 					},
		// 					Position: 0,
		// 					Path:     "Dockerfile",
		// 				},
		// 			},
		// 		},
		// 		{
		// 			Name:            "Tag And Digest",
		// 			DockerfilePaths: []string{"Dockerfile"},
		// 			DockerfileContents: [][]byte{
		// 				[]byte(`
		// FROM ubuntu:bionic@sha256:bae015c28bc7
		// `),
		// 			},
		// 			Expected: []*parse.DockerfileImage{
		// 				{
		// 					Image: &parse.Image{
		// 						Name:   "ubuntu",
		// 						Tag:    "bionic",
		// 						Digest: "bae015c28bc7",
		// 					},
		// 					Position: 0,
		// 					Path:     "Dockerfile",
		// 				},
		// 			},
		// 		},
		// 		{
		// 			Name:            "Port, Tag, And Digest",
		// 			DockerfilePaths: []string{"Dockerfile"},
		// 			DockerfileContents: [][]byte{
		// 				[]byte(`
		// FROM localhost:5000/ubuntu:bionic@sha256:bae015c28bc7
		// `),
		// 			},
		// 			Expected: []*parse.DockerfileImage{
		// 				{
		// 					Image: &parse.Image{
		// 						Name:   "localhost:5000/ubuntu",
		// 						Tag:    "bionic",
		// 						Digest: "bae015c28bc7",
		// 					},
		// 					Position: 0,
		// 					Path:     "Dockerfile",
		// 				},
		// 			},
		// 		},
		// 		{
		// 			Name:            "Local Arg",
		// 			DockerfilePaths: []string{"Dockerfile"},
		// 			DockerfileContents: [][]byte{
		// 				[]byte(`
		// ARG IMAGE=busybox
		// FROM ${IMAGE}
		// ARG IMAGE=ubuntu
		// FROM ${IMAGE}
		// `),
		// 			},
		// 			Expected: []*parse.DockerfileImage{
		// 				{
		// 					Image:    &parse.Image{Name: "busybox", Tag: "latest"},
		// 					Position: 0,
		// 					Path:     "Dockerfile",
		// 				},
		// 				{
		// 					Image:    &parse.Image{Name: "busybox", Tag: "latest"},
		// 					Position: 1,
		// 					Path:     "Dockerfile",
		// 				},
		// 			},
		// 		},
		// 		{
		// 			Name:            "Build Stage",
		// 			DockerfilePaths: []string{"Dockerfile"},
		// 			DockerfileContents: [][]byte{
		// 				[]byte(`
		// FROM busybox AS busy
		// FROM busy as anotherbusy
		// FROM ubuntu as worker
		// `),
		// 			},
		// 			Expected: []*parse.DockerfileImage{
		// 				{
		// 					Image:    &parse.Image{Name: "busybox", Tag: "latest"},
		// 					Position: 0,
		// 					Path:     "Dockerfile",
		// 				},
		// 				{
		// 					Image:    &parse.Image{Name: "ubuntu", Tag: "latest"},
		// 					Position: 1,
		// 					Path:     "Dockerfile",
		// 				},
		// 			},
		// 		},
		// 		{
		// 			Name:            "Multiple Files",
		// 			DockerfilePaths: []string{"Dockerfile-one", "Dockerfile-two"},
		// 			DockerfileContents: [][]byte{
		// 				[]byte(`
		// FROM busybox
		// FROM ubuntu
		// `),
		// 				[]byte(`
		// FROM ubuntu
		// FROM busybox
		// `),
		// 			},
		// 			Expected: []*parse.DockerfileImage{
		// 				{
		// 					Image:    &parse.Image{Name: "busybox", Tag: "latest"},
		// 					Position: 0,
		// 					Path:     "Dockerfile-one",
		// 				},
		// 				{
		// 					Image:    &parse.Image{Name: "ubuntu", Tag: "latest"},
		// 					Position: 1,
		// 					Path:     "Dockerfile-one",
		// 				},

		// 				{
		// 					Image:    &parse.Image{Name: "ubuntu", Tag: "latest"},
		// 					Position: 0,
		// 					Path:     "Dockerfile-two",
		// 				},
		// 				{
		// 					Image:    &parse.Image{Name: "busybox", Tag: "latest"},
		// 					Position: 1,
		// 					Path:     "Dockerfile-two",
		// 				},
		// 			},
		// 		},
		// 		{
		// 			Name:            "Invalid Arg",
		// 			DockerfilePaths: []string{"Dockerfile"},
		// 			DockerfileContents: [][]byte{
		// 				[]byte(`
		// ARG
		// FROM busybox
		// `),
		// 			},
		// 			ShouldFail: true,
		// 		},
		// 		{
		// 			Name:            "Invalid From",
		// 			DockerfilePaths: []string{"Dockerfile"},
		// 			DockerfileContents: [][]byte{
		// 				[]byte(`
		// FROM
		// `),
		// 			},
		// 			ShouldFail: true,
		// 		},
	}

	for _, test := range tests { // nolint: dupl
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := makeTempDir(t, kubernetesfileImageParserTestDir)
			defer os.RemoveAll(tempDir)

			makeParentDirsInTempDirFromFilePaths(
				t, tempDir, test.KubernetesfilePaths,
			)
			pathsToParse := writeFilesToTempDir(
				t, tempDir, test.KubernetesfilePaths,
				test.KubernetesfileContents,
			)

			pathsToParseCh := make(chan string, len(pathsToParse))
			for _, path := range pathsToParse {
				pathsToParseCh <- path
			}
			close(pathsToParseCh)

			done := make(chan struct{})

			kubernetesfileParser := &parse.KubernetesfileImageParser{}
			kubernetesfileImages := kubernetesfileParser.ParseFiles(
				pathsToParseCh, done,
			)

			var got []*parse.KubernetesfileImage

			for kubernetesfileImage := range kubernetesfileImages {
				if test.ShouldFail {
					if kubernetesfileImage.Err == nil {
						t.Fatal("expected error but did not get one")
					}

					return
				}

				if kubernetesfileImage.Err != nil {
					close(done)
					t.Fatal(kubernetesfileImage.Err)
				}

				got = append(got, kubernetesfileImage)
			}

			sortKubernetesfileImageParserResults(t, got)

			for _, dockerfileImage := range test.Expected {
				dockerfileImage.Path = filepath.Join(
					tempDir, dockerfileImage.Path,
				)
			}

			assertKubernetesfileImagesEqual(t, test.Expected, got)
		})
	}
}
