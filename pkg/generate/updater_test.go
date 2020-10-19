package generate_test

import (
	"testing"

	cmd_generate "github.com/safe-waters/docker-lock/cmd/generate"
	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/generate/registry"
	"github.com/safe-waters/docker-lock/pkg/generate/update"
)

func TestImageDigestUpdater(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name                    string
		AnyImages               []*generate.AnyImage
		ExpectedNumNetworkCalls uint64
		Expected                []*generate.AnyImage
	}{
		{
			Name: "Dockerfiles And Composefiles",
			AnyImages: []*generate.AnyImage{
				{
					DockerfileImage: &parse.DockerfileImage{
						Image: &parse.Image{
							Name: "redis",
							Tag:  "latest",
						},
						Position: 0,
						Path:     "Dockerfile",
					},
				},
				{
					DockerfileImage: &parse.DockerfileImage{
						Image: &parse.Image{
							Name:   "redis",
							Tag:    "latest",
							Digest: redisLatestSHA,
						},
						Position: 2,
						Path:     "Dockerfile",
					},
				},
				{
					DockerfileImage: &parse.DockerfileImage{
						Image: &parse.Image{
							Name: "busybox",
							Tag:  "latest",
						},
						Position: 1,
						Path:     "Dockerfile",
					},
				},
				{
					ComposefileImage: &parse.ComposefileImage{
						Image: &parse.Image{
							Name: "busybox",
							Tag:  "latest",
						},
						Position:    0,
						Path:        "docker-compose.yml",
						ServiceName: "svc",
					},
				},
				{
					ComposefileImage: &parse.ComposefileImage{
						Image: &parse.Image{
							Name: "golang",
							Tag:  "latest",
						},
						Position:    0,
						Path:        "docker-compose.yml",
						ServiceName: "anothersvc",
					},
				},
				{
					ComposefileImage: &parse.ComposefileImage{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: busyboxLatestSHA,
						},
						Position:    1,
						Path:        "docker-compose.yml",
						ServiceName: "svc",
					},
				},
			},
			Expected: []*generate.AnyImage{
				{
					DockerfileImage: &parse.DockerfileImage{
						Image: &parse.Image{
							Name:   "redis",
							Tag:    "latest",
							Digest: redisLatestSHA,
						},
						Position: 0,
						Path:     "Dockerfile",
					},
				},
				{
					DockerfileImage: &parse.DockerfileImage{
						Image: &parse.Image{
							Name:   "redis",
							Tag:    "latest",
							Digest: redisLatestSHA,
						},
						Position: 2,
						Path:     "Dockerfile",
					},
				},
				{
					DockerfileImage: &parse.DockerfileImage{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: busyboxLatestSHA,
						},
						Position: 1,
						Path:     "Dockerfile",
					},
				},
				{
					ComposefileImage: &parse.ComposefileImage{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: busyboxLatestSHA,
						},
						Position:    0,
						Path:        "docker-compose.yml",
						ServiceName: "svc",
					},
				},
				{
					ComposefileImage: &parse.ComposefileImage{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: golangLatestSHA,
						},
						Position:    0,
						Path:        "docker-compose.yml",
						ServiceName: "anothersvc",
					},
				},
				{
					ComposefileImage: &parse.ComposefileImage{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: busyboxLatestSHA,
						},
						Position:    1,
						Path:        "docker-compose.yml",
						ServiceName: "svc",
					},
				},
			},
			ExpectedNumNetworkCalls: 3,
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

			innerUpdater, err := update.NewImageDigestUpdater(wrapperManager)
			if err != nil {
				t.Fatal(err)
			}

			updater, err := generate.NewImageDigestUpdater(innerUpdater, false)
			if err != nil {
				t.Fatal(err)
			}

			done := make(chan struct{})

			anyImages := make(chan *generate.AnyImage, len(test.AnyImages))

			for _, anyImage := range test.AnyImages {
				anyImages <- anyImage
			}
			close(anyImages)

			updatedImages := updater.UpdateDigests(anyImages, done)

			var got []*generate.AnyImage

			for updatedImage := range updatedImages {
				if updatedImage.Err != nil {
					t.Fatal(updatedImage.Err)
				}

				got = append(got, updatedImage)
			}

			sortedExpected := sortAnyImages(t, test.Expected)
			sortedGot := sortAnyImages(t, got)

			assertAnyImagesEqual(t, sortedExpected, sortedGot)

			assertNumNetworkCallsEqual(
				t, test.ExpectedNumNetworkCalls, gotNumNetworkCalls,
			)
		})
	}
}
