package generate_test

import (
	"bytes"
	"encoding/json"
	"testing"

	cmd_generate "github.com/safe-waters/docker-lock/cmd/generate"
	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/generate/registry"
)

func TestGenerator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name       string
		Flags      *cmd_generate.Flags
		Expected   *generate.Lockfile
		ShouldFail bool
	}{
		{
			Name: "Normal Dockerfiles and Composefiles",
			Flags: makeFlags(t, "testdata/success", "docker-lock.json", "",
				".env", false, []string{"nocompose/Dockerfile"}, nil, nil, nil,
				false, false, false, false,
			),
			Expected: &generate.Lockfile{
				DockerfileImages: map[string][]*parse.DockerfileImage{
					"testdata/success/nocompose/Dockerfile": {
						{
							Image: &parse.Image{
								Name:   "redis",
								Tag:    "latest",
								Digest: redisLatestSHA,
							},
						},
						{
							Image: &parse.Image{
								Name:   "golang",
								Tag:    "latest",
								Digest: golangLatestSHA,
							},
						},
					},
				},
				ComposefileImages: map[string][]*parse.ComposefileImage{
					"testdata/success/docker-compose.yml": {
						{
							Image: &parse.Image{
								Name:   "redis",
								Tag:    "latest",
								Digest: redisLatestSHA,
							},
							DockerfilePath: "testdata/success/database/Dockerfile", // nolint: lll
							ServiceName:    "database",
						},
						{
							Image: &parse.Image{
								Name:   "golang",
								Tag:    "latest",
								Digest: golangLatestSHA,
							},
							ServiceName: "web",
						},
					},
				},
			},
		},
		{
			Name: "Exclude Dockerfiles",
			Flags: makeFlags(t, "testdata/success", "docker-lock.json", "",
				".env", false, []string{"nocompose/Dockerfile"}, nil, nil, nil,
				false, false, true, false,
			),
			Expected: &generate.Lockfile{
				DockerfileImages: nil,
				ComposefileImages: map[string][]*parse.ComposefileImage{
					"testdata/success/docker-compose.yml": {
						{
							Image: &parse.Image{
								Name:   "redis",
								Tag:    "latest",
								Digest: redisLatestSHA,
							},
							DockerfilePath: "testdata/success/database/Dockerfile", // nolint: lll
							ServiceName:    "database",
						},
						{
							Image: &parse.Image{
								Name:   "golang",
								Tag:    "latest",
								Digest: golangLatestSHA,
							},
							ServiceName: "web",
						},
					},
				},
			},
		},
		{
			Name: "Exclude Composefiles",
			Flags: makeFlags(t, "testdata/success", "docker-lock.json", "",
				".env", false, []string{"nocompose/Dockerfile"},
				[]string{"docker-compose.yml"}, nil, nil, false,
				false, false, true,
			),
			Expected: &generate.Lockfile{
				DockerfileImages: map[string][]*parse.DockerfileImage{
					"testdata/success/nocompose/Dockerfile": {
						{
							Image: &parse.Image{
								Name:   "redis",
								Tag:    "latest",
								Digest: redisLatestSHA,
							},
						},
						{
							Image: &parse.Image{
								Name:   "golang",
								Tag:    "latest",
								Digest: golangLatestSHA,
							},
						},
					},
				},
				ComposefileImages: nil,
			},
		},
		{
			Name: "Exclude All",
			Flags: makeFlags(t, "testdata/success", "docker-lock.json", "",
				".env", false, []string{"nocompose/Dockerfile"},
				[]string{"docker-compose.yml"}, nil, nil, false,
				false, true, true,
			),
			Expected: &generate.Lockfile{
				DockerfileImages:  nil,
				ComposefileImages: nil,
			},
		},
		{
			Name: "Service Typo",
			Flags: makeFlags(t, "testdata/fail", "docker-lock.json", "", ".env",
				false, nil, nil, nil, nil, false, false, false, false,
			),
			ShouldFail: true,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			var numNetworkCalls uint64

			server := mockServer(t, &numNetworkCalls)
			defer server.Close()

			client := &registry.HTTPClient{
				Client:      server.Client(),
				RegistryURL: server.URL,
				TokenURL:    server.URL + "?scope=repository%s",
			}

			generator, err := cmd_generate.SetupGenerator(client, test.Flags)
			if err != nil {
				t.Fatal(err)
			}

			var buf bytes.Buffer

			err = generator.GenerateLockfile(&buf)

			if test.ShouldFail {
				if err == nil {
					t.Fatal("expected error but did not get one")
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}

			var got generate.Lockfile
			if err = json.Unmarshal(buf.Bytes(), &got); err != nil {
				t.Fatal(err)
			}

			assertLockfilesEqual(t, test.Expected, &got)
		})
	}
}
