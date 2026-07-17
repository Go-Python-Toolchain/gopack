// Package nativelibs finds the external shared libraries that a bundle's C
// extensions depend on. It scans the staged packages for shared objects, asks
// the platform's linker tool which libraries they load, and separates the
// standard system libraries and libraries already inside the bundle from the
// external ones that would be missing on a bare target machine. Those external
// libraries can then be embedded into the bundle.
package nativelibs

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Dependency is one library that a shared object loads. Path is the resolved
// location, empty when the loader could not resolve it.
type Dependency struct {
	Name string
	Path string
}

// ExternalLib is a library that is neither a system library nor already inside
// the bundle, so it must be embedded for the bundle to run elsewhere.
type ExternalLib struct {
	Name string
	Path string
}

// Report summarizes a scan.
type Report struct {
	SharedObjects int
	External      []ExternalLib
}

// FindSharedObjects returns the shared object files under dir.
func FindSharedObjects(dir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if isSharedObject(path) {
			out = append(out, path)
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

func isSharedObject(path string) bool {
	base := filepath.Base(path)
	switch {
	case strings.HasSuffix(base, ".so"):
		return true
	case strings.Contains(base, ".so."):
		return true
	case strings.HasSuffix(base, ".pyd"):
		return true
	case strings.HasSuffix(base, ".dylib"):
		return true
	}
	return false
}

// Dependencies returns the libraries a shared object loads, using the platform's
// linker inspection tool.
func Dependencies(path string) ([]Dependency, error) {
	switch runtime.GOOS {
	case "linux":
		out, err := exec.Command("ldd", path).Output()
		if err != nil {
			// ldd exits non-zero for objects it cannot fully resolve, but still
			// prints useful lines, so parse whatever came out.
			if len(out) == 0 {
				return nil, err
			}
		}
		return ParseLdd(string(out)), nil
	case "darwin":
		out, err := exec.Command("otool", "-L", path).Output()
		if err != nil {
			return nil, err
		}
		return ParseOtool(string(out)), nil
	case "windows":
		out, err := exec.Command("dumpbin", "/dependents", path).Output()
		if err != nil {
			return nil, err
		}
		return ParseDumpbin(string(out)), nil
	}
	return nil, fmt.Errorf("dependency inspection is not supported on %s", runtime.GOOS)
}

// ParseLdd parses the output of ldd.
func ParseLdd(output string) []Dependency {
	var deps []Dependency
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if name, rest, ok := strings.Cut(line, "=>"); ok {
			name = strings.TrimSpace(name)
			rest = strings.TrimSpace(rest)
			if strings.HasPrefix(rest, "not found") {
				deps = append(deps, Dependency{Name: name})
				continue
			}
			deps = append(deps, Dependency{Name: name, Path: stripAddress(rest)})
			continue
		}
		// Lines without => are the dynamic linker or the vdso.
		field := strings.Fields(line)[0]
		if strings.HasPrefix(field, "/") {
			deps = append(deps, Dependency{Name: filepath.Base(field), Path: field})
		} else {
			deps = append(deps, Dependency{Name: field})
		}
	}
	return deps
}

// ParseOtool parses the output of otool -L.
func ParseOtool(output string) []Dependency {
	var deps []Dependency
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if i == 0 {
			continue // the first line echoes the inspected file
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		p := line
		if idx := strings.Index(p, " ("); idx >= 0 {
			p = p[:idx]
		}
		deps = append(deps, Dependency{Name: filepath.Base(p), Path: p})
	}
	return deps
}

// ParseDumpbin parses the dependents section of dumpbin output. It yields
// library names without paths, which is all Windows reports.
func ParseDumpbin(output string) []Dependency {
	var deps []Dependency
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(strings.ToLower(line), ".dll") && !strings.Contains(line, " ") {
			deps = append(deps, Dependency{Name: line})
		}
	}
	return deps
}

func stripAddress(s string) string {
	if idx := strings.Index(s, " (0x"); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return strings.TrimSpace(s)
}

// systemPrefixes lists the standard library directories per platform.
var systemPrefixes = map[string][]string{
	"linux":   {"/lib", "/lib64", "/usr/lib", "/usr/lib64"},
	"darwin":  {"/usr/lib", "/System/"},
	"windows": {"c:\\windows"},
}

// IsSystemPath reports whether a resolved path is a standard system library.
func IsSystemPath(path string) bool {
	if path == "" {
		return true
	}
	p := path
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	for _, prefix := range systemPrefixes[runtime.GOOS] {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

// Scan inspects every shared object under dir and returns the external libraries
// that are neither system libraries nor already inside dir.
func Scan(dir string) (*Report, error) {
	return scanWith(dir, Dependencies)
}

// scanWith is Scan with an injectable dependency resolver, which makes the
// categorization logic testable without the platform tool.
func scanWith(dir string, depFn func(string) ([]Dependency, error)) (*Report, error) {
	objects, err := FindSharedObjects(dir)
	if err != nil {
		return nil, err
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	report := &Report{SharedObjects: len(objects)}

	for _, obj := range objects {
		deps, err := depFn(obj)
		if err != nil {
			continue // skip objects the tool cannot read
		}
		for _, dep := range deps {
			if dep.Path == "" || IsSystemPath(dep.Path) {
				continue
			}
			absDep, err := filepath.Abs(dep.Path)
			if err == nil && strings.HasPrefix(absDep, absDir+string(os.PathSeparator)) {
				continue // already inside the bundle
			}
			if seen[dep.Path] {
				continue
			}
			seen[dep.Path] = true
			report.External = append(report.External, ExternalLib{Name: dep.Name, Path: dep.Path})
		}
	}
	sort.Slice(report.External, func(i, j int) bool { return report.External[i].Name < report.External[j].Name })
	return report, nil
}

// Embed copies the external libraries into destDir.
func Embed(externals []ExternalLib, destDir string) error {
	if len(externals) == 0 {
		return nil
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	for _, lib := range externals {
		if err := copyFile(lib.Path, filepath.Join(destDir, lib.Name)); err != nil {
			return fmt.Errorf("embedding %s: %w", lib.Name, err)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
