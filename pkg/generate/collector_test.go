package generate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/collect"
)

func TestPathCollector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name          string
		PathCollector *generate.PathCollector
		Expected      []*generate.AnyPath
		PathsToCreate []string
	}{
		{
			Name: "Dockerfiles And Composefiles",
			PathCollector: makePathCollector(
				t, "", []string{"Dockerfile"}, nil, nil, false,
				[]string{"docker-compose.yml"}, nil, nil, false, false,
			),
			PathsToCreate: []string{"Dockerfile", "docker-compose.yml"},
			Expected: []*generate.AnyPath{
				{
					DockerfilePath: "Dockerfile",
				},
				{
					ComposefilePath: "docker-compose.yml",
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

			dockerfileCollector := test.PathCollector.DockerfileCollector.(*collect.PathCollector)   // nolint: lll
			composefileCollector := test.PathCollector.ComposefileCollector.(*collect.PathCollector) // nolint: lll

			addTempDirToStringSlices(
				t, dockerfileCollector, tempDir,
			)
			addTempDirToStringSlices(
				t, composefileCollector, tempDir,
			)

			pathsToCreateContents := make([][]byte, len(test.PathsToCreate))
			writeFilesToTempDir(
				t, tempDir, test.PathsToCreate, pathsToCreateContents,
			)

			var got []*generate.AnyPath

			done := make(chan struct{})
			for anyPath := range test.PathCollector.CollectPaths(done) {
				if anyPath.Err != nil {
					close(done)
					t.Fatal(anyPath.Err)
				}
				got = append(got, anyPath)
			}

			for _, anyPath := range test.Expected {
				switch {
				case anyPath.DockerfilePath != "":
					anyPath.DockerfilePath = filepath.Join(
						tempDir, anyPath.DockerfilePath,
					)
				case anyPath.ComposefilePath != "":
					anyPath.ComposefilePath = filepath.Join(
						tempDir, anyPath.ComposefilePath,
					)
				}
			}

			sortAnyPaths(t, test.Expected)
			sortAnyPaths(t, got)

			assertAnyPathsEqual(t, test.Expected, got)
		})
	}
}
