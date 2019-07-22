package verify

import (
	"errors"
	"flag"
	"github.com/joho/godotenv"
	"os"
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
	if outfile == "" {
		return nil, errors.New("Outfile cannot be empty.")
	}
	fi, err := os.Stat(envFile)
	if err != nil && envFile != ".env" {
		return nil, err
	}
	if err == nil {
		if mode := fi.Mode(); mode.IsRegular() {
			if err := godotenv.Load(envFile); err != nil {
				return nil, err
			}
		}
	}
	if configFile != "" {
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			return nil, err
		}
	}
	if configFile == "" {
		defaultConfig := os.ExpandEnv("$HOME") + "/.docker/config.json"
		if _, err := os.Stat(defaultConfig); err == nil {
			configFile = defaultConfig
		}
	}
	return &Flags{Outfile: outfile, ConfigFile: configFile, EnvFile: envFile}, nil
}
