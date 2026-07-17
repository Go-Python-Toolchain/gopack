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
func PipInstallArgs(target string, reqs []string, reqFile string) []string {
	args := []string{"-m", "pip", "install", "--target", target, "--disable-pip-version-check", "--no-input"}
	if reqFile != "" {
		args = append(args, "-r", reqFile)
	}
	args = append(args, reqs...)
	return args
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
