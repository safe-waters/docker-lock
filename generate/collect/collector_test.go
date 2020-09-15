package collect_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/generate/collect"
)

const collectTestDir = "collect"

func TestNewPathCollector(t *testing.T) {
	t.Parallel()

	_, err := collect.NewPathCollector("", nil, nil, nil, true)
	if err == nil {
		t.Fatal(err)
	}
}

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
			PathCollector: &collect.PathCollector{
				DefaultPaths: []string{"Dockerfile"},
			},
			AddTempDirToCollector: true,
			PathsToCreate:         []string{"Dockerfile"},
			Expected:              []string{"Dockerfile"},
		},
		{
			Name: "Default Path Does Not Exist",
			PathCollector: &collect.PathCollector{
				DefaultPaths: []string{"Dockerfile"},
			},
		},
		{
			Name: "Do Not Use Default Paths If Other Methods Specified",
			PathCollector: &collect.PathCollector{
				DefaultPaths: []string{"Dockerfile"},
				ManualPaths:  []string{"Dockerfile-Manual"},
			},
			AddTempDirToCollector: true,
			Expected:              []string{"Dockerfile-Manual"},
			PathsToCreate:         []string{"Dockerfile-Manual"},
		},
		{
			Name: "Manual Paths",
			PathCollector: &collect.PathCollector{
				ManualPaths: []string{"Dockerfile"},
			},
			AddTempDirToCollector: true,
			Expected:              []string{"Dockerfile"},
			PathsToCreate:         []string{"Dockerfile"},
		},
		{
			Name: "Globs",
			PathCollector: &collect.PathCollector{
				Globs: []string{filepath.Join("**", "Dockerfile")},
			},
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
			PathCollector: &collect.PathCollector{
				DefaultPaths: []string{"Dockerfile"},
				Recursive:    true,
			},
			BaseDirIsTempDir: true,
			Expected: []string{
				filepath.Join("recursive-test", "Dockerfile"),
			},
			PathsToCreate: []string{
				filepath.Join("recursive-test", "Dockerfile"),
			},
		},
		{
			Name: "Path Outside Of Base Directory",
			PathCollector: &collect.PathCollector{
				ManualPaths: []string{filepath.Join("..", "Dockerfile")},
			},
			ShouldFail: true,
		},
		{
			Name: "Duplicate Paths",
			PathCollector: &collect.PathCollector{
				ManualPaths: []string{"Dockerfile", "Dockerfile"},
			},
			AddTempDirToCollector: true,
			Expected:              []string{"Dockerfile"},
			PathsToCreate:         []string{"Dockerfile"},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			var expected []string

			if len(test.PathsToCreate) != 0 {
				tempDir := makeTempDir(t, collectTestDir)
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
