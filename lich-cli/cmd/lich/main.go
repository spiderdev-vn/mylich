package main

import (
	"os"

	"github.com/spiderdev-vn/mylich/lich-cli/internal/cli"
)

func main() {
	exitCode := cli.Execute(os.Args[1:])
	os.Exit(exitCode)
}
