package bundle

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolveEntry turns a manifest entry into an absolute executable path and its
// arguments. A relative first element is the bundled interpreter or program and
// is joined with root. Later elements that name a file inside root are resolved
// to it; anything else is passed through unchanged.
func ResolveEntry(root string, entry []string) (string, []string) {
	exe := entry[0]
	if !filepath.IsAbs(exe) {
		exe = filepath.Join(root, exe)
	}
	args := make([]string, 0, len(entry)-1)
	for _, a := range entry[1:] {
		args = append(args, resolveArg(root, a))
	}
	return exe, args
}

func resolveArg(root, a string) string {
	if filepath.IsAbs(a) {
		return a
	}
	p := filepath.Join(root, a)
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return a
}

// ResolveEnv expands the ${ROOT} token in environment values and returns entries
// in KEY=VALUE form, sorted for stability.
func ResolveEnv(root string, env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sortStrings(keys)

	out := make([]string, 0, len(env))
	for _, k := range keys {
		out = append(out, k+"="+strings.ReplaceAll(env[k], "${ROOT}", root))
	}
	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
