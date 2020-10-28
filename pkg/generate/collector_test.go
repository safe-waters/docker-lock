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
		Expected      []*generate.CollectedPath
		PathsToCreate []string
	}{
		{
			Name: "Dockerfiles And Composefiles",
			PathCollector: makePathCollector(
				t, "", []string{"Dockerfile"}, nil, nil, false,
				[]string{"docker-compose.yml"}, nil, nil, false, false,
			),
			PathsToCreate: []string{"Dockerfile", "docker-compose.yml"},
			Expected: []*generate.CollectedPath{
				{
					Type: generate.Dockerfile,
					Path: "Dockerfile",
				},
				{
					Type: generate.Composefile,
					Path: "docker-compose.yml",
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

			var got []*generate.CollectedPath

			done := make(chan struct{})
			for collectedPath := range test.PathCollector.CollectPaths(done) {
				if collectedPath.Err != nil {
					close(done)
					t.Fatal(collectedPath.Err)
				}
				got = append(got, collectedPath)
			}

			for _, collectedPath := range test.Expected {
				switch collectedPath.Type {
				case generate.Dockerfile:
					collectedPath.Path = filepath.Join(
						tempDir, collectedPath.Path,
					)
				case generate.Composefile:
					collectedPath.Path = filepath.Join(
						tempDir, collectedPath.Path,
					)
				}
			}

			sortCollectedPaths(t, test.Expected)
			sortCollectedPaths(t, got)

			assertCollectedPathsEqual(t, test.Expected, got)
		})
	}
}
