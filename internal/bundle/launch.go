package bundle

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// HasPayload reports whether the current executable has a bundle appended, which
// is how gopack knows to act as a launcher rather than as the command line tool.
func HasPayload() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	r, err := Open(exe)
	if err != nil {
		return false
	}
	_ = r.Close()
	return true
}

// Launch runs the bundled application: it extracts the payload appended to the
// current executable and execs the entry command, passing through the process
// arguments. It returns the process exit code.
func Launch() int {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gopack: cannot find own path:", err)
		return 1
	}

	root, manifest, err := Prepare(exe)
	if err != nil {
		if errors.Is(err, ErrNoBundle) {
			fmt.Fprintln(os.Stderr, "gopack: this executable has no bundled application")
			return 1
		}
		fmt.Fprintln(os.Stderr, "gopack: preparing bundle:", err)
		return 1
	}

	prog, args := ResolveEntry(root, manifest.Entry)
	args = append(args, os.Args[1:]...)

	cmd := exec.Command(prog, args...)
	cmd.Dir = root
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), ResolveEnv(root, manifest.Env)...)

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
