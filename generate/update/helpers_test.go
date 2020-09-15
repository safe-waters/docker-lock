package update_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/safe-waters/docker-lock/generate/parse"
	"github.com/safe-waters/docker-lock/registry"
	"github.com/safe-waters/docker-lock/registry/contrib"
	"github.com/safe-waters/docker-lock/registry/firstparty"
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

func assertNumNetworkCallsEqual(t *testing.T, expected uint64, got uint64) {
	t.Helper()

	if expected != got {
		t.Fatalf("expected %d network calls, got %d", expected, got)
	}
}

func defaultWrapperManager(
	t *testing.T,
	server *httptest.Server,
) *registry.WrapperManager {
	t.Helper()

	client := &registry.HTTPClient{
		Client:      server.Client(),
		RegistryURL: server.URL,
		TokenURL:    server.URL + "?scope=repository%s",
	}
	configPath := defaultConfigPath()

	defaultWrapper, err := firstparty.DefaultWrapper(client, configPath)
	if err != nil {
		t.Fatal(err)
	}

	wrapperManager := registry.NewWrapperManager(defaultWrapper)
	wrapperManager.Add(firstparty.AllWrappers(client, configPath)...)
	wrapperManager.Add(contrib.AllWrappers(client, configPath)...)

	return wrapperManager
}

func defaultConfigPath() string {
	if homeDir, err := os.UserHomeDir(); err == nil {
		configPath := filepath.Join(homeDir, ".docker", "config.json")

		if _, err := os.Stat(configPath); err != nil {
			return ""
		}

		return configPath
	}

	return ""
}
