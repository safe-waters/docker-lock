package generate_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	cmd_generate "github.com/safe-waters/docker-lock/cmd/generate"
	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
)

const busyboxLatestSHA = "bae015c28bc7cdee3b7ef20d35db4299e3068554a769070950229d9f53f58572" // nolint: lll
const golangLatestSHA = "6cb55c08bbf44793f16e3572bd7d2ae18f7a858f6ae4faa474c0a6eae1174a5d"  // nolint: lll
const redisLatestSHA = "09c33840ec47815dc0351f1eca3befe741d7105b3e95bc8fdb9a7e4985b9e1e5"   // nolint: lll

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

type LockfileWithoutStructTags struct {
	DockerfileImages  map[string][]*DockerfileImageWithoutStructTags
	ComposefileImages map[string][]*ComposefileImageWithoutStructTags
}

type AnyImageWithoutStructTags struct {
	DockerfileImage  *DockerfileImageWithoutStructTags
	ComposefileImage *ComposefileImageWithoutStructTags
}

func assertAnyPathsEqual(
	t *testing.T,
	expected []*generate.AnyPath,
	got []*generate.AnyPath,
) {
	t.Helper()

	if !reflect.DeepEqual(expected, got) {
		t.Fatalf(
			"expected %v, got %v",
			jsonPrettyPrint(t, expected),
			jsonPrettyPrint(t, got),
		)
	}
}

func assertAnyImagesEqual(
	t *testing.T,
	expected []*generate.AnyImage,
	got []*generate.AnyImage,
) {
	t.Helper()

	if !reflect.DeepEqual(expected, got) {
		expectedWithoutStructTags := copyAnyImagesToAnyImagesWithoutStructTags(
			t, expected,
		)
		gotWithoutStructTags := copyAnyImagesToAnyImagesWithoutStructTags(
			t, got,
		)
		t.Fatalf(
			"expected %v, got %v",
			jsonPrettyPrint(t, expectedWithoutStructTags),
			jsonPrettyPrint(t, gotWithoutStructTags),
		)
	}
}

func assertLockfilesEqual(
	t *testing.T,
	expected *generate.Lockfile,
	got *generate.Lockfile,
) {
	t.Helper()

	if !reflect.DeepEqual(expected, got) {
		expectedWithoutStructTags := copyLockfileToLockfileWithoutStructTags(
			t, expected,
		)

		gotWithoutStructTags := copyLockfileToLockfileWithoutStructTags(
			t, got,
		)

		t.Fatalf(
			"expected %+v, got %+v",
			jsonPrettyPrint(t, expectedWithoutStructTags),
			jsonPrettyPrint(t, gotWithoutStructTags),
		)
	}
}

func assertDefaultValuesForOmittedJSONReadFromLockfile(
	t *testing.T,
	got *generate.Lockfile,
) {
	t.Helper()

	var buf bytes.Buffer
	if err := got.Write(&buf); err != nil {
		t.Fatal(err)
	}

	var readInLockfile generate.Lockfile
	if err := json.Unmarshal(buf.Bytes(), &readInLockfile); err != nil {
		t.Fatal(err)
	}

	for _, images := range readInLockfile.DockerfileImages {
		for _, image := range images {
			if image.Position != 0 {
				t.Fatal(
					"Written output contains unexpected key 'Position'",
				)
			}

			if image.Path != "" {
				t.Fatal(
					"Written output contains unexpected key 'Path'",
				)
			}

			if image.Err != nil {
				t.Fatal(
					"Written output contains unexpected key 'Err'",
				)
			}
		}
	}

	for _, images := range readInLockfile.ComposefileImages {
		for _, image := range images {
			if image.Position != 0 {
				t.Fatal(
					"Written output contains unexpected key 'Position'",
				)
			}

			if image.Path != "" {
				t.Fatal(
					"Written output contains unexpected key 'Path'",
				)
			}

			if image.Err != nil {
				t.Fatal(
					"Written output contains unexpected key 'Err'",
				)
			}
		}
	}
}

