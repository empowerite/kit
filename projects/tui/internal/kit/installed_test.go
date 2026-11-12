// Whether the kit is installed: read from kit.yml's version: line, or an old .kit/manifest's, as the loader reads it.
package kit

import (
	"os"
	"path/filepath"
	"testing"
)

// A clone has the kit installed where its kit.yml's first top-level version: line declares one, the comment beside
// it and the quotes around it aside, or, where that names none, its .kit/manifest records one; nowhere else.
func TestTheKitIsInstalledWhereKitYmlOrTheManifestDeclaresAVersion(t *testing.T) {
	for _, c := range []struct {
		name, settings, manifest string
		want                     bool
	}{
		{"no files", "", "", false},
		{"a version", "require_signing: true\nversion: 0.1.0\n", "", true},
		{"a comment beside it", "version: 0.1.0 # pinned\n", "", true},
		{"quoted", "version: \"0.1.0\"\n", "", true},
		{"single quoted", "version: '0.1.0'\n", "", true},
		{"a comment alone", "version: # none yet\n", "", false},
		{"empty quotes", "version: \"\"\n", "", false},
		{"a value with a # in it", "version: 0.1.0#x\n", "", true},
		{"nested, not top-level", "tools:\n  version: 1.0.0\n", "", false},
		{"only the first line counts", "version:\nversion: 0.1.0\n", "", false},
		{"no version: the repository's own keys", "log_days: 3\n", "", false},
		{"an old manifest", "", "version 0.0.9\npackages kit\n", true},
		{"a manifest with none", "log_days: 3\n", "packages kit\nversion \n", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			if c.settings != "" {
				write(t, filepath.Join(root, "kit.yml"), c.settings)
			}
			if c.manifest != "" {
				write(t, filepath.Join(root, ".kit", "manifest"), c.manifest)
			}
			if got := Installed(root); got != c.want {
				t.Errorf("Installed = %v, want %v", got, c.want)
			}
		})
	}
}

// write writes the file, making its folder first.
func write(t *testing.T, file, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}
