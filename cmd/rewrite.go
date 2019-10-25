package cmd

import (
	"github.com/michaelperel/docker-lock/rewrite"
	"github.com/spf13/cobra"
)

func NewRewriteCmd() *cobra.Command {
	rewriteCmd := &cobra.Command{
		Use:   "rewrite",
		Short: "Rewrites Dockerfiles and docker-compose files referenced in the Lockfile to use digests.",
		Long: `After generating a Lockfile with "docker lock generate", running "docker lock rewrite"
will rewrite all referenced base images to include the digests from the Lockfile.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rewriter, err := rewrite.NewRewriter(cmd)
			if err != nil {
				return err
			}
			if err := rewriter.Rewrite(); err != nil {
				return err
			}
			return nil
		},
	}
	rewriteCmd.Flags().String("outpath", "docker-lock.json", "Path to load Lockfile.")
	rewriteCmd.Flags().String("suffix", "", "String to append to rewritten Dockerfiles and docker-compose files.")
	rewriteCmd.Flags().String("tempdir", "", "Directory where temporary files will be written during the rewrite transaction.")
	return rewriteCmd
}
