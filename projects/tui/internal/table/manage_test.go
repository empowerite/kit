// The presets dialog lists every view, the favorites first, and stars, orders, renames, and deletes them.
package table

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// The dialog lists the favorites first, a rule under them, then the table's own, then the person's, each with its star
// and where it comes from; Rename… and Delete stand under them for one of the person's own alone. Up and down move;
// space or * stars the view under the keyboard, or no longer; shift with up or down moves a favorite among the
// favorites; r renames one of the person's own, closing the dialog, and d deletes it; esc closes it.
func TestThePresetsDialogListsTheViewsTheFavoritesFirst(t *testing.T) {
	h := presetting()
	h.views.Starred = []string{"Grouped"}
	d := Manage{Open: true}
	if got := d.Listed(h); !slices.Equal(got, []string{"Grouped", Default, "Mine"}) {
		t.Fatalf("listed: %q", got)
	}
	box, top, _ := d.Box(h)
	text := ansi.Strip(strings.Join(box, "\n"))
	if !strings.Contains(text, "things presets") || !strings.Contains(text, Starred+" Grouped") ||
		strings.Contains(text, RenameIt) || d.Row(h, top+1) != -1 || d.Row(h, top+2) != 0 || d.Row(h, top+3) != -1 ||
		d.Row(h, top+4) != 1 || d.Row(h, top+5) != 2 || d.Row(h, top+40) != -1 {
		t.Fatalf("the dialog:\n%s", text)
	}
	if got := ansi.Strip(d.Line(h, "Mine")); !strings.HasPrefix(got, Unstarred+" Mine") || !strings.HasSuffix(got, MineHeading) {
		t.Errorf("Mine's line: %q", got)
	}
	if keys := (shell.Path{d.Keys(h)}).String(); keys != "esc close ↑↓ move ␣ star" {
		t.Errorf("Grouped's keys: %q", keys)
	}
	// on Mine, Rename… and Delete stand under the views, and their keys on the keys bar
	d.Key(h, key("down"))
	d.Key(h, key("down"))
	box, _, _ = d.Box(h)
	if keys := (shell.Path{d.Keys(h)}).String(); !strings.Contains(keys, "r rename") || !strings.Contains(keys, "d delete") ||
		!strings.Contains(ansi.Strip(box[len(box)-2]), RenameIt) {
		t.Errorf("Mine's keys: %q", keys)
	}
	// space stars Mine; shift with up moves it ahead of Grouped; not past the first
	if do := d.Key(h, key("space")); do.Act != ManageFavorites || do.Name != "Mine" ||
		!slices.Equal(do.Favorites, []string{"Grouped", "Mine"}) {
		t.Fatalf("space on Mine: %+v", do)
	}
	h.views.Starred = []string{"Grouped", "Mine"}
	d.At = 1
	if keys := (shell.Path{d.Keys(h)}).String(); !strings.Contains(keys, "⇧↑↓ order") {
		t.Errorf("a favorite's keys: %q", keys)
	}
	if do := d.Key(h, press("shift+up")); !slices.Equal(do.Favorites, []string{"Mine", "Grouped"}) || d.At != 0 {
		t.Errorf("shift+up on Mine: %+v, at %d", do, d.At)
	}
	h.views.Starred = []string{"Mine", "Grouped"}
	if do := d.Key(h, press("shift+up")); do.Act != ManageNothing {
		t.Errorf("shift+up on the first favorite: %+v", do)
	}
	// r renames Mine, closing the dialog; d deletes it; neither acts on a table's own
	if do := d.Key(h, tea.KeyPressMsg{Code: 'r', Text: "r"}); do.Act != ManageRename || do.Name != "Mine" || d.Open {
		t.Errorf("r on Mine: %+v", do)
	}
	d = Manage{Open: true}
	if do := d.Key(h, tea.KeyPressMsg{Code: 'd', Text: "d"}); do.Act != ManageDelete || do.Name != "Mine" {
		t.Errorf("d on Mine: %+v", do)
	}
	d.At = 2
	for _, k := range []string{"r", "d", "x"} {
		if do := d.Key(h, tea.KeyPressMsg{Code: rune(k[0]), Text: k}); do.Act != ManageNothing {
			t.Errorf("%s on Default: %+v", k, do)
		}
	}
	// past the views, nothing acts, and the keys bar offers no star
	d.At = 9
	if do := d.Key(h, key("space")); do.Act != ManageNothing || strings.Contains((shell.Path{d.Keys(h)}).String(), "star") {
		t.Errorf("space past the views: %+v", do)
	}
	if d.Key(h, key("esc")); d.Open {
		t.Error("esc left it open")
	}
}

// A click on a view's star stars it, or no longer, the keyboard on it; one on its name puts the keyboard there; one on
// Rename… or Delete does it; one between them, or on the headings, does nothing; one off the dialog closes it.
func TestThePresetsDialogTakesClicks(t *testing.T) {
	h := presetting()
	d := Manage{Open: true}
	box, top, left := d.Box(h)
	if do := d.Click(h, left+2, top+3); do.Act != ManageFavorites || do.Name != "Grouped" || d.At != 1 {
		t.Errorf("a click on Grouped's star: %+v", do)
	}
	if do := d.Click(h, left+6, top+4); do.Act != ManageNothing || d.At != 2 {
		t.Errorf("a click on Mine's name: %+v, at %d", do, d.At)
	}
	box, top, left = d.Box(h)
	actions, y := d.actions(h), top+len(box)-2
	if do := d.Click(h, left+3, y); do.Act != ManageRename || d.Open {
		t.Errorf("a click on Rename…: %+v", do)
	}
	d = Manage{Open: true, At: 2}
	if do := d.Click(h, left+2+ansi.StringWidth(actions[0].text), y); do.Act != ManageNothing {
		t.Errorf("a click between the actions: %+v", do)
	}
	if do := d.Click(h, left+2+ansi.StringWidth(actions[0].text)+3, y); do.Act != ManageDelete || do.Name != "Mine" {
		t.Errorf("a click on Delete: %+v", do)
	}
	if do := d.Click(h, left+3, top+1); do.Act != ManageNothing || !d.Open {
		t.Errorf("a click on the headings: %+v", do)
	}
	if d.Click(h, 0, 0); d.Open {
		t.Error("a click off it left it open")
	}
}
