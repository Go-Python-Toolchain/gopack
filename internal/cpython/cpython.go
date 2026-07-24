// Package cpython acquires a relocatable CPython runtime for a target platform.
// It uses the python-build-standalone project, which publishes ready to run,
// relocatable CPython builds. gopack downloads the build that matches the
// requested version and platform, caches it, and extracts it so the bundler can
// pack it into an application.
package cpython

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// pbsRepo is the GitHub repository that publishes the standalone builds.
const pbsRepo = "astral-sh/python-build-standalone"

// TripleFor returns the python-build-standalone target triple for a Go OS and
// architecture.
func TripleFor(goos, goarch string) (string, error) {
	switch goos {
	case "linux":
		switch goarch {
		case "amd64":
			return "x86_64-unknown-linux-gnu", nil
		case "arm64":
			return "aarch64-unknown-linux-gnu", nil
		}
	case "darwin":
		switch goarch {
		case "amd64":
			return "x86_64-apple-darwin", nil
		case "arm64":
			return "aarch64-apple-darwin", nil
		}
	case "windows":
		if goarch == "amd64" {
			return "x86_64-pc-windows-msvc", nil
		}
	}
	return "", fmt.Errorf("no CPython build available for %s/%s", goos, goarch)
}

// PythonExe returns the interpreter path inside an extracted runtime.
func PythonExe(runtimeDir, goos string) string {
	if goos == "windows" {
		return filepath.Join(runtimeDir, "python", "python.exe")
	}
	return filepath.Join(runtimeDir, "python", "bin", "python3")
}

// Runtime is an acquired CPython runtime.
type Runtime struct {
	Dir         string // extraction root, containing the python directory
	Exe         string // interpreter path
	FullVersion string // for example 3.12.8
	Triple      string
}

// Client acquires runtimes. The zero value is usable; it uses a default HTTP
// client and the standard cache directory.
type Client struct {
	HTTP     *http.Client
	CacheDir string
	Token    string // optional GitHub token to lift API rate limits
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 10 * time.Minute}
}

func (c *Client) cacheRoot() (string, error) {
	if c.CacheDir != "" {
		return c.CacheDir, nil
	}
	if base := os.Getenv("GOPACK_CACHE"); base != "" {
		return filepath.Join(base, "runtimes"), nil
	}
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "gopack", "runtimes"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "gopack", "runtimes"), nil
}

type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	// Digest is the asset's content digest as the index publishes it, for
	// example "sha256:abcd...". Empty on older releases that predate the field.
	Digest string `json:"digest"`
}

type ghRelease struct {
	TagName   string `json:"tag_name"`
	AssetsURL string `json:"assets_url"`
}

func (c *Client) get(ctx context.Context, url string, accept string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	return body, nil
}

// resolveAsset finds the best install_only asset for the version prefix and
// triple in the latest python-build-standalone release. It prefers the stripped
// variant, which omits the interpreter's debug symbols and is roughly a third of
// the size, and falls back to the full install_only build for a platform that
// does not publish a stripped one.
func (c *Client) resolveAsset(ctx context.Context, versionPrefix, triple string) (asset ghAsset, fullVersion string, err error) {
	body, err := c.get(ctx, "https://api.github.com/repos/"+pbsRepo+"/releases/latest", "application/vnd.github+json")
	if err != nil {
		return ghAsset{}, "", err
	}
	var rel ghRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return ghAsset{}, "", err
	}

	assets, err := c.listAssets(ctx, rel.AssetsURL)
	if err != nil {
		return ghAsset{}, "", err
	}

	re := assetRegexp(triple)
	bestVersion := ""
	bestStripped := false
	found := false
	for _, a := range assets {
		m := re.FindStringSubmatch(a.Name)
		if m == nil {
			continue
		}
		full, stripped := m[1], m[2] == "_stripped"
		if !versionMatchesPrefix(full, versionPrefix) {
			continue
		}
		// A newer patch version always wins; the stripped build is a size
		// optimization, not worth downgrading Python for. At the same version the
		// stripped build wins, which is the common case since a release publishes
		// both.
		if !found || betterAsset(full, stripped, bestVersion, bestStripped) {
			bestVersion, bestStripped = full, stripped
			asset, fullVersion, found = a, full, true
		}
	}
	if !found {
		return ghAsset{}, "", fmt.Errorf("no CPython %s build for %s in release %s", versionPrefix, triple, rel.TagName)
	}
	return asset, fullVersion, nil
}

