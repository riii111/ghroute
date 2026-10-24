package main

import (
	"os"

	"github.com/riii111/ghroute/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args, os.Stdout, os.Stderr))
}
