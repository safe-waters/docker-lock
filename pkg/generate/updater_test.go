package generate_test

import (
	"testing"

	"github.com/safe-waters/docker-lock/internal/testutils"
	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/generate/update"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

func TestImageDigestUpdater(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name                    string
		Images                  []parse.IImage
		ExpectedNumNetworkCalls uint64
		Expected                []parse.IImage
	}{
		{
			Name: "Dockerfiles, Composefiles, And Kubernetesfiles",
			Images: []parse.IImage{
				parse.NewImage(
					kind.Dockerfile, "redis", "latest", "",
					map[string]interface{}{
						"position": 0,
						"path":     "Dockerfile",
					}, nil,
				),
				parse.NewImage(
					kind.Dockerfile, "redis", "latest",
					testutils.RedisLatestSHA, map[string]interface{}{
						"position": 2,
						"path":     "Dockerfile",
					}, nil,
				),
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest", "",
					map[string]interface{}{
						"position": 1,
						"path":     "Dockerfile",
					}, nil,
				),
				parse.NewImage(
					kind.Composefile, "busybox", "latest", "",
					map[string]interface{}{
						"position":    0,
						"path":        "docker-compose.yml",
						"serviceName": "svc",
					}, nil,
				),
				parse.NewImage(
					kind.Composefile, "golang", "latest", "",
					map[string]interface{}{
						"position":    0,
						"path":        "docker-compose.yml",
						"serviceName": "anothersvc",
					}, nil,
				),
				parse.NewImage(
					kind.Kubernetesfile, "busybox", "latest", "",
					map[string]interface{}{
						"path":          "pod.yml",
						"containerName": "busybox",
						"docPosition":   0,
						"imagePosition": 1,
					}, nil,
				),
				parse.NewImage(
					kind.Kubernetesfile, "golang", "latest",
					testutils.GolangLatestSHA, map[string]interface{}{
						"path":          "pod.yml",
						"containerName": "golang",
						"docPosition":   0,
						"imagePosition": 0,
					}, nil,
				),
			},
			Expected: []parse.IImage{
				parse.NewImage(
					kind.Dockerfile, "redis", "latest",
					testutils.RedisLatestSHA, map[string]interface{}{
						"position": 0,
						"path":     "Dockerfile",
					}, nil,
				),
				parse.NewImage(
					kind.Dockerfile, "redis", "latest",
					testutils.RedisLatestSHA, map[string]interface{}{
						"position": 2,
						"path":     "Dockerfile",
					}, nil,
				),
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest",
					testutils.BusyboxLatestSHA, map[string]interface{}{
						"position": 1,
						"path":     "Dockerfile",
					}, nil,
				),
				parse.NewImage(
					kind.Composefile, "busybox", "latest",
					testutils.BusyboxLatestSHA, map[string]interface{}{
						"position":    0,
						"path":        "docker-compose.yml",
						"serviceName": "svc",
					}, nil,
				),
				parse.NewImage(
					kind.Composefile, "golang", "latest",
					testutils.GolangLatestSHA, map[string]interface{}{
						"position":    0,
						"path":        "docker-compose.yml",
						"serviceName": "anothersvc",
					}, nil,
				),
				parse.NewImage(
					kind.Kubernetesfile, "busybox", "latest",
					testutils.BusyboxLatestSHA, map[string]interface{}{
						"path":          "pod.yml",
						"containerName": "busybox",
						"docPosition":   0,
						"imagePosition": 1,
					}, nil,
				),
				parse.NewImage(
					kind.Kubernetesfile, "golang", "latest",
					testutils.GolangLatestSHA, map[string]interface{}{
						"path":          "pod.yml",
						"containerName": "golang",
						"docPosition":   0,
						"imagePosition": 0,
					}, nil,
				),
			},
			ExpectedNumNetworkCalls: 3,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			var gotNumNetworkCalls uint64

			digestRequester := testutils.NewMockDigestRequester(
				t, &gotNumNetworkCalls,
			)
			innerUpdater, err := update.NewImageDigestUpdater(
				digestRequester, false, false,
			)
			if err != nil {
				t.Fatal(err)
			}

			updater, err := generate.NewImageDigestUpdater(innerUpdater)
			if err != nil {
				t.Fatal(err)
			}

			done := make(chan struct{})
			defer close(done)

			imagesToUpdate := make(chan parse.IImage, len(test.Images))

			for _, anyImage := range test.Images {
				imagesToUpdate <- anyImage
			}
			close(imagesToUpdate)

			updatedImages := updater.UpdateDigests(imagesToUpdate, done)

			var got []parse.IImage

			for image := range updatedImages {
				if image.Err() != nil {
					t.Fatal(image.Err())
				}

				got = append(got, image)
			}

			testutils.SortImages(t, test.Expected)
			testutils.SortImages(t, got)

			testutils.AssertImagesEqual(t, test.Expected, got)
			testutils.AssertNumNetworkCallsEqual(
				t, test.ExpectedNumNetworkCalls, gotNumNetworkCalls,
			)
		})
	}
}
