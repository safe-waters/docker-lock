package verify_test

import (
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/cmd/verify"
	"github.com/safe-waters/docker-lock/internal/testutils"
)

func TestFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name       string
		Expected   *verify.Flags
		ShouldFail bool
	}{
		{
			Name: "Lockfile Name With Slashes",
			Expected: &verify.Flags{
				LockfileName: filepath.Join("lockfile", "path"),
			},
			ShouldFail: true,
		},
		{
			Name: "Normal",
			Expected: &verify.Flags{
				LockfileName: "docker-lock.json",
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			got, err := verify.NewFlags(
				test.Expected.LockfileName,
				test.Expected.IgnoreMissingDigests,
				test.Expected.UpdateExistingDigests,
				test.Expected.ExcludeTags,
			)
			if test.ShouldFail {
				if err == nil {
					t.Fatal("expected error but did not get one")
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}

			testutils.AssertFlagsEqual(t, test.Expected, got)
		})
	}
}
