package generate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/internal/testutils"
	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
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
		Expected               []parse.IImage
	}{
		{
			Name:                "Dockerfiles, Composefiles, Kubernetesfiles",
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
			Expected: []parse.IImage{
				parse.NewImage(
					kind.Dockerfile, "ubuntu", "bionic", "",
					map[string]interface{}{
						"position": 0,
						"path":     "Dockerfile",
					}, nil,
				),
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest", "",
					map[string]interface{}{
						"position": 1,
						"path":     "Dockerfile",
					}, nil,
				),
				parse.NewImage(
					kind.Composefile, "busybox", "latest", "",
					map[string]interface{}{
						"servicePosition": 0,
						"path":            "docker-compose.yml",
						"serviceName":     "svc",
					}, nil,
				),
				parse.NewImage(
					kind.Composefile, "golang", "latest", "",
					map[string]interface{}{
						"servicePosition": 0,
						"path":            "docker-compose.yml",
						"serviceName":     "anothersvc",
					}, nil,
				),
				parse.NewImage(
					kind.Kubernetesfile, "redis", "latest", "",
					map[string]interface{}{
						"path":          "pod.yml",
						"imagePosition": 0,
						"docPosition":   0,
						"containerName": "redis",
					}, nil,
				),
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := testutils.MakeTempDir(t, "")
			defer os.RemoveAll(tempDir)

			dockerfilePaths := testutils.WriteFilesToTempDir(
				t, tempDir, test.DockerfilePaths, test.DockerfileContents,
			)
			composefilePaths := testutils.WriteFilesToTempDir(
				t, tempDir, test.ComposefilePaths, test.ComposefileContents,
			)
			kubernetesfilePaths := testutils.WriteFilesToTempDir(
				t, tempDir, test.KubernetesfilePaths,
				test.KubernetesfileContents,
			)

			paths := make(
				chan collect.IPath,
				len(dockerfilePaths)+
					len(composefilePaths)+
					len(kubernetesfilePaths),
			)
			for _, path := range dockerfilePaths {
				paths <- collect.NewPath(kind.Dockerfile, path, nil)
			}
			for _, path := range composefilePaths {
				paths <- collect.NewPath(kind.Composefile, path, nil)
			}
			for _, path := range kubernetesfilePaths {
				paths <- collect.NewPath(kind.Kubernetesfile, path, nil)
			}

			close(paths)

			dockerfileImageParser := parse.NewDockerfileImageParser()
			composefileImageParser, err := parse.NewComposefileImageParser(
				dockerfileImageParser,
			)
			if err != nil {
				t.Fatal(err)
			}
			kubernetesfileImageParser := parse.NewKubernetesfileImageParser()

			imageParser, err := generate.NewImageParser(
				dockerfileImageParser, composefileImageParser,
				kubernetesfileImageParser,
			)
			if err != nil {
				t.Fatal(err)
			}

			done := make(chan struct{})
			defer close(done)

			gotImages := imageParser.ParseFiles(paths, done)

			var got []parse.IImage

			for anyImage := range gotImages {
				if anyImage.Err() != nil {
					t.Fatal(anyImage.Err())
				}
				got = append(got, anyImage)
			}

			for _, image := range test.Expected {
				metadata := image.Metadata()
				metadata["path"] = filepath.Join(
					tempDir, image.Metadata()["path"].(string),
				)

				image.SetMetadata(metadata)
			}

			testutils.SortImages(t, test.Expected)
			testutils.SortImages(t, got)

			testutils.AssertImagesEqual(t, test.Expected, got)
		})
	}
}
