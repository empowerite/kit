// The Open Repository… dialog takes a folder typed, completes it a folder at a time, and asks to open it.
package opener

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// folders makes the folders under a directory of the test's own, and a file beside them, and gives the directory.
func folders(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		if err := os.MkdirAll(filepath.Join(dir, n), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "kitfile"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// typed gives the dialog after each rune of the text is typed.
func typedIn(d Dialog, text string) Dialog {
	for _, r := range text {
		d, _ = d.keyed(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return d
}

func TestOpenStartsAtTheFolderGivenWritingHomeAsTilde(t *testing.T) {
	home := t.TempDir()
	d := newOpen(filepath.Join(home, "Code"), home)
	if !d.Open || d.Typed != "~/Code/" {
		t.Errorf("opened: %+v", d)
	}
	if d := newOpen("/srv", home); d.Typed != "/srv/" {
		t.Errorf("outside home: %q", d.Typed)
	}
	if d := newOpen("", home); d.Typed != "~/" {
		t.Errorf("with none: %q", d.Typed)
	}
}

func TestOpenTypesErasesAndClears(t *testing.T) {
	d := typedIn(newOpen("/", "/home"), "srv")
	if d.Typed != "/srv" {
		t.Errorf("typed: %q", d.Typed)
	}
	d, _ = d.keyed(key("backspace"))
	if d.Typed != "/sr" {
		t.Errorf("erased: %q", d.Typed)
	}
	d.Refusal = "x"
	d, _ = d.keyed(key("ctrl+u"))
	if d.Typed != "" || d.Refusal != "" {
		t.Errorf("cleared: %+v", d)
	}
	if d, path := d.keyed(key("enter")); path != "" || !d.Open {
		t.Errorf("enter with nothing typed: %q, open %v", path, d.Open)
	}
	if d, _ := d.keyed(key("esc")); d.Open {
		t.Error("esc leaves it open")
	}
}

func TestOpenTabCompletesAFolderAtATime(t *testing.T) {
	dir := folders(t, "kit", "kittens", "edge", ".hidden")
	d := newOpen(dir, "/nowhere")
	if got := d.Folders(); strings.Join(got, "|") != "edge|kit|kittens" {
		t.Errorf("the folders offered: %q", got)
	}
	d = typedIn(d, "k")
	if got := d.Folders(); strings.Join(got, "|") != "kit|kittens" {
		t.Errorf("after k: %q", got)
	}
	d, _ = d.keyed(key("tab"))
	if d.Typed != dir+"/kit" {
		t.Errorf("tab to the common part: %q", d.Typed)
	}
	d = typedIn(d, "t")
	d, _ = d.keyed(key("tab"))
	if d.Typed != dir+"/kittens/" {
		t.Errorf("tab to the one folder: %q", d.Typed)
	}
	d = typedIn(newOpen(dir, "/nowhere"), ".")
	if got := d.Folders(); strings.Join(got, "|") != ".hidden" {
		t.Errorf("a dot offers the hidden ones: %q", got)
	}
	d = typedIn(newOpen(dir, "/nowhere"), "zzz")
	if d, _ = d.keyed(key("tab")); d.Typed != dir+"/zzz" {
		t.Errorf("tab with none: %q", d.Typed)
	}
}

func TestOpenEnterAsksForTheFolderWhole(t *testing.T) {
	home := folders(t, "Code/kit")
	d := newOpen(filepath.Join(home, "Code"), home)
	d = typedIn(d, "kit")
	d, path := d.keyed(key("enter"))
	if path != filepath.Join(home, "Code", "kit") || !d.Open || !d.Busy {
		t.Errorf("enter: %q, %+v", path, d)
	}
	// while kit is asked, nothing more is asked
	if _, again := d.keyed(key("enter")); again != "" {
		t.Errorf("asked again: %q", again)
	}
	d = d.Refused("'x' is no git working tree")
	if d.Busy || !d.Open || d.Refusal == "" {
		t.Errorf("refused: %+v", d)
	}
	lines := d.Overlay(make([]string, 20), 100)
	if !strings.Contains(ansi.Strip(strings.Join(lines, "\n")), "is no git working tree") {
		t.Errorf("the refusal is not drawn:\n%s", ansi.Strip(strings.Join(lines, "\n")))
	}
	rel := typedIn(newOpen("", home), "Code/../Code/kit")
	if _, path := rel.keyed(key("enter")); path != filepath.Join(home, "Code", "kit") {
		t.Errorf("a path written from ~: %q", path)
	}
}

// Every line of the dialog does what it says on a plain click: a folder completes to it, Open opens what is typed, and
// a click off the box closes it.
func TestOpenIsDoneByPlainClicks(t *testing.T) {
	dir := folders(t, "edge", "kit")
	d := newOpen(dir, "/nowhere")
	p := d.placed(30, 100)
	lines := ansi.Strip(strings.Join(p.Lines, "\n"))
	if !strings.Contains(lines, Folder+" edge/") || !strings.Contains(lines, Opening+" Open") {
		t.Fatalf("the box:\n%s", lines)
	}
	row := func(d Dialog, text string) int {
		p := d.placed(30, 100)
		for k, l := range p.Lines {
			if strings.Contains(ansi.Strip(l), text) {
				return p.Top + k
			}
		}
		t.Fatalf("no line %q", text)
		return 0
	}
	click := func(d Dialog, y int) (Dialog, string) {
		return d.clicked(tea.MouseClickMsg{X: p.Left + 3, Y: y, Button: tea.MouseLeft}, 30, 100)
	}
	d2, path := click(d, row(d, "kit/"))
	if path != "" || d2.Typed != dir+"/kit/" {
		t.Errorf("a click on kit/: %q, %q", path, d2.Typed)
	}
	if _, path := click(d2, row(d2, Opening+" Open")); path != dir+"/kit" {
		t.Errorf("a click on Open: %q", path)
	}
	if d3, _ := d.clicked(tea.MouseClickMsg{X: 0, Y: 29, Button: tea.MouseLeft}, 30, 100); d3.Open {
		t.Error("a click off the box leaves it open")
	}
}

// many makes so many folders, f00 on, under a directory of the test's own, and gives the directory.
func many(t *testing.T, n int) string {
	t.Helper()
	var names []string
	for k := range n {
		names = append(names, fmt.Sprintf("f%02d", k))
	}
	return folders(t, names...)
}

// drawn gives the dialog's box as plain text, over a screen of 30 lines, 100 wide.
func drawn(d Dialog) string { return ansi.Strip(strings.Join(d.placed(30, 100).Lines, "\n")) }

// ↓ from what is typed puts the highlight on the first folder, and ↑ from it back on what is typed; the list windows to
// keep the highlight in view, saying how many folders it hides above and below; ⇞ and ⇟ move it by a page.
func TestOpenMovesAHighlightOverTheFoldersAndScrollsToIt(t *testing.T) {
	d := newOpen(many(t, 20), "/nowhere")
	if v := drawn(d); !strings.Contains(v, Folder+" f00/") || !strings.Contains(v, "… 12 more") ||
		strings.Contains(v, "above") {
		t.Errorf("at the top:\n%s", v)
	}
	lit := func(d Dialog, name string) bool {
		t.Helper()
		on, _, _ := strings.Cut(style.Selected("x"), "x")
		for _, l := range d.placed(30, 100).Lines {
			if strings.Contains(l, on+" "+Folder+" "+name+"/ ") {
				return true
			}
		}
		return false
	}
	d, _ = d.keyed(key("down"))
	if d.At != 1 || !lit(d, "..") {
		t.Errorf("down from what is typed, onto ../: %d\n%s", d.At, drawn(d))
	}
	d, _ = d.keyed(key("down"))
	if !lit(d, "f00") || lit(d, "f01") || lit(d, "..") {
		t.Errorf("f00 is not the one highlighted:\n%s", drawn(d))
	}
	if up, _ := d.keyed(key("up")); up.At != 1 {
		t.Errorf("up from the first folder: %d", up.At)
	} else if up, _ = up.keyed(key("up")); up.At != 0 {
		t.Errorf("up from ../: %d", up.At)
	}
	if d, _ := newOpen(t.TempDir(), "/nowhere").keyed(key("up")); d.At != 0 {
		t.Errorf("up from what is typed: %d", d.At)
	}
	for range 9 {
		d, _ = d.keyed(key("down"))
	}
	if v := drawn(d); !lit(d, "f09") || strings.Contains(v, " f00/") || !strings.Contains(v, "above") ||
		!strings.Contains(v, "more") {
		t.Errorf("down to f09:\n%s", v)
	}
	if v := drawn(d); !strings.Contains(v, Folder+" ../") {
		t.Errorf("../ scrolled away:\n%s", v)
	}
	d, _ = d.keyed(tea.KeyPressMsg{Code: tea.KeyPgDown})
	if d.At != 11+openPage || !lit(d, "f16") {
		t.Errorf("a page down: %d\n%s", d.At, drawn(d))
	}
	for range 5 {
		d, _ = d.keyed(tea.KeyPressMsg{Code: tea.KeyPgDown})
	}
	if v := drawn(d); d.At != 21 || !lit(d, "f19") || strings.Contains(v, "more") {
		t.Errorf("pages to the end: %d\n%s", d.At, v)
	}
	for range 5 {
		d, _ = d.keyed(tea.KeyPressMsg{Code: tea.KeyPgUp})
	}
	if v := drawn(d); d.At != 1 || strings.Contains(v, "above") {
		t.Errorf("pages to the top: %d\n%s", d.At, v)
	}
}

// Enter on a folder highlighted goes into it, as a click on it does; typing, tab, or backspace drops the highlight.
func TestOpenEnterGoesIntoTheFolderHighlighted(t *testing.T) {
	dir := many(t, 3)
	d, _ := newOpen(dir, "/nowhere").keyed(key("down"))
	d, _ = d.keyed(key("down"))
	d, _ = d.keyed(key("down"))
	d, path := d.keyed(key("enter"))
	if path != "" || d.Typed != dir+"/f01/" || d.At != 0 {
		t.Errorf("enter on f01: %q, %+v", path, d)
	}
	for _, k := range []tea.KeyPressMsg{key("f"), key("tab"), {Code: tea.KeyBackspace}} {
		d, _ := newOpen(dir, "/nowhere").keyed(key("down"))
		if d, _ = d.keyed(k); d.At != 0 {
			t.Errorf("%s leaves the highlight: %d", k, d.At)
		}
	}
}

// The mouse wheel scrolls the folders, never past their ends, the highlight staying where it is.
func TestOpenWheelScrollsTheFolders(t *testing.T) {
	d := newOpen(many(t, 20), "/nowhere")
	down := tea.MouseWheelMsg{Button: tea.MouseWheelDown}
	d = d.Wheel(down, 30, 100)
	if v := drawn(d); strings.Contains(v, " f00/") || !strings.Contains(v, "above") || d.At != 0 {
		t.Errorf("a turn down:\n%s", v)
	}
	for range 10 {
		d = d.Wheel(down, 30, 100)
	}
	if v := drawn(d); !strings.Contains(v, " f19/") || strings.Contains(v, "more") {
		t.Errorf("turned to the end:\n%s", v)
	}
	for range 10 {
		d = d.Wheel(tea.MouseWheelMsg{Button: tea.MouseWheelUp}, 30, 100)
	}
	if v := drawn(d); !strings.Contains(v, " f00/") || strings.Contains(v, "above") {
		t.Errorf("turned to the top:\n%s", v)
	}
}

// The keys bar names ↑↓ while there are lines to move over, ⇞⇟ while there are more folders than the list shows, and
// going up while there is a folder above.
func TestOpenNamesTheMovesWhereTheyApply(t *testing.T) {
	keys := func(d Dialog) string { return (shell.Path{d.Keys()}).String() }
	if got := keys(newOpen(t.TempDir(), "")); got != "esc close ↑↓ move ⇥ complete ⇧⌫·⌃W up ⌃F find clones ⏎ open" {
		t.Errorf("no folders: %q", got)
	}
	if got := keys(newOpen(many(t, 20), "")); got != "esc close ↑↓ move ⇞⇟ page ⇥ complete ⇧⌫·⌃W up ⌃F find clones ⏎ open" {
		t.Errorf("many: %q", got)
	}
	if got := keys(newOpen("/", "")); strings.Contains(got, " up") {
		t.Errorf("at /: %q", got)
	}
}

// ⇧⌫ and ⌃W take the last folder off what is typed, ~/ giving the home folder's parent, written whole, and / staying;
// both are bound, as iTerm sends ⇧⌫ as ⌫ unless told otherwise.
func TestOpenGoesUpAFolder(t *testing.T) {
	for typed, want := range map[string]string{"~/a/b/c": "~/a/b/", "~/a/b/c/": "~/a/b/", "/x": "/", "/": "/",
		"~/": "/home/", "~": "/home/", "~/a": "~/"} {
		for _, k := range []tea.KeyPressMsg{{Code: tea.KeyBackspace, Mod: tea.ModShift}, {Code: 'w', Mod: tea.ModCtrl}} {
			d := Dialog{Open: true, Home: "/home/me", Typed: typed, At: 1, Refusal: "x"}
			if d, _ = d.keyed(k); d.Typed != want || d.At != 0 || d.Refusal != "" {
				t.Errorf("%s on %q: %+v, want %q", k, typed, d, want)
			}
		}
	}
}

// The list starts with ../ while there is a folder above; a click on it, or enter on it, goes up a folder.
func TestOpenListsTheFolderAboveFirst(t *testing.T) {
	dir := folders(t, "kit/a")
	d := newOpen(filepath.Join(dir, "kit"), "/nowhere")
	p := d.placed(30, 100)
	row := -1
	for k, l := range p.Lines {
		if strings.Contains(ansi.Strip(l), Folder+" ../") {
			row = p.Top + k
			break
		}
		if strings.Contains(ansi.Strip(l), Folder+" a/") {
			t.Fatalf("a/ stands before ../:\n%s", drawn(d))
		}
	}
	if row < 0 {
		t.Fatalf("no ../:\n%s", drawn(d))
	}
	if up, path := d.clicked(tea.MouseClickMsg{X: p.Left + 3, Y: row, Button: tea.MouseLeft}, 30, 100); path != "" ||
		up.Typed != dir+"/" {
		t.Errorf("a click on ../: %q, %q", path, up.Typed)
	}
	d, _ = d.keyed(key("down"))
	if d, _ = d.keyed(key("enter")); d.Typed != dir+"/" {
		t.Errorf("enter on ../: %q", d.Typed)
	}
	if v := drawn(newOpen("/", "")); strings.Contains(v, "../") {
		t.Errorf("at /:\n%s", v)
	}
}

// clones makes a tree of folders under a directory of the test's own, some of them git working trees, a .git folder
// or file in each, and gives the directory.
func cloneTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, p := range []string{"a/.git/modules/z/.git", "b/c/.git", ".hidden/x/.git", "node_modules/y/.git",
		"d/1/2/3/4/5/.git", "d/1/2/3/4/5/6/7/.git", "plain/e"} {
		if err := os.MkdirAll(filepath.Join(dir, p), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "a/worktrees/w1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a/worktrees/w1/.git"), []byte("gitdir: ../../.git/worktrees/w1"),
		0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// The walk finds every git working tree under a folder, into a clone for its worktrees, but never into .git, a hidden
// folder, or node_modules, and no deeper than it is let; it stops at so many, and says so.
func TestFindClonesWalksTheFolderForWorkingTrees(t *testing.T) {
	dir := cloneTree(t)
	got, desks, capped := findClones(dir, cloneDepth, cloneMost)
	if strings.Join(got, "|") != "a|a/worktrees/w1|b/c|d/1/2/3/4/5" || capped {
		t.Errorf("found %q, capped %v", got, capped)
	}
	if want := []string{"a", "a", "b/c", "d/1/2/3/4/5"}; len(desks) != len(want) {
		t.Errorf("desks %q", desks)
	} else {
		for k, w := range want {
			if desks[k] != filepath.Join(dir, w) {
				t.Errorf("the desk of %s: %q", got[k], desks[k])
			}
		}
	}
	if got, _, capped := findClones(dir, cloneDepth, 2); strings.Join(got, "|") != "a|a/worktrees/w1" || !capped {
		t.Errorf("at most 2: %q, capped %v", got, capped)
	}
	if got, _, _ := findClones(filepath.Join(dir, "b", "c"), cloneDepth, cloneMost); strings.Join(got, "|") != "." {
		t.Errorf("a clone itself: %q", got)
	}
}

// Home itself is written ~, and with no home known nothing is: a path is written as it is.
func TestOpenWritesHomeItselfAsTilde(t *testing.T) {
	home := t.TempDir()
	if d := newOpen(home, home); d.Typed != "~/" || d.Path() != home {
		t.Errorf("at home: %q, %q", d.Typed, d.Path())
	}
	if d := newOpen("/srv", ""); d.Typed != "/srv/" {
		t.Errorf("with no home: %q", d.Typed)
	}
}

// A path without a / at its start is read from home, and its folders offered from there.
func TestOpenReadsAPathWithoutASlashFromHome(t *testing.T) {
	home := folders(t, "Code/kit", "Codex")
	d, _ := newOpen("", home).keyed(key("ctrl+u"))
	d = typedIn(d, "Co")
	if got := d.Folders(); strings.Join(got, "|") != "Code|Codex" {
		t.Errorf("the folders offered: %q", got)
	}
	d = typedIn(d, "de/kit")
	if _, path := d.keyed(key("enter")); path != filepath.Join(home, "Code", "kit") {
		t.Errorf("asked for %q", path)
	}
}

// A folder that is not there offers none, and says so; tab completes the start folders share, however short.
func TestOpenSaysNoFolderStartsSoAndCompletesWhatFoldersShare(t *testing.T) {
	dir := folders(t, "kitab", "kitcd")
	d := typedIn(newOpen(dir, "/nowhere"), "gone/")
	if got := d.Folders(); got != nil {
		t.Errorf("under a folder not there: %q", got)
	}
	if v := ansi.Strip(strings.Join(d.Overlay(make([]string, 20), 100), "\n")); !strings.Contains(v,
		"no folder here starts so") {
		t.Errorf("drawn:\n%s", v)
	}
	if d, _ = typedIn(newOpen(dir, "/nowhere"), "k").keyed(key("tab")); d.Typed != dir+"/kit" {
		t.Errorf("tab: %q", d.Typed)
	}
}

// The dialog lists so many folders, and counts the rest.
func TestOpenListsEightFoldersAndCountsTheRest(t *testing.T) {
	var names []string
	for k := range openShown + 3 {
		names = append(names, "f"+string(rune('a'+k)))
	}
	d := newOpen(folders(t, names...), "/nowhere")
	v := ansi.Strip(strings.Join(d.Overlay(make([]string, 30), 100), "\n"))
	if !strings.Contains(v, Folder+" fh/") || strings.Contains(v, Folder+" fi/") || !strings.Contains(v, "… 3 more") {
		t.Errorf("drawn:\n%s", v)
	}
}

// Open is dim while nothing is typed, or kit is asked; a click on it then asks nothing, and a right click does nothing
// anywhere. Closed, the dialog draws nothing.
func TestOpenIsDimAndAsksNothingWhileItCannotOpen(t *testing.T) {
	d, _ := newOpen("/", "/home").keyed(key("ctrl+u"))
	body := d.body(60)
	if last := body[len(body)-1]; !last.opens || last.text != style.Off(Opening+" Open") {
		t.Errorf("Open with nothing typed: %+v", last)
	}
	p := d.placed(30, 100)
	at := func(text string) int {
		for k, l := range p.Lines {
			if strings.Contains(ansi.Strip(l), text) {
				return p.Top + k
			}
		}
		t.Fatalf("no line %q", text)
		return 0
	}
	if d2, path := d.clicked(tea.MouseClickMsg{X: p.Left + 3, Y: at(Opening + " Open"), Button: tea.MouseLeft}, 30,
		100); path != "" || !d2.Open {
		t.Errorf("a click on Open with nothing typed: %q, open %v", path, d2.Open)
	}
	if d2, _ := d.clicked(tea.MouseClickMsg{X: 0, Y: 29, Button: tea.MouseRight}, 30, 100); !d2.Open {
		t.Error("a right click off the box closed it")
	}
	if got := (Dialog{}).Overlay([]string{"a"}, 100); strings.Join(got, "|") != "a" {
		t.Errorf("closed, drawn: %q", got)
	}
}

// A key that types nothing, as ←, changes nothing.
func TestOpenTakesAKeyThatTypesNothingAsNothing(t *testing.T) {
	d := newOpen("/srv", "/home")
	if got, path := d.keyed(tea.KeyPressMsg{Code: tea.KeyLeft}); !reflect.DeepEqual(got, d) || path != "" {
		t.Errorf("←: %+v, %q", got, path)
	}
}

// The dialog's keys take every press: with no folder to move over, esc, tab, and enter, as the keys bar names them.
func TestOpenNamesItsKeys(t *testing.T) {
	s := newOpen(t.TempDir(), "").Keys()
	if got := (shell.Path{shell.ScreenKeys(), s}).String(); got != "esc close ↑↓ move ⇥ complete ⇧⌫·⌃W up ⌃F find clones ⏎ open" || !s.Owns {
		t.Errorf("the keys: %q, owns %v", got, s.Owns)
	}
}

// newOpen gives the dialog open as New does, over no workspace.
func newOpen(start, home string) Dialog { return New(start, home, kit.Workspace{}) }

// keyed takes a key as Key does, and gives the folder it asks to open.
func (d Dialog) keyed(k tea.KeyPressMsg) (Dialog, string) {
	d, ask := d.Key(k)
	return d, ask.Open
}

// clicked takes a click as Click does, and gives the folder it asks to open.
func (d Dialog) clicked(msg tea.MouseClickMsg, height, width int) (Dialog, string) {
	d, ask := d.Click(msg, height, width)
	return d, ask.Open
}

// key gives the press a key's name stands for, as testkit.Press does.
func key(s string) tea.KeyPressMsg { return testkit.Press(s) }
