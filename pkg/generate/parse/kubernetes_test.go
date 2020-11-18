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
		{
			Name:                "Doc Position",
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
---
apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
    app: test
spec:
  containers:
  - name: redis
    image: redis:1.0@sha256:123
    ports:
    - containerPort: 80
  - name: bash
    image: bash:v1
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
				{
					Image: &parse.Image{
						Name:   "redis",
						Tag:    "1.0",
						Digest: "123",
					},
					ImagePosition: 0,
					DocPosition:   1,
					ContainerName: "redis",
					Path:          "pod.yaml",
				},
				{
					Image:         &parse.Image{Name: "bash", Tag: "v1"},
					ImagePosition: 1,
					DocPosition:   1,
					ContainerName: "bash",
					Path:          "pod.yaml",
				},
			},
		},
		{
			Name:                "Multiple Files",
			KubernetesfilePaths: []string{"deployment.yaml", "pod.yaml"},
			KubernetesfileContents: [][]byte{
				[]byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: nginx
  name: nginx-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - image: nginx
        name: nginx
        ports:
        - containerPort: 80
`),
				[]byte(`---
apiVersion: v1
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
`),
			},
			Expected: []*parse.KubernetesfileImage{
				{
					Image:         &parse.Image{Name: "nginx", Tag: "latest"},
					ContainerName: "nginx",
					Path:          "deployment.yaml",
				},
				{
					Image:         &parse.Image{Name: "busybox", Tag: "latest"},
					ContainerName: "busybox",
					Path:          "pod.yaml",
				},
			},
		},
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
