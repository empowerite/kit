package shell

import (
	"encoding/json"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Settings are kittui's own, the same on every screen, which the Settings dialog changes and kittui's file keeps:
// whether the keys bar shows on the screen's last line. With no file, or one that does not read, each is as kittui
// starts, the keys bar shown.
type Settings struct {
	Path    string `json:"-"`
	KeysBar bool   `json:"keysBar"`
}

// SettingsFile gives the file kittui keeps its settings in: kit/kittui.json under XDG_CONFIG_HOME, else under
// ~/.config, beside the searches; nothing when neither is known.
func SettingsFile() string {
	return filepath.Join(filepath.Dir(RecentFile()), "kittui.json")
}

// LoadSettings gives the settings the file keeps, each it does not keep as kittui starts.
func LoadSettings(path string) Settings {
	s := Settings{KeysBar: true}
	if data, err := os.ReadFile(path); path != "" && err == nil {
		_ = json.Unmarshal(data, &s)
	}
	s.Path = path
	return s
}

// Save writes the settings to their file. A file that cannot be written leaves them for the session.
func (s Settings) Save() {
	if s.Path == "" {
		return
	}
	if data, err := json.MarshalIndent(s, "", "  "); err == nil && os.MkdirAll(filepath.Dir(s.Path), 0o755) == nil {
		_ = os.WriteFile(s.Path, append(data, '\n'), 0o644)
	}
}

// SettingsScreen is what the app's menu's Settings… asks for: not a screen, but the dialog kittui opens over the one
// shown.
const SettingsScreen Screen = "settings"

// setting is one line of the Settings dialog: its name, what it says, and how it reads and flips its value.
type setting struct {
	name, says string
	get        func(Settings) bool
	flip       func(Settings) Settings
}

// settings are the Settings dialog's lines, in order.
var settings = []setting{
	{"Keys bar", "the keys that act where the keyboard is, on the last line",
		func(s Settings) bool { return s.KeysBar },
		func(s Settings) Settings { s.KeysBar = !s.KeysBar; return s }},
}

// SettingsDialog is the dialog the app's menu's Settings… opens over the screen shown: whether it is open, which then
// has the keyboard, and the setting under the keyboard.
type SettingsDialog struct {
	Open bool
	At   int
}

// settingsKeys are the dialog's keys, on its bottom edge and on the keys bar.
const settingsKeys = "esc close · ↑↓ move · ␣ toggle"

// Key takes a key while the dialog is open: up and down move between the settings, space or enter flips the one under
// the keyboard, which is saved at once, and esc closes the dialog, as = does. It gives the dialog and the settings.
func (d SettingsDialog) Key(k tea.KeyPressMsg, s Settings) (SettingsDialog, Settings) {
	switch k.String() {
	case "up":
		d.At = max(d.At-1, 0)
	case "down":
		d.At = min(d.At+1, len(settings)-1)
	case "space", "enter":
		s = settings[d.At].flip(s)
		s.Save()
	case "esc", "=":
		d.Open = false
	}
	return d, s
}

// Keys gives the dialog's segment of the keys bar: it takes every press.
func (d SettingsDialog) Keys() Segment {
	s := DialogKeys(settingsKeys)
	s.Bindings[1].Presses = []string{"up", "down"}
	s.Bindings[2].Presses = []string{"space", "enter"}
	return s
}

// Overlay draws the open dialog over the screen's lines, centered across a screen so wide, near its top: each setting
// a box picked or not, its name, and what it says, dim, the one under the keyboard inverse.
func (d SettingsDialog) Overlay(lines []string, s Settings, width int) []string {
	if !d.Open {
		return lines
	}
	var body []string
	for k, set := range settings {
		box := style.Unpicked
		if set.get(s) {
			box = style.Picked
		}
		name := box + " " + set.name
		if k == d.At {
			name = style.Cursor(name)
		}
		body = append(body, name+"  "+style.Off(set.says))
	}
	box := Dialog("Settings", body, settingsKeys, width-2)
	return Over(lines, box, min(2, max(0, len(lines)-len(box))), max(0, (width-ansi.StringWidth(box[0]))/2))
}
