package rewrite_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/safe-waters/docker-lock/internal/testutils"
	"github.com/safe-waters/docker-lock/pkg/kind"
	"github.com/safe-waters/docker-lock/pkg/rewrite"
	"github.com/safe-waters/docker-lock/pkg/rewrite/write"
)

func TestWriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name       string
		Lockfile   map[kind.Kind]map[string][]interface{}
		Contents   [][]byte
		Expected   [][]byte
		ShouldFail bool
	}{
		{
			Name: "Dockerfile, Composefile, And Kubernetesfile",
			Lockfile: map[kind.Kind]map[string][]interface{}{
				kind.Dockerfile: {
					"Dockerfile": {
						map[string]interface{}{
							"name":   "golang",
							"tag":    "latest",
							"digest": "golang",
						},
					},
				},
				kind.Composefile: {
					"docker-compose.yml": {
						map[string]interface{}{
							"name":    "busybox",
							"tag":     "latest",
							"digest":  "busybox",
							"service": "svc-compose",
						},
					},
				},
				kind.Kubernetesfile: {
					"pod.yml": {
						map[string]interface{}{
							"name":      "redis",
							"tag":       "latest",
							"digest":    "redis",
							"container": "redis",
						},
					},
				},
			},
			Contents: [][]byte{
				[]byte(`FROM golang
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox
`,
				),
				[]byte(`apiVersion: v1
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
			Expected: [][]byte{
				[]byte(`FROM golang:latest@sha256:golang
`,
				),
				[]byte(`
version: '3'

services:
  svc-compose:
    image: busybox:latest@sha256:busybox
`,
				),
				[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
    app: test
spec:
  containers:
  - name: redis
    image: redis:latest@sha256:redis
    ports:
    - containerPort: 80
`),
			},
		},
	}
	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := testutils.MakeTempDirInCurrentDir(t)
			defer os.RemoveAll(tempDir)

			uniquePathsToWrite := map[string]struct{}{}

			lockfileWithTempDir := map[kind.Kind]map[string][]interface{}{}

			lockfileWithTempDir[kind.Dockerfile] = map[string][]interface{}{}
			lockfileWithTempDir[kind.Kubernetesfile] = map[string][]interface{}{} // nolint: lll
			lockfileWithTempDir[kind.Composefile] = map[string][]interface{}{}

			for composefilePath, images := range test.Lockfile[kind.Composefile] { // nolint: lll
				for _, image := range images {
					image := image.(map[string]interface{})
					dockerfilePath := image["dockerfile"]
					if dockerfilePath != nil {
						dockerfilePath := dockerfilePath.(string)
						uniquePathsToWrite[dockerfilePath] = struct{}{}
						image["dockerfile"] = filepath.Join(
							tempDir, dockerfilePath,
						)
					}
				}

				uniquePathsToWrite[composefilePath] = struct{}{}

				composefilePath = filepath.Join(tempDir, composefilePath)
				lockfileWithTempDir[kind.Composefile][composefilePath] = images
			}

			for dockerfilePath, images := range test.Lockfile[kind.Dockerfile] {
				uniquePathsToWrite[dockerfilePath] = struct{}{}

				dockerfilePath = filepath.Join(tempDir, dockerfilePath)
				lockfileWithTempDir[kind.Dockerfile][dockerfilePath] = images
			}

			for kubernetesfilePath, images := range test.Lockfile[kind.Kubernetesfile] { // nolint: lll
				uniquePathsToWrite[kubernetesfilePath] = struct{}{}

				kubernetesfilePath = filepath.Join(tempDir, kubernetesfilePath)
				lockfileWithTempDir[kind.Kubernetesfile][kubernetesfilePath] = images // nolint: lll
			}

			var pathsToWrite []string
			for path := range uniquePathsToWrite {
				pathsToWrite = append(pathsToWrite, path)
			}

			sort.Strings(pathsToWrite)

			testutils.WriteFilesToTempDir(
				t, tempDir, pathsToWrite, test.Contents,
			)

			dockerfileWriter := write.NewDockerfileWriter(false)
			composefileWriter, err := write.NewComposefileWriter(
				dockerfileWriter, false,
			)
			if err != nil {
				t.Fatal(err)
			}
			kubernetesfileWriter := write.NewKubernetesfileWriter(false)

			writer, err := rewrite.NewWriter(
				dockerfileWriter, composefileWriter, kubernetesfileWriter,
			)
			if err != nil {
				t.Fatal(err)
			}

			done := make(chan struct{})
			defer close(done)

			writtenPathResults := writer.WriteFiles(
				lockfileWithTempDir, tempDir, done,
			)

			var got []string

			for writtenPath := range writtenPathResults {
				if writtenPath.Err() != nil {
					err = writtenPath.Err()
				}
				got = append(got, writtenPath.NewPath())
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

			testutils.AssertWrittenFilesEqual(t, test.Expected, got)
		})
	}
}
