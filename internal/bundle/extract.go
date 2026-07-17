package bundle

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// manifestName is the payload entry that holds the run manifest.
const manifestName = "gopack.json"

// completeMarker signals that a cache directory holds a fully extracted bundle.
const completeMarker = ".gopack-complete"

// Prepare finds the payload in exePath, extracts it to a content-addressed cache
// directory if it has not been extracted already, and returns the extraction
// root along with the run manifest.
func Prepare(exePath string) (root string, m *Manifest, err error) {
	r, err := Open(exePath)
	if err != nil {
		return "", nil, err
	}
	defer r.Close()

	key, err := payloadKey(r.Section)
	if err != nil {
		return "", nil, err
	}

	root, err = cacheDir(key)
	if err != nil {
		return "", nil, err
	}

	if !extracted(root) {
		if err := extractAll(r.Section, r.Size, root); err != nil {
			return "", nil, err
		}
		if err := os.WriteFile(filepath.Join(root, completeMarker), nil, 0o644); err != nil {
			return "", nil, err
		}
	}

	m, err = readManifest(root)
	if err != nil {
		return "", nil, err
	}
	return root, m, nil
}

// payloadKey returns a short content hash of the payload.
func payloadKey(section *io.SectionReader) (string, error) {
	if _, err := section.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	h := sha256.New()
	if _, err := io.Copy(h, section); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

func cacheDir(key string) (string, error) {
	base := os.Getenv("GOPACK_CACHE")
	if base == "" {
		if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
			base = filepath.Join(xdg, "gopack")
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, ".cache", "gopack")
		}
	}
	return filepath.Join(base, key), nil
}

func extracted(root string) bool {
	_, err := os.Stat(filepath.Join(root, completeMarker))
	return err == nil
}

// extractAll unpacks the zip payload into root, preserving file modes and
// guarding against paths that escape the destination.
func extractAll(section *io.SectionReader, size int64, root string) error {
	zr, err := zip.NewReader(section, size)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	for _, zf := range zr.File {
		dest, err := safeJoin(root, zf.Name)
		if err != nil {
			return err
		}
		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := extractFile(zf, dest); err != nil {
			return err
		}
	}
	return nil
}

func extractFile(zf *zip.File, dest string) error {
	rc, err := zf.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	mode := zf.Mode()
	if mode == 0 {
		mode = 0o644
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, rc); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// safeJoin joins root and name, refusing any name that would escape root.
func safeJoin(root, name string) (string, error) {
	clean := filepath.Clean("/" + strings.ReplaceAll(name, "\\", "/"))
	dest := filepath.Join(root, clean)
	if dest != root && !strings.HasPrefix(dest, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("payload entry %q escapes the extraction root", name)
	}
	return dest, nil
}

func readManifest(root string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(root, manifestName))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", manifestName, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", manifestName, err)
	}
	if len(m.Entry) == 0 {
		return nil, fmt.Errorf("%s has no entry command", manifestName)
	}
	return &m, nil
}
