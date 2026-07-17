// Command launcher is a minimal stub that gopack can append a payload to. When
// run, it extracts the bundled application and runs it. gopack normally uses a
// copy of itself as the stub, so this standalone launcher is used mainly for
// testing the bundle mechanism.
package main

import (
	"os"

	"github.com/Go-Python-Toolchain/gopack/internal/bundle"
)

func main() {
	os.Exit(bundle.Launch())
}
