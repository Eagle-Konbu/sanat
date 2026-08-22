package main

import (
	"os"

	"github.com/Eagle-Konbu/sanat/cmd"
)

var version = "1.0.0-alpha.1"

func main() {
	cmd.SetVersion(version)

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
