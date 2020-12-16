// Package rewrite provides the "rewrite" command.
package rewrite

import (
	"errors"
	"fmt"
	"os"

	"github.com/safe-waters/docker-lock/pkg/rewrite"
	"github.com/safe-waters/docker-lock/pkg/rewrite/preprocess"
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

			err = rewriter.RewriteLockfile(reader, flags.TempDir)
			if err == nil {
				fmt.Println(
					"successfully rewrote files referenced by lockfile!",
				)
			}

			return err
		},
	}
	rewriteCmd.Flags().String(
		"lockfile-name", "docker-lock.json", "Lockfile to read from",
	)
	rewriteCmd.Flags().String(
		"tempdir", ".",
		"Directory where a temporary directory will be created/deleted "+
			"during a rewrite transaction",
	)
	rewriteCmd.Flags().Bool(
		"exclude-tags", false, "Exclude image tags from rewritten files",
	)

	return rewriteCmd, nil
}

// SetupRewriter creates a Rewriter configured for docker-lock's cli.
func SetupRewriter(flags *Flags) (rewrite.IRewriter, error) {
	if flags == nil {
		return nil, errors.New("'flags' cannot be nil")
	}

	if _, err := os.Stat(flags.LockfileName); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(
				"lockfile '%s' does not exist", flags.LockfileName,
			)
		}

		return nil, err
	}

	dockerfileWriter := write.NewDockerfileWriter(flags.ExcludeTags)

	composefileWriter, err := write.NewComposefileWriter(
		dockerfileWriter, flags.ExcludeTags,
	)
	if err != nil {
		return nil, err
	}

	kubernetesfileWriter := write.NewKubernetesfileWriter(flags.ExcludeTags)

	writer, err := rewrite.NewWriter(
		dockerfileWriter, composefileWriter, kubernetesfileWriter,
	)
	if err != nil {
		return nil, err
	}

	renamer := rewrite.NewRenamer()

	composefilePreprocessor := preprocess.NewComposefilePreprocessor()

	preprocessor, err := rewrite.NewPreprocessor(composefilePreprocessor)
	if err != nil {
		return nil, err
	}

	return rewrite.NewRewriter(preprocessor, writer, renamer)
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

func parseFlags() (*Flags, error) {
	var (
		lockfileName = viper.GetString(
			fmt.Sprintf("%s.%s", namespace, "lockfile-name"),
		)
		tempDir = viper.GetString(
			fmt.Sprintf("%s.%s", namespace, "tempdir"),
		)
		excludeTags = viper.GetBool(
			fmt.Sprintf("%s.%s", namespace, "exclude-tags"),
		)
	)

	return NewFlags(lockfileName, tempDir, excludeTags)
}
