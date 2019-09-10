package cmd

import (
	"fmt"
	"os"

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

func Execute() {
	rootCmd := NewRootCmd()
	lockCmd := NewLockCmd()
	generateCmd := NewGenerateCmd()
	verifyCmd := NewVerifyCmd()
	rewriteCmd := NewRewriteCmd()
	rootCmd.AddCommand(lockCmd)
	lockCmd.AddCommand(generateCmd)
	lockCmd.AddCommand(verifyCmd)
	lockCmd.AddCommand(rewriteCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
