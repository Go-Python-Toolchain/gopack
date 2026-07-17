package nativelibs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseLdd(t *testing.T) {
	out := `	linux-vdso.so.1 (0x00007ffd)
	libpython3.12.so.1.0 => /bundle/site-packages/pkg/../pkg.libs/libpython3.12.so.1.0 (0x00007f00)
	libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6 (0x00007f01)
	libgfortran.so.5 => /opt/lib/libgfortran.so.5 (0x00007f02)
	libmissing.so => not found
	/lib64/ld-linux-x86-64.so.2 (0x00007f03)
`
	deps := ParseLdd(out)
	byName := map[string]Dependency{}
	for _, d := range deps {
		byName[d.Name] = d
	}
	if got := byName["libc.so.6"].Path; got != "/lib/x86_64-linux-gnu/libc.so.6" {
		t.Errorf("libc path = %q", got)
	}
	if got := byName["libgfortran.so.5"].Path; got != "/opt/lib/libgfortran.so.5" {
		t.Errorf("libgfortran path = %q", got)
	}
	if got := byName["libmissing.so"].Path; got != "" {
		t.Errorf("missing lib should have no path, got %q", got)
	}
}

func TestParseOtool(t *testing.T) {
	out := "thing.dylib:\n" +
		"\t/usr/lib/libSystem.B.dylib (compatibility version 1.0.0)\n" +
		"\t@rpath/libfoo.dylib (compatibility version 1.0.0)\n"
	deps := ParseOtool(out)
	if len(deps) != 2 {
		t.Fatalf("expected 2 deps, got %d: %v", len(deps), deps)
	}
	if deps[0].Path != "/usr/lib/libSystem.B.dylib" {
		t.Errorf("first dep path = %q", deps[0].Path)
	}
}

func TestIsSystemPath(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("system path cases are written for linux")
	}
	if !IsSystemPath("/lib/x86_64-linux-gnu/libc.so.6") {
		t.Error("libc should be a system path")
	}
	if !IsSystemPath("/usr/lib/libm.so.6") {
		t.Error("/usr/lib should be system")
	}
	if IsSystemPath("/opt/lib/libgfortran.so.5") {
		t.Error("/opt/lib should not be system")
	}
	if !IsSystemPath("") {
		t.Error("empty path counts as system")
	}
}

func TestFindSharedObjects(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"ext.so", "lib.so.1", "mod.pyd", "thing.dylib", "notes.txt", "script.py"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	objs, err := FindSharedObjects(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(objs) != 4 {
		t.Fatalf("expected 4 shared objects, got %d: %v", len(objs), objs)
	}
}

func TestScanCategorizesExternal(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("uses linux system paths")
	}
	dir := t.TempDir()
	so := filepath.Join(dir, "ext.so")
	if err := os.WriteFile(so, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A vendored library that lives inside the scanned directory.
	vendored := filepath.Join(dir, "vendor", "libvendored.so")
	os.MkdirAll(filepath.Dir(vendored), 0o755)
	os.WriteFile(vendored, []byte("x"), 0o644)

	depFn := func(string) ([]Dependency, error) {
		return []Dependency{
			{Name: "libc.so.6", Path: "/lib/x86_64-linux-gnu/libc.so.6"},  // system
			{Name: "libvendored.so", Path: vendored},                      // in bundle
			{Name: "libgfortran.so.5", Path: "/opt/lib/libgfortran.so.5"}, // external
			{Name: "libmissing.so", Path: ""},                             // unresolved
		}, nil
	}

	report, err := scanWith(dir, depFn)
	if err != nil {
		t.Fatal(err)
	}
	// Both ext.so and the vendored library under dir are shared objects.
	if report.SharedObjects != 2 {
		t.Fatalf("expected 2 shared objects, got %d", report.SharedObjects)
	}
	if len(report.External) != 1 || report.External[0].Name != "libgfortran.so.5" {
		t.Fatalf("expected only libgfortran to be external, got %v", report.External)
	}
}

func TestEmbed(t *testing.T) {
	srcDir := t.TempDir()
	lib := filepath.Join(srcDir, "libgfortran.so.5")
	if err := os.WriteFile(lib, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "libs")
	if err := Embed([]ExternalLib{{Name: "libgfortran.so.5", Path: lib}}, dest); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dest, "libgfortran.so.5"))
	if err != nil || string(data) != "binary" {
		t.Fatalf("embedded library not copied correctly: %v %q", err, data)
	}
}
