package verify

import (
	"flag"
	"github.com/joho/godotenv"
	"os"
	"path/filepath"
)

type Flags struct {
	Outfile    string
	ConfigFile string
	EnvFile    string
}

func NewFlags(cmdLineArgs []string) (*Flags, error) {
	var outfile string
	var configFile string
	var envFile string
	command := flag.NewFlagSet("verify", flag.ExitOnError)
	command.StringVar(&outfile, "o", "docker-lock.json", "Path to save Lockfile from current directory.")
	command.StringVar(&configFile, "c", "", "Path to config file for auth credentials.")
	command.StringVar(&envFile, "e", ".env", "Path to .env file.")
	command.Parse(cmdLineArgs)
	if _, err := os.Stat(envFile); err != nil {
		if envFile != ".env" {
			return nil, err
		}
	} else if err := godotenv.Load(envFile); err != nil {
		return nil, err
	}
	if configFile != "" {
		if _, err := os.Stat(configFile); err != nil {
			return nil, err
		}
	} else if homeDir, err := os.UserHomeDir(); err == nil {
		defaultConfig := filepath.Join(homeDir, ".docker", "config.json")
		if _, err := os.Stat(defaultConfig); err == nil {
			configFile = defaultConfig
		}
	}
	return &Flags{Outfile: outfile, ConfigFile: configFile, EnvFile: envFile}, nil
}