func assertNumNetworkCallsEqual(t *testing.T, expected uint64, got uint64) {
	t.Helper()

	if expected != got {
		t.Fatalf("expected %d network calls, got %d", expected, got)
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

func copyAnyImagesToAnyImagesWithoutStructTags(
	t *testing.T,
	anyImages []*generate.AnyImage,
) []*AnyImageWithoutStructTags {
	var dockerfileImages []*parse.DockerfileImage

	var composefileImages []*parse.ComposefileImage

	for _, anyImage := range anyImages {
		switch {
		case anyImage.DockerfileImage != nil:
			dockerfileImages = append(
				dockerfileImages, anyImage.DockerfileImage,
			)

		case anyImage.ComposefileImage != nil:
			composefileImages = append(
				composefileImages, anyImage.ComposefileImage,
			)
		}
	}

	dockerfileImagesWithoutStructTags := copyDockerfileImagesToDockerfileImagesWithoutStructTags( // nolint: lll
		t, dockerfileImages,
	)
	composefileImagesWithoutStructTags := copyComposefileImagesToComposefileImagesWithoutStructTags( // nolint: lll
		t, composefileImages,
	)

	anyImagesWithoutStructTags := make(
		[]*AnyImageWithoutStructTags,
		len(dockerfileImages)+len(composefileImages),
	)

	var i int

	for _, dockerfileImage := range dockerfileImagesWithoutStructTags {
		anyImagesWithoutStructTags[i] = &AnyImageWithoutStructTags{
			DockerfileImage: dockerfileImage,
		}
		i++
	}

	for _, composefileImage := range composefileImagesWithoutStructTags {
		anyImagesWithoutStructTags[i] = &AnyImageWithoutStructTags{
			ComposefileImage: composefileImage,
		}
		i++
	}

	return anyImagesWithoutStructTags
}

func copyLockfileToLockfileWithoutStructTags(
	t *testing.T,
	lockfile *generate.Lockfile,
) *LockfileWithoutStructTags {
	t.Helper()

	lockfileWithoutStructTags := &LockfileWithoutStructTags{
		ComposefileImages: map[string][]*ComposefileImageWithoutStructTags{},
		DockerfileImages:  map[string][]*DockerfileImageWithoutStructTags{},
	}

	for p := range lockfile.DockerfileImages {
		lockfileWithoutStructTags.DockerfileImages[p] = copyDockerfileImagesToDockerfileImagesWithoutStructTags( // nolint: lll
			t, lockfile.DockerfileImages[p],
		)
	}

	for p := range lockfile.ComposefileImages {
		lockfileWithoutStructTags.ComposefileImages[p] = copyComposefileImagesToComposefileImagesWithoutStructTags( // nolint: lll
			t, lockfile.ComposefileImages[p],
		)
	}

	return lockfileWithoutStructTags
}

func mockServer(t *testing.T, numNetworkCalls *uint64) *httptest.Server {
	t.Helper()

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
				atomic.AddUint64(numNetworkCalls, 1)

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

func jsonPrettyPrint(t *testing.T, i interface{}) string {
	t.Helper()

	byt, err := json.MarshalIndent(i, "", "\t")
	if err != nil {
		t.Fatal(err)
	}

	return string(byt)
}

// nolint: unparam
func makeFlags(
	t *testing.T,
	baseDir string,
	lockfileName string,
	configPath string,
	envPath string,
	ignoreMissingDigests bool,
	dockerfilePaths []string,
	composefilePaths []string,
	dockerfileGlobs []string,
	composefileGlobs []string,
	dockerfileRecursive bool,
	composefileRecursive bool,
	dockerfileExcludeAll bool,
	composefileExcludeAll bool,
) *cmd_generate.Flags {
	t.Helper()

	flags, err := cmd_generate.NewFlags(
		baseDir, lockfileName, configPath, envPath, ignoreMissingDigests,
		dockerfilePaths, composefilePaths, dockerfileGlobs, composefileGlobs,
		dockerfileRecursive, composefileRecursive,
		dockerfileExcludeAll, composefileExcludeAll,
	)
	if err != nil {
		t.Fatal(err)
	}

	return flags
}

func makePathCollector(
	t *testing.T,
	baseDir string,
	defaultDockerfilePaths []string,
	manualDockerfilePaths []string,
	dockerfileGlobs []string,
	dockerfileRecursive bool,
	defaultComposefilePaths []string,
	manualComposefilePaths []string,
	composefileGlobs []string,
	composefileRecursive bool,
	shouldFail bool,
) *generate.PathCollector {
	t.Helper()

	dockerfileCollector := makeCollectPathCollector(
		t, baseDir, defaultDockerfilePaths, manualDockerfilePaths,
		dockerfileGlobs, dockerfileRecursive, shouldFail,
	)
	composefileCollector := makeCollectPathCollector(
		t, baseDir, defaultComposefilePaths, manualComposefilePaths,
		composefileGlobs, composefileRecursive, shouldFail,
	)

	return &generate.PathCollector{
		DockerfileCollector:  dockerfileCollector,
		ComposefileCollector: composefileCollector,
	}
}

func makeCollectPathCollector(
	t *testing.T,
	baseDir string,
	defaultPaths []string,
	manualPaths []string,
	globs []string,
	recursive bool,
	shouldFail bool,
) *collect.PathCollector {
	t.Helper()

	pathCollector, err := collect.NewPathCollector(
		baseDir, defaultPaths, manualPaths, globs, recursive,
	)
	if shouldFail {
		if err == nil {
			t.Fatal("expected error but did not get one")
		}

		return nil
	}

	if err != nil {
		t.Fatal(err)
	}

	return pathCollector
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

func makeTempDir(t *testing.T, dirName string) string {
	t.Helper()

	dir, err := ioutil.TempDir("", dirName)
	if err != nil {
		t.Fatal(err)
	}

	return dir
}

func sortAnyPaths(
	t *testing.T,
	anyPaths []*generate.AnyPath,
) {
	t.Helper()

	sort.Slice(anyPaths, func(i, j int) bool {
		switch {
		case anyPaths[i].DockerfilePath != anyPaths[j].DockerfilePath:
			return anyPaths[i].DockerfilePath < anyPaths[j].DockerfilePath
		default:
			return anyPaths[i].ComposefilePath < anyPaths[j].ComposefilePath
		}
	})
}

func sortAnyImages(
	t *testing.T,
	anyImages []*generate.AnyImage,
) []*generate.AnyImage {
	t.Helper()

	var dockerfileImages []*parse.DockerfileImage

	var composefileImages []*parse.ComposefileImage

	for _, anyImage := range anyImages {
		switch {
		case anyImage.DockerfileImage != nil:
			dockerfileImages = append(
				dockerfileImages, anyImage.DockerfileImage,
			)
		case anyImage.ComposefileImage != nil:
			composefileImages = append(
				composefileImages, anyImage.ComposefileImage,
			)
		}
	}

	sort.Slice(dockerfileImages, func(i, j int) bool {
		switch {
		case dockerfileImages[i].Path != dockerfileImages[j].Path:
			return dockerfileImages[i].Path < dockerfileImages[j].Path
		default:
			return dockerfileImages[i].Position < dockerfileImages[j].Position
		}
	})

	sort.Slice(composefileImages, func(i, j int) bool {
		// nolint: lll
		switch {
		case composefileImages[i].Path != composefileImages[j].Path:
			return composefileImages[i].Path < composefileImages[j].Path
		case composefileImages[i].ServiceName != composefileImages[j].ServiceName:
			return composefileImages[i].ServiceName < composefileImages[j].ServiceName
		case composefileImages[i].DockerfilePath != composefileImages[j].DockerfilePath:
			return composefileImages[i].DockerfilePath < composefileImages[j].DockerfilePath
		default:
			return composefileImages[i].Position < composefileImages[j].Position
		}
	})

	sortedAnyImages := make(
		[]*generate.AnyImage, len(dockerfileImages)+len(composefileImages),
	)

	var i int

	for _, dockerfileImage := range dockerfileImages {
		sortedAnyImages[i] = &generate.AnyImage{
			DockerfileImage: dockerfileImage,
		}

		i++
	}

	for _, composefileImage := range composefileImages {
		sortedAnyImages[i] = &generate.AnyImage{
			ComposefileImage: composefileImage,
		}

		i++
	}

	return sortedAnyImages
}
