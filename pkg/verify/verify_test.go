package verify_test

import (
	"os"
	"path/filepath"
	"testing"

	cmd_verify "github.com/safe-waters/docker-lock/cmd/verify"
	"github.com/safe-waters/docker-lock/pkg/generate/registry"
)

var dTestDir = filepath.Join("testdata", "docker")  // nolint: gochecknoglobals
var cTestDir = filepath.Join("testdata", "compose") // nolint: gochecknoglobals

func TestVerifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name       string
		Flags      *cmd_verify.Flags
		ShouldFail bool
	}{
		{
			Name: "Different Number of Images in Dockerfile",
			Flags: &cmd_verify.Flags{
				LockfilePath: filepath.Join(
					dTestDir, "diffnumimages", "docker-lock.json",
				),
				EnvPath: ".env",
			},
			ShouldFail: true,
		},
		{
			Name: "Different Digests in Dockerfile",
			Flags: &cmd_verify.Flags{
				LockfilePath: filepath.Join(
					dTestDir, "diffdigests", "docker-lock.json",
				),
				EnvPath: ".env",
			},
			ShouldFail: true,
		},
		{
			Name: "Different Number of Images in Composefile",
			Flags: &cmd_verify.Flags{
				LockfilePath: filepath.Join(
					cTestDir, "diffnumimages", "docker-lock.json",
				),
				EnvPath: ".env",
			},
			ShouldFail: true,
		},
		{
			Name: "Different Digests in Composefile",
			Flags: &cmd_verify.Flags{
				LockfilePath: filepath.Join(
					cTestDir, "diffdigests", "docker-lock.json",
				),
				EnvPath: ".env",
			},
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

			verifier, err := cmd_verify.SetupVerifier(client, test.Flags)
			if err != nil {
				t.Fatal(err)
			}

			reader, err := os.Open(test.Flags.LockfilePath)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()

			if err := verifier.VerifyLockfile(reader); err != nil &&
				!test.ShouldFail {
				t.Fatal(err)
			}
		})
	}
}
