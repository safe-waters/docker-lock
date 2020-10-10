package rewrite_test

import (
	"os"
	"path/filepath"
	"testing"

	cmd_rewrite "github.com/safe-waters/docker-lock/cmd/rewrite"
)

func TestRewriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name         string
		LockfilePath string
		ShouldFail   bool
	}{
		{
			Name: "Composefile Overrides Dockerfile",
			LockfilePath: filepath.Join(
				"testdata", "override_dockerfile", "docker-lock.json",
			),
		},
		{
			Name: "Duplicate Services Same Dockerfile Images",
			LockfilePath: filepath.Join(
				"testdata", "duplicate_svc_same_images", "docker-lock.json",
			),
		},
		{
			Name: "Different Composefiles Same Dockerfile Images",
			LockfilePath: filepath.Join(
				"testdata", "duplicate_files_same_images", "docker-lock.json",
			),
		},
		{
			Name: "Duplicate Services Different Dockerfile Images",
			LockfilePath: filepath.Join(
				"testdata", "duplicate_svc_diff_images", "docker-lock.json",
			),
			ShouldFail: true,
		},
		{
			Name: "Different Composefiles Different Dockerfile Images",
			LockfilePath: filepath.Join(
				"testdata", "duplicate_files_diff_images", "docker-lock.json",
			),
			ShouldFail: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.Name, func(t *testing.T) {
			tempDir := generateUUID(t)
			makeDir(t, tempDir)

			defer os.RemoveAll(tempDir)

			flags := &cmd_rewrite.Flags{
				TempDir:      tempDir,
				LockfilePath: test.LockfilePath,
			}

			rewriter, err := cmd_rewrite.SetupRewriter(flags)
			if err != nil {
				t.Fatal(err)
			}

			reader, err := os.Open(flags.LockfilePath)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()

			err = rewriter.RewriteLockfile(reader)

			if test.ShouldFail {
				if err == nil {
					t.Fatal("expected error but did not get one")
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
