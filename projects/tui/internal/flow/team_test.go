// The team's presets: read from the clone kit trusts, shared, copied, saved, renamed, and deleted a file at a time.
package flow

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// The team's folder is .kit/presets at the root of the clone kit flow runs in, where kit trusts it; none where it does
// not, or where there is no clone.
func TestTheTeamsFolderIsInTheCloneKitTrusts(t *testing.T) {
	repo, _ := filepath.EvalSymlinks(t.TempDir())
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	r := runner(t)
	r.Dir = filepath.Join(repo)
	if got := teamFolder(r); got != "" {
		t.Errorf("an untrusted clone: %q", got)
	}
	t.Setenv("FAKE_FLOW_TRUSTED", "1")
	if got := teamFolder(r); got != filepath.Join(repo, table.TeamDir) {
		t.Errorf("a trusted clone: %q", got)
	}
	r.Dir = t.TempDir()
	if got := teamFolder(r); got != "" {
		t.Errorf("no clone: %q", got)
	}
}

// teamed gives the screen on the presets' menu's line, its person's presets in file and the team's in a folder that
// holds Ours for issues, as a clone kit trusts does.
func teamed(t *testing.T) (m Model, file, team string) {
	t.Helper()
	file, team = filepath.Join(t.TempDir(), "presets.json"), filepath.Join(t.TempDir(), table.TeamDir)
	if err := table.WriteTeam(team, "issues", table.Preset{Name: "Ours"}); err != nil {
		t.Fatal(err)
	}
	o := viewing(file)
	o.Team = team
	return toPresets(loaded(t, o, tea.WindowSizeMsg{Width: 140, Height: 40})), file, team
}

// teamFile says whether the team's folder holds the file of issues' preset of the slug.
func teamFile(team, slug string) bool {
	_, err := os.Stat(filepath.Join(team, "issues", slug+".json"))
	return err == nil
}

// The team's presets stand in the presets' menu under team. One of the person's, shared, moves to the team's folder;
// one of the team's, copied, is the person's under a name of its own; saved, renamed, or deleted, the team's file
// changes; and each says so, and to commit it.
func TestTheTeamsPresetsAreSharedCopiedAndChanged(t *testing.T) {
	m, _, team := teamed(t)
	if items := table.PresetItems(&m); items[table.ItemViews+1].Name != table.TeamHeading || items[table.ItemViews+2].Name != "Ours" {
		t.Fatalf("the menu: %+v", items)
	}
	// Triage, saved, then shared: it is the team's, and no longer the person's
	m.choosePreset(table.ItemSaveAs)
	m.stages[0].Naming = "Triage"
	m = drive(m, press(tea.KeyEnter))
	if m.managedPreset(table.ManageDo{Act: table.ManageShare, Name: "Triage"}); !teamFile(team, "triage") ||
		len(m.viewsOf(0).Saved) != 0 || !strings.Contains(m.note.Text, "shared the preset Triage with the team") {
		t.Fatalf("Triage shared: %q", m.note.Text)
	}
	// a name the team has is refused for the person's, and for sharing
	m.choosePreset(table.ItemSaveAs)
	m.stages[0].Naming = "TRIAGE"
	if m = drive(m, press(tea.KeyEnter)); !strings.Contains(m.note.Text, "is the team's preset") {
		t.Errorf("save as the team's: %q", m.note.Text)
	}
	m = drive(m, esc)
	m.saved["issues"] = []table.Preset{{Name: "triage!"}}
	if m.managedPreset(table.ManageDo{Act: table.ManageShare, Name: "triage!"}); !strings.Contains(m.note.Text, "the team has a preset") {
		t.Errorf("share over the team's: %q", m.note.Text)
	}
	m.saved["issues"] = nil
	// Ours, copied, is the person's as Ours copy
	if m.managedPreset(table.ManageDo{Act: table.ManageCopy, Name: "Ours"}); !slices.ContainsFunc(m.viewsOf(0).Saved,
		func(p table.Preset) bool { return p.Name == "Ours copy" }) || !strings.Contains(m.note.Text, "as Ours copy") {
		t.Errorf("Ours copied: %q", m.note.Text)
	}
	// set as Ours, then changed, Save keeps the panel in Ours' file
	ours, _ := m.viewsOf(0).Shared("Ours")
	m.apply(ours)
	m.setPanel(m.panel(0).Placed("age", grid.Hidden))
	if m.choosePreset(table.ItemSave); !strings.Contains(m.note.Text, "saved the team's preset Ours") {
		t.Errorf("Save on Ours: %q", m.note.Text)
	}
	if p, _ := m.viewsOf(0).Shared("Ours"); !m.same(0, p, m.current(0)) {
		t.Error("Ours' file does not hold the panel")
	}
	// starred, then renamed, Ours is Theirs, its file and its star with it; to the same slug, it keeps its file
	m.favor("Ours", []string{"Ours"})
	m.managedPreset(table.ManageDo{Act: table.ManageRename, Name: "Ours"})
	m.stages[0].Naming = "Theirs"
	if m = drive(m, press(tea.KeyEnter)); !teamFile(team, "theirs") || teamFile(team, "ours") ||
		m.stages[0].ViewName != "Theirs" || !slices.Equal(m.viewsOf(0).Favorites(), []string{"Theirs"}) {
		t.Fatalf("renamed to Theirs: %q", m.note.Text)
	}
	m.managedPreset(table.ManageDo{Act: table.ManageRename, Name: "Theirs"})
	m.stages[0].Naming = "THEIRS"
	if m = drive(m, press(tea.KeyEnter)); !teamFile(team, "theirs") || m.stages[0].ViewName != "THEIRS" {
		t.Errorf("renamed to its slug: %q", m.note.Text)
	}
	m.managedPreset(table.ManageDo{Act: table.ManageRename, Name: "THEIRS"})
	m.stages[0].Naming = "Compact"
	if m = drive(m, press(tea.KeyEnter)); !strings.Contains(m.note.Text, "is the stage's own") {
		t.Errorf("renamed to a stage's own: %q", m.note.Text)
	}
	m = drive(m, esc)
	// deleted, its file goes, and the stage is set as the view it opens on
	if m.managedPreset(table.ManageDo{Act: table.ManageDelete, Name: "THEIRS"}); teamFile(team, "theirs") ||
		m.is(0) != "All Issues" || !strings.Contains(m.note.Text, "deleted the team's preset THEIRS") {
		t.Errorf("deleted: %q, is %q", m.note.Text, m.is(0))
	}
	if m.managedPreset(table.ManageDo{Act: table.ManageDelete, Name: "Triage"}); teamFile(team, "triage") ||
		m.is(0) != "All Issues" {
		t.Errorf("Triage deleted while the stage is All Issues: %q", m.note.Text)
	}
}

