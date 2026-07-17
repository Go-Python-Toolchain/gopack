package assemble

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Go-Python-Toolchain/gopack/internal/cpython"
	"github.com/Go-Python-Toolchain/gopack/internal/stage"
)

// TestFullBundleReal builds a complete bundle around a real CPython runtime and
// a pip-installed dependency, then runs the finished single binary in a minimal
// environment to show it needs no system Python. Skipped unless
// GOPACK_NETWORK_TESTS=1.
func TestFullBundleReal(t *testing.T) {
	if os.Getenv("GOPACK_NETWORK_TESTS") == "" {
		t.Skip("set GOPACK_NETWORK_TESTS=1 to run tests that download CPython and install packages")
	}
	if runtime.GOOS == "windows" {
		t.Skip("this test runs the finished binary on the host")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	sharedCache := t.TempDir()
	rt, err := (&cpython.Client{CacheDir: sharedCache, Token: os.Getenv("GITHUB_TOKEN")}).
		Ensure(ctx, "3.12", runtime.GOOS, runtime.GOARCH, true)
	if err != nil {
		t.Fatal(err)
	}

	// An app that uses a third party dependency and the standard library.
	appDir := t.TempDir()
	app := "import sys, six\n" +
		"print('python', sys.version.split()[0])\n" +
		"print('six', six.__version__)\n" +
		"print('args', sys.argv[1:])\n"
	if err := os.WriteFile(filepath.Join(appDir, "main.py"), []byte(app), 0o644); err != nil {
		t.Fatal(err)
	}

	staging := t.TempDir()
	manifest, err := stage.Build(stage.Options{
		Name:         "demo",
		AppDir:       appDir,
		Entry:        "main.py",
		Requirements: []string{"six"},
		PythonExe:    rt.Exe,
		TargetOS:     runtime.GOOS,
	}, staging)
	if err != nil {
		t.Fatal(err)
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

	outPath := filepath.Join(t.TempDir(), "myapp")
	if err := Assemble(launcher, manifest, staging, rt.Dir, outPath); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(outPath)
	t.Logf("bundled single binary: %.1f MB", float64(info.Size())/(1024*1024))

	// Run it in a minimal environment with no Python on the PATH, to prove the
	// bundle is self-contained.
	runCache := t.TempDir()
	cmd := exec.Command(outPath, "one", "two")
	cmd.Env = []string{
		"PATH=/nonexistent",
		"GOPACK_CACHE=" + runCache,
		"HOME=" + t.TempDir(),
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("running the bundle with no system Python: %v\n%s", err, out)
	}
	got := string(out)
	for _, want := range []string{"python 3.12", "six 1.", "args ['one', 'two']"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got:\n%s", want, got)
		}
	}
	t.Logf("bundle output with no system Python:\n%s", strings.TrimSpace(got))
}
