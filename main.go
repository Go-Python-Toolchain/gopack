package main

import (
	"os"

	"github.com/Go-Python-Toolchain/gopack/cmd"
	"github.com/Go-Python-Toolchain/gopack/internal/bundle"
)

func main() {
	// When gopack has a bundle appended to it, it is a finished application and
	// acts as the launcher. Otherwise it is the command line tool.
	if bundle.HasPayload() {
		os.Exit(bundle.Launch())
	}
	cmd.Execute()
}
