package generate_test

import (
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/cmd/generate"
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
			BaseDir:    getAbsPath(t),
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
				ManualPaths: []string{getAbsPath(t)},
			},
			ShouldFail: true,
		},
		{
			Name:    "Globs Absolute Paths",
			BaseDir: "",
			Expected: &generate.FlagsWithSharedNames{
				Globs: []string{filepath.Join(
					getAbsPath(t), "**", "Dockerfile"),
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

			assertFlagsEqual(t, test.Expected, got)
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
				BaseDir: getAbsPath(t),
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
				ConfigPath:   filepath.Join("~", ".docker", "config.json"),
				EnvPath:      ".env",
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			got, err := generate.NewFlagsWithSharedValues(
				test.Expected.BaseDir, test.Expected.LockfileName,
				test.Expected.ConfigPath, test.Expected.EnvPath,
				test.Expected.IgnoreMissingDigests,
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

			assertFlagsEqual(t, test.Expected, got)
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
				DockerfileFlags:  &generate.FlagsWithSharedNames{},
				ComposefileFlags: &generate.FlagsWithSharedNames{},
			},
			ShouldFail: true,
		},
		{
			Name: "Dockerfile Absolute Paths",
			Expected: &generate.Flags{
				FlagsWithSharedValues: &generate.FlagsWithSharedValues{},
				DockerfileFlags: &generate.FlagsWithSharedNames{
					ManualPaths: []string{getAbsPath(t)},
				},
				ComposefileFlags: &generate.FlagsWithSharedNames{},
			},
			ShouldFail: true,
		},
		{
			Name: "Composefile Absolute Paths",
			Expected: &generate.Flags{
				FlagsWithSharedValues: &generate.FlagsWithSharedValues{},
				DockerfileFlags:       &generate.FlagsWithSharedNames{},
				ComposefileFlags: &generate.FlagsWithSharedNames{
					ManualPaths: []string{getAbsPath(t)},
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
					ConfigPath:   filepath.FromSlash("~/.docker/config.json"),
					EnvPath:      ".env",
				},
				DockerfileFlags: &generate.FlagsWithSharedNames{
					ManualPaths: []string{"Dockerfile"},
				},
				ComposefileFlags: &generate.FlagsWithSharedNames{
					ManualPaths: []string{"docker-compose.yml"},
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
				test.Expected.FlagsWithSharedValues.ConfigPath,
				test.Expected.FlagsWithSharedValues.EnvPath,
				test.Expected.FlagsWithSharedValues.IgnoreMissingDigests,
				test.Expected.DockerfileFlags.ManualPaths,
				test.Expected.ComposefileFlags.ManualPaths,
				test.Expected.DockerfileFlags.Globs,
				test.Expected.ComposefileFlags.Globs,
				test.Expected.DockerfileFlags.Recursive,
				test.Expected.ComposefileFlags.Recursive,
				test.Expected.DockerfileFlags.ExcludePaths,
				test.Expected.ComposefileFlags.ExcludePaths,
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

			assertFlagsEqual(t, test.Expected, got)
		})
	}
}
