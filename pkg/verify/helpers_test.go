package verify_test

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mockServer(t *testing.T) *httptest.Server {
	t.Helper()

	const busyboxLatestSHA = "busybox"

	const golangLatestSHA = "golang"

	const redisLatestSHA = "redis"

	server := httptest.NewServer(
		http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			switch url := req.URL.String(); {
			case strings.Contains(url, "scope"):
				byt := []byte(`{"token": "NOT_USED"}`)
				_, err := res.Write(byt)
				if err != nil {
					t.Fatal(err)
				}
			case strings.Contains(url, "manifests"):
				urlParts := strings.Split(url, "/")
				repo, ref := urlParts[2], urlParts[len(urlParts)-1]

				var digest string
				switch fmt.Sprintf("%s:%s", repo, ref) {
				case "busybox:latest":
					digest = busyboxLatestSHA
				case "redis:latest":
					digest = redisLatestSHA
				case "golang:latest":
					digest = golangLatestSHA
				default:
					digest = fmt.Sprintf(
						"repo %s with ref %s not defined for testing",
						repo, ref,
					)
				}

				res.Header().Set("Docker-Content-Digest", digest)
			}
		}))

	return server
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

func makeTempDirInCurrentDir(t *testing.T) string {
	t.Helper()

	tempDir := generateUUID(t)
	makeDir(t, tempDir)

	return tempDir
}

func makeDir(t *testing.T, dirPath string) {
	t.Helper()

	err := os.MkdirAll(dirPath, 0777)
	if err != nil {
		t.Fatal(err)
	}
}

func generateUUID(t *testing.T) string {
	t.Helper()

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
