package cmd

import (
	"fmt"

	"github.com/michaelperel/docker-lock/rewrite"
	"github.com/spf13/cobra"
)

func NewRewriteCmd() *cobra.Command {
	rewriteCmd := &cobra.Command{
		Use:   "rewrite",
		Short: "Rewrites Dockerfiles and docker-compose files referenced in the Lockfile to use digests.",
		Long: `After generating a Lockfile with "docker lock generate", running "docker lock rewrite"
will rewrite all referenced base images to include the digests from the Lockfile.`,
		Run: func(cmd *cobra.Command, args []string) {
			rewriter, err := rewrite.NewRewriter(cmd)
			if err != nil {
				err = fmt.Errorf("err creating rewrite command: %v", err)
				handleError(err)
			}
			err = rewriter.Rewrite()
			if err != nil {
				err = fmt.Errorf("err rewriting: %v", err)
			}
			handleError(err)
		},
	}
	rewriteCmd.Flags().String("outfile", "docker-lock.json", "Path to load Lockfile.")
	rewriteCmd.Flags().String("suffix", "", "String to append to rewritten Dockerfiles and docker-compose files.")
	return rewriteCmd
}
