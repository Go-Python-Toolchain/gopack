// Package stage builds the staging tree for a bundle: the application code and
// its dependencies, installed with pip into a target directory, together with
// the run manifest. The CPython runtime is added later, at assembly time, so the
// staging step stays fast and does not copy the interpreter.
package stage

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Go-Python-Toolchain/gopack/internal/bundle"
)

// Options describes what to stage.
type Options struct {
	Name             string   // bundle name
	AppDir           string   // directory holding the application source
	Entry            string   // entry script, relative to AppDir
	Requirements     []string // requirement strings to install
	RequirementsFile string   // path to a requirements file to install
	PythonExe        string   // interpreter used to run pip
	TargetOS         string   // the os the bundle will run on
}

// Build stages the application and its dependencies into stagingDir and returns
// the run manifest.
func Build(opts Options, stagingDir string) (*bundle.Manifest, error) {
	if opts.Entry == "" {
		return nil, fmt.Errorf("an entry script is required")
	}

	appDest := filepath.Join(stagingDir, "app")
	if err := copyTree(opts.AppDir, appDest); err != nil {
		return nil, fmt.Errorf("copying application: %w", err)
	}
	if _, err := os.Stat(filepath.Join(appDest, opts.Entry)); err != nil {
		return nil, fmt.Errorf("entry script %q not found in the application", opts.Entry)
	}

	site := filepath.Join(stagingDir, "site-packages")
	if err := os.MkdirAll(site, 0o755); err != nil {
		return nil, err
	}
	if len(opts.Requirements) > 0 || opts.RequirementsFile != "" {
		if err := pipInstall(opts.PythonExe, site, opts.Requirements, opts.RequirementsFile); err != nil {
			return nil, fmt.Errorf("installing dependencies: %w", err)
		}
		if err := compile(opts.PythonExe, site); err != nil {
			return nil, fmt.Errorf("byte-compiling dependencies: %w", err)
		}
	}

	manifest := &bundle.Manifest{
		Name:  opts.Name,
		Entry: []string{pythonRel(opts.TargetOS), filepath.ToSlash(filepath.Join("app", opts.Entry))},
		Env:   map[string]string{"PYTHONPATH": "${ROOT}/site-packages"},
	}
	return manifest, nil
}

// pythonRel is the interpreter path inside the bundle, relative to the root.
func pythonRel(goos string) string {
	if goos == "windows" {
		return "python/python.exe"
	}
	return "python/bin/python3"
}

// PipInstallArgs builds the pip command used to install into a target directory.
// It passes --no-compile so pip does not write byte-code caches; those are
// created afterwards, deterministically, by CompileArgs. pip's own compilation
// stamps each .pyc with the source file's modification time, which differs
// between builds and would make bundles non-reproducible.
func PipInstallArgs(target string, reqs []string, reqFile string) []string {
	args := []string{"-m", "pip", "install", "--target", target, "--no-compile", "--disable-pip-version-check", "--no-input"}
	if reqFile != "" {
		args = append(args, "-r", reqFile)
	}
	args = append(args, reqs...)
	return args
}

// CompileArgs builds the command that byte-compiles a directory reproducibly.
// Two choices make the output identical across builds. Hash-based invalidation
// (PEP 552) records a hash of the source rather than its modification time, so
// the .pyc does not depend on when the file was written. Stripping the target
// path makes the source path recorded inside each .pyc relative, so it does not
// depend on the build's temporary directory.
func CompileArgs(target string) []string {
	return []string{
		"-m", "compileall",
		"-q",                                    // quiet
		"-f",                                    // rewrite existing caches
		"--invalidation-mode", "unchecked-hash", // hash-based, not timestamp-based
		"-s", target, // strip this prefix from recorded source paths
		target,
	}
}

func pipInstall(pythonExe, target string, reqs []string, reqFile string) error {
	if pythonExe == "" {
		return fmt.Errorf("no interpreter available to run pip")
	}
	cmd := exec.Command(pythonExe, PipInstallArgs(target, reqs, reqFile)...)
	cmd.Stdout = os.Stderr // pip progress goes to stderr so command stdout stays clean
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// compile byte-compiles the installed dependencies so a bundle carries ready
// .pyc files, generated deterministically so two builds of the same inputs
// produce identical bundles.
func compile(pythonExe, target string) error {
	if pythonExe == "" {
		return fmt.Errorf("no interpreter available to byte-compile")
	}
	cmd := exec.Command(pythonExe, CompileArgs(target)...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Exclude removes staged files and directories matching any of the glob
// patterns, and reports how many entries and how many bytes were removed. It is
// how a build keeps chosen paths, such as test suites or documentation shipped
// inside a dependency, out of a bundle.
//
// A pattern with no slash is matched against an entry's base name, so `tests`
// removes any file or directory named tests at any depth and `*.pyi` removes
// every stub file. A pattern with a slash is matched against the entry's path
// relative to the staging root, so `site-packages/scipy/misc` removes exactly
// that directory. Matching a directory removes the whole subtree.
func Exclude(stagingDir string, patterns []string) (removed int, freed int64, err error) {
	if len(patterns) == 0 {
		return 0, 0, nil
	}
	for _, p := range patterns {
		if _, err := filepath.Match(p, "probe"); err != nil {
			return 0, 0, fmt.Errorf("invalid exclude pattern %q: %w", p, err)
		}
	}

	err = filepath.WalkDir(stagingDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == stagingDir {
			return nil
		}
		rel, relErr := filepath.Rel(stagingDir, path)
		if relErr != nil {
			return relErr
		}
		if !excludeMatch(patterns, filepath.ToSlash(rel), d.Name()) {
			return nil
		}

		size, sizeErr := entrySize(path, d)
		if sizeErr != nil {
			return sizeErr
		}
		if err := os.RemoveAll(path); err != nil {
			return err
		}
		removed++
		freed += size
		if d.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
	return removed, freed, err
}

// excludeMatch reports whether any pattern matches the entry. A pattern with a
// slash is matched against the whole relative path; one without is matched
// against the base name.
func excludeMatch(patterns []string, rel, base string) bool {
	for _, p := range patterns {
		target := base
		if strings.Contains(p, "/") {
			target = rel
		}
		if ok, _ := filepath.Match(p, target); ok {
			return true
		}
	}
	return false
}

// entrySize returns the size of a file, or the total size of a directory's
// regular files.
func entrySize(path string, d fs.DirEntry) (int64, error) {
	if !d.IsDir() {
		info, err := d.Info()
		if err != nil {
			return 0, err
		}
		return info.Size(), nil
	}
	var total int64
	err := filepath.WalkDir(path, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.Type().IsRegular() {
			info, err := e.Info()
			if err != nil {
				return err
			}
			total += info.Size()
		}
		return nil
	})
	return total, err
}

// copyTree copies the file tree at src into dst, preserving file modes and
// recreating symlinks.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.Type()&fs.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			return os.Symlink(link, target)
		}
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
