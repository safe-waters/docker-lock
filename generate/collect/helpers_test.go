package collect_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/safe-waters/docker-lock/generate/collect"
)

func assertCollectedPathsEqual(t *testing.T, expected []string, got []string) {
	t.Helper()

	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func addTempDirToStringSlices(
	t *testing.T,
	collector *collect.PathCollector,
	tempDir string,
) {
	t.Helper()

	collectorValue := reflect.ValueOf(collector).Elem()

	for i := 0; i < collectorValue.NumField(); i++ {
		field := collectorValue.Field(i)

		if field.Kind() == reflect.Slice {
			concreteSliceField := field.Interface().([]string)

			for i := 0; i < len(concreteSliceField); i++ {
				concreteSliceField[i] = filepath.Join(
					tempDir, concreteSliceField[i],
				)
			}
		}
	}
}

func writeFilesToTempDir(
	t *testing.T,
	tempDir string,
	fileNames []string,
	fileContents [][]byte,
) []string {
	t.Helper()

	if len(fileNames) != len(fileContents) {
		t.Fatalf(
			"different number of names and contents: %d names, %d contents",
			len(fileNames), len(fileContents))
	}

	fullPaths := make([]string, len(fileNames))

	for i, name := range fileNames {
		fullPath := filepath.Join(tempDir, name)

		if err := ioutil.WriteFile(
			fullPath, fileContents[i], 0777,
		); err != nil {
			t.Fatal(err)
		}

		fullPaths[i] = fullPath
	}

	return fullPaths
}

func makeDir(t *testing.T, dirPath string) {
	t.Helper()

	err := os.MkdirAll(dirPath, 0777)
	if err != nil {
		t.Fatal(err)
	}
}

func makeTempDir(t *testing.T, dirName string) string {
	t.Helper()

	dir, err := ioutil.TempDir("", dirName)
	if err != nil {
		t.Fatal(err)
	}

	return dir
}

func makeParentDirsInTempDirFromFilePaths(
	t *testing.T,
	tempDir string,
	paths []string,
) {
	t.Helper()

	for _, p := range paths {
		dir, _ := filepath.Split(p)
		fullDir := filepath.Join(tempDir, dir)

		makeDir(t, fullDir)
	}
}
