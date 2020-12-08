// Package generate provides the "generate" command.
package generate

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const namespace = "generate"

// NewGenerateCmd creates the command 'generate' used in 'docker lock generate'.
func NewGenerateCmd() (*cobra.Command, error) {
	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a Lockfile to track image digests",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return bindPFlags(cmd, []string{
				"base-dir",
				"dockerfiles",
				"composefiles",
				"kubernetesfiles",
				"lockfile-name",
				"dockerfile-globs",
				"composefile-globs",
				"kubernetesfile-globs",
				"dockerfile-recursive",
				"composefile-recursive",
				"kubernetesfile-recursive",
				"exclude-all-dockerfiles",
				"exclude-all-composefiles",
				"exclude-all-kubernetesfiles",
				"ignore-missing-digests",
				"update-existing-digests",
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			flags, err := parseFlags()
			if err != nil {
				return err
			}

			generator, err := SetupGenerator(flags)
			if err != nil {
				return err
			}

			var lockfileByt bytes.Buffer

			err = generator.GenerateLockfile(&lockfileByt)
			if err != nil {
				return err
			}

			lockfileContents := lockfileByt.Bytes()

			if len(lockfileContents) == 0 {
				return errors.New("no images found")
			}

			writer, err := os.Create(
				flags.FlagsWithSharedValues.LockfileName,
			)
			if err != nil {
				return err
			}
			defer writer.Close()

			_, err = writer.Write(lockfileContents)
			if err == nil {
				fmt.Println("successfully generated lockfile!")
			}

			return err
		},
	}
	generateCmd.Flags().String(
		"base-dir", ".", "Top level directory to collect files from",
	)
	generateCmd.Flags().StringSlice(
		"dockerfiles", []string{}, "Paths to Dockerfiles",
	)
	generateCmd.Flags().StringSlice(
		"composefiles", []string{}, "Paths to docker-compose files",
	)
	generateCmd.Flags().StringSlice(
		"kubernetesfiles", []string{}, "Paths to kubernetes files",
	)
	generateCmd.Flags().String(
		"lockfile-name", "docker-lock.json",
		"Lockfile name to be output in the current working directory",
	)
	generateCmd.Flags().StringSlice(
		"dockerfile-globs", []string{}, "Glob pattern to select Dockerfiles",
	)
	generateCmd.Flags().StringSlice(
		"composefile-globs", []string{},
		"Glob pattern to select docker-compose files",
	)
	generateCmd.Flags().StringSlice(
		"kubernetesfile-globs", []string{},
		"Glob pattern to select kubernetes files",
	)
	generateCmd.Flags().Bool(
		"dockerfile-recursive", false, "Recursively collect Dockerfiles",
	)
	generateCmd.Flags().Bool(
		"composefile-recursive", false,
		"Recursively collect docker-compose files",
	)
	generateCmd.Flags().Bool(
		"kubernetesfile-recursive", false,
		"Recursively collect kubernetes files",
	)
	generateCmd.Flags().Bool(
		"exclude-all-dockerfiles", false,
		"Do not collect Dockerfiles unless referenced by docker-compose files",
	)
	generateCmd.Flags().Bool(
		"exclude-all-composefiles", false,
		"Do not collect docker-compose files",
	)
	generateCmd.Flags().Bool(
		"exclude-all-kubernetesfiles", false,
		"Do not collect kubernetes files",
	)
	generateCmd.Flags().Bool(
		"ignore-missing-digests", false,
		"Do not fail if unable to find digests",
	)
	generateCmd.Flags().Bool(
		"update-existing-digests", false,
		"Query registries for new digests even if they are hardcoded in files",
	)

	return generateCmd, nil
}

// SetupGenerator creates a Generator configured for docker-lock's cli.
func SetupGenerator(
	flags *Flags,
) (generate.IGenerator, error) {
	if err := ensureFlagsNotNil(flags); err != nil {
		return nil, err
	}

	collector, err := DefaultPathCollector(flags)
	if err != nil {
		return nil, err
	}

	parser, err := DefaultImageParser(flags)
	if err != nil {
		return nil, err
	}

	updater, err := DefaultImageDigestUpdater(flags)
	if err != nil {
		return nil, err
	}

	sorter, err := DefaultImageFormatter(flags)
	if err != nil {
		return nil, err
	}

	generator, err := generate.NewGenerator(collector, parser, updater, sorter)
	if err != nil {
		return nil, err
	}

	return generator, nil
}

func bindPFlags(cmd *cobra.Command, flagNames []string) error {
	for _, name := range flagNames {
		if err := viper.BindPFlag(
			fmt.Sprintf("%s.%s", namespace, name), cmd.Flags().Lookup(name),
		); err != nil {
			return err
		}
	}

	return nil
}

func parseFlags() (*Flags, error) {
	var (
		baseDir = viper.GetString(
			fmt.Sprintf("%s.%s", namespace, "base-dir"),
		)
		lockfileName = viper.GetString(
			fmt.Sprintf("%s.%s", namespace, "lockfile-name"),
		)
		dockerfilePaths = viper.GetStringSlice(
			fmt.Sprintf("%s.%s", namespace, "dockerfiles"),
		)
		composefilePaths = viper.GetStringSlice(
			fmt.Sprintf("%s.%s", namespace, "composefiles"),
		)
		kubernetesfilePaths = viper.GetStringSlice(
			fmt.Sprintf("%s.%s", namespace, "kubernetesfiles"),
		)
		dockerfileGlobs = viper.GetStringSlice(
			fmt.Sprintf("%s.%s", namespace, "dockerfile-globs"),
		)
		composefileGlobs = viper.GetStringSlice(
			fmt.Sprintf("%s.%s", namespace, "composefile-globs"),
		)
		kubernetesfileGlobs = viper.GetStringSlice(
			fmt.Sprintf("%s.%s", namespace, "kubernetesfile-globs"),
		)
		dockerfileRecursive = viper.GetBool(
			fmt.Sprintf("%s.%s", namespace, "dockerfile-recursive"),
		)
		composefileRecursive = viper.GetBool(
			fmt.Sprintf("%s.%s", namespace, "composefile-recursive"),
		)
		kubernetesfileRecursive = viper.GetBool(
			fmt.Sprintf("%s.%s", namespace, "kubernetesfile-recursive"),
		)
		dockerfileExcludeAll = viper.GetBool(
			fmt.Sprintf("%s.%s", namespace, "exclude-all-dockerfiles"),
		)
		composefileExcludeAll = viper.GetBool(
			fmt.Sprintf("%s.%s", namespace, "exclude-all-composefiles"),
		)
		kubernetesfileExcludeAll = viper.GetBool(
			fmt.Sprintf("%s.%s", namespace, "exclude-all-kubernetesfiles"),
		)
		ignoreMissingDigests = viper.GetBool(
			fmt.Sprintf("%s.%s", namespace, "ignore-missing-digests"),
		)
		updateExistingDigests = viper.GetBool(
			fmt.Sprintf("%s.%s", namespace, "update-existing-digests"),
		)
	)

	return NewFlags(
		baseDir, lockfileName, ignoreMissingDigests, updateExistingDigests,
		dockerfilePaths, composefilePaths, kubernetesfilePaths,
		dockerfileGlobs, composefileGlobs, kubernetesfileGlobs,
		dockerfileRecursive, composefileRecursive, kubernetesfileRecursive,
		dockerfileExcludeAll, composefileExcludeAll, kubernetesfileExcludeAll,
	)
}
