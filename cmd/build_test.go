package cmd

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestDefaultOutputName(t *testing.T) {
	got := defaultOutputName("/home/user/myproject")
	want := "myproject"
	if runtime.GOOS == "windows" {
		want += ".exe"
	}
	if got != want {
		t.Fatalf("defaultOutputName = %q, want %q", got, want)
	}
}

func TestDefaultOutputNameCurrentDir(t *testing.T) {
	// A relative directory resolves to a real name, not the empty string.
	got := defaultOutputName(".")
	if got == "" || got == string(filepath.Separator) {
		t.Fatalf("unexpected default output name: %q", got)
	}
}

func TestLibraryPathVar(t *testing.T) {
	cases := map[string]string{
		"linux":   "LD_LIBRARY_PATH",
		"darwin":  "DYLD_LIBRARY_PATH",
		"windows": "PATH",
	}
	for goos, want := range cases {
		if got := libraryPathVar(goos); got != want {
			t.Errorf("libraryPathVar(%s) = %q, want %q", goos, got, want)
		}
	}
}

func TestTargetPythonExplicit(t *testing.T) {
	if got := targetPython("3.11"); got != "3.11" {
		t.Fatalf("targetPython(3.11) = %q", got)
	}
}
