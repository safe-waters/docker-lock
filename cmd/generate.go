package cmd

import (
	"errors"
	"os"

	"github.com/safe-waters/docker-lock/generate"
	"github.com/safe-waters/docker-lock/registry"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewGenerateCmd creates the command 'generate' used in 'docker lock generate'.
func NewGenerateCmd(client *registry.HTTPClient) (*cobra.Command, error) {
	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a Lockfile to track image digests",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags, err := generatorFlags(cmd)
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
		"config-file", defaultConfigPath(),
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
	flags *generate.Flags,
) (*generate.Generator, error) {
	if flags == nil {
		return nil, errors.New("flags cannot be nil")
	}

	var err error

	if err = loadEnv(flags.FlagsWithSharedValues.EnvPath); err != nil {
		return nil, err
	}

	var dockerfileCollector *generate.Collector

	if !flags.DockerfileFlags.ExcludePaths {
		dockerfileCollector, err = generate.DockerfileCollector(flags)
		if err != nil {
			return nil, err
		}
	}

	var composefileCollector *generate.Collector

	if !flags.ComposefileFlags.ExcludePaths {
		composefileCollector, err = generate.ComposefileCollector(flags)
		if err != nil {
			return nil, err
		}
	}

	wrapperManager, err := defaultWrapperManager(
		client, flags.FlagsWithSharedValues.ConfigPath,
	)
	if err != nil {
		return nil, err
	}

	updater, err := generate.NewUpdater(wrapperManager)
	if err != nil {
		return nil, err
	}

	dockerfileParser := &generate.DockerfileParser{}
	composefileParser := &generate.ComposefileParser{}

	generator, err := generate.NewGenerator(
		dockerfileCollector, composefileCollector,
		dockerfileParser, composefileParser, updater,
	)
	if err != nil {
		return nil, err
	}

	return generator, nil
}

func generatorFlags(cmd *cobra.Command) (*generate.Flags, error) {
	var (
		baseDir, lockfileName, configPath, envPath  string
		dockerfilePaths, composefilePaths           []string
		dockerfileGlobs, composefileGlobs           []string
		dockerfileRecursive, composefileRecursive   bool
		dockerfileExcludeAll, composefileExcludeAll bool
		err                                         error
	)

	switch viper.ConfigFileUsed() {
	case "":
		baseDir, err = cmd.Flags().GetString("base-dir")
		if err != nil {
			return nil, err
		}

		lockfileName, err = cmd.Flags().GetString("lockfile-name")
		if err != nil {
			return nil, err
		}

		configPath, err = cmd.Flags().GetString("config-file")
		if err != nil {
			return nil, err
		}

		envPath, err = cmd.Flags().GetString("env-file")
		if err != nil {
			return nil, err
		}

		dockerfilePaths, err = cmd.Flags().GetStringSlice("dockerfiles")
		if err != nil {
			return nil, err
		}

		composefilePaths, err = cmd.Flags().GetStringSlice("composefiles")
		if err != nil {
			return nil, err
		}

		dockerfileGlobs, err = cmd.Flags().GetStringSlice("dockerfile-globs")
		if err != nil {
			return nil, err
		}

		composefileGlobs, err = cmd.Flags().GetStringSlice("composefile-globs")
		if err != nil {
			return nil, err
		}

		dockerfileRecursive, err = cmd.Flags().GetBool("dockerfile-recursive")
		if err != nil {
			return nil, err
		}

		composefileRecursive, err = cmd.Flags().GetBool("composefile-recursive")
		if err != nil {
			return nil, err
		}

		dockerfileExcludeAll, err = cmd.Flags().GetBool(
			"exclude-all-dockerfiles",
		)
		if err != nil {
			return nil, err
		}

		composefileExcludeAll, err = cmd.Flags().GetBool(
			"exclude-all-composefiles",
		)
		if err != nil {
			return nil, err
		}

	default:
		baseDir = viper.GetString("base-dir")
		lockfileName = viper.GetString("lockfile-name")
		configPath = viper.GetString("config-file")
		envPath = viper.GetString("env-file")
		dockerfilePaths = viper.GetStringSlice("dockerfiles")
		composefilePaths = viper.GetStringSlice("composefiles")
		dockerfileGlobs = viper.GetStringSlice("dockerfile-globs")
		composefileGlobs = viper.GetStringSlice("composefile-globs")
		dockerfileRecursive = viper.GetBool("dockerfile-recursive")
		composefileRecursive = viper.GetBool("composefile-recursive")
		dockerfileExcludeAll = viper.GetBool("exclude-all-dockerfiles")
		composefileExcludeAll = viper.GetBool("exclude-all-composefiles")
	}

	return generate.NewFlags(
		baseDir, lockfileName, configPath, envPath,
		dockerfilePaths, composefilePaths, dockerfileGlobs, composefileGlobs,
		dockerfileRecursive, composefileRecursive,
		dockerfileExcludeAll, composefileExcludeAll,
	)
}
