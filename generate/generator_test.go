package generate_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/safe-waters/docker-lock/cmd"
	"github.com/safe-waters/docker-lock/generate"
	"github.com/safe-waters/docker-lock/generate/parse"
	"github.com/safe-waters/docker-lock/registry"
)

func TestGenerator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name       string
		Flags      *generate.Flags
		Expected   *generate.Lockfile
		ShouldFail bool
	}{
		{
			Name: "Normal Dockerfiles and Composefiles",
			Flags: makeFlags(t, "testdata/success", "docker-lock.json", "",
				".env", []string{"nocompose/Dockerfile"}, nil, nil, nil, false,
				false, false, false,
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
			Name: "Fail Compose Build",
			Flags: makeFlags(t, "testdata/fail", "docker-lock.json", "", ".env",
				nil, nil, nil, nil, false, false, false, false,
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

			generator, err := cmd.SetupGenerator(client, test.Flags)
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

func makeFlags(
	t *testing.T,
	baseDir string,
	lockfileName string,
	configPath string,
	envPath string,
	dockerfilePaths []string,
	composefilePaths []string,
	dockerfileGlobs []string,
	composefileGlobs []string,
	dockerfileRecursive bool,
	composefileRecursive bool,
	dockerfileExcludeAll bool,
	composefileExcludeAll bool,
) *generate.Flags {
	t.Helper()

	flags, err := generate.NewFlags(
		baseDir, lockfileName, configPath, envPath, dockerfilePaths,
		composefilePaths, dockerfileGlobs, composefileGlobs,
		dockerfileRecursive, composefileRecursive,
		dockerfileExcludeAll, composefileExcludeAll,
	)
	if err != nil {
		t.Fatal(err)
	}

	return flags
}
