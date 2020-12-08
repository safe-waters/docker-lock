// Package verify provides the "verify" command.
package verify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	cmd_generate "github.com/safe-waters/docker-lock/cmd/generate"
	"github.com/safe-waters/docker-lock/pkg/kind"
	"github.com/safe-waters/docker-lock/pkg/verify"
	"github.com/safe-waters/docker-lock/pkg/verify/diff"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const namespace = "verify"

// NewVerifyCmd creates the command 'verify' used in 'docker lock verify'.
func NewVerifyCmd() (*cobra.Command, error) {
	verifyCmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify that a Lockfile is up-to-date",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return bindPFlags(cmd, []string{
				"lockfile-name",
				"ignore-missing-digests",
				"update-existing-digests",
				"exclude-tags",
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			flags, err := parseFlags()
			if err != nil {
				return err
			}

			verifier, err := SetupVerifier(flags)
			if err != nil {
				return err
			}

			reader, err := os.Open(flags.LockfileName)
			if err != nil {
				return err
			}
			defer reader.Close()

			err = verifier.VerifyLockfile(reader)
			if err == nil {
				fmt.Println("successfully verified lockfile!")
			}

			return err
		},
	}
	verifyCmd.Flags().String(
		"lockfile-name", "docker-lock.json", "Lockfile to read from",
	)
	verifyCmd.Flags().Bool(
		"ignore-missing-digests", false,
		"Do not fail if unable to find digests",
	)
	verifyCmd.Flags().Bool(
		"update-existing-digests", false,
		"Query registries for new digests even if they are hardcoded in files",
	)
	verifyCmd.Flags().Bool(
		"exclude-tags", false, "Exclude image tags from verification",
	)

	return verifyCmd, nil
}

// SetupVerifier creates a Verifier configured for docker-lock's cli.
func SetupVerifier(flags *Flags) (verify.IVerifier, error) {
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

	existingLByt, err := ioutil.ReadFile(flags.LockfileName)
	if err != nil {
		return nil, err
	}

	var existingLockfile map[kind.Kind]map[string][]interface{}
	if err = json.Unmarshal(existingLByt, &existingLockfile); err != nil {
		return nil, err
	}

	var (
		dockerfilePaths = make(
			[]string, len(existingLockfile[kind.Dockerfile]),
		)
		composefilePaths = make(
			[]string, len(existingLockfile[kind.Composefile]),
		)
		kubernetesfilePaths = make(
			[]string, len(existingLockfile[kind.Kubernetesfile]),
		)
		i, j, k int
	)

	for p := range existingLockfile[kind.Dockerfile] {
		dockerfilePaths[i] = p
		i++
	}

	for p := range existingLockfile[kind.Composefile] {
		composefilePaths[j] = p
		j++
	}

	for p := range existingLockfile[kind.Kubernetesfile] {
		kubernetesfilePaths[k] = p
		k++
	}

	generatorFlags, err := cmd_generate.NewFlags(
		".", "", flags.IgnoreMissingDigests, flags.UpdateExistingDigests,
		dockerfilePaths, composefilePaths, kubernetesfilePaths,
		nil, nil, nil, false, false, false,
		len(dockerfilePaths) == 0, len(composefilePaths) == 0,
		len(kubernetesfilePaths) == 0,
	)
	if err != nil {
		return nil, err
	}

	generator, err := cmd_generate.SetupGenerator(generatorFlags)
	if err != nil {
		return nil, err
	}

	var (
		dockerfileDifferentiator = diff.NewDockerfileDifferentiator(
			flags.ExcludeTags,
		)
		composefileDifferentiator = diff.NewComposefileDifferentiator(
			flags.ExcludeTags,
		)
		kubernetesfileDifferentiator = diff.NewKubernetesfileDifferentiator(
			flags.ExcludeTags,
		)
	)

	return verify.NewVerifier(
		generator, dockerfileDifferentiator, composefileDifferentiator,
		kubernetesfileDifferentiator,
	)
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
		ignoreMissingDigests = viper.GetBool(
			fmt.Sprintf("%s.%s", namespace, "ignore-missing-digests"),
		)
		updateExistingDigests = viper.GetBool(
			fmt.Sprintf("%s.%s", namespace, "update-existing-digests"),
		)
		excludeTags = viper.GetBool(
			fmt.Sprintf("%s.%s", namespace, "exclude-tags"),
		)
	)

	return NewFlags(
		lockfileName, ignoreMissingDigests,
		updateExistingDigests, excludeTags,
	)
}
