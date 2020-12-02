package parse_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/internal/testutils"
	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

const kubernetesfileImageParserTestDir = "kubernetesfileParser-tests"

func TestKubernetesfileImageParser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name                   string
		KubernetesfilePaths    []string
		KubernetesfileContents [][]byte
		Expected               []parse.IImage
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
			Expected: []parse.IImage{
				parse.NewImage(
					kind.Kubernetesfile, "busybox", "latest", "",
					map[string]interface{}{
						"path":          "pod.yaml",
						"docPosition":   0,
						"imagePosition": 0,
						"containerName": "busybox",
					}, nil,
				),
				parse.NewImage(
					kind.Kubernetesfile, "golang", "latest", "",
					map[string]interface{}{
						"path":          "pod.yaml",
						"docPosition":   0,
						"imagePosition": 1,
						"containerName": "golang",
					}, nil,
				),
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
			Expected: []parse.IImage{
				parse.NewImage(
					kind.Kubernetesfile, "busybox", "latest", "",
					map[string]interface{}{
						"path":          "pod.yaml",
						"docPosition":   0,
						"imagePosition": 0,
						"containerName": "busybox",
					}, nil,
				),
				parse.NewImage(
					kind.Kubernetesfile, "golang", "latest", "",
					map[string]interface{}{
						"path":          "pod.yaml",
						"docPosition":   0,
						"imagePosition": 1,
						"containerName": "golang",
					}, nil,
				),
				parse.NewImage(
					kind.Kubernetesfile, "redis", "1.0", "123",
					map[string]interface{}{
						"path":          "pod.yaml",
						"docPosition":   1,
						"imagePosition": 0,
						"containerName": "redis",
					}, nil,
				),
				parse.NewImage(
					kind.Kubernetesfile, "bash", "v1", "",
					map[string]interface{}{
						"path":          "pod.yaml",
						"docPosition":   1,
						"imagePosition": 1,
						"containerName": "bash",
					}, nil,
				),
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
			Expected: []parse.IImage{
				parse.NewImage(
					kind.Kubernetesfile, "nginx", "latest", "",
					map[string]interface{}{
						"path":          "deployment.yaml",
						"docPosition":   0,
						"imagePosition": 0,
						"containerName": "nginx",
					}, nil,
				),
				parse.NewImage(
					kind.Kubernetesfile, "busybox", "latest", "",
					map[string]interface{}{
						"path":          "pod.yaml",
						"docPosition":   0,
						"imagePosition": 0,
						"containerName": "busybox",
					}, nil,
				),
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := testutils.MakeTempDir(
				t, kubernetesfileImageParserTestDir,
			)
			defer os.RemoveAll(tempDir)

			testutils.MakeParentDirsInTempDirFromFilePaths(
				t, tempDir, test.KubernetesfilePaths,
			)
			pathsToParse := testutils.WriteFilesToTempDir(
				t, tempDir, test.KubernetesfilePaths,
				test.KubernetesfileContents,
			)

			pathsToParseCh := make(chan collect.IPath, len(pathsToParse))
			for _, path := range pathsToParse {
				pathsToParseCh <- collect.NewPath(
					kind.Kubernetesfile, path, nil,
				)
			}
			close(pathsToParseCh)

			done := make(chan struct{})
			defer close(done)

			parser := parse.NewKubernetesfileImageParser()
			images := parser.ParseFiles(
				pathsToParseCh, done,
			)

			var got []parse.IImage

			for image := range images {
				if test.ShouldFail {
					if image.Err() == nil {
						t.Fatal("expected error but did not get one")
					}

					return
				}

				if image.Err() != nil {
					t.Fatal(image.Err())
				}

				got = append(got, image)
			}

			for _, image := range test.Expected {
				metadata := image.Metadata()
				metadata["path"] = filepath.Join(
					tempDir, metadata["path"].(string),
				)
				image.SetMetadata(metadata)
			}

			testutils.SortKubernetesfileImages(t, got)

			testutils.AssertImagesEqual(t, test.Expected, got)
		})
	}
}
