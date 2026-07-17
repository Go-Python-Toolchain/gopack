package bundle

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAppendOpenRoundtrip(t *testing.T) {
	launcher := []byte("this is a pretend launcher binary")
	payload := []byte("this is the payload archive content")

	image := Append(launcher, payload)
	path := filepath.Join(t.TempDir(), "image.bin")
	if err := os.WriteFile(path, image, 0o644); err != nil {
		t.Fatal(err)
	}

	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	if r.Size != int64(len(payload)) {
		t.Fatalf("size = %d, want %d", r.Size, len(payload))
	}
	got, err := io.ReadAll(r.Section)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload round trip mismatch: got %q", got)
	}
}

func TestOpenNoBundle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plain.bin")
	if err := os.WriteFile(path, []byte("no trailer here"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path); err != ErrNoBundle {
		t.Fatalf("expected ErrNoBundle, got %v", err)
	}
}

func TestResolveEntry(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "app.py"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	exe, args := ResolveEntry(root, []string{"python/bin/python3", "app.py", "--flag"})
	if exe != filepath.Join(root, "python/bin/python3") {
		t.Fatalf("exe = %q", exe)
	}
	if args[0] != filepath.Join(root, "app.py") {
		t.Fatalf("first arg should resolve to root: %q", args[0])
	}
	if args[1] != "--flag" {
		t.Fatalf("literal arg should pass through: %q", args[1])
	}
}

func TestResolveEnv(t *testing.T) {
	env := ResolveEnv("/opt/app", map[string]string{
		"PYTHONHOME": "${ROOT}/python",
		"OTHER":      "value",
	})
	joined := strings.Join(env, " ")
	if !strings.Contains(joined, "PYTHONHOME=/opt/app/python") {
		t.Fatalf("ROOT not expanded: %v", env)
	}
	if !strings.Contains(joined, "OTHER=value") {
		t.Fatalf("plain value missing: %v", env)
	}
}

// makePayload builds a zip payload with a manifest and a small shell script.
func makePayload(t *testing.T, manifest Manifest, script string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	mf, _ := json.Marshal(manifest)
	w, err := zw.Create(manifestName)
	if err != nil {
		t.Fatal(err)
	}
	w.Write(mf)

	sh, err := zw.Create("run.sh")
	if err != nil {
		t.Fatal(err)
	}
	sh.Write([]byte(script))

	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestLauncherEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the end to end test uses a shell script")
	}

	// Build the launcher for this platform.
	launcherPath := filepath.Join(t.TempDir(), "launcher")
	build := exec.Command("go", "build", "-o", launcherPath, "github.com/Go-Python-Toolchain/gopack/launcher")
	build.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building launcher failed: %v\n%s", err, out)
	}
	launcherBytes, err := os.ReadFile(launcherPath)
	if err != nil {
		t.Fatal(err)
	}

	// A payload whose entry runs the bundled script and echoes a marker and its
	// own arguments.
	payload := makePayload(t,
		Manifest{Entry: []string{"/bin/sh", "run.sh"}},
		"echo BUNDLE_RAN arg=$1\n")

	image := Append(launcherBytes, payload)
	binPath := filepath.Join(t.TempDir(), "app")
	if err := os.WriteFile(binPath, image, 0o755); err != nil {
		t.Fatal(err)
	}

	// Use an isolated cache so the test is hermetic.
	cache := t.TempDir()

	runOnce := func() string {
		cmd := exec.Command(binPath, "hello")
		cmd.Env = append(os.Environ(), "GOPACK_CACHE="+cache)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("running bundle failed: %v\n%s", err, out)
		}
		return string(out)
	}

	first := runOnce()
	if !strings.Contains(first, "BUNDLE_RAN arg=hello") {
		t.Fatalf("unexpected output: %q", first)
	}

	// A second run reuses the extracted cache and still works.
	second := runOnce()
	if !strings.Contains(second, "BUNDLE_RAN arg=hello") {
		t.Fatalf("second run output: %q", second)
	}

	// With GOPACK_CACHE set, the cache holds exactly one extracted bundle keyed
	// by the payload hash.
	entries, err := os.ReadDir(cache)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		t.Fatalf("expected one cached bundle directory, found %v", entries)
	}
}