// dirRegexp matches an extracted runtime's cache directory name, which is an
// asset name without the .tar.gz suffix, capturing the version and variant the
// same way assetRegexp does.
func dirRegexp(triple string) *regexp.Regexp {
	return regexp.MustCompile(`^cpython-(\d+\.\d+\.\d+)\+\d+-` + regexp.QuoteMeta(triple) + `-install_only(_stripped)?$`)
}

// findCached returns an already-extracted runtime that matches the version
// prefix and triple, if one is cached. It is what lets a build with a warm cache
// make no network request at all, and what makes an offline build possible once
// a runtime has been acquired. It prefers the same build resolveAsset would: the
// newest version, and the stripped variant at an equal version.
func (c *Client) findCached(versionPrefix, goos, triple string) (*Runtime, bool) {
	root, err := c.cacheRoot()
	if err != nil {
		return nil, false
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, false
	}

	re := dirRegexp(triple)
	var best *Runtime
	bestVersion := ""
	bestStripped := false
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m := re.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		full, stripped := m[1], m[2] == "_stripped"
		if !versionMatchesPrefix(full, versionPrefix) {
			continue
		}
		dir := filepath.Join(root, e.Name())
		if !isComplete(dir) {
			continue
		}
		if best == nil || betterAsset(full, stripped, bestVersion, bestStripped) {
			bestVersion, bestStripped = full, stripped
			best = &Runtime{Dir: dir, Exe: PythonExe(dir, goos), FullVersion: full, Triple: triple}
		}
	}
	if best == nil {
		return nil, false
	}
	return best, true
}

// betterAsset reports whether a candidate (version, stripped) should replace the
// current best. A higher version always wins; at an equal version the stripped
// build wins.
func betterAsset(version string, stripped bool, bestVersion string, bestStripped bool) bool {
	switch cmp := compareVersions(version, bestVersion); {
	case cmp > 0:
		return true
	case cmp < 0:
		return false
	default:
		return stripped && !bestStripped
	}
}

func (c *Client) listAssets(ctx context.Context, assetsURL string) ([]ghAsset, error) {
	var all []ghAsset
	for page := 1; page <= 20; page++ {
		url := fmt.Sprintf("%s?per_page=100&page=%d", assetsURL, page)
		body, err := c.get(ctx, url, "application/vnd.github+json")
		if err != nil {
			return nil, err
		}
		var batch []ghAsset
		if err := json.Unmarshal(body, &batch); err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < 100 {
			break
		}
	}
	return all, nil
}

// assetRegexp matches a python-build-standalone asset for the triple, capturing
// the version in group 1 and the variant suffix in group 2, which is empty for
// the full install_only build and "_stripped" for the stripped one.
func assetRegexp(triple string) *regexp.Regexp {
	return regexp.MustCompile(`^cpython-(\d+\.\d+\.\d+)\+\d+-` + regexp.QuoteMeta(triple) + `-install_only(_stripped)?\.tar\.gz$`)
}

func versionMatchesPrefix(full, prefix string) bool {
	return full == prefix || strings.HasPrefix(full, prefix+".")
}

func compareVersions(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) && i < len(bs); i++ {
		ai, _ := strconv.Atoi(as[i])
		bi, _ := strconv.Atoi(bs[i])
		if ai != bi {
			if ai < bi {
				return -1
			}
			return 1
		}
	}
	return len(as) - len(bs)
}

