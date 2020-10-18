// Package lock provides the "generate" command.
package generate

import (
	"os"

	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/registry"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewGenerateCmd creates the command 'generate' used in 'docker lock generate'.
func NewGenerateCmd(client *registry.HTTPClient) (*cobra.Command, error) {
	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a Lockfile to track image digests",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags, err := parseFlags(cmd)
			if err != nil {
				return err
			}

			generator, err := SetupGenerator(client, flags)
			if err != nil {
				return err
			}

			writer, err := os.Create(
				flags.FlagsWithSharedValues.LockfileName,
			)
			if err != nil {
				return err
			}
			defer writer.Close()

			return generator.GenerateLockfile(writer)
		},
	}
	generateCmd.Flags().StringP(
		"base-dir", "b", ".", "Top level directory to collect files from",
	)
	generateCmd.Flags().StringSliceP(
		"dockerfiles", "d", []string{}, "Path to Dockerfiles",
	)
	generateCmd.Flags().StringSliceP(
		"composefiles", "c", []string{}, "Path to docker-compose files",
	)
	generateCmd.Flags().StringP(
		"lockfile-name", "l", "docker-lock.json",
		"Lockfile name to be output in the current working directory",
	)
	generateCmd.Flags().StringSlice(
		"dockerfile-globs", []string{}, "Glob pattern to select Dockerfiles",
	)
	generateCmd.Flags().StringSlice(
		"composefile-globs", []string{},
		"Glob pattern to select docker-compose files",
	)
	generateCmd.Flags().Bool(
		"dockerfile-recursive", false, "Recursively collect Dockerfiles",
	)
	generateCmd.Flags().Bool(
		"composefile-recursive", false,
		"Recursively collect docker-compose files",
	)
	generateCmd.Flags().String(
		"config-file", DefaultConfigPath(),
		"Path to config file for auth credentials",
	)
	generateCmd.Flags().StringP(
		"env-file", "e", ".env", "Path to .env file",
	)
	generateCmd.Flags().Bool(
		"exclude-all-dockerfiles", false,
		"Do not collect Dockerfiles unless referenced by docker-compose files",
	)
	generateCmd.Flags().Bool(
		"exclude-all-composefiles", false,
		"Do not collect docker-compose files",
	)

	if err := viper.BindPFlags(generateCmd.Flags()); err != nil {
		return nil, err
	}

	return generateCmd, nil
}

// SetupGenerator creates a Generator configured for docker-lock's cli.
func SetupGenerator(
	client *registry.HTTPClient,
	flags *Flags,
) (*generate.Generator, error) {
	if err := ensureFlagsNotNil(flags); err != nil {
		return nil, err
	}

	var err error

	if err = DefaultLoadEnv(flags.FlagsWithSharedValues.EnvPath); err != nil {
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

	updater, err := DefaultImageDigestUpdater(client, flags)
	if err != nil {
		return nil, err
	}

	generator, err := generate.NewGenerator(collector, parser, updater)
	if err != nil {
		return nil, err
	}

	return generator, nil
}

func parseFlags(cmd *cobra.Command) (*Flags, error) {
	baseDir := viper.GetString("base-dir")
	lockfileName := viper.GetString("lockfile-name")
	configPath := viper.GetString("config-file")
	envPath := viper.GetString("env-file")
	dockerfilePaths := viper.GetStringSlice("dockerfiles")
	composefilePaths := viper.GetStringSlice("composefiles")
	dockerfileGlobs := viper.GetStringSlice("dockerfile-globs")
	composefileGlobs := viper.GetStringSlice("composefile-globs")
	dockerfileRecursive := viper.GetBool("dockerfile-recursive")
	composefileRecursive := viper.GetBool("composefile-recursive")
	dockerfileExcludeAll := viper.GetBool("exclude-all-dockerfiles")
	composefileExcludeAll := viper.GetBool("exclude-all-composefiles")

	return NewFlags(
		baseDir, lockfileName, configPath, envPath,
		dockerfilePaths, composefilePaths, dockerfileGlobs, composefileGlobs,
		dockerfileRecursive, composefileRecursive,
		dockerfileExcludeAll, composefileExcludeAll,
	)
}
