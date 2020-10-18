// Package rewrite provides the "rewrite" command.
package rewrite

import (
	"fmt"
	"os"

	"github.com/safe-waters/docker-lock/pkg/rewrite"
	"github.com/safe-waters/docker-lock/pkg/rewrite/write"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const namespace = "rewrite"

// NewRewriteCmd creates the command 'rewrite' used in 'docker lock rewrite'.
func NewRewriteCmd() (*cobra.Command, error) {
	rewriteCmd := &cobra.Command{
		Use:   "rewrite",
		Short: "Rewrite files referenced by a Lockfile to use image digests",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return bindPFlags(cmd, []string{
				"lockfile-name",
				"tempdir",
				"exclude-tags",
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			flags, err := parseFlags()
			if err != nil {
				return err
			}

			rewriter, err := SetupRewriter(flags)
			if err != nil {
				return err
			}

			reader, err := os.Open(flags.LockfileName)
			if err != nil {
				return err
			}
			defer reader.Close()

			return rewriter.RewriteLockfile(reader)
		},
	}
	rewriteCmd.Flags().String(
		"lockfile-name", "docker-lock.json", "Lockfile to read from",
	)
	rewriteCmd.Flags().String(
		"tempdir", "",
		"Directory where a temporary directory will be created/deleted "+
			"during a rewrite transaction",
	)
	rewriteCmd.Flags().Bool(
		"exclude-tags", false, "Exclude image tags from rewritten files",
	)

	return rewriteCmd, nil
}

// SetupRewriter creates a Rewriter configured for docker-lock's cli.
func SetupRewriter(flags *Flags) (*rewrite.Rewriter, error) {
	dockerfileWriter := &write.DockerfileWriter{
		ExcludeTags: flags.ExcludeTags,
		Directory:   flags.TempDir,
	}

	composefileWriter := &write.ComposefileWriter{
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

func bindPFlags(cmd *cobra.Command, flagNames []string) error {
	for _, name := range flagNames {
		if err := viper.BindPFlag(
			fmt.Sprintf("%s.%s", namespace, name), cmd.Flags().Lookup(name),
		); err != nil {
			return err
		}
	}

	return nil
}

// parseFlags gets values from the command and uses them to
// create Flags.
func parseFlags() (*Flags, error) {
	lockfileName := viper.GetString(
		fmt.Sprintf("%s.%s", namespace, "lockfile-name"),
	)
	tempDir := viper.GetString(
		fmt.Sprintf("%s.%s", namespace, "tempdir"),
	)
	excludeTags := viper.GetBool(
		fmt.Sprintf("%s.%s", namespace, "exclude-tags"),
	)

	return NewFlags(lockfileName, tempDir, excludeTags)
}
