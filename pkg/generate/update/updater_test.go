package update_test

import (
	"testing"

	cmd_generate "github.com/safe-waters/docker-lock/cmd/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/generate/registry"
	"github.com/safe-waters/docker-lock/pkg/generate/update"
)

func TestImageDigestUpdater(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name                    string
		Images                  []*parse.Image
		ExpectedNumNetworkCalls uint64
		ExpectedImages          []*parse.Image
	}{
		{
			Name: "Image Without Digest",
			Images: []*parse.Image{
				{
					Name: "busybox",
					Tag:  "latest",
				},
			},
			ExpectedNumNetworkCalls: 1,
			ExpectedImages: []*parse.Image{
				{
					Name:   "busybox",
					Tag:    "latest",
					Digest: busyboxLatestSHA,
				},
			},
		},
		{
			Name: "Image With Digest",
			Images: []*parse.Image{
				{
					Name:   "busybox",
					Tag:    "latest",
					Digest: busyboxLatestSHA,
				},
			},
			ExpectedNumNetworkCalls: 0,
			ExpectedImages: []*parse.Image{
				{
					Name:   "busybox",
					Tag:    "latest",
					Digest: busyboxLatestSHA,
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

			client := &registry.HTTPClient{
				Client:      server.Client(),
				RegistryURL: server.URL,
				TokenURL:    server.URL + "?scope=repository%s",
			}

			wrapperManager, err := cmd_generate.DefaultWrapperManager(
				client, cmd_generate.DefaultConfigPath(),
			)
			if err != nil {
				t.Fatal(err)
			}

			updater, err := update.NewImageDigestUpdater(wrapperManager)
			if err != nil {
				t.Fatal(err)
			}

			done := make(chan struct{})

			images := make(chan *parse.Image, len(test.Images))

			for _, image := range test.Images {
				images <- image
			}
			close(images)

			updatedImages := updater.UpdateDigests(images, done)

			var gotImages []*parse.Image

			for updatedImage := range updatedImages {
				if updatedImage.Err != nil {
					t.Fatal(updatedImage.Err)
				}
				gotImages = append(gotImages, updatedImage.Image)
			}

			assertImagesEqual(
				t, test.ExpectedImages, gotImages,
			)

			assertNumNetworkCallsEqual(
				t, test.ExpectedNumNetworkCalls, gotNumNetworkCalls,
			)
		})
	}
}
