package rewrite_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func assertOriginalContentsEqualPathContents(
	t *testing.T,
	expected [][]byte,
	got [][]byte,
) {
	for i := range expected {
		if !bytes.Equal(expected[i], got[i]) {
			t.Fatalf("expected %s, got %s", expected[i], got[i])
		}
	}
}

func writeFile(t *testing.T, path string, contents []byte) {
	if err := ioutil.WriteFile(
		path, contents, 0777,
	); err != nil {
		t.Fatal(err)
	}
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
