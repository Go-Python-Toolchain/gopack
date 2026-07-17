// Package assemble writes the finished bundle: it packs the run manifest, the
// staged application and its dependencies, and the CPython runtime into a zip,
// appends that zip to the target launcher, and writes the result as a single
// executable. The zip is streamed straight to the output file so large bundles
// do not have to be held in memory.
package assemble

import (
	"archive/zip"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Go-Python-Toolchain/gopack/internal/bundle"
)

// Assemble writes a finished bundle to outPath. launcher is the launcher binary
// for the target platform, manifest is the run manifest, stagingDir holds the
// app and site-packages trees, and runtimeDir holds the extracted CPython (its
// python subdirectory becomes the bundle's python directory).
func Assemble(launcher []byte, manifest *bundle.Manifest, stagingDir, runtimeDir, outPath string) error {
	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := out.Write(launcher); err != nil {
		return err
	}

	counter := &countWriter{w: out}
	zw := zip.NewWriter(counter)

	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := addBytes(zw, "gopack.json", manifestJSON, 0o644); err != nil {
		return err
	}
	if err := addTree(zw, filepath.Join(stagingDir, "app"), "app"); err != nil {
		return err
	}
	if err := addTree(zw, filepath.Join(stagingDir, "site-packages"), "site-packages"); err != nil {
		return err
	}
	// Embedded external libraries, if any were detected and copied in.
	if err := addTree(zw, filepath.Join(stagingDir, "libs"), "libs"); err != nil {
		return err
	}
	if err := addTree(zw, filepath.Join(runtimeDir, "python"), "python"); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}

	trailer := bundle.MakeTrailer(uint64(len(launcher)), uint64(counter.n))
	if _, err := out.Write(trailer); err != nil {
		return err
	}
	return out.Chmod(0o755)
}

// addTree adds the file tree under srcDir to the zip under prefix. Symlinks are
// followed and stored as regular files, so the extracted bundle needs no symlink
// support and works the same on every platform.
func addTree(zw *zip.Writer, srcDir, prefix string) error {
	root, err := os.Stat(srcDir)
	if err != nil || !root.IsDir() {
		return nil // nothing to add
	}
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		name := prefix + "/" + filepath.ToSlash(rel)

		// Resolve symlinks and directories through a stat that follows links.
		info, err := os.Stat(path)
		if err != nil {
			return nil // skip dangling links
		}
		if info.IsDir() {
			return nil // directories are implied by their entries
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return addBytes(zw, name, data, info.Mode())
	})
}

func addBytes(zw *zip.Writer, name string, data []byte, mode fs.FileMode) error {
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetMode(mode)
	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// countWriter counts the bytes written through it.
type countWriter struct {
	w io.Writer
	n int64
}

func (c *countWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}
