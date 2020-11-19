package generate_test

import (
	"testing"

	cmd_generate "github.com/safe-waters/docker-lock/cmd/generate"
)

func TestDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name       string
		Flags      *cmd_generate.Flags
		ShouldFail bool
	}{
		{
			Name:       "Nil Flags",
			Flags:      nil,
			ShouldFail: true,
		},
		{
			Name: "Nil DockerfileFlags",
			Flags: &cmd_generate.Flags{
				ComposefileFlags:      &cmd_generate.FlagsWithSharedNames{},
				KubernetesfileFlags:   &cmd_generate.FlagsWithSharedNames{},
				FlagsWithSharedValues: &cmd_generate.FlagsWithSharedValues{},
			},
			ShouldFail: true,
		},
		{
			Name: "Nil ComposefileFlags",
			Flags: &cmd_generate.Flags{
				DockerfileFlags:       &cmd_generate.FlagsWithSharedNames{},
				KubernetesfileFlags:   &cmd_generate.FlagsWithSharedNames{},
				FlagsWithSharedValues: &cmd_generate.FlagsWithSharedValues{},
			},
			ShouldFail: true,
		},
		{
			Name: "Nil KubernetesfileFlags",
			Flags: &cmd_generate.Flags{
				DockerfileFlags:       &cmd_generate.FlagsWithSharedNames{},
				ComposefileFlags:      &cmd_generate.FlagsWithSharedNames{},
				FlagsWithSharedValues: &cmd_generate.FlagsWithSharedValues{},
			},
			ShouldFail: true,
		},
		{
			Name: "Nil FlagsWithSharedValues",
			Flags: &cmd_generate.Flags{
				DockerfileFlags:     &cmd_generate.FlagsWithSharedNames{},
				ComposefileFlags:    &cmd_generate.FlagsWithSharedNames{},
				KubernetesfileFlags: &cmd_generate.FlagsWithSharedNames{},
			},
			ShouldFail: true,
		},
		{
			Name: "Normal",
			Flags: &cmd_generate.Flags{
				DockerfileFlags:       &cmd_generate.FlagsWithSharedNames{},
				ComposefileFlags:      &cmd_generate.FlagsWithSharedNames{},
				KubernetesfileFlags:   &cmd_generate.FlagsWithSharedNames{},
				FlagsWithSharedValues: &cmd_generate.FlagsWithSharedValues{},
			},
		},
		{
			Name: "Exclude Dockerfiles",
			Flags: &cmd_generate.Flags{
				DockerfileFlags: &cmd_generate.FlagsWithSharedNames{
					ExcludePaths: true,
				},
				ComposefileFlags:      &cmd_generate.FlagsWithSharedNames{},
				KubernetesfileFlags:   &cmd_generate.FlagsWithSharedNames{},
				FlagsWithSharedValues: &cmd_generate.FlagsWithSharedValues{},
			},
		},
		{
			Name: "Exclude Composefiles",
			Flags: &cmd_generate.Flags{
				DockerfileFlags: &cmd_generate.FlagsWithSharedNames{},
				ComposefileFlags: &cmd_generate.FlagsWithSharedNames{
					ExcludePaths: true,
				},
				KubernetesfileFlags:   &cmd_generate.FlagsWithSharedNames{},
				FlagsWithSharedValues: &cmd_generate.FlagsWithSharedValues{},
			},
		},
		{
			Name: "Exclude Kubernetesfiles",
			Flags: &cmd_generate.Flags{
				DockerfileFlags:  &cmd_generate.FlagsWithSharedNames{},
				ComposefileFlags: &cmd_generate.FlagsWithSharedNames{},
				KubernetesfileFlags: &cmd_generate.FlagsWithSharedNames{
					ExcludePaths: true,
				},
				FlagsWithSharedValues: &cmd_generate.FlagsWithSharedValues{},
			},
		},
		{
			Name: "Exclude Dockerfiles, Composefiles, And Kubernetesfiles",
			Flags: &cmd_generate.Flags{
				DockerfileFlags: &cmd_generate.FlagsWithSharedNames{
					ExcludePaths: true,
				},
				ComposefileFlags: &cmd_generate.FlagsWithSharedNames{
					ExcludePaths: true,
				},
				KubernetesfileFlags: &cmd_generate.FlagsWithSharedNames{
					ExcludePaths: true,
				},
				FlagsWithSharedValues: &cmd_generate.FlagsWithSharedValues{},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			assertPathCollector(t, test.Flags, test.ShouldFail)
			assertImageParser(t, test.Flags, test.ShouldFail)
			assertImageDigestUpdater(t, test.Flags, test.ShouldFail)
		})
	}
}
