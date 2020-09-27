package collect_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
)

const testDir = "collect"

func TestPathCollector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name                  string
		PathCollector         *collect.PathCollector
		AddTempDirToCollector bool
		BaseDirIsTempDir      bool
		ShouldFail            bool
		Expected              []string
		PathsToCreate         []string
	}{
		{
			Name: "Default Path Exists",
			PathCollector: makePathCollector(
				t, "", []string{"Dockerfile"}, nil, nil, false, false,
			),
			AddTempDirToCollector: true,
			PathsToCreate:         []string{"Dockerfile"},
			Expected:              []string{"Dockerfile"},
		},
		{
			Name: "Default Path Does Not Exist",
			PathCollector: makePathCollector(
				t, "", []string{"Dockerfile"}, nil, nil, false, false,
			),
		},
		{
			Name: "Do Not Use Default Paths If Other Methods Specified",
			PathCollector: makePathCollector(
				t, "", []string{"Dockerfile"}, []string{"Dockerfile-Manual"},
				nil, false, false,
			),
			AddTempDirToCollector: true,
			Expected:              []string{"Dockerfile-Manual"},
			PathsToCreate:         []string{"Dockerfile-Manual"},
		},
		{
			Name: "Manual Paths",
			PathCollector: makePathCollector(
				t, "", nil, []string{"Dockerfile"}, nil, false, false,
			),
			AddTempDirToCollector: true,
			Expected:              []string{"Dockerfile"},
			PathsToCreate:         []string{"Dockerfile"},
		},
		{
			Name: "Globs",
			PathCollector: makePathCollector(
				t, "", nil, nil, []string{filepath.Join("**", "Dockerfile")},
				false, false,
			),
			AddTempDirToCollector: true,
			Expected: []string{
				filepath.Join("globs-test", "Dockerfile"),
			},
			PathsToCreate: []string{
				filepath.Join("globs-test", "Dockerfile"),
			},
		},
		{
			Name: "Recursive",
			PathCollector: makePathCollector(
				t, "", []string{"Dockerfile"}, nil, nil, true, false,
			),
			BaseDirIsTempDir: true,
			Expected: []string{
				filepath.Join("recursive-test", "Dockerfile"),
			},
			PathsToCreate: []string{
				filepath.Join("recursive-test", "Dockerfile"),
			},
		},
		{
			Name: "Duplicate Paths",
			PathCollector: makePathCollector(
				t, "", nil, []string{"Dockerfile", "Dockerfile"}, nil,
				false, false,
			),
			AddTempDirToCollector: true,
			Expected:              []string{"Dockerfile"},
			PathsToCreate:         []string{"Dockerfile"},
		},
		{
			Name: "Default Path Outside Of Base Directory",
			PathCollector: makePathCollector(
				t, "", []string{filepath.Join("..", "Dockerfile")}, nil, nil,
				false, false,
			),
			ShouldFail: true,
		},
		{
			Name: "Manual Path Outside Of Base Directory",
			PathCollector: makePathCollector(
				t, "", nil, []string{filepath.Join("..", "Dockerfile")}, nil,
				false, false,
			),
			ShouldFail: true,
		},
		{
			Name: "No Default Paths And Recursive",
			PathCollector: makePathCollector(
				t, "", nil, nil, nil, true, true,
			),
			ShouldFail: true,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			if test.PathCollector == nil {
				return
			}

			var expected []string

			if len(test.PathsToCreate) != 0 {
				tempDir := makeTempDir(t, testDir)
				defer os.RemoveAll(tempDir)

				if test.BaseDirIsTempDir {
					test.PathCollector.BaseDir = tempDir
				}

				if test.AddTempDirToCollector {
					addTempDirToStringSlices(t, test.PathCollector, tempDir)
				}

				makeParentDirsInTempDirFromFilePaths(
					t, tempDir, test.PathsToCreate,
				)
				pathsToCreateContents := make([][]byte, len(test.PathsToCreate))
				writeFilesToTempDir(
					t, tempDir, test.PathsToCreate, pathsToCreateContents,
				)

				for _, path := range test.Expected {
					expected = append(
						expected, filepath.Join(tempDir, path),
					)
				}
			}

			var got []string

			var err error

			done := make(chan struct{})
			for pathResult := range test.PathCollector.CollectPaths(done) {
				if pathResult.Err != nil {
					close(done)
					err = pathResult.Err
					break
				}
				got = append(got, pathResult.Path)
			}

			if test.ShouldFail {
				if err == nil {
					t.Fatal("expected an error but did not get one")
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assertCollectedPathsEqual(t, expected, got)
		})
	}
}
