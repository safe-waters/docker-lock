package generate_test

import (
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/cmd/generate"
	"github.com/safe-waters/docker-lock/internal/testutils"
)

func TestFlagsWithSharedNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name       string
		BaseDir    string
		Expected   *generate.FlagsWithSharedNames
		ShouldFail bool
	}{
		{
			Name:       "Absolute Path Base Dir",
			BaseDir:    testutils.GetAbsPath(t),
			Expected:   &generate.FlagsWithSharedNames{},
			ShouldFail: true,
		},
		{
			Name:       "Base Dir Outside CWD",
			BaseDir:    "..",
			Expected:   &generate.FlagsWithSharedNames{},
			ShouldFail: true,
		},
		{
			Name:       "Base Dir is a File",
			BaseDir:    "flags.go",
			Expected:   &generate.FlagsWithSharedNames{},
			ShouldFail: true,
		},
		{
			Name:       "Base Dir does not exist",
			BaseDir:    "does-not-exist",
			Expected:   &generate.FlagsWithSharedNames{},
			ShouldFail: true,
		},
		{
			Name:    "Paths Outside CWD",
			BaseDir: "",
			Expected: &generate.FlagsWithSharedNames{
				ManualPaths: []string{filepath.Join("..", "Dockerfile")},
			},
			ShouldFail: true,
		},
		{
			Name:    "Absolute Paths",
			BaseDir: "",
			Expected: &generate.FlagsWithSharedNames{
				ManualPaths: []string{testutils.GetAbsPath(t)},
			},
			ShouldFail: true,
		},
		{
			Name:    "Globs Absolute Paths",
			BaseDir: "",
			Expected: &generate.FlagsWithSharedNames{
				Globs: []string{filepath.Join(
					testutils.GetAbsPath(t), "**", "Dockerfile"),
				},
			},
			ShouldFail: true,
		},
		{
			Name:    "Normal",
			BaseDir: ".",
			Expected: &generate.FlagsWithSharedNames{
				ManualPaths: []string{"Dockerfile",
					filepath.Join("some-path", "Dockerfile-one"),
				},
				Globs:     []string{filepath.Join("**", "Dockerfile")},
				Recursive: true,
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			got, err := generate.NewFlagsWithSharedNames(
				test.BaseDir, test.Expected.ManualPaths,
				test.Expected.Globs, test.Expected.Recursive,
				test.Expected.ExcludePaths,
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

func TestFlagsWithSharedValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name       string
		Expected   *generate.FlagsWithSharedValues
		ShouldFail bool
	}{
		{
			Name: "Absolute Path Base Dir",
			Expected: &generate.FlagsWithSharedValues{
				BaseDir: testutils.GetAbsPath(t),
			},
			ShouldFail: true,
		},
		{
			Name: "Base Dir Outside CWD",
			Expected: &generate.FlagsWithSharedValues{
				BaseDir: "..",
			},
			ShouldFail: true,
		},
		{
			Name: "Base Dir is a File",
			Expected: &generate.FlagsWithSharedValues{
				BaseDir: "flags.go",
			},
			ShouldFail: true,
		},
		{
			Name: "Base Dir does not exist",
			Expected: &generate.FlagsWithSharedValues{
				BaseDir: "does-not-exist",
			},
			ShouldFail: true,
		},
		{
			Name: "Normal",
			Expected: &generate.FlagsWithSharedValues{
				BaseDir:      ".",
				LockfileName: "docker-lock.json",
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			got, err := generate.NewFlagsWithSharedValues(
				test.Expected.BaseDir, test.Expected.LockfileName,
				test.Expected.IgnoreMissingDigests,
				test.Expected.UpdateExistingDigests,
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

func TestFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name       string
		Expected   *generate.Flags
		ShouldFail bool
	}{
		{
			Name: "Lockfile With Slashes",
			Expected: &generate.Flags{
				FlagsWithSharedValues: &generate.FlagsWithSharedValues{
					LockfileName: filepath.FromSlash(
						"my/lockfile/docker-lock.json",
					),
				},
				DockerfileFlags:     &generate.FlagsWithSharedNames{},
				ComposefileFlags:    &generate.FlagsWithSharedNames{},
				KubernetesfileFlags: &generate.FlagsWithSharedNames{},
			},
			ShouldFail: true,
		},
		{
			Name: "Dockerfile Absolute Paths",
			Expected: &generate.Flags{
				FlagsWithSharedValues: &generate.FlagsWithSharedValues{},
				DockerfileFlags: &generate.FlagsWithSharedNames{
					ManualPaths: []string{testutils.GetAbsPath(t)},
				},
				ComposefileFlags:    &generate.FlagsWithSharedNames{},
				KubernetesfileFlags: &generate.FlagsWithSharedNames{},
			},
			ShouldFail: true,
		},
		{
			Name: "Composefile Absolute Paths",
			Expected: &generate.Flags{
				FlagsWithSharedValues: &generate.FlagsWithSharedValues{},
				DockerfileFlags:       &generate.FlagsWithSharedNames{},
				ComposefileFlags: &generate.FlagsWithSharedNames{
					ManualPaths: []string{testutils.GetAbsPath(t)},
				},
				KubernetesfileFlags: &generate.FlagsWithSharedNames{},
			},
			ShouldFail: true,
		},
		{
			Name: "Kubernetesfile Absolute Paths",
			Expected: &generate.Flags{
				FlagsWithSharedValues: &generate.FlagsWithSharedValues{},
				DockerfileFlags:       &generate.FlagsWithSharedNames{},
				ComposefileFlags:      &generate.FlagsWithSharedNames{},
				KubernetesfileFlags: &generate.FlagsWithSharedNames{
					ManualPaths: []string{testutils.GetAbsPath(t)},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Normal",
			Expected: &generate.Flags{
				FlagsWithSharedValues: &generate.FlagsWithSharedValues{
					BaseDir:      ".",
					LockfileName: "docker-lock.json",
				},
				DockerfileFlags: &generate.FlagsWithSharedNames{
					ManualPaths: []string{"Dockerfile"},
				},
				ComposefileFlags: &generate.FlagsWithSharedNames{
					ManualPaths: []string{"docker-compose.yml"},
				},
				KubernetesfileFlags: &generate.FlagsWithSharedNames{
					ManualPaths: []string{"pod.yaml"},
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			got, err := generate.NewFlags(
				test.Expected.FlagsWithSharedValues.BaseDir,
				test.Expected.FlagsWithSharedValues.LockfileName,
				test.Expected.FlagsWithSharedValues.IgnoreMissingDigests,
				test.Expected.FlagsWithSharedValues.UpdateExistingDigests,
				test.Expected.DockerfileFlags.ManualPaths,
				test.Expected.ComposefileFlags.ManualPaths,
				test.Expected.KubernetesfileFlags.ManualPaths,
				test.Expected.DockerfileFlags.Globs,
				test.Expected.ComposefileFlags.Globs,
				test.Expected.KubernetesfileFlags.Globs,
				test.Expected.DockerfileFlags.Recursive,
				test.Expected.ComposefileFlags.Recursive,
				test.Expected.KubernetesfileFlags.Recursive,
				test.Expected.DockerfileFlags.ExcludePaths,
				test.Expected.ComposefileFlags.ExcludePaths,
				test.Expected.KubernetesfileFlags.ExcludePaths,
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