// A team's folder that cannot be written says so, and changes nothing: sharing, saving, renaming, and deleting; a
// presets file that cannot be written keeps a rename's star from following; a file in the team's folder that does not
// read is said as the screen opens.
func TestTheTeamsPresetsSayWhatCouldNotBeDone(t *testing.T) {
	m, file, team := teamed(t)
	m.choosePreset(table.ItemSaveAs)
	m.stages[0].Naming = "Mine"
	m = drive(m, press(tea.KeyEnter))
	// the team's folder moved under a file, so nothing is written there, and Ours' file is no longer there
	m.teamDir = filepath.Join(file, "under")
	if m.managedPreset(table.ManageDo{Act: table.ManageShare, Name: "Mine"}); !strings.Contains(m.note.Text, "could not be shared") {
		t.Errorf("share: %q", m.note.Text)
	}
	ours, _ := m.viewsOf(0).Shared("Ours")
	m.apply(ours)
	m.setPanel(m.panel(0).Placed("age", grid.Hidden))
	if m.choosePreset(table.ItemSave); !strings.Contains(m.note.Text, "could not be saved") {
		t.Errorf("save: %q", m.note.Text)
	}
	if m.managedPreset(table.ManageDo{Act: table.ManageDelete, Name: "Ours"}); !strings.Contains(m.note.Text, "could not be deleted") {
		t.Errorf("delete: %q", m.note.Text)
	}
	m.managedPreset(table.ManageDo{Act: table.ManageRename, Name: "Ours"})
	m.stages[0].Naming = "Theirs"
	if m = drive(m, press(tea.KeyEnter)); !strings.Contains(m.note.Text, "could not be renamed") || m.stages[0].ViewName != "Ours" {
		t.Errorf("rename: %q", m.note.Text)
	}
	m = drive(m, esc)
	// the folder back, the presets file under a file: the rename's star cannot follow it
	m.teamDir, m.presetsPath = team, filepath.Join(file, "under", "presets.json")
	m.managedPreset(table.ManageDo{Act: table.ManageRename, Name: "Ours"})
	m.stages[0].Naming = "Theirs"
	if m = drive(m, press(tea.KeyEnter)); !strings.Contains(m.note.Text, "could not be renamed") {
		t.Errorf("rename, the presets file unwritable: %q", m.note.Text)
	}
	// a file that does not read, said as the screen opens; the rest are read
	if err := os.WriteFile(filepath.Join(team, "issues", "bad.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	o := viewing(file)
	o.Team = team
	if n := New(runner(t), o); !strings.Contains(n.note.Text, "does not read as a preset") || len(n.team["issues"]) == 0 {
		t.Errorf("a file that does not read: %q", n.note.Text)
	}
}
