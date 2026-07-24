package cpython

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestTripleFor(t *testing.T) {
	cases := []struct {
		goos, goarch, want string
	}{
		{"linux", "amd64", "x86_64-unknown-linux-gnu"},
		{"linux", "arm64", "aarch64-unknown-linux-gnu"},
		{"darwin", "amd64", "x86_64-apple-darwin"},
		{"darwin", "arm64", "aarch64-apple-darwin"},
		{"windows", "amd64", "x86_64-pc-windows-msvc"},
	}
	for _, tc := range cases {
		got, err := TripleFor(tc.goos, tc.goarch)
		if err != nil || got != tc.want {
			t.Errorf("TripleFor(%s,%s) = %q, %v; want %q", tc.goos, tc.goarch, got, err, tc.want)
		}
	}
	if _, err := TripleFor("plan9", "amd64"); err == nil {
		t.Error("expected an error for an unsupported platform")
	}
}

func TestPythonExe(t *testing.T) {
	if got := PythonExe("/rt", "linux"); got != filepath.FromSlash("/rt/python/bin/python3") {
		t.Errorf("linux exe = %q", got)
	}
	if got := PythonExe("/rt", "windows"); got != filepath.FromSlash("/rt/python/python.exe") {
		t.Errorf("windows exe = %q", got)
	}
}

func TestAssetMatching(t *testing.T) {
	triple := "x86_64-unknown-linux-gnu"
	re := assetRegexp(triple)

	// Both the full and the stripped install_only builds match, and the variant
	// is captured so the resolver can prefer the smaller one.
	full := "cpython-3.12.8+20241206-x86_64-unknown-linux-gnu-install_only.tar.gz"
	if m := re.FindStringSubmatch(full); m == nil || m[1] != "3.12.8" || m[2] != "" {
		t.Fatalf("full build: expected version 3.12.8 and empty variant, got %v", m)
	}
	stripped := "cpython-3.12.8+20241206-x86_64-unknown-linux-gnu-install_only_stripped.tar.gz"
	if m := re.FindStringSubmatch(stripped); m == nil || m[1] != "3.12.8" || m[2] != "_stripped" {
		t.Fatalf("stripped build: expected version 3.12.8 and _stripped variant, got %v", m)
	}

	for _, bad := range []string{
		"cpython-3.12.8+20241206-x86_64-apple-darwin-install_only.tar.gz",                // wrong triple
		"cpython-3.12.8+20241206-x86_64-unknown-linux-gnu-debug-full.tar.zst",            // wrong kind
		"cpython-3.12.8+20241206-x86_64-unknown-linux-gnu-install_only_stripped.tar.zst", // wrong extension
	} {
		if re.MatchString(bad) {
			t.Errorf("should not match %q", bad)
		}
	}
}

// The stripped build is preferred at the same version, since it is a third of
// the size, but a newer patch version wins over an older stripped one, because a
// smaller bundle is not worth an older interpreter.
func TestBetterAsset(t *testing.T) {
	cases := []struct {
		name         string
		version      string
		stripped     bool
		bestVersion  string
		bestStripped bool
		want         bool
	}{
		{"stripped beats full at same version", "3.12.8", true, "3.12.8", false, true},
		{"full does not beat stripped at same version", "3.12.8", false, "3.12.8", true, false},
		{"newer full beats older stripped", "3.12.9", false, "3.12.8", true, true},
		{"older stripped does not beat newer full", "3.12.8", true, "3.12.9", false, false},
		{"same variant same version does not replace", "3.12.8", true, "3.12.8", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := betterAsset(tc.version, tc.stripped, tc.bestVersion, tc.bestStripped); got != tc.want {
				t.Errorf("betterAsset = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestVersionMatchesPrefix(t *testing.T) {
	if !versionMatchesPrefix("3.12.8", "3.12") {
		t.Error("3.12.8 should match prefix 3.12")
	}
	if !versionMatchesPrefix("3.12.8", "3.12.8") {
		t.Error("exact version should match")
	}
	if versionMatchesPrefix("3.121.0", "3.12") {
		t.Error("3.121.0 should not match prefix 3.12")
	}
	if versionMatchesPrefix("3.11.0", "3.12") {
		t.Error("3.11.0 should not match prefix 3.12")
	}
}

func TestCompareVersions(t *testing.T) {
	if compareVersions("3.12.8", "3.12.7") <= 0 {
		t.Error("3.12.8 should be greater than 3.12.7")
	}
	if compareVersions("3.12.10", "3.12.9") <= 0 {
		t.Error("3.12.10 should be greater than 3.12.9 (numeric, not lexical)")
	}
	if compareVersions("3.12.8", "3.12.8") != 0 {
		t.Error("equal versions should compare equal")
	}
}

// TestEnsureReal downloads a real CPython runtime and runs it. Skipped by
// default; enable with GOPACK_NETWORK_TESTS=1.
func TestEnsureReal(t *testing.T) {
	if os.Getenv("GOPACK_NETWORK_TESTS") == "" {
		t.Skip("set GOPACK_NETWORK_TESTS=1 to run tests that download CPython")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	c := &Client{CacheDir: t.TempDir(), Token: os.Getenv("GITHUB_TOKEN")}
	rt, err := c.Ensure(ctx, "3.12", runtime.GOOS, runtime.GOARCH, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("acquired CPython %s (%s) at %s", rt.FullVersion, rt.Triple, rt.Dir)

	out, err := exec.Command(rt.Exe, "-c", "import sys; print(sys.version)").CombinedOutput()
	if err != nil {
		t.Fatalf("running bundled interpreter: %v\n%s", err, out)
	}
	t.Logf("interpreter says: %s", out)
}
