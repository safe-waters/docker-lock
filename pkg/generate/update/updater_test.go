package update_test

import (
	"testing"

	cmd_generate "github.com/safe-waters/docker-lock/cmd/generate"
	"github.com/safe-waters/docker-lock/internal/testutils"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/generate/registry"
	"github.com/safe-waters/docker-lock/pkg/generate/update"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

func TestImageDigestUpdater(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name                    string
		Images                  []parse.IImage
		UpdateExistingDigests   bool
		ExpectedNumNetworkCalls uint64
		ExpectedImages          []parse.IImage
	}{
		{
			Name: "Image Without Digest",
			Images: []parse.IImage{
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest", "", nil, nil,
				),
			},
			ExpectedNumNetworkCalls: 1,
			ExpectedImages: []parse.IImage{
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest",
					testutils.BusyboxLatestSHA, nil, nil,
				),
			},
		},
		{
			Name: "Image With Digest",
			Images: []parse.IImage{
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest",
					testutils.BusyboxLatestSHA, nil, nil,
				),
			},
			ExpectedNumNetworkCalls: 0,
			ExpectedImages: []parse.IImage{
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest",
					testutils.BusyboxLatestSHA, nil, nil,
				),
			},
		},
		{
			Name: "Update Existing Digests",
			Images: []parse.IImage{
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest",
					"digest-to-update", nil, nil,
				),
			},
			UpdateExistingDigests:   true,
			ExpectedNumNetworkCalls: 1,
			ExpectedImages: []parse.IImage{
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest",
					testutils.BusyboxLatestSHA, nil, nil,
				),
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			var gotNumNetworkCalls uint64

			server := testutils.MakeMockServer(t, &gotNumNetworkCalls)
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

			updater, err := update.NewImageDigestUpdater(
				wrapperManager, false, test.UpdateExistingDigests,
			)
			if err != nil {
				t.Fatal(err)
			}

			done := make(chan struct{})
			defer close(done)

			images := make(chan parse.IImage, len(test.Images))

			for _, image := range test.Images {
				images <- image
			}
			close(images)

			updatedImages := updater.UpdateDigests(images, done)

			var got []parse.IImage

			for image := range updatedImages {
				if image.Err() != nil {
					t.Fatal(image.Err())
				}
				got = append(got, image)
			}

			testutils.AssertImagesEqual(
				t, test.ExpectedImages, got,
			)

			testutils.AssertNumNetworkCallsEqual(
				t, test.ExpectedNumNetworkCalls, gotNumNetworkCalls,
			)
		})
	}
}
