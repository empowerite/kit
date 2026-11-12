// Whether the kit is installed in a clone, read from its files as the loader, bin/kit, reads them, so a screen asks
// it without starting kit: a clone without it gets nothing written under .kit/, nor into kit.yml.
package kit

import (
	"os"
	"path/filepath"
	"strings"
)

// Installed says whether the kit is installed in the clone whose root is given: its kit.yml declares a kit version in
// its first top-level version: line, as the loader's pinned reads it, or, without one, its .kit/manifest records one,
// as a release before the declaration wrote it.
func Installed(root string) bool {
	if data, err := os.ReadFile(filepath.Join(root, "kit.yml")); err == nil {
		for line := range strings.Lines(string(data)) {
			if value, ok := strings.CutPrefix(line, "version:"); ok {
				if declared(value) != "" {
					return true
				}
				break
			}
		}
	}
	data, err := os.ReadFile(filepath.Join(root, ".kit", "manifest"))
	if err != nil {
		return false
	}
	for line := range strings.Lines(string(data)) {
		if value, ok := strings.CutPrefix(strings.TrimRight(line, "\n"), "version "); ok && value != "" {
			return true
		}
	}
	return false
}

// declared gives the value of a version: line, what follows its colon: without the spaces about it, a comment beside
// it, or the quotes around it; empty for a comment alone.
func declared(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "#") {
		return ""
	}
	for at := 1; at < len(value); at++ {
		if value[at] == '#' && (value[at-1] == ' ' || value[at-1] == '\t') {
			value = strings.TrimSpace(value[:at])
			break
		}
	}
	for _, quote := range []string{`"`, `'`} {
		if len(value) >= 2 && strings.HasPrefix(value, quote) && strings.HasSuffix(value, quote) {
			return value[1 : len(value)-1]
		}
	}
	return value
}
