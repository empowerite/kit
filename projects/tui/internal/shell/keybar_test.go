// The keys bar is made from the path to the keyboard: each control's keys between rules, a deeper control's press
// hiding an ancestor's key for it, and, too narrow, whole controls dropped between the screen's and the last.
package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// rows is a control under the screen with its own keys, as a stage's rows are.
var rows = Segment{Bindings: []Binding{Bind("↑↓", "move", "up", "down"), Bind("␣", "select", "space")}}

// A screen with a workspace menu names - after =, in the title's order, then the search and its own keys.
func TestAWorkspaceScreenNamesMinusAfterEquals(t *testing.T) {
	if got := ansi.Strip(Path{WorkspaceScreenKeys(Bind("?", "legend"))}.Line(80)); got !=
		" = menu - workspace / search ? legend" {
		t.Errorf("the bar: %q", got)
	}
}

func TestEachControlsKeysStandBetweenRules(t *testing.T) {
	path := Path{ScreenKeys(Bind("?", "legend")), {}, rows}
	if got := ansi.Strip(path.Line(80)); got != " = menu / search ? legend │ ↑↓ move ␣ select" {
		t.Errorf("the bar: %q", got)
	}
	// the keys where the keyboard is are bright, and the screen's dim
	line := path.Line(80)
	if !strings.Contains(line, style.BarKey("↑↓", true)) || !strings.Contains(line, style.BarDoes("menu", false)) {
		t.Errorf("the keys are not drawn by where the keyboard is: %q", line)
	}
}

