package write_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func assertWrittenFiles(t *testing.T, expected [][]byte, got []string) {
	t.Helper()

	if len(expected) != len(got) {
		t.Fatalf("expected %d contents, got %d", len(expected), len(got))
	}

	for i := range expected {
		gotContents, err := ioutil.ReadFile(got[i])
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(expected[i], gotContents) {
			t.Fatalf(
				"expected:\n%s\ngot:\n%s",
				string(expected[i]),
				string(gotContents),
			)
		}
	}
}

func makeTempDirInCurrentDir(t *testing.T) string {
	t.Helper()

	tempDir := generateUUID(t)
	makeDir(t, tempDir)

	return tempDir
}

func writeFilesToTempDir(
	t *testing.T,
	tempDir string,
	fileNames []string,
	fileContents [][]byte,
) []string { // nolint: unparam
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

func generateUUID(t *testing.T) string {
	b := make([]byte, 16)

	_, err := rand.Read(b)
	if err != nil {
		t.Fatal(err)
	}

	uuid := fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:],
	)

	return uuid
}
