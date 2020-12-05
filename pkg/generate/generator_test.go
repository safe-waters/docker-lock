package generate_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/internal/testutils"
	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/generate/format"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/generate/update"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

func TestGenerator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name          string
		PathsToCreate []string
		Contents      [][]byte
		Expected      map[kind.Kind]map[string][]interface{}
	}{
		{
			Name:          "One Kind",
			PathsToCreate: []string{"Dockerfile"},
			Contents: [][]byte{
				[]byte(`
FROM golang
FROM busybox
`,
				),
			},
			Expected: map[kind.Kind]map[string][]interface{}{
				kind.Dockerfile: {
					"Dockerfile": {
						map[string]string{
							"name":   "golang",
							"tag":    "latest",
							"digest": testutils.GolangLatestSHA,
						},
						map[string]string{
							"name":   "busybox",
							"tag":    "latest",
							"digest": testutils.BusyboxLatestSHA,
						},
					},
				},
			},
		},
		{
			Name: "All Kinds",
			PathsToCreate: []string{
				"Dockerfile", "docker-compose.yml", "pod.yml",
			},
			Contents: [][]byte{
				[]byte(`
FROM golang
FROM busybox
`,
				),
				[]byte(`
version: '3'
services:
  web:
    image: golang
  database:
    image: unused
    build: .
`),
				[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
    app: test
spec:
  containers:
  - name: busybox
    image: busybox:v1@sha256:busybox
    ports:
    - containerPort: 80
  - name: golang
    image: golang@sha256:golang
    ports:
    - containerPort: 88
`),
			},
			Expected: map[kind.Kind]map[string][]interface{}{
				kind.Dockerfile: {
					"Dockerfile": {
						map[string]string{
							"name":   "golang",
							"tag":    "latest",
							"digest": testutils.GolangLatestSHA,
						},
						map[string]string{
							"name":   "busybox",
							"tag":    "latest",
							"digest": testutils.BusyboxLatestSHA,
						},
					},
				},
				kind.Composefile: {
					"docker-compose.yml": {
						map[string]string{
							"name":       "golang",
							"tag":        "latest",
							"digest":     testutils.GolangLatestSHA,
							"service":    "database",
							"dockerfile": "Dockerfile",
						},
						map[string]string{
							"name":       "busybox",
							"tag":        "latest",
							"digest":     testutils.BusyboxLatestSHA,
							"service":    "database",
							"dockerfile": "Dockerfile",
						},
						map[string]string{
							"name":    "golang",
							"tag":     "latest",
							"digest":  testutils.GolangLatestSHA,
							"service": "web",
						},
					},
				},
				kind.Kubernetesfile: {
					"pod.yml": {
						map[string]string{
							"name":      "busybox",
							"tag":       "v1",
							"digest":    "busybox",
							"container": "busybox",
						},
						map[string]string{
							"name":      "golang",
							"tag":       "",
							"digest":    "golang",
							"container": "golang",
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := testutils.MakeTempDirInCurrentDir(t)
			defer os.RemoveAll(tempDir)

			testutils.WriteFilesToTempDir(
				t, tempDir, test.PathsToCreate, test.Contents,
			)

			dockerfileCollector, err := collect.NewPathCollector(
				kind.Dockerfile, tempDir, []string{"Dockerfile"},
				nil, nil, false,
			)
			if err != nil {
				t.Fatal(err)
			}

			composefileCollector, err := collect.NewPathCollector(
				kind.Composefile, tempDir, []string{"docker-compose.yml"},
				nil, nil, false,
			)
			if err != nil {
				t.Fatal(err)
			}

			kubernetesfileCollector, err := collect.NewPathCollector(
				kind.Kubernetesfile, tempDir, []string{"pod.yml"},
				nil, nil, false,
			)
			if err != nil {
				t.Fatal(err)
			}

			collector, err := generate.NewPathCollector(
				dockerfileCollector, composefileCollector,
				kubernetesfileCollector,
			)
			if err != nil {
				t.Fatal(err)
			}

			dockerfileImageParser := parse.NewDockerfileImageParser()
			composefileImageParser, err := parse.NewComposefileImageParser(
				dockerfileImageParser,
			)
			if err != nil {
				t.Fatal(err)
			}

			kubernetesfileImageParser := parse.NewKubernetesfileImageParser()

			parser, err := generate.NewImageParser(
				dockerfileImageParser, composefileImageParser,
				kubernetesfileImageParser,
			)
			if err != nil {
				t.Fatal(err)
			}

			var gotNumNetworkCalls uint64

			digestRequester := testutils.NewMockDigestRequester(
				t, &gotNumNetworkCalls,
			)
			innerUpdater, err := update.NewImageDigestUpdater(
				digestRequester, false, false,
			)
			if err != nil {
				t.Fatal(err)
			}

			updater, err := generate.NewImageDigestUpdater(innerUpdater)
			if err != nil {
				t.Fatal(err)
			}

			dockerfileFormatter := format.NewDockerfileImageFormatter()
			composefileFormatter := format.NewComposefileImageFormatter()
			kubernetesfileFormatter := format.NewKubernetesfileImageFormatter()

			formatter, err := generate.NewImageFormatter(
				dockerfileFormatter, composefileFormatter,
				kubernetesfileFormatter,
			)
			if err != nil {
				t.Fatal(err)
			}

			generator, err := generate.NewGenerator(
				collector, parser, updater, formatter,
			)
			if err != nil {
				t.Fatal(err)
			}

			var gotByt bytes.Buffer

			err = generator.GenerateLockfile(&gotByt)
			if err != nil {
				t.Fatal(err)
			}

			sortedGot := map[kind.Kind]map[string][]interface{}{}

			if err = json.Unmarshal(gotByt.Bytes(), &sortedGot); err != nil {
				t.Fatal(err)
			}

			expectedWithTempDir := map[kind.Kind]map[string][]interface{}{}
			for k, pathImages := range test.Expected {
				expectedWithTempDir[k] = map[string][]interface{}{}
				for path, images := range pathImages {
					for _, image := range images {
						image := image.(map[string]string)
						if dockerfile, ok := image["dockerfile"]; ok {
							image["dockerfile"] = filepath.ToSlash(
								filepath.Join(tempDir, dockerfile),
							)
						}
					}

					tempPath := filepath.ToSlash(filepath.Join(tempDir, path))
					expectedWithTempDir[k][tempPath] = pathImages[path]
				}
			}

			expected, err := json.MarshalIndent(expectedWithTempDir, "", "\t")
			if err != nil {
				t.Fatal(err)
			}

			got, err := json.MarshalIndent(sortedGot, "", "\t")
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(expected, got) {
				t.Fatalf("expected:\n%s\ngot:\n%s",
					string(expected),
					string(got),
				)
			}
		})
	}
}
