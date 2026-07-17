package assemble

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Go-Python-Toolchain/gopack/internal/bundle"
)

// TestAssembleAndRun builds a real launcher, assembles a bundle around a fake
// interpreter and app, and runs the finished single binary. The fake
// interpreter is a shell script, so the whole pipeline is exercised without
// downloading CPython.
func TestAssembleAndRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this test uses shell scripts")
	}

	// Build the launcher for this platform.
	launcherPath := filepath.Join(t.TempDir(), "launcher")
	build := exec.Command("go", "build", "-o", launcherPath, "github.com/Go-Python-Toolchain/gopack/launcher")
	build.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building launcher: %v\n%s", err, out)
	}
	launcher, err := os.ReadFile(launcherPath)
	if err != nil {
		t.Fatal(err)
	}

	// A fake runtime whose python runs its script argument with the shell and
	// reports the PYTHONPATH it was given.
	runtimeDir := t.TempDir()
	pyDir := filepath.Join(runtimeDir, "python", "bin")
	if err := os.MkdirAll(pyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	fakePython := "#!/bin/sh\necho \"python ran with PYTHONPATH=$PYTHONPATH\"\nexec /bin/sh \"$@\"\n"
	if err := os.WriteFile(filepath.Join(pyDir, "python3"), []byte(fakePython), 0o755); err != nil {
		t.Fatal(err)
	}

	// A staged app and a stand-in installed dependency.
	staging := t.TempDir()
	appDir := filepath.Join(staging, "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.py"), []byte("echo APP_RAN with args: $@\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	site := filepath.Join(staging, "site-packages", "mydep")
	if err := os.MkdirAll(site, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(site, "__init__.py"), []byte("v = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest := &bundle.Manifest{
		Name:  "demo",
		Entry: []string{"python/bin/python3", "app/main.py"},
		Env:   map[string]string{"PYTHONPATH": "${ROOT}/site-packages"},
	}

	outPath := filepath.Join(t.TempDir(), "myapp")
	if err := Assemble(launcher, manifest, staging, runtimeDir, outPath); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("assembled single binary: %d bytes", info.Size())

	// Run the finished binary with an isolated cache and pass an argument.
	cmd := exec.Command(outPath, "hello")
	cmd.Env = append(os.Environ(), "GOPACK_CACHE="+t.TempDir())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("running bundle: %v\n%s", err, out)
	}
	got := string(out)
	if !strings.Contains(got, "APP_RAN with args: hello") {
		t.Fatalf("app did not run with args: %q", got)
	}
	if !strings.Contains(got, "site-packages") {
		t.Fatalf("PYTHONPATH not set to site-packages: %q", got)
	}
	t.Logf("bundle output: %s", got)
}
