package update_test

import (
	"testing"

	"github.com/safe-waters/docker-lock/generate/parse"
	"github.com/safe-waters/docker-lock/generate/update"
)

func TestComposefileImageDigestUpdater(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name                      string
		ComposefileImages         []*parse.ComposefileImage
		ExpectedNumNetworkCalls   uint64
		ExpectedComposefileImages []*parse.ComposefileImage
	}{
		{
			Name: "Composefile Image Without Digest",
			ComposefileImages: []*parse.ComposefileImage{
				{
					Image: &parse.Image{
						Name: "busybox",
						Tag:  "latest",
					},
				},
			},
			ExpectedNumNetworkCalls: 1,
			ExpectedComposefileImages: []*parse.ComposefileImage{
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
			Name: "Composefile Image With Digest",
			ComposefileImages: []*parse.ComposefileImage{
				{
					Image: &parse.Image{
						Name:   "busybox",
						Tag:    "latest",
						Digest: busyboxLatestSHA,
					},
				},
			},
			ExpectedNumNetworkCalls: 0,
			ExpectedComposefileImages: []*parse.ComposefileImage{
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
			ComposefileImages: []*parse.ComposefileImage{
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
			ExpectedComposefileImages: []*parse.ComposefileImage{
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

			updater := &update.ComposefileImageDigestUpdater{
				QueryExecutor: queryExecutor,
			}

			done := make(chan struct{})

			composefileImages := make(
				chan *parse.ComposefileImage, len(test.ComposefileImages),
			)

			for _, composefileImage := range test.ComposefileImages {
				composefileImages <- composefileImage
			}
			close(composefileImages)

			updatedComposefileImages := updater.UpdateDigests(
				composefileImages, done,
			)

			var gotComposefileImages []*parse.ComposefileImage

			for composefileImage := range updatedComposefileImages {
				if composefileImage.Err != nil {
					t.Fatal(composefileImage.Err)
				}
				gotComposefileImages = append(
					gotComposefileImages, composefileImage,
				)
			}

			assertComposefileImagesEqual(
				t, test.ExpectedComposefileImages, gotComposefileImages,
			)

			assertNumNetworkCallsEqual(
				t, test.ExpectedNumNetworkCalls, gotNumNetworkCalls,
			)
		})
	}
}
