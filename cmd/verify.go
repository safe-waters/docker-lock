package cmd

import (
	"github.com/joho/godotenv"
	"github.com/michaelperel/docker-lock/registry"
	"github.com/michaelperel/docker-lock/verify"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verifies that base images in Dockerfiles and docker-compose files refer to the same images as in the Lockfile.",
	Long: `After generating a Lockfile with "docker lock generate", running "docker lock verify"
will verify that all base images in files referenced in the Lockfile exist in the Lockfile and have up-to-date digests.`,
	Run: func(cmd *cobra.Command, args []string) {
		envFile, err := cmd.Flags().GetString("env-file")
		handleError(err)
		godotenv.Load(envFile)
		verifier, err := verify.NewVerifier(cmd)
		handleError(err)
		configFile, err := cmd.Flags().GetString("config-file")
		handleError(err)
		defaultWrapper := &registry.DockerWrapper{ConfigFile: configFile}
		wrapperManager := registry.NewWrapperManager(defaultWrapper)
		wrappers := []registry.Wrapper{&registry.ElasticWrapper{}, &registry.MCRWrapper{}}
		wrapperManager.Add(wrappers...)
		handleError(verifier.VerifyLockfile(wrapperManager))
	},
}

func init() {
	lockCmd.AddCommand(verifyCmd)
	verifyCmd.Flags().String("outfile", "docker-lock.json", "Path to load Lockfile.")
	verifyCmd.Flags().String("config-file", getDefaultConfigFile(), "Path to config file for auth credentials.")
	verifyCmd.Flags().String("env-file", ".env", "Path to .env file.")
}