// What stands at the bar's right end ends a cell short of the screen's edge, and gives way to the keys: too narrow for
// both, the keys stand alone, whole.
func TestTheRightEndStandsAfterTheKeysWhereBothFit(t *testing.T) {
	path := Path{ScreenKeys(Bind("?", "legend")), {}, rows}
	keys := ansi.Strip(path.Line(80))
	got := ansi.Strip(path.LineWith(60, "🐙 4871/5000"))
	if !strings.HasPrefix(got, keys) || !strings.HasSuffix(got, "🐙 4871/5000") || ansi.StringWidth(got) != 59 {
		t.Errorf("the right end: %q, %d cells", got, ansi.StringWidth(got))
	}
	if got := ansi.Strip(path.LineWith(ansi.StringWidth(keys)+12, "🐙 4871/5000")); got != keys {
		t.Errorf("too narrow for both: %q", got)
	}
	if got := path.LineWith(80, ""); got != path.Line(80) {
		t.Errorf("nothing at the right end: %q", got)
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
		{"what types", Path{ScreenKeys(Bind("esc", "quit")), SearchKeys("rows", "co", true)},
			"⏎ rows type to search ⌃U clear ⌃R recent"},
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

// Too narrow, the bar keeps the screen's keys and the last control's, and drops the controls between them; too narrow
// for those two, it clips the last; too narrow for the screen's alone, it drops it too.
func TestTooNarrowTheBarKeepsTheScreensKeysAndDropsTheControlsBetween(t *testing.T) {
	path := Path{ScreenKeys(), {Bindings: []Binding{Bind("←→", "tab", "left", "right")}}, rows}
	for _, c := range []struct {
		width int
		want  string
	}{
		{80, " = menu / search │ ←→ tab │ ↑↓ move ␣ select"},
		{44, " = menu / search │ ←→ tab │ ↑↓ move ␣ select"},
		{43, " = menu / search │ … │ ↑↓ move ␣ select"},
		{39, " = menu / search │ … │ ↑↓ move ␣ select"},
		{38, " = menu / search │ … │ ↑↓ move ␣ sele…"},
		{23, " = menu / search │ … │…"},
		{22, " … │ ↑↓ move ␣ select"},
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
	if v := ansi.Strip(strings.Join(lines, "\n")); !strings.Contains(v, "╭─ Settings… ") ||
		!strings.Contains(v, style.Unpicked+" Key hints") || strings.Contains(v, "esc close") ||
		!strings.Contains(strings.Join(lines, "\n"), style.Selected(style.Unpicked+" Key hints")) {
		t.Errorf("the dialog:\n%s", v)
	}
	if got := (Path{ScreenKeys(), d.Keys()}).String(); got != "esc close ↑↓ move ␣ toggle" {
		t.Errorf("the dialog's keys: %q", got)
	}
	if d, _ = d.Key(key("esc"), s); d.Open {
		t.Error("esc did not close the dialog")
	}
	// shared values fold, empty columns hide, and shared emoji are dropped, until the dialog flips them, for every
	// view, and the file keeps what it flips
	if !s.FoldShared || !s.HideBlank || s.SharedEmoji {
		t.Errorf("with no file: fold %v, hide %v, emoji %v", s.FoldShared, s.HideBlank, s.SharedEmoji)
	}
	d = SettingsDialog{Open: true}
	lines = d.Overlay(strings.Split(strings.Repeat(strings.Repeat(" ", 120)+"\n", 10), "\n"), s, 120)
	if v := ansi.Strip(strings.Join(lines, "\n")); !strings.Contains(v, style.Picked+" Auto-group") ||
		!strings.Contains(v, style.Picked+" Auto-hide") || !strings.Contains(v, style.Unpicked+" Repeated emoji") {
		t.Errorf("the dialog's settings:\n%s", v)
	}
	d, s = d.Key(key("down"), s)
	if d, s = d.Key(key("space"), s); s.FoldShared || LoadSettings(path).FoldShared {
		t.Errorf("space on Auto-group: %v, saved %v", s.FoldShared, LoadSettings(path).FoldShared)
	}
	d, s = d.Key(key("down"), s)
	if d, s = d.Key(key("space"), s); s.HideBlank || LoadSettings(path).HideBlank {
		t.Errorf("space on Auto-hide: %v, saved %v", s.HideBlank, LoadSettings(path).HideBlank)
	}
	d, s = d.Key(key("down"), s)
	if _, s = d.Key(key("space"), s); !s.SharedEmoji || !LoadSettings(path).SharedEmoji {
		t.Errorf("space on Repeated emoji: %v, saved %v", s.SharedEmoji, LoadSettings(path).SharedEmoji)
	}
}

// The dialog's settings say what they do in one column, two cells after the longest name, the keyboard's highlight on
// the name alone.
func TestTheSettingsDialogsDescriptionsStandInOneColumn(t *testing.T) {
	s := LoadSettings(filepath.Join(t.TempDir(), "kittui.json"))
	box := (SettingsDialog{Open: true}).box(s, 200)
	at := map[int]bool{}
	for k, set := range settings {
		at[strings.Index(ansi.Strip(box[k+1]), set.says)] = true
	}
	if len(at) != 1 {
		t.Errorf("the descriptions start at %v:\n%s", at, ansi.Strip(strings.Join(box, "\n")))
	}
	if !strings.Contains(box[1], style.Selected(style.Unpicked+" Key hints")+"     ") &&
		!strings.Contains(box[1], style.Selected(style.Picked+" Key hints")+"     ") {
		t.Errorf("Key hints: %q", box[1])
	}
}

// A left click on a setting puts the keyboard on it and flips it, saved at once; one on the box's edge changes
// nothing; one off the box closes the dialog; another button does nothing.
func TestAClickOnTheSettingsDialogFlipsASettingOrClosesIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kittui.json")
	s, d := LoadSettings(path), SettingsDialog{Open: true}
	box := d.box(s, 120)
	top, left := d.place(box, 30, 120)
	click := func(x, y int, b tea.MouseButton) tea.MouseClickMsg { return tea.MouseClickMsg{X: x, Y: y, Button: b} }
	if d, s = d.Click(click(left+3, top+2, tea.MouseLeft), s, 30, 120); d.At != 1 || s.FoldShared ||
		LoadSettings(path).FoldShared {
		t.Errorf("a click on Auto-group: at %d, fold %v", d.At, s.FoldShared)
	}
	if d, s = d.Click(click(left+3, top+2, tea.MouseRight), s, 30, 120); s.FoldShared || !d.Open {
		t.Errorf("a right click flipped: fold %v", s.FoldShared)
	}
	if d, s = d.Click(click(left+3, top, tea.MouseLeft), s, 30, 120); !d.Open || s.FoldShared {
		t.Errorf("a click on the edge: open %v, fold %v", d.Open, s.FoldShared)
	}
	if d, _ = d.Click(click(0, 0, tea.MouseLeft), s, 30, 120); d.Open {
		t.Error("a click off the box kept the dialog")
	}
}

// Keys of one control that do the same thing are drawn as one, joined by /, and a key a deeper control takes leaves the
// others standing.
func TestKeysThatDoTheSameThingAreDrawnAsOne(t *testing.T) {
	sidebar := Segment{Bindings: []Binding{Bind("⇥", "issues", "tab"), Bind("esc", "issues"), Bind("↑↓", "move",
		"up", "down")}}
	if got := (Path{sidebar}).String(); got != "⇥·esc issues ↑↓ move" {
		t.Errorf("the sidebar: %q", got)
	}
	// the dot is dim, where the keyboard is too, and the keys either side of it are not
	if got := (Path{sidebar}).Line(80); !strings.Contains(got, style.BarKey("⇥", true)+style.Off("·")+style.BarKey("esc", true)) {
		t.Errorf("the merged keys drawn: %q", got)
	}
	renaming := Segment{Bindings: []Binding{Bind("esc", "cancel")}}
	if got := (Path{sidebar, renaming}).String(); got != "⇥ issues ↑↓ move │ esc cancel" {
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

// With no home, and no XDG_CONFIG_HOME, kittui keeps no file: the searches and the settings are the session's alone,
// the dialog flipping a setting writes nothing, not even where kittui runs.
func TestWithNoHomeTheSettingsAreTheSessionsAlone(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	t.Chdir(t.TempDir())
	if RecentFile() != "" || SettingsFile() != "" {
		t.Errorf("the files: %q, %q", RecentFile(), SettingsFile())
	}
	s := LoadSettings(SettingsFile())
	if _, s = (SettingsDialog{Open: true}).Key(key("space"), s); s.KeysBar {
		t.Error("space did not flip Key hints")
	}
	if entries, _ := os.ReadDir("."); len(entries) != 0 {
		t.Errorf("written where kittui runs: %v", entries)
	}
	// closed, the dialog draws nothing over the screen
	if got := (SettingsDialog{}).Overlay([]string{"a", "b"}, s, 40); strings.Join(got, "|") != "a|b" {
		t.Errorf("closed, drawn: %q", got)
	}
}

// A binding with a key and nothing it does draws the key alone.
func TestAKeyThatSaysNothingIsDrawnAlone(t *testing.T) {
	if got := ansi.Strip((Path{{Bindings: []Binding{Bind("=", "menu"), Bind("q", "")}}}).Line(40)); got != " = menu q" {
		t.Errorf("drawn: %q", got)
	}
}
