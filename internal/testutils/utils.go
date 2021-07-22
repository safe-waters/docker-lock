package testutils

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"sync/atomic"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/generate/update"
)

const (
	BusyboxLatestSHA = "bae015c28bc7cdee3b7ef20d35db4299e3068554a769070950229d9f53f58572" // nolint: lll
	GolangLatestSHA  = "6cb55c08bbf44793f16e3572bd7d2ae18f7a858f6ae4faa474c0a6eae1174a5d" // nolint: lll
	RedisLatestSHA   = "09c33840ec47815dc0351f1eca3befe741d7105b3e95bc8fdb9a7e4985b9e1e5" // nolint: lll
)

type mockDigestRequester struct {
	numNetworkCalls *uint64
}

func NewMockDigestRequester(
	t *testing.T,
	numNetworkCalls *uint64,
) update.IDigestRequester {
	t.Helper()

	return &mockDigestRequester{
		numNetworkCalls: numNetworkCalls,
	}
}

func (m *mockDigestRequester) Digest(name string, tag string) (string, error) {
	if m.numNetworkCalls != nil {
		atomic.AddUint64(m.numNetworkCalls, 1)
	}

	nameTag := fmt.Sprintf("%s:%s", name, tag)

	switch nameTag {
	case "busybox:latest":
		return BusyboxLatestSHA, nil
	case "redis:latest":
		return RedisLatestSHA, nil
	case "golang:latest":
		return GolangLatestSHA, nil
	default:
		return "", fmt.Errorf("no digest found for %s", nameTag)
	}
}

func AssertImagesEqual(
	t *testing.T,
	expected []parse.IImage,
	got []parse.IImage,
) {
	t.Helper()

	if len(expected) != len(got) {
		t.Fatalf("expected %d images, got %d", len(expected), len(got))
	}

	var i int

	for i < len(expected) {
		if expected[i].Kind() != got[i].Kind() {
			t.Fatalf(
				"expected kind %s, got %s", expected[i].Kind(), got[i].Kind(),
			)
		}

		if expected[i].Name() != got[i].Name() {
			t.Fatalf(
				"expected name %s, got %s", expected[i].Name(), got[i].Name(),
			)
		}

		if expected[i].Tag() != got[i].Tag() {
			t.Fatalf(
				"expected digest %s, got %s", expected[i].Tag(), got[i].Tag(),
			)
		}

		if expected[i].Digest() != got[i].Digest() {
			t.Fatalf(
				"expected digest %s, got %s", expected[i].Digest(),
				got[i].Digest(),
			)
		}

		expectedMetadataByt, err := json.MarshalIndent(
			expected[i].Metadata(), "", "\t",
		)
		if err != nil {
			t.Fatal(err)
		}

		gotMetadataByt, err := json.MarshalIndent(got[i].Metadata(), "", "\t")
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(expectedMetadataByt, gotMetadataByt) {
			t.Fatalf(
				"expected metadata %#v, got %#v",
				string(expectedMetadataByt), string(gotMetadataByt),
			)
		}

		i++
	}
}

func AssertFlagsEqual(
	t *testing.T,
	expected interface{},
	got interface{},
) {
	t.Helper()

	if !reflect.DeepEqual(expected, got) {
		t.Fatalf(
			"expected %+v, got %+v",
			jsonPrettyPrint(t, expected), jsonPrettyPrint(t, got),
		)
	}
}

func AssertWrittenFilesEqual(t *testing.T, expected [][]byte, got []string) {
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

func AssertNumNetworkCallsEqual(t *testing.T, expected uint64, got uint64) {
	t.Helper()

	if expected != got {
		t.Fatalf("expected %d network calls, got %d", expected, got)
	}
}

func MakeDir(t *testing.T, dirPath string) {
	t.Helper()

	err := os.MkdirAll(dirPath, 0777) // nolint: gomnd
	if err != nil {
		t.Fatal(err)
	}
}

func MakeTempDir(t *testing.T, dirName string) string {
	t.Helper()

	dir, err := ioutil.TempDir("", dirName)
	if err != nil {
		t.Fatal(err)
	}

	return dir
}

func MakeTempDirInCurrentDir(t *testing.T) string {
	t.Helper()

	const tempLen = 16

	b := make([]byte, tempLen)

	_, err := rand.Read(b)
	if err != nil {
		t.Fatal(err)
	}

	uuid := fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:],
	)
	MakeDir(t, uuid)

	return uuid
}

