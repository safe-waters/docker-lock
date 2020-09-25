// Package rewrite provides the "rewrite" command.
package rewrite

import (
	"os"

	"github.com/safe-waters/docker-lock/rewrite"
	"github.com/safe-waters/docker-lock/rewrite/writers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewRewriteCmd creates the command 'rewrite' used in 'docker lock rewrite'.
func NewRewriteCmd() (*cobra.Command, error) {
	rewriteCmd := &cobra.Command{
		Use:   "rewrite",
		Short: "Rewrite files referenced by a Lockfile to use image digests",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags, err := parseFlags(cmd)
			if err != nil {
				return err
			}

			rewriter, err := SetupRewriter(flags)
			if err != nil {
				return err
			}

			reader, err := os.Open(flags.LockfilePath)
			if err != nil {
				return err
			}
			defer reader.Close()

			return rewriter.RewriteLockfile(reader)
		},
	}
	rewriteCmd.Flags().StringP(
		"lockfile-path", "l", "docker-lock.json", "Path to Lockfile",
	)
	rewriteCmd.Flags().StringP(
		"tempdir", "t", "",
		"Directory where a temporary directory will be created/deleted "+
			"during a rewrite transaction",
	)
	rewriteCmd.Flags().BoolP(
		"exclude-tags", "e", false, "Exclude image tags from rewritten files",
	)

	if err := viper.BindPFlags(rewriteCmd.Flags()); err != nil {
		return nil, err
	}

	return rewriteCmd, nil
}

// SetupRewriter creates a Rewriter configured for docker-lock's cli.
func SetupRewriter(flags *Flags) (*rewrite.Rewriter, error) {
	dockerfileWriter := &writers.DockerfileWriter{
		ExcludeTags: flags.ExcludeTags,
		Directory:   flags.TempDir,
	}

	composefileWriter := &writers.ComposefileWriter{
		DockerfileWriter: dockerfileWriter,
		ExcludeTags:      flags.ExcludeTags,
		Directory:        flags.TempDir,
	}

	writer, err := rewrite.NewWriter(dockerfileWriter, composefileWriter)
	if err != nil {
		return nil, err
	}

	renamer := &rewrite.Renamer{}

	return rewrite.NewRewriter(writer, renamer)
}

// parseFlags gets values from the command and uses them to
// create Flags.
func parseFlags(cmd *cobra.Command) (*Flags, error) {
	var (
		lockfilePath, tempDir string
		excludeTags           bool
		err                   error
	)

	switch viper.ConfigFileUsed() {
	case "":
		lockfilePath, err = cmd.Flags().GetString("lockfile-path")
		if err != nil {
			return nil, err
		}

		tempDir, err = cmd.Flags().GetString("tempdir")
		if err != nil {
			return nil, err
		}

		excludeTags, err = cmd.Flags().GetBool("exclude-tags")
		if err != nil {
			return nil, err
		}
	default:
		lockfilePath = viper.GetString("lockfile-path")
		tempDir = viper.GetString("tempdir")
		excludeTags = viper.GetBool("exclude-tags")
	}

	return &Flags{
		LockfilePath: lockfilePath,
		TempDir:      tempDir,
		ExcludeTags:  excludeTags,
	}, nil
}