package parse_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
)

type DockerfileImageWithoutStructTags struct {
	*parse.Image
	Position int
	Path     string
	Err      error
}

type ComposefileImageWithoutStructTags struct {
	*parse.Image
	DockerfilePath string
	Position       int
	ServiceName    string
	Path           string
	Err            error
}

func assertDockerfileImagesEqual(
	t *testing.T,
	expected []*parse.DockerfileImage,
	got []*parse.DockerfileImage,
) {
	t.Helper()

	if !reflect.DeepEqual(expected, got) {
		expectedWithoutStructTags := copyDockerfileImagesToDockerfileImagesWithoutStructTags( // nolint: lll
			t, expected,
		)

		gotWithoutStructTags := copyDockerfileImagesToDockerfileImagesWithoutStructTags( // nolint: lll
			t, got,
		)

		t.Fatalf(
			"expected %+v, got %+v",
			jsonPrettyPrint(t, expectedWithoutStructTags),
			jsonPrettyPrint(t, gotWithoutStructTags),
		)
	}
}

func assertComposefileImagesEqual(
	t *testing.T,
	expected []*parse.ComposefileImage,
	got []*parse.ComposefileImage,
) {
	t.Helper()

	if !reflect.DeepEqual(expected, got) {
		expectedWithoutStructTags := copyComposefileImagesToComposefileImagesWithoutStructTags( // nolint: lll
			t, expected,
		)

		gotWithoutStructTags := copyComposefileImagesToComposefileImagesWithoutStructTags( // nolint: lll
			t, got,
		)

		t.Fatalf(
			"expected %+v, got %+v",
			jsonPrettyPrint(t, expectedWithoutStructTags),
			jsonPrettyPrint(t, gotWithoutStructTags),
		)
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

func copyDockerfileImagesToDockerfileImagesWithoutStructTags(
	t *testing.T,
	dockerfileImages []*parse.DockerfileImage,
) []*DockerfileImageWithoutStructTags {
	t.Helper()

	dockerfileImagesWithoutStructTags := make(
		[]*DockerfileImageWithoutStructTags, len(dockerfileImages),
	)

	for i, image := range dockerfileImages {
		dockerfileImagesWithoutStructTags[i] =
			&DockerfileImageWithoutStructTags{
				Image:    image.Image,
				Position: image.Position,
				Path:     image.Path,
				Err:      image.Err,
			}
	}

	return dockerfileImagesWithoutStructTags
}

func copyComposefileImagesToComposefileImagesWithoutStructTags(
	t *testing.T,
	composefileImages []*parse.ComposefileImage,
) []*ComposefileImageWithoutStructTags {
	t.Helper()

	composefileImagesWithoutStructTags := make(
		[]*ComposefileImageWithoutStructTags, len(composefileImages),
	)

	for i, image := range composefileImages {
		composefileImagesWithoutStructTags[i] =
			&ComposefileImageWithoutStructTags{
				Image:          image.Image,
				DockerfilePath: image.DockerfilePath,
				Position:       image.Position,
				ServiceName:    image.ServiceName,
				Path:           image.Path,
				Err:            image.Err,
			}
	}

	return composefileImagesWithoutStructTags
}

func jsonPrettyPrint(t *testing.T, i interface{}) string {
	t.Helper()

	byt, err := json.MarshalIndent(i, "", "\t")
	if err != nil {
		t.Fatal(err)
	}

	return string(byt)
}

func sortDockerfileImageParserResults(
	t *testing.T,
	results []*parse.DockerfileImage,
) {
	t.Helper()

	sort.Slice(results, func(i, j int) bool {
		switch {
		case results[i].Path != results[j].Path:
			return results[i].Path < results[j].Path
		default:
			return results[i].Position < results[j].Position
		}
	})
}

func sortComposefileImageParserResults(
	t *testing.T,
	results []*parse.ComposefileImage,
) {
	t.Helper()

	sort.Slice(results, func(i, j int) bool {
		switch {
		case results[i].Path != results[j].Path:
			return results[i].Path < results[j].Path
		case results[i].ServiceName != results[j].ServiceName:
			return results[i].ServiceName < results[j].ServiceName
		case results[i].DockerfilePath != results[j].DockerfilePath:
			return results[i].DockerfilePath < results[j].DockerfilePath
		default:
			return results[i].Position < results[j].Position
		}
	})
}
