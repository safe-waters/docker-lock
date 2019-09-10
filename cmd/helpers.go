package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

func handleError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func getDefaultConfigFile() string {
	if homeDir, err := os.UserHomeDir(); err == nil {
		cFile := filepath.Join(homeDir, ".docker", "config.json")
		if _, err := os.Stat(cFile); err != nil {
			return ""
		}
		return cFile
	}
	return ""
}
