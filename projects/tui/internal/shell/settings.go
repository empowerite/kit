package shell

import (
	"encoding/json"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Settings are kittui's own, the same on every screen and every view, which the Settings dialog changes and kittui's
// file keeps: whether the keys bar shows on the screen's last line; whether a table folds a value every row of a group
// shares, said once in the group's heading; and whether it draws an emoji every row shown shares, which it drops
// otherwise. With no file, or one that does not read, each is as kittui starts: the keys bar shown, shared values
// folded, and shared emoji dropped.
type Settings struct {
	Path        string `json:"-"`
	KeysBar     bool   `json:"keysBar"`
	FoldShared  bool   `json:"foldShared"`
	SharedEmoji bool   `json:"sharedEmoji"`
}

// SettingsMsg tells a screen the settings, when it is made and whenever the dialog changes one, so a table folds and
// draws as they say.
type SettingsMsg struct{ Settings Settings }

// SettingsFile gives the file kittui keeps its settings in: kit/kittui.json under XDG_CONFIG_HOME, else under
// ~/.config, beside the searches; nothing when neither is known.
func SettingsFile() string {
	return filepath.Join(filepath.Dir(RecentFile()), "kittui.json")
}

// LoadSettings gives the settings the file keeps, each it does not keep as kittui starts.
func LoadSettings(path string) Settings {
	s := Settings{KeysBar: true, FoldShared: true}
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
	{"Key bar", "shows the keys you can press now, on the bottom line",
		func(s Settings) bool { return s.KeysBar },
		func(s Settings) Settings { s.KeysBar = !s.KeysBar; return s }},
	{"Auto-group", "puts a value every row shares in one heading over them, while no grouping is chosen",
		func(s Settings) bool { return s.FoldShared },
		func(s Settings) Settings { s.FoldShared = !s.FoldShared; return s }},
	{"Repeated emoji", "keeps an emoji every row shows, which is otherwise dropped",
		func(s Settings) bool { return s.SharedEmoji },
		func(s Settings) Settings { s.SharedEmoji = !s.SharedEmoji; return s }},
}

// SettingsDialog is the dialog the app's menu's Settings… opens over the screen shown: whether it is open, which then
// has the keyboard, and the setting under the keyboard.
type SettingsDialog struct {
	Open bool
	At   int
}

// settingsKeys are the dialog's keys, on the keys bar.
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

// Click takes a click while the dialog is open: a left one on a setting puts the keyboard on it and flips it, saved at
// once, as space does; a left one outside the box closes the dialog, as esc does; anything else, as a click on the
// box's edges, changes nothing. The dialog stands where Overlay draws it over a screen of so many lines, so wide. It
// gives the dialog and the settings.
func (d SettingsDialog) Click(msg tea.MouseClickMsg, s Settings, height, width int) (SettingsDialog, Settings) {
	mouse := msg.Mouse()
	if mouse.Button != tea.MouseLeft {
		return d, s
	}
	box := d.box(s, width)
	top, left := d.place(box, height, width)
	switch k := mouse.Y - top - 1; {
	case mouse.Y < top || mouse.Y >= top+len(box) || mouse.X < left || mouse.X >= left+ansi.StringWidth(box[0]):
		d.Open = false
	case k >= 0 && k < len(settings):
		d.At = k
		s = settings[k].flip(s)
		s.Save()
	}
	return d, s
}

// Overlay draws the open dialog over the screen's lines, centered across a screen so wide, near its top: each setting
// a box picked or not, its name, and what it says, dim, the one under the keyboard inverse.
func (d SettingsDialog) Overlay(lines []string, s Settings, width int) []string {
	if !d.Open {
		return lines
	}
	box := d.box(s, width)
	top, left := d.place(box, len(lines), width)
	return Over(lines, box, top, left)
}

// box gives the dialog's lines, as Overlay draws them on a screen so wide.
func (d SettingsDialog) box(s Settings, width int) []string {
	var body []string
	for k, set := range settings {
		box := style.Unpicked
		if set.get(s) {
			box = style.Picked
		}
		name := box + " " + set.name
		if k == d.At {
			name = style.Selected(name)
		}
		body = append(body, name+"  "+style.Off(set.says))
	}
	return Dialog("Settings", body, width-2)
}

// place gives the line and the column the dialog's box starts at on a screen of so many lines, so wide: its third
// line, or as near as the screen lets, and centered across it.
func (d SettingsDialog) place(box []string, height, width int) (top, left int) {
	return min(2, max(0, height-len(box))), max(0, (width-ansi.StringWidth(box[0]))/2)
}
