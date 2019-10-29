package cmd

import (
	"github.com/joho/godotenv"
	"github.com/michaelperel/docker-lock/generate"
	"github.com/michaelperel/docker-lock/registry"
	"github.com/spf13/cobra"
)

func NewGenerateCmd() *cobra.Command {
	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generates a Lockfile.",
		Long: `"docker lock generate" generates a Lockfile that can be used with
	docker lock's 'verify' and 'rewrite' subcommands. The Lockfile contains image
	digests for all base images used by selected Dockerfiles and docker-compose
	files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			envFile, err := cmd.Flags().GetString("env-file")
			if err != nil {
				return err
			}
			godotenv.Load(envFile)
			generator, err := generate.NewGenerator(cmd)
			if err != nil {
				return err
			}
			configFile, err := cmd.Flags().GetString("config-file")
			if err != nil {
				return err
			}
			defaultWrapper, err := registry.NewDockerWrapper(configFile)
			if err != nil {
				return err
			}
			ACRWrapper, err := registry.NewACRWrapper(configFile)
			if err != nil {
				return err
			}
			wrapperManager := registry.NewWrapperManager(defaultWrapper)
			wrappers := []registry.Wrapper{&registry.ElasticWrapper{}, &registry.MCRWrapper{}, ACRWrapper}
			wrapperManager.Add(wrappers...)
			if err := generator.GenerateLockfile(wrapperManager); err != nil {
				return err
			}
			return nil
		},
	}
	generateCmd.Flags().String("base-dir", ".", "Top level directory to collect files from.")
	generateCmd.Flags().StringSlice("dockerfiles", []string{}, "Path to Dockerfiles.")
	generateCmd.Flags().StringSlice("compose-files", []string{}, "Path to docker-compose files.")
	generateCmd.Flags().StringSlice("dockerfile-globs", []string{}, "Glob pattern to select Dockerfiles.")
	generateCmd.Flags().StringSlice("compose-file-globs", []string{}, "Glob pattern to select docker-compose files.")
	generateCmd.Flags().Bool("dockerfile-recursive", false, "Recursively collect Dockerfiles.")
	generateCmd.Flags().Bool("compose-file-recursive", false, "Recursively collect docker-compose files.")
	generateCmd.Flags().String("outpath", "docker-lock.json", "Path to save Lockfile.")
	generateCmd.Flags().String("config-file", getDefaultConfigFile(), "Path to config file for auth credentials.")
	generateCmd.Flags().String("env-file", ".env", "Path to .env file.")
	generateCmd.Flags().Bool("dockerfile-env-build-args", false, "Use environment variables as build args for Dockerfiles.")
	return generateCmd
}
