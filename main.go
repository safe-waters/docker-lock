package main

import (
	"errors"
	"fmt"
	"github.com/michaelperel/docker-lock/generate"
	"github.com/michaelperel/docker-lock/registry"
	"github.com/michaelperel/docker-lock/verify"
	"os"
)

func main() {
	// Boilerplate required by every cli-plugin to show up in the 'docker' command.
	if os.Args[1] == "docker-cli-plugin-metadata" {
		metadata, err := getMetadata()
		handleError(err)
		fmt.Println(metadata)
		os.Exit(0)
	}
	if len(os.Args) <= 2 {
		handleError(errors.New("Expected 'generate' or 'verify' subcommands."))
	}
	subCommandIndex := 2
	switch subCommand := os.Args[subCommandIndex]; subCommand {
	case "generate":
		flags, err := generate.NewFlags(os.Args[subCommandIndex+1:])
		handleError(err)
		generator, err := generate.NewGenerator(flags)
		handleError(err)
		defaultWrapper := &registry.DockerWrapper{ConfigFile: flags.ConfigFile}
		wrapperManager := registry.NewWrapperManager(defaultWrapper)
		wrappers := []registry.Wrapper{&registry.ElasticWrapper{}, &registry.MCRWrapper{}}
		wrapperManager.Add(wrappers...)
		handleError(generator.GenerateLockfile(wrapperManager))
	case "verify":
		flags, err := verify.NewFlags(os.Args[subCommandIndex+1:])
		handleError(err)
		verifier, err := verify.NewVerifier(flags)
		handleError(err)
		defaultWrapper := &registry.DockerWrapper{ConfigFile: flags.ConfigFile}
		wrapperManager := registry.NewWrapperManager(defaultWrapper)
		wrappers := []registry.Wrapper{&registry.ElasticWrapper{}, &registry.MCRWrapper{}}
		wrapperManager.Add(wrappers...)
		handleError(verifier.VerifyLockfile(wrapperManager))
	default:
		handleError(errors.New("Expected 'generate' or 'verify' subcommands."))
	}
}

func handleError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
