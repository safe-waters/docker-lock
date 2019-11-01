package cmd

import (
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "docker",
		Short: "Root command for docker lock.",
		Long: `Root command for docker lock, referenced by docker when listing
	commands to the console.`,
	}
	return rootCmd
}

func Execute() error {
	rootCmd := NewRootCmd()
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	lockCmd := NewLockCmd()
	generateCmd := NewGenerateCmd()
	verifyCmd := NewVerifyCmd()
	rewriteCmd := NewRewriteCmd()
	rootCmd.AddCommand(lockCmd)
	lockCmd.AddCommand(generateCmd)
	lockCmd.AddCommand(verifyCmd)
	lockCmd.AddCommand(rewriteCmd)
	if err := rootCmd.Execute(); err != nil {
		return err
	}
	return nil
}
