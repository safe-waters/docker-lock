package generate

import (
	"errors"
	"flag"
	"fmt"
	"github.com/joho/godotenv"
	"os"
)

type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return fmt.Sprintf("%v", []string(*s))
}

func (s *stringSliceFlag) Set(filePath string) error {
	*s = append(*s, filePath)
	return nil
}

type Flags struct {
	Dockerfiles      []string
	Composefiles     []string
	Globs            []string
	ComposeGlobs     []string
	Recursive        bool
	ComposeRecursive bool
	Outfile          string
	ConfigFile       string
	EnvFile          string
}

func NewFlags(cmdLineArgs []string) (*Flags, error) {
	var dockerfiles, composefiles stringSliceFlag
	var globs, composeGlobs stringSliceFlag
	var recursive, composeRecursive bool
	var outfile string
	var configFile string
	var envFile string
	command := flag.NewFlagSet("generate", flag.ExitOnError)
	command.Var(&dockerfiles, "f", "Path to Dockerfile from current directory.")
	command.Var(&composefiles, "cf", "Path to docker-compose file from current directory.")
	command.Var(&globs, "g", "Glob pattern to select Dockerfiles from current directory.")
	command.Var(&composeGlobs, "cg", "Glob pattern to select docker-compose files from current directory.")
	command.BoolVar(&recursive, "r", false, "recursively collect Dockerfiles from current directory.")
	command.BoolVar(&composeRecursive, "cr", false, "recursively collect docker-compose files from current directory.")
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
	return &Flags{Dockerfiles: []string(dockerfiles),
		Composefiles:     []string(composefiles),
		Globs:            []string(globs),
		ComposeGlobs:     []string(composeGlobs),
		Recursive:        recursive,
		ComposeRecursive: composeRecursive,
		Outfile:          outfile,
		ConfigFile:       configFile,
		EnvFile:          envFile,
	}, nil
}
