package rewrite

import (
	"flag"
)

type Flags struct {
	Outfile string
	Postfix string
}

func NewFlags(cmdLineArgs []string) (*Flags, error) {
	var outfile string
	var postfix string
	command := flag.NewFlagSet("rewrite", flag.ExitOnError)
	command.StringVar(&outfile, "o", "docker-lock.json", "Path to read Lockfile from current directory.")
	command.StringVar(&postfix, "p", "", "String to append to Dockerfile names.")
	command.Parse(cmdLineArgs)
	return &Flags{Outfile: outfile, Postfix: postfix}, nil
}
