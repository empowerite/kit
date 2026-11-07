// The keys bar is made from the path to the keyboard: each control's keys between rules, a deeper control's press
// hiding an ancestor's key for it, and, too narrow, whole controls dropped from the screen's end.
package shell

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// rows is a control under the screen with its own keys, as a stage's rows are.
var rows = Segment{Bindings: []Binding{Bind("↑↓", "move", "up", "down"), Bind("␣", "select", "space")}}

func TestEachControlsKeysStandBetweenRules(t *testing.T) {
	path := Path{ScreenKeys(Bind("?", "sidebar")), {}, rows}
	if got := ansi.Strip(path.Line(80)); got != " = menu / search ? sidebar │ ↑↓ move ␣ select" {
		t.Errorf("the bar: %q", got)
	}
	// the keys where the keyboard is are bright, and the screen's dim
	line := path.Line(80)
	if !strings.Contains(line, style.BarKey("↑↓", true)) || !strings.Contains(line, style.BarDoes("menu", false)) {
		t.Errorf("the keys are not drawn by where the keyboard is: %q", line)
	}
}

func TestADeeperControlHidesTheKeysItTakes(t *testing.T) {
	for _, c := range []struct {
		name string
		path Path
		want string
	}{
		{"a press it names", Path{ScreenKeys(), {Bindings: []Binding{Bind("=", "equals")}}},
			"/ search │ = equals"},
		{"a press it takes unnamed", Path{ScreenKeys(), {Takes: []string{"/"}, Bindings: []Binding{Bind("x", "ex")}}},
			"= menu │ x ex"},
		{"what types", Path{ScreenKeys(Bind("esc", "quit")), SearchKeys(true)},
			"esc quit │ type to search ⌃R recent"},
		{"every press", Path{ScreenKeys(Bind("1-9", "jump", Digits...)), MenuKeys()},
			"esc close ↑↓ move ⏎ choose"},
		{"every press but those let through", Path{ScreenKeys(Bind("1-9", "jump", Digits...)),
			{Owns: true, Passes: Digits, Bindings: []Binding{Bind("x", "ex")}}}, "1-9 jump │ x ex"},
		{"through two that own", Path{ScreenKeys(Bind("1-9", "jump", Digits...)),
			{Owns: true, Passes: append([]string{"="}, Digits...)}, {Owns: true, Passes: []string{"="}}}, "= menu"},
	} {
		if got := strings.TrimSpace(ansi.Strip(c.path.Line(80))); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestTooNarrowTheBarDropsWholeControlsFromTheScreensEnd(t *testing.T) {
	path := Path{ScreenKeys(), {Bindings: []Binding{Bind("←→", "tab", "left", "right")}}, rows}
	for _, c := range []struct {
		width int
		want  string
	}{
		{80, " = menu / search │ ←→ tab │ ↑↓ move ␣ select"},
		{44, " = menu / search │ ←→ tab │ ↑↓ move ␣ select"},
		{43, " … │ ←→ tab │ ↑↓ move ␣ select"},
		{29, " … │ ↑↓ move ␣ select"},
		{19, " … │ ↑↓ move ␣ sel…"},
		{1, ""},
	} {
		if got := ansi.Strip(path.Line(c.width)); got != c.want {
			t.Errorf("at %d: %q, want %q", c.width, got, c.want)
		}
	}
}

// The settings are kept in their file, each the file does not keep as kittui starts; the dialog flips the one under the
// keyboard and saves it at once, on the menu's color under the keyboard, and its keys take every press, named on the
// keys bar alone.
func TestTheSettingsDialogFlipsASettingAndSavesIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kit", "kittui.json")
	s := LoadSettings(path)
	if !s.KeysBar {
		t.Fatal("with no file, the keys bar is hidden")
	}
	d := SettingsDialog{Open: true}
	if d, s = d.Key(key("space"), s); s.KeysBar || LoadSettings(path).KeysBar || !d.Open {
		t.Errorf("space: shown %v, saved shown %v, open %v", s.KeysBar, LoadSettings(path).KeysBar, d.Open)
	}
	lines := d.Overlay(strings.Split(strings.Repeat(strings.Repeat(" ", 100)+"\n", 10), "\n"), s, 100)
	if v := ansi.Strip(strings.Join(lines, "\n")); !strings.Contains(v, "╭─ Settings ") ||
		!strings.Contains(v, style.Unpicked+" Key bar") || strings.Contains(v, "esc close") ||
		!strings.Contains(strings.Join(lines, "\n"), style.Selected(style.Unpicked+" Key bar")) {
		t.Errorf("the dialog:\n%s", v)
	}
	if got := (Path{ScreenKeys(), d.Keys()}).String(); got != "esc close ↑↓ move ␣ toggle" {
		t.Errorf("the dialog's keys: %q", got)
	}
	if d, _ = d.Key(key("esc"), s); d.Open {
		t.Error("esc did not close the dialog")
	}
	// shared values fold, and shared emoji are dropped, until the dialog flips them, for every view, and the file keeps
	// what it flips
	if !s.FoldShared || s.SharedEmoji {
		t.Errorf("with no file: fold %v, emoji %v", s.FoldShared, s.SharedEmoji)
	}
	d = SettingsDialog{Open: true}
	lines = d.Overlay(strings.Split(strings.Repeat(strings.Repeat(" ", 120)+"\n", 10), "\n"), s, 120)
	if v := ansi.Strip(strings.Join(lines, "\n")); !strings.Contains(v, style.Picked+" Fold shared values") ||
		!strings.Contains(v, style.Unpicked+" Show shared emoji") {
		t.Errorf("the dialog's settings:\n%s", v)
	}
	d, s = d.Key(key("down"), s)
	if d, s = d.Key(key("space"), s); s.FoldShared || LoadSettings(path).FoldShared {
		t.Errorf("space on Fold shared values: %v, saved %v", s.FoldShared, LoadSettings(path).FoldShared)
	}
	d, s = d.Key(key("down"), s)
	if _, s = d.Key(key("space"), s); !s.SharedEmoji || !LoadSettings(path).SharedEmoji {
		t.Errorf("space on Show shared emoji: %v, saved %v", s.SharedEmoji, LoadSettings(path).SharedEmoji)
	}
}

// Keys of one control that do the same thing are drawn as one, joined by /, and a key a deeper control takes leaves the
// others standing.
func TestKeysThatDoTheSameThingAreDrawnAsOne(t *testing.T) {
	sidebar := Segment{Bindings: []Binding{Bind("⇥", "backlog", "tab"), Bind("esc", "backlog"), Bind("↑↓", "move",
		"up", "down")}}
	if got := (Path{sidebar}).String(); got != "⇥·esc backlog ↑↓ move" {
		t.Errorf("the sidebar: %q", got)
	}
	// the dot is dim, where the keyboard is too, and the keys either side of it are not
	if got := (Path{sidebar}).Line(80); !strings.Contains(got, style.BarKey("⇥", true)+style.Off("·")+style.BarKey("esc", true)) {
		t.Errorf("the merged keys drawn: %q", got)
	}
	renaming := Segment{Bindings: []Binding{Bind("esc", "cancel")}}
	if got := (Path{sidebar, renaming}).String(); got != "⇥ backlog ↑↓ move │ esc cancel" {
		t.Errorf("under a rename: %q", got)
	}
}

// With adds bindings after a segment's own, and leaves the segment it was given as it was.
func TestWithAddsBindingsAfterTheSegmentsOwn(t *testing.T) {
	s := Segment{Bindings: []Binding{Bind("esc", "close")}}
	if got := s.With(Bind("⏎", "open", "enter")); len(got.Bindings) != 2 || got.Bindings[1].Does != "open" ||
		len(s.Bindings) != 1 {
		t.Errorf("with: %+v, was %+v", got, s)
	}
}

// The settings file is kit/kittui.json, beside the searches.
func TestTheSettingsFileIsBesideTheSearches(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/somewhere")
	if SettingsFile() != "/somewhere/kit/kittui.json" {
		t.Errorf("the file: %q", SettingsFile())
	}
}
