package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/michaelperel/docker-lock/generate"
	"github.com/michaelperel/docker-lock/registry"
	"github.com/michaelperel/docker-lock/verify"
)

type metadata struct {
	SchemaVersion    string
	Vendor           string
	Version          string
	ShortDescription string
}

func main() {
	if len(os.Args) <= 1 {
		handleError(errors.New("Expected 'lock' subcommand."))
	}
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

func getMetadata() (string, error) {
	m := metadata{
		SchemaVersion:    "0.1.0",
		Vendor:           "https://github.com/michaelperel/docker-lock",
		Version:          "v0.1.0",
		ShortDescription: "Generate and validate lock files for Docker",
	}
	var jsonData []byte
	jsonData, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

func handleError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
