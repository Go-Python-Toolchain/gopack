package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/Go-Python-Toolchain/gopack/internal/assemble"
	"github.com/Go-Python-Toolchain/gopack/internal/cpython"
	"github.com/Go-Python-Toolchain/gopack/internal/nativelibs"
	"github.com/Go-Python-Toolchain/gopack/internal/stage"
	"github.com/spf13/cobra"
)

var buildOpts struct {
	python       string
	entry        string
	requirements string
	output       string
	noEmbed      bool
	exclude      []string
}

var buildCmd = &cobra.Command{
	Use:   "build [app-directory]",
	Short: "Bundle a Python app into a single self-contained executable",
	Long: `build packs a Python application, a CPython runtime, and the app's
dependencies into one executable that runs on a machine with no Python installed.

The application directory defaults to the current directory. Its dependencies are
read from a requirements file with -r. External native libraries that C
extensions need are detected and embedded so the result is self-contained.

The bundle targets the current operating system and architecture.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBuild,
}

func runBuild(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()

	appDir := "."
	if len(args) == 1 {
		appDir = args[0]
	}
	entry := buildOpts.entry
	output := buildOpts.output
	if output == "" {
		output = defaultOutputName(appDir)
	}
	py := targetPython(buildOpts.python)
	goos, goarch := runtime.GOOS, runtime.GOARCH

	// A bundle is the gopack binary itself with a payload appended: gopack runs as
	// the command line tool when it has no payload, and as the launcher when it
	// does. So the finished bundle is built from a copy of the running gopack.
	self, err := os.Executable()
	if err != nil {
		return err
	}
	runner, err := os.ReadFile(self)
	if err != nil {
		return err
	}

	ctx := context.Background()
	fmt.Fprintf(out, "acquiring CPython %s for %s/%s\n", py, goos, goarch)
	rt, err := (&cpython.Client{Token: githubToken()}).Ensure(ctx, py, goos, goarch, true)
	if err != nil {
		return err
	}

	staging, err := os.MkdirTemp("", "gopack-staging-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)

	fmt.Fprintln(out, "staging application and dependencies")
	manifest, err := stage.Build(stage.Options{
		Name:             filepath.Base(output),
		AppDir:           appDir,
		Entry:            entry,
		RequirementsFile: buildOpts.requirements,
		PythonExe:        rt.Exe,
		TargetOS:         goos,
	}, staging)
	if err != nil {
		return err
	}

	if len(buildOpts.exclude) > 0 {
		removed, freed, err := stage.Exclude(staging, buildOpts.exclude)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "excluded %d entrie(s), %.1f MB\n", removed, float64(freed)/(1024*1024))
	}

	report, err := nativelibs.Scan(filepath.Join(staging, "site-packages"))
	if err == nil && len(report.External) > 0 {
		fmt.Fprintf(out, "found %d external native librar(y/ies):\n", len(report.External))
		for _, lib := range report.External {
			fmt.Fprintf(out, "  %s (%s)\n", lib.Name, lib.Path)
		}
		if buildOpts.noEmbed {
			fmt.Fprintln(out, "not embedding them (--no-embed); the target must provide them")
		} else {
			if err := nativelibs.Embed(report.External, filepath.Join(staging, "libs")); err != nil {
				return err
			}
			manifest.Env[libraryPathVar(goos)] = "${ROOT}/libs"
			fmt.Fprintln(out, "embedded them into the bundle")
		}
	}

	fmt.Fprintln(out, "assembling")
	if err := assemble.Assemble(runner, manifest, staging, rt.Dir, output); err != nil {
		return err
	}

	if info, err := os.Stat(output); err == nil {
		fmt.Fprintf(out, "wrote %s (%.1f MB)\n", output, float64(info.Size())/(1024*1024))
	}
	return nil
}

func defaultOutputName(appDir string) string {
	abs, err := filepath.Abs(appDir)
	if err != nil || filepath.Base(abs) == "" || filepath.Base(abs) == string(filepath.Separator) {
		return "app"
	}
	name := filepath.Base(abs)
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func libraryPathVar(goos string) string {
	switch goos {
	case "darwin":
		return "DYLD_LIBRARY_PATH"
	case "windows":
		return "PATH"
	default:
		return "LD_LIBRARY_PATH"
	}
}

// githubToken returns a GitHub token from the environment, if set, used to lift
// the anonymous rate limit when resolving the CPython runtime release. gopack
// works without one; setting it just avoids the 60-requests-per-hour anonymous
// cap when building many bundles in a row. GOPACK_GITHUB_TOKEN takes precedence,
// then the conventional GITHUB_TOKEN and GH_TOKEN.
func githubToken() string {
	for _, name := range []string{"GOPACK_GITHUB_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"} {
		if t := os.Getenv(name); t != "" {
			return t
		}
	}
	return ""
}

var pyVersionRe = regexp.MustCompile(`(\d+\.\d+)`)

// targetPython returns the explicit version, or the detected local interpreter
// version, or a sensible default.
func targetPython(explicit string) string {
	if explicit != "" {
		return explicit
	}
	for _, exe := range []string{"python3", "python"} {
		if out, err := exec.Command(exe, "--version").CombinedOutput(); err == nil {
			if m := pyVersionRe.FindString(string(out)); m != "" {
				return m
			}
		}
	}
	return "3.12"
}

func init() {
	f := buildCmd.Flags()
	f.StringVar(&buildOpts.python, "python", "", "target Python version, for example 3.12")
	f.StringVar(&buildOpts.entry, "entry", "main.py", "entry script, relative to the app directory")
	f.StringVarP(&buildOpts.requirements, "requirements", "r", "", "requirements file to install")
	f.StringVarP(&buildOpts.output, "output", "o", "", "output path (default the app directory name)")
	f.BoolVar(&buildOpts.noEmbed, "no-embed", false, "do not embed detected external native libraries")
	f.StringArrayVar(&buildOpts.exclude, "exclude", nil, "glob of staged paths to leave out of the bundle (repeatable), for example --exclude tests --exclude '*.pyi'")
	rootCmd.AddCommand(buildCmd)
}
