package rewrite

import (
	"flag"
)

type Flags struct {
	Outfile string
	Suffix  string
}

func NewFlags(cmdLineArgs []string) (*Flags, error) {
	var outfile string
	var suffix string
	command := flag.NewFlagSet("rewrite", flag.ExitOnError)
	command.StringVar(&outfile, "o", "docker-lock.json", "Path to read Lockfile from current directory.")
	command.StringVar(&suffix, "s", "", "String to append to rewritten Dockerfiles and Composefiles.")
	command.Parse(cmdLineArgs)
	return &Flags{Outfile: outfile, Suffix: suffix}, nil
}
