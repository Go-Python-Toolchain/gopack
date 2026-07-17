// Command launcher is the small program that gopack appends a payload to. When
// run, it extracts the bundled application and interpreter to a cache and runs
// the entry command. It is not used on its own; gopack turns it into a finished
// executable by appending a payload.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/Go-Python-Toolchain/gopack/internal/bundle"
)

func main() {
	os.Exit(run())
}

func run() int {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gopack: cannot find own path:", err)
		return 1
	}

	root, manifest, err := bundle.Prepare(exe)
	if err != nil {
		if errors.Is(err, bundle.ErrNoBundle) {
			fmt.Fprintln(os.Stderr, "gopack: this executable has no bundled application")
			return 1
		}
		fmt.Fprintln(os.Stderr, "gopack: preparing bundle:", err)
		return 1
	}

	prog, args := bundle.ResolveEntry(root, manifest.Entry)
	args = append(args, os.Args[1:]...)

	cmd := exec.Command(prog, args...)
	cmd.Dir = root
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), bundle.ResolveEnv(root, manifest.Env)...)

	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode()
		}
		fmt.Fprintln(os.Stderr, "gopack: running application:", err)
		return 1
	}
	return 0
}
