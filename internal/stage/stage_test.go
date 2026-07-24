package stage

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
)

func TestPipInstallArgs(t *testing.T) {
	got := PipInstallArgs("/site", []string{"six"}, "reqs.txt")
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "install --target /site") {
		t.Fatalf("missing target: %q", joined)
	}
	if !strings.Contains(joined, "-r reqs.txt") {
		t.Fatalf("missing requirements file: %q", joined)
	}
	if !strings.HasSuffix(joined, "six") {
		t.Fatalf("missing requirement: %q", joined)
	}
	// pip must not byte-compile: it stamps each .pyc with the source mtime, which
	// would make bundles non-reproducible. Compilation is done separately.
	if !strings.Contains(joined, "--no-compile") {
		t.Fatalf("pip should be told not to compile: %q", joined)
	}
}

// The byte-compile step has to be deterministic, or bundles would differ between
// builds. Both properties that make it so are checked here.
func TestCompileArgs(t *testing.T) {
	got := CompileArgs("/stage/site-packages")
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "-m compileall") {
		t.Fatalf("not a compileall command: %q", joined)
	}
	// Hash-based invalidation records a source hash, not a modification time.
	if !strings.Contains(joined, "--invalidation-mode unchecked-hash") {
		t.Fatalf("must use hash-based invalidation: %q", joined)
	}
	// Stripping the target makes the source path recorded in each .pyc relative,
	// so it does not carry the build's temporary directory.
	if !strings.Contains(joined, "-s /stage/site-packages") {
		t.Fatalf("must strip the target path: %q", joined)
	}
	if !strings.HasSuffix(joined, "/stage/site-packages") {
		t.Fatalf("must compile the target directory: %q", joined)
	}
}

func TestPythonRel(t *testing.T) {
	if pythonRel("linux") != "python/bin/python3" {
		t.Fatal("linux python path wrong")
	}
	if pythonRel("windows") != "python/python.exe" {
		t.Fatal("windows python path wrong")
	}
}

func TestBuildStagingNoDeps(t *testing.T) {
	appDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(appDir, "main.py"), []byte("print('hi')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A nested package to confirm the tree is copied.
	if err := os.MkdirAll(filepath.Join(appDir, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "pkg", "util.py"), []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	staging := t.TempDir()
	m, err := Build(Options{
		Name:     "demo",
		AppDir:   appDir,
		Entry:    "main.py",
		TargetOS: "linux",
	}, staging)
	if err != nil {
		t.Fatal(err)
	}

	if m.Entry[0] != "python/bin/python3" || m.Entry[1] != "app/main.py" {
		t.Fatalf("unexpected entry: %v", m.Entry)
	}
	if m.Env["PYTHONPATH"] != "${ROOT}/site-packages" {
		t.Fatalf("unexpected env: %v", m.Env)
	}
	for _, want := range []string{"app/main.py", "app/pkg/util.py", "site-packages"} {
		if _, err := os.Stat(filepath.Join(staging, filepath.FromSlash(want))); err != nil {
			t.Errorf("expected %s in staging: %v", want, err)
		}
	}
}

func TestBuildMissingEntry(t *testing.T) {
	appDir := t.TempDir()
	os.WriteFile(filepath.Join(appDir, "main.py"), []byte("x\n"), 0o644)
	if _, err := Build(Options{AppDir: appDir, Entry: "nope.py", TargetOS: "linux"}, t.TempDir()); err == nil {
		t.Fatal("expected an error for a missing entry script")
	}
}

// TestStageAndRunReal stages an app with a real dependency and runs it with a
// downloaded interpreter. Skipped unless GOPACK_NETWORK_TESTS=1.
func TestStageAndRunReal(t *testing.T) {
	if os.Getenv("GOPACK_NETWORK_TESTS") == "" {
		t.Skip("set GOPACK_NETWORK_TESTS=1 to run tests that download CPython and install packages")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	rt, err := (&cpython.Client{CacheDir: t.TempDir(), Token: os.Getenv("GITHUB_TOKEN")}).
		Ensure(ctx, "3.12", runtime.GOOS, runtime.GOARCH, true)
	if err != nil {
		t.Fatal(err)
	}

	appDir := t.TempDir()
	app := "import six\nprint('six version', six.__version__)\n"
	if err := os.WriteFile(filepath.Join(appDir, "main.py"), []byte(app), 0o644); err != nil {
		t.Fatal(err)
	}

	staging := t.TempDir()
	m, err := Build(Options{
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

	// Run the staged app with the runtime, as the launcher will.
	cmd := exec.Command(rt.Exe, filepath.Join(staging, "app", "main.py"))
	cmd.Env = append(os.Environ(), "PYTHONPATH="+filepath.Join(staging, "site-packages"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("running staged app: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "six version") {
		t.Fatalf("app did not import its dependency: %s", out)
	}
	t.Logf("staged app output: %s", strings.TrimSpace(string(out)))
	t.Logf("manifest entry: %v", m.Entry)
}