// Ensure returns the CPython runtime for the version, os, and arch, downloading
// and extracting it if it is not already cached. When verify is true it runs the
// interpreter to confirm the build works, which is only possible when building
// for the host platform.
//
// A cached runtime is used without contacting the index, so repeated builds make
// no network request and a build works offline once a runtime has been acquired.
// A freshly downloaded runtime is checked against the digest the index publishes
// before it is extracted, so a corrupted or tampered download is refused.
func (c *Client) Ensure(ctx context.Context, version, goos, goarch string, verify bool) (*Runtime, error) {
	triple, err := TripleFor(goos, goarch)
	if err != nil {
		return nil, err
	}

	if rt, ok := c.findCached(version, goos, triple); ok {
		if verify {
			if err := verifyInterpreter(rt.Exe, rt.FullVersion); err != nil {
				return nil, err
			}
		}
		return rt, nil
	}

	asset, full, err := c.resolveAsset(ctx, version, triple)
	if err != nil {
		return nil, err
	}

	root, err := c.cacheRoot()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(root, strings.TrimSuffix(asset.Name, ".tar.gz"))
	rt := &Runtime{Dir: dir, Exe: PythonExe(dir, goos), FullVersion: full, Triple: triple}

	if !isComplete(dir) {
		if err := c.downloadAndExtract(ctx, asset.URL, asset.Digest, dir); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(dir, ".gopack-complete"), nil, 0o644); err != nil {
			return nil, err
		}
	}

	if verify {
		if err := verifyInterpreter(rt.Exe, full); err != nil {
			return nil, err
		}
	}
	return rt, nil
}

func isComplete(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".gopack-complete"))
	return err == nil
}

func (c *Client) downloadAndExtract(ctx context.Context, url, digest, dir string) error {
	tmp, err := os.CreateTemp("", "gopack-cpython-*.tar.gz")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		tmp.Close()
		return err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		tmp.Close()
		return err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		tmp.Close()
		return fmt.Errorf("downloading runtime: status %d", resp.StatusCode)
	}
	// Hash the download as it is written, so the interpreter can be checked
	// against the digest the index published before anything is extracted.
	hasher := sha256.New()
	_, err = io.Copy(tmp, io.TeeReader(resp.Body, hasher))
	resp.Body.Close()
	tmp.Close()
	if err != nil {
		return err
	}
	if err := verifyDigest(digest, hasher.Sum(nil)); err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return extractTarGz(tmp.Name(), dir)
}

// verifyDigest checks a downloaded runtime against the digest the index
// published. A mismatch is refused. A digest gopack cannot check, because the
// release predates the field or uses an algorithm gopack does not implement, is
// not treated as a failure: there is nothing to check against, and refusing
// every such build would be worse than the status quo. Only sha256 is verified,
// which is what python-build-standalone publishes.
func verifyDigest(digest string, sum []byte) error {
	algo, want, ok := strings.Cut(digest, ":")
	if !ok || algo != "sha256" {
		return nil
	}
	got := hex.EncodeToString(sum)
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("runtime digest mismatch: the index published sha256:%s but the download hashed to sha256:%s; refusing it", want, got)
	}
	return nil
}

func verifyInterpreter(exe, full string) error {
	out, err := exec.Command(exe, "--version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("running %s: %w", exe, err)
	}
	if !strings.Contains(string(out), full) {
		return fmt.Errorf("interpreter reported %q, expected version %s", strings.TrimSpace(string(out)), full)
	}
	return nil
}

// extractTarGz unpacks a gzip-compressed tar into dir, preserving modes and
// symlinks and refusing entries that would escape dir.
func extractTarGz(src, dir string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target, err := safeJoin(dir, hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode).Perm())
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		}
	}
	return nil
}

func safeJoin(root, name string) (string, error) {
	clean := filepath.Clean("/" + strings.ReplaceAll(name, "\\", "/"))
	dest := filepath.Join(root, clean)
	if dest != root && !strings.HasPrefix(dest, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry %q escapes the destination", name)
	}
	return dest, nil
}
