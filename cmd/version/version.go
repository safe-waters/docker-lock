// Package version provides the "version" command.
package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the docker-lock version. In the CD pipeline,
// this value is overridden to match the git tag.
var Version = "v0.0.1" // nolint: gochecknoglobals

// NewVersionCmd creates the command 'version' used in 'docker lock version'.
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the docker-lock version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(Version)
			return nil
		},
	}
}
