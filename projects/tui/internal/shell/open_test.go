// The Open… dialog takes a folder typed, completes it a folder at a time, and asks to open it.
package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
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
func typedIn(d OpenDialog, text string) OpenDialog {
	for _, r := range text {
		d, _ = d.Key(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return d
}

func TestOpenStartsAtTheFolderGivenWritingHomeAsTilde(t *testing.T) {
	home := t.TempDir()
	d := NewOpen(filepath.Join(home, "Code"), home)
	if !d.Open || d.Typed != "~/Code/" {
		t.Errorf("opened: %+v", d)
	}
	if d := NewOpen("/srv", home); d.Typed != "/srv/" {
		t.Errorf("outside home: %q", d.Typed)
	}
	if d := NewOpen("", home); d.Typed != "~/" {
		t.Errorf("with none: %q", d.Typed)
	}
}

func TestOpenTypesErasesAndClears(t *testing.T) {
	d := typedIn(NewOpen("/", "/home"), "srv")
	if d.Typed != "/srv" {
		t.Errorf("typed: %q", d.Typed)
	}
	d, _ = d.Key(key("backspace"))
	if d.Typed != "/sr" {
		t.Errorf("erased: %q", d.Typed)
	}
	d.Refusal = "x"
	d, _ = d.Key(key("ctrl+u"))
	if d.Typed != "" || d.Refusal != "" {
		t.Errorf("cleared: %+v", d)
	}
	if d, path := d.Key(key("enter")); path != "" || !d.Open {
		t.Errorf("enter with nothing typed: %q, open %v", path, d.Open)
	}
	if d, _ := d.Key(key("esc")); d.Open {
		t.Error("esc leaves it open")
	}
}

func TestOpenTabCompletesAFolderAtATime(t *testing.T) {
	dir := folders(t, "kit", "kittens", "edge", ".hidden")
	d := NewOpen(dir, "/nowhere")
	if got := d.Folders(); strings.Join(got, "|") != "edge|kit|kittens" {
		t.Errorf("the folders offered: %q", got)
	}
	d = typedIn(d, "k")
	if got := d.Folders(); strings.Join(got, "|") != "kit|kittens" {
		t.Errorf("after k: %q", got)
	}
	d, _ = d.Key(key("tab"))
	if d.Typed != dir+"/kit" {
		t.Errorf("tab to the common part: %q", d.Typed)
	}
	d = typedIn(d, "t")
	d, _ = d.Key(key("tab"))
	if d.Typed != dir+"/kittens/" {
		t.Errorf("tab to the one folder: %q", d.Typed)
	}
	d = typedIn(NewOpen(dir, "/nowhere"), ".")
	if got := d.Folders(); strings.Join(got, "|") != ".hidden" {
		t.Errorf("a dot offers the hidden ones: %q", got)
	}
	d = typedIn(NewOpen(dir, "/nowhere"), "zzz")
	if d, _ = d.Key(key("tab")); d.Typed != dir+"/zzz" {
		t.Errorf("tab with none: %q", d.Typed)
	}
}

func TestOpenEnterAsksForTheFolderWhole(t *testing.T) {
	home := folders(t, "Code/kit")
	d := NewOpen(filepath.Join(home, "Code"), home)
	d = typedIn(d, "kit")
	d, path := d.Key(key("enter"))
	if path != filepath.Join(home, "Code", "kit") || !d.Open || !d.Busy {
		t.Errorf("enter: %q, %+v", path, d)
	}
	// while kit is asked, nothing more is asked
	if _, again := d.Key(key("enter")); again != "" {
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
	rel := typedIn(NewOpen("", home), "Code/../Code/kit")
	if _, path := rel.Key(key("enter")); path != filepath.Join(home, "Code", "kit") {
		t.Errorf("a path written from ~: %q", path)
	}
}

// Every line of the dialog does what it says on a plain click: a folder completes to it, Open opens what is typed, and
// a click off the box closes it.
func TestOpenIsDoneByPlainClicks(t *testing.T) {
	dir := folders(t, "edge", "kit")
	d := NewOpen(dir, "/nowhere")
	p := d.placed(30, 100)
	lines := ansi.Strip(strings.Join(p.Lines, "\n"))
	if !strings.Contains(lines, Folder+" edge/") || !strings.Contains(lines, Opening+" Open") {
		t.Fatalf("the box:\n%s", lines)
	}
	row := func(d OpenDialog, text string) int {
		p := d.placed(30, 100)
		for k, l := range p.Lines {
			if strings.Contains(ansi.Strip(l), text) {
				return p.Top + k
			}
		}
		t.Fatalf("no line %q", text)
		return 0
	}
	click := func(d OpenDialog, y int) (OpenDialog, string) {
		return d.Click(tea.MouseClickMsg{X: p.Left + 3, Y: y, Button: tea.MouseLeft}, 30, 100)
	}
	d2, path := click(d, row(d, "kit/"))
	if path != "" || d2.Typed != dir+"/kit/" {
		t.Errorf("a click on kit/: %q, %q", path, d2.Typed)
	}
	if _, path := click(d2, row(d2, Opening+" Open")); path != dir+"/kit" {
		t.Errorf("a click on Open: %q", path)
	}
	if d3, _ := d.Click(tea.MouseClickMsg{X: 0, Y: 29, Button: tea.MouseLeft}, 30, 100); d3.Open {
		t.Error("a click off the box leaves it open")
	}
}
