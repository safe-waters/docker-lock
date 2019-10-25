package cmd

import (
	"os"
	"path/filepath"
)

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
