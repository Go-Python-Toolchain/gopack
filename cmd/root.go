package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gopack",
	Short: "Bundle a Python app and its interpreter into one executable",
	Long: `gopack packs a Python application together with a self-contained CPython
runtime and its dependencies into a single executable. The result runs on a
machine that has no Python installed: it extracts what it needs on first run and
launches the app.

gopack does not ask you to restructure your app. It bundles what pip installs.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command and exits non-zero on failure.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "gopack:", err)
		os.Exit(1)
	}
}
