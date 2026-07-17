package assemble

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Go-Python-Toolchain/gopack/internal/cpython"
	"github.com/Go-Python-Toolchain/gopack/internal/nativelibs"
	"github.com/Go-Python-Toolchain/gopack/internal/stage"
)

// TestBundleScientificPackages is the milestone acceptance test: it bundles
// NumPy, Pandas, and FastAPI, each of which relies on compiled extensions, into
// standalone binaries and runs them with no system Python. Skipped unless
// GOPACK_NETWORK_TESTS=1. It is heavy: each bundle downloads packages and writes
// a few hundred megabytes.
func TestBundleScientificPackages(t *testing.T) {
	if os.Getenv("GOPACK_NETWORK_TESTS") == "" {
		t.Skip("set GOPACK_NETWORK_TESTS=1 to run the heavy bundling acceptance test")
	}
	if runtime.GOOS == "windows" {
		t.Skip("this test runs the finished binaries on the host")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// One shared runtime for all three bundles.
	rt, err := (&cpython.Client{CacheDir: t.TempDir(), Token: os.Getenv("GITHUB_TOKEN")}).
		Ensure(ctx, "3.12", runtime.GOOS, runtime.GOARCH, true)
	if err != nil {
		t.Fatal(err)
	}

	// Build the gopack binary once; a bundle is gopack with a payload appended.
	launcherPath := filepath.Join(t.TempDir(), "launcher")
	build := exec.Command("go", "build", "-o", launcherPath, "github.com/Go-Python-Toolchain/gopack")
	build.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building gopack: %v\n%s", err, out)
	}
	launcher, err := os.ReadFile(launcherPath)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name    string
		require string
		app     string
		expect  string
	}{
		{
			name:    "numpy",
			require: "numpy",
			app:     "import numpy as np\nprint('numpy', np.__version__)\nprint('sum', int(np.arange(10).sum()))\n",
			expect:  "sum 45",
		},
		{
			name:    "pandas",
			require: "pandas",
			app:     "import pandas as pd\nprint('pandas', pd.__version__)\nprint('total', int(pd.DataFrame({'a': [1, 2, 3]})['a'].sum()))\n",
			expect:  "total 6",
		},
		{
			name:    "fastapi",
			require: "fastapi",
			app:     "import fastapi\nfrom fastapi import FastAPI\napp = FastAPI()\nprint('fastapi', fastapi.__version__)\nprint('app_ok', app is not None)\n",
			expect:  "app_ok True",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			appDir := t.TempDir()
			if err := os.WriteFile(filepath.Join(appDir, "main.py"), []byte(tc.app), 0o644); err != nil {
				t.Fatal(err)
			}

			staging := t.TempDir()
			manifest, err := stage.Build(stage.Options{
				Name:         tc.name,
				AppDir:       appDir,
				Entry:        "main.py",
				Requirements: []string{tc.require},
				PythonExe:    rt.Exe,
				TargetOS:     runtime.GOOS,
			}, staging)
			if err != nil {
				t.Fatalf("staging %s: %v", tc.name, err)
			}

			// Report the native library scan for the record.
			if report, err := nativelibs.Scan(filepath.Join(staging, "site-packages")); err == nil {
				t.Logf("%s: %d shared objects, %d external libraries", tc.name, report.SharedObjects, len(report.External))
				for _, lib := range report.External {
					t.Logf("  external: %s (%s)", lib.Name, lib.Path)
				}
				if len(report.External) > 0 {
					if err := nativelibs.Embed(report.External, filepath.Join(staging, "libs")); err != nil {
						t.Fatal(err)
					}
					manifest.Env["LD_LIBRARY_PATH"] = "${ROOT}/libs"
				}
			}

			outPath := filepath.Join(t.TempDir(), tc.name+"-app")
			if err := Assemble(launcher, manifest, staging, rt.Dir, outPath); err != nil {
				t.Fatalf("assembling %s: %v", tc.name, err)
			}
			if info, err := os.Stat(outPath); err == nil {
				t.Logf("%s bundle: %.0f MB", tc.name, float64(info.Size())/(1024*1024))
			}

			cmd := exec.Command(outPath)
			cmd.Env = []string{"PATH=/nonexistent", "GOPACK_CACHE=" + t.TempDir(), "HOME=" + t.TempDir()}
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("running %s bundle with no system Python: %v\n%s", tc.name, err, out)
			}
			if !strings.Contains(string(out), tc.expect) {
				t.Fatalf("%s bundle output missing %q:\n%s", tc.name, tc.expect, out)
			}
			t.Logf("%s bundle ran with no system Python:\n%s", tc.name, strings.TrimSpace(string(out)))
		})
	}
}
