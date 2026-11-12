// A team's presets are kept a file each, by slug, and stand beside the table's own and the person's.
package table

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
)

func TestASlugIsTheNameInLowerCaseDashed(t *testing.T) {
	for name, want := range map[string]string{"Mine: open": "mine-open", "  By Owner ": "by-owner", "ÉtÉ 2": "été-2",
		"***": "preset", "a/b": "a-b"} {
		if got := Slug(name); got != want {
			t.Errorf("Slug(%q) = %q, not %q", name, got, want)
		}
	}
}

// A team preset is written in its table's folder, a file by its slug, read back by its names' order, and removed; a
// file that does not read, or names no preset, is the error, and the rest are read; no folder named reads none.
func TestATeamsPresetsAreAFileEach(t *testing.T) {
	dir := filepath.Join(t.TempDir(), TeamDir)
	if ps, err := ReadTeam("", "issues"); ps != nil || err != nil {
		t.Errorf("no folder: %v %v", ps, err)
	}
	if ps, err := ReadTeam(dir, "issues"); len(ps) != 0 || err != nil {
		t.Errorf("no presets: %v %v", ps, err)
	}
	for _, name := range []string{"Triage", "by owner"} {
		if err := WriteTeam(dir, "welcome/terms", Preset{Name: name}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "welcome", "terms", "by-owner.json")); err != nil {
		t.Errorf("by owner's file: %v", err)
	}
	ps, err := ReadTeam(dir, "welcome/terms")
	if err != nil || len(ps) != 2 || ps[0].Name != "by owner" || ps[1].Name != "Triage" {
		t.Fatalf("read back: %+v %v", ps, err)
	}
	for file, data := range map[string]string{"bad.json": "{", "nameless.json": "{}"} {
		if err := os.WriteFile(filepath.Join(dir, "welcome", "terms", file), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if ps, err := ReadTeam(dir, "welcome/terms"); len(ps) != 2 || err == nil || !strings.Contains(err.Error(), "nameless.json") {
		t.Errorf("files that do not read: %+v %v", ps, err)
	}
	if err := RemoveTeam(dir, "welcome/terms", "Triage"); err != nil || RemoveTeam(dir, "welcome/terms", "Triage") == nil {
		t.Errorf("removed: %v", err)
	}
	// a folder that cannot be made, as a file stands where it would be, nor written in
	if err := WriteTeam(filepath.Join(dir, "welcome", "terms", "bad.json"), "x", Preset{Name: "y"}); err == nil {
		t.Error("written under a file")
	}
	locked := filepath.Join(t.TempDir(), "locked")
	if err := os.MkdirAll(filepath.Join(locked, "issues"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(locked, "issues"), 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(locked, "issues"), 0o700) })
	if err := WriteTeam(locked, "issues", Preset{Name: "y"}); err == nil {
		t.Error("written in a folder that cannot be written to")
	}
}

// The team's views stand among the table's views, after its own, before the person's; a name the team has, by its
// slug, is taken for the person's; sharing one of the person's moves it to the team; a copy of the team's takes a
// name of its own; and renaming the team's refuses what renaming the person's refuses.
func TestTheTeamsViewsStandBesideTheTablesAndThePersons(t *testing.T) {
	h := presetting()
	h.views.Team, h.views.Shares = []Preset{{Name: "Ours"}, {Name: "Ours copy"}}, true
	v := h.views
	if names := itemNames(h); !slices.Equal(names, []string{SaveIt, SaveAs, ManageIt, "", TeamHeading, "Ours", "Ours copy",
		BuiltinHeading, Default, "Grouped", MineHeading, "Mine"}) {
		t.Fatalf("the menu: %q", names)
	}
	if p, ok := v.Shared("Ours"); !ok || p.Name != "Ours" || len(v.All()) != 5 {
		t.Errorf("Ours: %+v", p)
	}
	if _, err := v.SaveAs("ours", h.panel); !errors.Is(err, ErrTaken) {
		t.Errorf("save as the team's: %v", err)
	}
	if _, err := v.Rename("Mine", "OURS"); !errors.Is(err, ErrTaken) {
		t.Errorf("rename to the team's: %v", err)
	}
	if p, saved, err := v.ShareAs("Mine"); err != nil || p.Name != "Mine" || len(saved) != 0 {
		t.Errorf("share Mine: %+v %v %v", p, saved, err)
	}
	if _, _, err := v.ShareAs("Ours"); !errors.Is(err, ErrUnnamed) {
		t.Errorf("share the team's own: %v", err)
	}
	v.Saved = append(v.Saved, Preset{Name: "ours!"})
	if _, _, err := v.ShareAs("ours!"); !errors.Is(err, ErrTaken) {
		t.Errorf("share over the team's: %v", err)
	}
	if saved, name := v.CopyOf("Ours"); name != "Ours copy 2" || saved[len(saved)-1].Name != name {
		t.Errorf("a copy of Ours: %q", name)
	}
	if p, err := v.RenameShared("Ours", "Theirs"); err != nil || p.Name != "Theirs" {
		t.Errorf("rename Ours: %+v %v", p, err)
	}
	for to, want := range map[string]error{"": ErrUnnamed, Default: ErrBuiltin, "ours copy": ErrTaken, "Mine": ErrTaken} {
		if _, err := v.RenameShared("Ours", to); !errors.Is(err, want) {
			t.Errorf("rename Ours to %q: %v", to, err)
		}
	}
	// Save acts on a team view the table has changed from
	h.tbl.ViewName = "Ours"
	h.tbl.View = h.tbl.View.Placed("state", grid.Hidden)
	if !PresetEnabled(h, ItemSave) {
		t.Error("Save on a changed team view")
	}
}

// The presets dialog says where each view comes from; on one of the person's it offers Share with team, where the
// table keeps the team's; on one of the team's, Copy to mine, Rename…, and Delete; and their keys, and clicks on
// them, ask for them.
func TestThePresetsDialogSharesAndCopies(t *testing.T) {
	h := presetting()
	h.views.Team, h.views.Shares = []Preset{{Name: "Ours"}}, true
	d := Manage{Open: true}
	if got := ansi.Strip(d.Line(h, "Ours")); !strings.HasSuffix(got, TeamHeading) {
		t.Errorf("Ours' line: %q", got)
	}
	d.At = slices.Index(d.Listed(h), "Mine")
	if keys := (shell.Path{d.Keys(h)}).String(); !strings.Contains(keys, "s share") {
		t.Errorf("Mine's keys: %q", keys)
	}
	if do := d.Key(h, tea.KeyPressMsg{Code: 's', Text: "s"}); do.Act != ManageShare || do.Name != "Mine" {
		t.Errorf("s on Mine: %+v", do)
	}
	box, top, left := d.Box(h)
	share := left + 2 + strings.Index(ansi.Strip(box[len(box)-2]), Team)
	if do := d.Click(h, share, top+len(box)-2); do.Act != ManageShare {
		t.Errorf("a click on Share with team: %+v", do)
	}
	d.At = slices.Index(d.Listed(h), "Ours")
	if keys := (shell.Path{d.Keys(h)}).String(); !strings.Contains(keys, "c copy") || !strings.Contains(keys, "r rename") ||
		strings.Contains(keys, "s share") {
		t.Errorf("Ours' keys: %q", keys)
	}
	if do := d.Key(h, tea.KeyPressMsg{Code: 'c', Text: "c"}); do.Act != ManageCopy || do.Name != "Ours" {
		t.Errorf("c on Ours: %+v", do)
	}
	if do := d.Key(h, tea.KeyPressMsg{Code: 's', Text: "s"}); do.Act != ManageNothing {
		t.Errorf("s on Ours: %+v", do)
	}
	// where the table keeps no team's, there is no Share
	h.views.Shares = false
	d.At = slices.Index(d.Listed(h), "Mine")
	if keys := (shell.Path{d.Keys(h)}).String(); strings.Contains(keys, "share") {
		t.Errorf("Mine's keys, no team's: %q", keys)
	}
}
