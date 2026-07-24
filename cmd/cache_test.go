package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Go-Python-Toolchain/gopack/internal/bundle"
)

// writeFile creates a file with some bytes under dir.
func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

// scanCaches separates downloaded runtimes from extracted bundles and reports
// their sizes, which is what info and clear both rely on.
func TestScanCaches(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, bundle.RuntimesSubdir, "cpython-3.12.13-linux", "python", "bin", "python3"), 2048)
	writeFile(t, filepath.Join(root, "0b6c693bc5666917", "gopack.json"), 100)
	writeFile(t, filepath.Join(root, "0b6c693bc5666917", "app", "main.py"), 400)
	writeFile(t, filepath.Join(root, "9e3af725b9cb0c64", "gopack.json"), 100)

	runtimes, bundles, err := scanCaches(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(runtimes) != 1 || runtimes[0].size != 2048 {
		t.Fatalf("runtimes = %+v", runtimes)
	}
	if len(bundles) != 2 {
		t.Fatalf("bundles = %+v, want 2", bundles)
	}
	// Bundles are sorted by name, so the ordering is stable.
	if bundles[0].name != "0b6c693bc5666917" || bundles[0].size != 500 {
		t.Fatalf("first bundle = %+v", bundles[0])
	}
}

// A cache root that does not exist yet is an empty cache, not an error.
func TestScanCachesMissingRoot(t *testing.T) {
	runtimes, bundles, err := scanCaches(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatal(err)
	}
	if len(runtimes) != 0 || len(bundles) != 0 {
		t.Fatal("a missing cache should be empty")
	}
}

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{
		0:          "0 B",
		512:        "512 B",
		1024:       "1.0 KB",
		1048576:    "1.0 MB",
		1073741824: "1.0 GB",
	}
	for n, want := range cases {
		if got := humanBytes(n); got != want {
			t.Errorf("humanBytes(%d) = %q, want %q", n, got, want)
		}
	}
}
