package generate_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/safe-waters/docker-lock/internal/testutils"
	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

func TestPathCollector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name          string
		Expected      []collect.IPath
		PathsToCreate []string
	}{
		{
			Name: "Dockerfiles, Composefiles And Kubernetesfiles",
			PathsToCreate: []string{
				"Dockerfile", "docker-compose.yml", "pod.yml",
			},
			Expected: []collect.IPath{
				collect.NewPath(kind.Dockerfile, "Dockerfile", nil),
				collect.NewPath(kind.Composefile, "docker-compose.yml", nil),
				collect.NewPath(kind.Kubernetesfile, "pod.yml", nil),
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := testutils.MakeTempDirInCurrentDir(t)
			defer os.RemoveAll(tempDir)

			var expected []collect.IPath

			pathsToCreateContents := make([][]byte, len(test.PathsToCreate))
			testutils.WriteFilesToTempDir(
				t, tempDir, test.PathsToCreate, pathsToCreateContents,
			)

			for _, path := range test.Expected {
				expected = append(
					expected, collect.NewPath(
						path.Kind(), filepath.Join(tempDir, path.Val()), nil,
					),
				)
			}

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

			var got []collect.IPath

			done := make(chan struct{})
			defer close(done)

			for path := range collector.CollectPaths(done) {
				if path.Err() != nil {
					t.Fatal(path.Err())
				}
				got = append(got, path)
			}

			testutils.SortPaths(t, expected)
			testutils.SortPaths(t, got)

			if !reflect.DeepEqual(expected, got) {
				t.Fatalf("expected %v, got %v", test.Expected, got)
			}
		})
	}
}
