package main

import (
	"os"

	"lich-cli/internal/cli"
)

func main() {
	exitCode := cli.Execute(os.Args[1:])
	os.Exit(exitCode)
}
