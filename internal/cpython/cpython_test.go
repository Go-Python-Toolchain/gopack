package cpython

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

func TestVerifyDigest(t *testing.T) {
	// sha256 of the bytes "hello".
	sum := sha256.Sum256([]byte("hello"))
	hexsum := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

	if err := verifyDigest("sha256:"+hexsum, sum[:]); err != nil {
		t.Errorf("a matching digest should pass: %v", err)
	}
	if err := verifyDigest("sha256:"+strings.ToUpper(hexsum), sum[:]); err != nil {
		t.Errorf("digest comparison should be case-insensitive: %v", err)
	}
	if err := verifyDigest("sha256:0000000000000000000000000000000000000000000000000000000000000000", sum[:]); err == nil {
		t.Error("a mismatching digest must be refused")
	}
	// Nothing to check against is not a failure.
	if err := verifyDigest("", sum[:]); err != nil {
		t.Errorf("an empty digest should be tolerated: %v", err)
	}
	if err := verifyDigest("md5:whatever", sum[:]); err != nil {
		t.Errorf("an unimplemented algorithm should be tolerated: %v", err)
	}
}

// completeRuntime creates a fake extracted runtime directory with the completion
// marker, so cache selection can be tested without a download.
func completeRuntime(t *testing.T, root, name string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Join(dir, "python", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gopack-complete"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

// A cached runtime is found and used without any network, which is what makes a
// warm-cache build make no API request and an offline build possible.
func TestFindCachedUsesTheCache(t *testing.T) {
	root := t.TempDir()
	triple := "x86_64-unknown-linux-gnu"
	completeRuntime(t, root, "cpython-3.12.13+20260718-"+triple+"-install_only_stripped")

	c := &Client{CacheDir: root}
	rt, ok := c.findCached("3.12", "linux", triple)
	if !ok {
		t.Fatal("the cached runtime should have been found")
	}
	if rt.FullVersion != "3.12.13" {
		t.Errorf("version = %q, want 3.12.13", rt.FullVersion)
	}
	if rt.Exe != filepath.Join(rt.Dir, "python", "bin", "python3") {
		t.Errorf("exe = %q", rt.Exe)
	}
}

// When both variants of the same version are cached, the stripped one is used,
// and a newer version wins over an older stripped one, matching resolveAsset.
func TestFindCachedPrefersStrippedAndNewest(t *testing.T) {
	root := t.TempDir()
	triple := "x86_64-unknown-linux-gnu"
	completeRuntime(t, root, "cpython-3.12.13+20260718-"+triple+"-install_only")
	completeRuntime(t, root, "cpython-3.12.13+20260718-"+triple+"-install_only_stripped")
	completeRuntime(t, root, "cpython-3.12.11+20260101-"+triple+"-install_only_stripped")

	c := &Client{CacheDir: root}
	rt, ok := c.findCached("3.12", "linux", triple)
	if !ok {
		t.Fatal("expected to find a cached runtime")
	}
	if rt.FullVersion != "3.12.13" {
		t.Errorf("version = %q, want the newest 3.12.13", rt.FullVersion)
	}
	if !strings.HasSuffix(rt.Dir, "install_only_stripped") {
		t.Errorf("dir = %q, want the stripped variant", rt.Dir)
	}
}

// A directory without the completion marker is a half-finished extraction and
// must not be used, or a build would run against a truncated runtime.
func TestFindCachedIgnoresIncompleteRuntimes(t *testing.T) {
	root := t.TempDir()
	triple := "x86_64-unknown-linux-gnu"
	dir := filepath.Join(root, "cpython-3.12.13+20260718-"+triple+"-install_only")
	if err := os.MkdirAll(dir, 0o755); err != nil { // no completion marker
		t.Fatal(err)
	}

	c := &Client{CacheDir: root}
	if _, ok := c.findCached("3.12", "linux", triple); ok {
		t.Error("an incomplete runtime should not be used")
	}
}

// A cached runtime for one platform must not answer for another.
func TestFindCachedRespectsTheTriple(t *testing.T) {
	root := t.TempDir()
	completeRuntime(t, root, "cpython-3.12.13+20260718-aarch64-apple-darwin-install_only_stripped")

	c := &Client{CacheDir: root}
	if _, ok := c.findCached("3.12", "linux", "x86_64-unknown-linux-gnu"); ok {
		t.Error("a darwin runtime should not satisfy a linux build")
	}
}

func TestDirRegexp(t *testing.T) {
	re := dirRegexp("x86_64-unknown-linux-gnu")
	m := re.FindStringSubmatch("cpython-3.12.13+20260718-x86_64-unknown-linux-gnu-install_only_stripped")
	if m == nil || m[1] != "3.12.13" || m[2] != "_stripped" {
		t.Fatalf("stripped dir: got %v", m)
	}
	m = re.FindStringSubmatch("cpython-3.12.13+20260718-x86_64-unknown-linux-gnu-install_only")
	if m == nil || m[1] != "3.12.13" || m[2] != "" {
		t.Fatalf("full dir: got %v", m)
	}
	// A .tar.gz name is an asset, not a directory, and must not match.
	if re.MatchString("cpython-3.12.13+20260718-x86_64-unknown-linux-gnu-install_only.tar.gz") {
		t.Error("an asset filename should not match the directory pattern")
	}
}

// A download whose bytes do not match the published digest is refused before it
// is extracted, so a corrupted or tampered runtime never reaches a bundle. This
// uses a local server, so it needs no network.
func TestDownloadRefusesADigestMismatch(t *testing.T) {
	payload := []byte("this is not really a runtime tarball")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(payload)
	}))
	defer srv.Close()

	c := &Client{}
	dir := filepath.Join(t.TempDir(), "rt")

	// A digest that does not match the served bytes must be rejected.
	wrong := "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	err := c.downloadAndExtract(context.Background(), srv.URL, wrong, dir)
	if err == nil {
		t.Fatal("a digest mismatch must be refused")
	}
	if !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("error = %v, want a digest mismatch", err)
	}
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Error("nothing should have been extracted after a rejected download")
	}

	// The matching digest gets past verification (extraction then fails because
	// the bytes are not a real tar.gz, which is a different, later error).
	sum := sha256.Sum256(payload)
	err = c.downloadAndExtract(context.Background(), srv.URL, "sha256:"+hex.EncodeToString(sum[:]), dir)
	if err != nil && strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("a correct digest must not be reported as a mismatch: %v", err)
	}
}
