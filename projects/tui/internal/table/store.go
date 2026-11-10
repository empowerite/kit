// The person's presets, kept in a file of their own, presets.json under the kit's config directory, so their views
// follow them to every repository: each table's, by the table's name, in the catalog's vocabulary, its fields by id.
package table

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// PresetsVersion is the version of the presets file.
const PresetsVersion = 2

// presetsFile is the presets file: its version, and the person's presets, by table.
type presetsFile struct {
	Version int                 `json:"version"`
	Stages  map[string][]Preset `json:"stages"`
}

// PresetsPath gives the file the person's presets are kept in: kit/presets.json under XDG_CONFIG_HOME, else under
// ~/.config; nothing when neither is known.
func PresetsPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "kit", "presets.json")
}

// ReadPresets gives the presets the file keeps, by table; none for no file named. A file that is not there is the
// error, as os.ReadFile says, so a screen can read what it kept before; one that does not read, or is of another
// version, is the error too.
func ReadPresets(path string) (map[string][]Preset, error) {
	out := map[string][]Preset{}
	if path == "" {
		return out, nil
	}
	data, err := os.ReadFile(path)
	if err != nil && os.IsNotExist(err) {
		return out, err
	}
	var file presetsFile
	if err == nil {
		err = json.Unmarshal(data, &file)
	}
	if err == nil && file.Version != PresetsVersion {
		err = fmt.Errorf("version %d, not %d", file.Version, PresetsVersion)
	}
	if err != nil {
		return out, fmt.Errorf("%s does not read as presets: %w", path, err)
	}
	for name, ps := range file.Stages {
		out[name] = ps
	}
	return out, nil
}

// WritePresets keeps the presets in the file, written whole beside it first, so a reader never sees half of it; none
// for no file named.
func WritePresets(path string, tables map[string][]Preset) error {
	if path == "" {
		return nil
	}
	// presets are strings, numbers, and maps of them, which always marshal
	data, _ := json.MarshalIndent(presetsFile{Version: PresetsVersion, Stages: tables}, "", "  ")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".new"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// KeepPresets keeps one table's presets in the file, the person's presets of every other table as the file holds them
// now: it reads the file again, so what another screen saved since it was first read stays, sets the table's presets
// under its name, and writes the file whole, as WritePresets does. It gives every table's presets, as written. A file
// that is not there yet is written new; one that does not read is the error, and nothing is written.
func KeepPresets(path, name string, saved []Preset) (map[string][]Preset, error) {
	all, err := ReadPresets(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	all[name] = saved
	if err := WritePresets(path, all); err != nil {
		return nil, err
	}
	return all, nil
}
