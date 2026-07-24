package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/Go-Python-Toolchain/gopack/internal/bundle"
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Inspect and clear gopack's caches",
	Long: `cache shows where gopack keeps what it downloads and extracts, how much space
that is, and clears it.

gopack keeps two caches under one root. Downloaded CPython runtimes are the
build-time cache: acquiring one is a download, so it is worth keeping between
builds. Extracted bundles are the run-time cache: a bundle unpacks itself here on
first launch and reuses it after, and anything removed is simply re-extracted on
the next run.`,
}

var cacheDirCmd = &cobra.Command{
	Use:   "dir",
	Short: "Print the cache root",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := bundle.CacheRoot()
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), root)
		return nil
	},
}

var cacheInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show what the caches hold",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := bundle.CacheRoot()
		if err != nil {
			return err
		}
		runtimes, bundles, err := scanCaches(root)
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "cache root: %s\n", root)

		fmt.Fprintf(out, "\nruntimes (downloaded CPython, build-time): %d, %s\n", len(runtimes), humanBytes(sumSizes(runtimes)))
		for _, e := range runtimes {
			fmt.Fprintf(out, "  %s  %s\n", humanBytes(e.size), e.name)
		}
		fmt.Fprintf(out, "\nextracted bundles (run-time): %d, %s\n", len(bundles), humanBytes(sumSizes(bundles)))
		for _, e := range bundles {
			fmt.Fprintf(out, "  %s  %s\n", humanBytes(e.size), e.name)
		}

		if len(runtimes) == 0 && len(bundles) == 0 {
			fmt.Fprintln(out, "\nthe cache is empty")
		}
		return nil
	},
}

var cacheClearOpts struct {
	all bool
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete extracted bundles, and with --all the downloaded runtimes too",
	Long: `clear removes the extracted-bundle cache, which is regenerated automatically
the next time a bundle runs. Downloaded runtimes are kept by default, since each
one is a download; pass --all to remove them as well.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := bundle.CacheRoot()
		if err != nil {
			return err
		}
		runtimes, bundles, err := scanCaches(root)
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		removed := 0
		var freed int64

		for _, e := range bundles {
			if err := os.RemoveAll(e.path); err != nil {
				return err
			}
			removed++
			freed += e.size
		}
		if cacheClearOpts.all {
			for _, e := range runtimes {
				if err := os.RemoveAll(e.path); err != nil {
					return err
				}
				removed++
				freed += e.size
			}
		}

		if removed == 0 {
			fmt.Fprintln(out, "nothing to remove")
			return nil
		}
		what := "extracted bundle(s)"
		if cacheClearOpts.all {
			what = "cache entrie(s), runtimes included"
		}
		fmt.Fprintf(out, "removed %d %s, freed %s\n", removed, what, humanBytes(freed))
		if !cacheClearOpts.all && len(runtimes) > 0 {
			fmt.Fprintf(out, "kept %d downloaded runtime(s), %s; use --all to remove them\n",
				len(runtimes), humanBytes(sumSizes(runtimes)))
		}
		return nil
	},
}

// cacheEntry is one cached item: a runtime or an extracted bundle.
type cacheEntry struct {
	name string
	path string
	size int64
}

// scanCaches enumerates the two caches under root, each sorted by name. A root
// that does not exist yet is reported as two empty lists, since an unused cache
// is a normal state.
func scanCaches(root string) (runtimes, bundles []cacheEntry, err error) {
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if e.Name() == bundle.RuntimesSubdir {
			runtimes, err = scanDir(filepath.Join(root, e.Name()))
			if err != nil {
				return nil, nil, err
			}
			continue
		}
		size, err := dirSize(filepath.Join(root, e.Name()))
		if err != nil {
			return nil, nil, err
		}
		bundles = append(bundles, cacheEntry{name: e.Name(), path: filepath.Join(root, e.Name()), size: size})
	}
	sort.Slice(bundles, func(i, j int) bool { return bundles[i].name < bundles[j].name })
	return runtimes, bundles, nil
}

// scanDir enumerates the immediate subdirectories of dir as cache entries.
func scanDir(dir string) ([]cacheEntry, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []cacheEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		size, err := dirSize(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, cacheEntry{name: e.Name(), path: filepath.Join(dir, e.Name()), size: size})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, nil
}

func sumSizes(entries []cacheEntry) int64 {
	var total int64
	for _, e := range entries {
		total += e.size
	}
	return total
}

// dirSize returns the total size of the regular files under dir.
func dirSize(dir string) (int64, error) {
	var total int64
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			total += info.Size()
		}
		return nil
	})
	return total, err
}

// humanBytes renders a size the way a person reads it.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n/div >= unit && exp < 3 {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGT"[exp])
}

func init() {
	cacheClearCmd.Flags().BoolVar(&cacheClearOpts.all, "all", false, "also remove downloaded CPython runtimes")
	cacheCmd.AddCommand(cacheDirCmd, cacheInfoCmd, cacheClearCmd)
	rootCmd.AddCommand(cacheCmd)
}
