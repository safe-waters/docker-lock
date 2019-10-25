package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/michaelperel/docker-lock/cmd"
)

func main() {
	if os.Args[1] == "docker-cli-plugin-metadata" {
		m := map[string]string{
			"SchemaVersion":    "0.1.0",
			"Vendor":           "https://github.com/michaelperel/docker-lock",
			"Version":          "v0.1.0",
			"ShortDescription": "Manage lockfiles",
		}
		j, _ := json.Marshal(m)
		fmt.Println(string(j))
		os.Exit(0)
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
