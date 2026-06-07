package main

import (
	"os"

	"github.com/sean2077/jira-cli/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr))
}
