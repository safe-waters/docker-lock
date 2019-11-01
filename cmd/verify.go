package cmd

import (
	"github.com/joho/godotenv"
	"github.com/michaelperel/docker-lock/registry"
	"github.com/michaelperel/docker-lock/verify"
	"github.com/spf13/cobra"
)

func NewVerifyCmd() *cobra.Command {
	verifyCmd := &cobra.Command{
		Use:   "verify",
		Short: "Verifies that base images in Dockerfiles and docker-compose files refer to the same images as in the Lockfile.",
		Long: `After generating a Lockfile with "docker lock generate", running "docker lock verify"
will verify that all base images in files referenced in the Lockfile exist in the Lockfile and have up-to-date digests.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			envFile, err := cmd.Flags().GetString("env-file")
			if err != nil {
				return err
			}
			godotenv.Load(envFile)
			verifier, err := verify.NewVerifier(cmd)
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
			if err := verifier.VerifyLockfile(wrapperManager); err != nil {
				return err
			}
			return nil
		},
	}
	verifyCmd.Flags().String("outpath", "docker-lock.json", "Path to load Lockfile.")
	verifyCmd.Flags().String("config-file", getDefaultConfigFile(), "Path to config file for auth credentials.")
	verifyCmd.Flags().String("env-file", ".env", "Path to .env file.")
	verifyCmd.Flags().Bool("dockerfile-env-build-args", false, "Use environment variables as build args for Dockerfiles.")
	return verifyCmd
}
