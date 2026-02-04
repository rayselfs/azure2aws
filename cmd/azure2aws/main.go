package main

import (
	"os"

	"github.com/user/azure2aws/internal/cmd"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	rootCmd := cmd.NewRootCmd(version, commit, buildDate)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
