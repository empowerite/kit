// The person's presets, kept in a file of their own, presets.json under the kit's config directory, so their views
// follow them to every repository: each table's, by the table's name, in the catalog's vocabulary, its fields by id;
// and each table's favorites, the names of the views the person starred, in their order.
package table

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
)

// PresetsVersion is the version of the presets file.
const PresetsVersion = 2

// presetsFile is the presets file: its version, the person's presets, by table, and their favorites, by table.
type presetsFile struct {
	Version   int                 `json:"version"`
	Stages    map[string][]Preset `json:"stages"`
	Favorites map[string][]string `json:"favorites,omitempty"`
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
	file, err := readFile(path)
	return file.Stages, err
}

// ReadFavorites gives the favorites the file keeps, by table, a map never nil; none for no file named, nor for one that
// does not read, which ReadPresets says.
func ReadFavorites(path string) map[string][]string {
	file, _ := readFile(path)
	return file.Favorites
}

// readFile gives the presets file as it reads, its maps never nil; empty for no file named. A file that is not there is
// the error, as os.ReadFile says; one that does not read, or is of another version, is the error too.
func readFile(path string) (presetsFile, error) {
	out := presetsFile{Stages: map[string][]Preset{}, Favorites: map[string][]string{}}
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
	maps.Copy(out.Stages, file.Stages)
	maps.Copy(out.Favorites, file.Favorites)
	return out, nil
}

// WritePresets keeps the presets in the file, the favorites as it holds them, written whole beside it first, so a
// reader never sees half of it; none for no file named.
func WritePresets(path string, tables map[string][]Preset) error {
	file, _ := readFile(path)
	file.Stages = tables
	return writeFile(path, file)
}

// writeFile keeps the presets file whole, written beside it first, so a reader never sees half of it; nothing for no
// file named.
func writeFile(path string, file presetsFile) error {
	if path == "" {
		return nil
	}
	file.Version = PresetsVersion
	if len(file.Favorites) == 0 {
		file.Favorites = nil
	}
	// presets are strings, numbers, and maps of them, which always marshal
	data, _ := json.MarshalIndent(file, "", "  ")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".new"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// KeepPresets keeps one table's presets in the file, and its favorites, the person's presets and favorites of every
// other table as the file holds them now: it reads the file again, so what another screen saved since it was first read
// stays, sets the table's presets and favorites under its name, and writes the file whole. A table's favorites are given
// as they stand, those that name a view since gone already dropped, so saving drops them. It gives every table's
// presets, as written. A file that is not there yet is written new; one that does not read is the error, and nothing
// is written.
func KeepPresets(path, name string, saved []Preset, favorites []string) (map[string][]Preset, error) {
	file, err := readFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	file.Stages[name] = saved
	file.Favorites[name] = favorites
	if len(favorites) == 0 {
		delete(file.Favorites, name)
	}
	if err := writeFile(path, file); err != nil {
		return nil, err
	}
	return file.Stages, nil
}

// KeepFavorites keeps one table's favorites in the file, as KeepPresets does, its presets as the file holds them.
func KeepFavorites(path, name string, favorites []string) error {
	file, err := readFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	_, err = KeepPresets(path, name, file.Stages[name], favorites)
	return err
}
