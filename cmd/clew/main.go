package main

import (
	"fmt"
	"os"

	"github.com/adamancini/clew/internal/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := cmd.Execute(version, commit, date); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