func MakeParentDirsInTempDirFromFilePaths(
	t *testing.T,
	tempDir string,
	paths []string,
) {
	t.Helper()

	for _, p := range paths {
		dir, _ := filepath.Split(p)
		fullDir := filepath.Join(tempDir, dir)

		MakeDir(t, fullDir)
	}
}

func WriteFilesToTempDir(
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
			fullPath, fileContents[i], 0777, // nolint: gomnd
		); err != nil {
			t.Fatal(err)
		}

		fullPaths[i] = fullPath
	}

	return fullPaths
}

func SortImages(t *testing.T, images []parse.IImage) {
	t.Helper()

	sort.Slice(images, func(i, j int) bool {
		return images[i].Kind() < images[j].Kind()
	})
	SortDockerfileImages(t, images)
	SortComposefileImages(t, images)
	SortKubernetesfileImages(t, images)
}

func SortDockerfileImages(t *testing.T, images []parse.IImage) {
	t.Helper()

	sort.Slice(images, func(i, j int) bool {
		var (
			path1, _     = images[i].Metadata()["path"].(string)
			path2, _     = images[j].Metadata()["path"].(string)
			position1, _ = images[i].Metadata()["position"].(int)
			position2, _ = images[j].Metadata()["position"].(int)
		)

		switch {
		case path1 != path2:
			return path1 < path2
		default:
			return position1 < position2
		}
	})
}

func SortKubernetesfileImages(t *testing.T, images []parse.IImage) {
	t.Helper()

	sort.Slice(images, func(i, j int) bool {
		var (
			path1, _          = images[i].Metadata()["path"].(string)
			path2, _          = images[j].Metadata()["path"].(string)
			docPosition1, _   = images[i].Metadata()["docPosition"].(int)
			docPosition2, _   = images[j].Metadata()["docPosition"].(int)
			imagePosition1, _ = images[i].Metadata()["imagePosition"].(int)
			imagePosition2, _ = images[j].Metadata()["imagePosition"].(int)
		)

		switch {
		case path1 != path2:
			return path1 < path2
		case docPosition1 != docPosition2:
			return docPosition1 < docPosition2
		default:
			return imagePosition1 < imagePosition2
		}
	})
}

func SortComposefileImages(t *testing.T, images []parse.IImage) {
	t.Helper()

	sort.Slice(images, func(i, j int) bool {
		var (
			path1, _            = images[i].Metadata()["path"].(string)
			path2, _            = images[j].Metadata()["path"].(string)
			serviceName1, _     = images[i].Metadata()["serviceName"].(string)
			serviceName2, _     = images[j].Metadata()["serviceName"].(string)
			servicePosition1, _ = images[i].Metadata()["servicePosition"].(int)
			servicePosition2, _ = images[j].Metadata()["servicePosition"].(int)
		)

		switch {
		case path1 != path2:
			return path1 < path2
		case serviceName1 != serviceName2:
			return serviceName1 < serviceName2
		default:
			return servicePosition1 < servicePosition2
		}
	})
}

func SortPaths(t *testing.T, paths []collect.IPath) {
	t.Helper()

	sort.Slice(paths, func(i, j int) bool {
		switch {
		case paths[i].Kind() != paths[j].Kind():
			return paths[i].Kind() < paths[j].Kind()
		default:
			return paths[i].Val() < paths[j].Val()
		}
	})
}

func GetAbsPath(t *testing.T) string {
	t.Helper()

	absPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		t.Fatal(err)
	}

	return absPath
}

func jsonPrettyPrint(t *testing.T, i interface{}) string {
	t.Helper()

	byt, err := json.MarshalIndent(i, "", "\t")
	if err != nil {
		t.Fatal(err)
	}

	return string(byt)
}
