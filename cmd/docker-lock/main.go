// Package main is a cli tool that automates managing image digests
// by tracking them in a separate Lockfile
// (think package-lock.json or Pipfile.lock) - with docker-lock,
// you can refer to images in Dockerfiles, docker-compose files,
// and Kubernetes manifests by mutable tags (as in python:3.6)
// yet receive the same benefits as if you had specified immutable
// digests (as in python:3.6@sha256:25a189a536ae4d...).
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/safe-waters/docker-lock/cmd/docker"
	"github.com/safe-waters/docker-lock/cmd/generate"
	"github.com/safe-waters/docker-lock/cmd/lock"
	"github.com/safe-waters/docker-lock/cmd/rewrite"
	"github.com/safe-waters/docker-lock/cmd/verify"
	"github.com/safe-waters/docker-lock/cmd/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "docker-cli-plugin-metadata" {
		m := map[string]string{
			"SchemaVersion":    "0.1.0",
			"Vendor":           "https://github.com/safe-waters/docker-lock",
			"Version":          version.Version,
			"ShortDescription": "Manage Lockfiles",
		}
		j, _ := json.Marshal(m)

		fmt.Println(string(j))

		os.Exit(0)
	}

	if err := execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}

func execute() error {
	if err := initViper(); err != nil {
		return err
	}

	dockerCmd := docker.NewDockerCmd()

	dockerCmd.SilenceUsage = true
	dockerCmd.SilenceErrors = true

	lockCmd := lock.NewLockCmd()
	versionCmd := version.NewVersionCmd()

	generateCmd, err := generate.NewGenerateCmd()
	if err != nil {
		return err
	}

	verifyCmd, err := verify.NewVerifyCmd()
	if err != nil {
		return err
	}

	rewriteCmd, err := rewrite.NewRewriteCmd()
	if err != nil {
		return err
	}

	dockerCmd.AddCommand(lockCmd)
	lockCmd.AddCommand(
		[]*cobra.Command{versionCmd, generateCmd, verifyCmd, rewriteCmd}...,
	)

	return dockerCmd.Execute()
}

func initViper() error {
	const cfgFilePrefix = ".docker-lock"

	// works with variety of files such as .docker-lock.[yaml|json|toml] etc.
	viper.SetConfigName(cfgFilePrefix)
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("malformed '%s' file: %v", cfgFilePrefix, err)
		}
	}

	return nil
}
