// Package docker provides the "docker" command.
package docker

import (
	"github.com/spf13/cobra"
)

// NewDockerCmd creates the root command for docker-lock.
func NewDockerCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "docker",
		Short: "Root command for docker lock",
	}

	return rootCmd
}
