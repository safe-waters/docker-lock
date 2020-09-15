package update_test

import (
	"testing"

	"github.com/safe-waters/docker-lock/generate/parse"
	"github.com/safe-waters/docker-lock/generate/update"
)

func TestDockerfileImageDigestUpdater(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name                     string
		DockerfileImages         []*parse.DockerfileImage
		ExpectedNumNetworkCalls  uint64
		ExpectedDockerfileImages []*parse.DockerfileImage
	}{
		{
			Name: "Dockerfile Image Without Digest",
			DockerfileImages: []*parse.DockerfileImage{
				{
					Image: &parse.Image{
						Name: "busybox",
						Tag:  "latest",
					},
				},
			},
			ExpectedNumNetworkCalls: 1,
			ExpectedDockerfileImages: []*parse.DockerfileImage{
				{
					Image: &parse.Image{
						Name:   "busybox",
						Tag:    "latest",
						Digest: busyboxLatestSHA,
					},
				},
			},
		},
		{
			Name: "Dockerfile Image With Digest",
			DockerfileImages: []*parse.DockerfileImage{
				{
					Image: &parse.Image{
						Name:   "busybox",
						Tag:    "latest",
						Digest: busyboxLatestSHA,
					},
				},
			},
			ExpectedNumNetworkCalls: 0,
			ExpectedDockerfileImages: []*parse.DockerfileImage{
				{
					Image: &parse.Image{
						Name:   "busybox",
						Tag:    "latest",
						Digest: busyboxLatestSHA,
					},
				},
			},
		},
		{
			Name: "No Duplicate Network Calls",
			DockerfileImages: []*parse.DockerfileImage{
				{
					Image: &parse.Image{
						Name:   "busybox",
						Tag:    "latest",
						Digest: busyboxLatestSHA,
					},
				},
				{
					Image: &parse.Image{
						Name: "busybox",
						Tag:  "latest",
					},
				},
			},
			ExpectedNumNetworkCalls: 1,
			ExpectedDockerfileImages: []*parse.DockerfileImage{
				{
					Image: &parse.Image{
						Name:   "busybox",
						Tag:    "latest",
						Digest: busyboxLatestSHA,
					},
				},
				{
					Image: &parse.Image{
						Name:   "busybox",
						Tag:    "latest",
						Digest: busyboxLatestSHA,
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			var gotNumNetworkCalls uint64

			server := mockServer(t, &gotNumNetworkCalls)
			defer server.Close()

			wrapperManager := defaultWrapperManager(t, server)

			queryExecutor, err := update.NewQueryExecutor(wrapperManager)
			if err != nil {
				t.Fatal(err)
			}

			updater := &update.DockerfileImageDigestUpdater{
				QueryExecutor: queryExecutor,
			}

			done := make(chan struct{})

			dockerfileImages := make(
				chan *parse.DockerfileImage, len(test.DockerfileImages),
			)

			for _, dockerfileImage := range test.DockerfileImages {
				dockerfileImages <- dockerfileImage
			}
			close(dockerfileImages)

			updatedDockerfileImages := updater.UpdateDigests(
				dockerfileImages, done,
			)

			var gotDockerfileImages []*parse.DockerfileImage

			for dockerfileImage := range updatedDockerfileImages {
				if dockerfileImage.Err != nil {
					t.Fatal(dockerfileImage.Err)
				}
				gotDockerfileImages = append(
					gotDockerfileImages, dockerfileImage,
				)
			}

			assertDockerfileImagesEqual(
				t, test.ExpectedDockerfileImages, gotDockerfileImages,
			)

			assertNumNetworkCallsEqual(
				t, test.ExpectedNumNetworkCalls, gotNumNetworkCalls,
			)
		})
	}
}
