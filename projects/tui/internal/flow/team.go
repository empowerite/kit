// The team's presets, as internal/table keeps them: under .kit/presets at the root of the clone kit flow runs in, a
// folder per stage, read only where kit trusts the clone and the kit is installed there, so a clone without it gets no
// .kit/. The presets dialog shares one of the person's with the team,
// moving it there, and copies one of the team's into the person's; renaming, deleting, or saving one of the team's
// changes its file, a change in the working tree the person commits as any other.
package flow

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// The team's words: Shared starts what sharing a preset did, and Copied what copying one did.
const (
	Shared = table.Team
	Copied = table.Mine
)

// teamFolder gives the folder the team's presets are kept in, in the clone the runner runs in: table.TeamDir under its
// root; none where kit does not trust it, it is no clone, or the kit is not installed there, as its kit.yml declares
// no kit version.
func teamFolder(r kit.Runner) string {
	if !r.Trusted() {
		return ""
	}
	return teamUnder(r.Dir)
}

// teamUnder gives table.TeamDir under the root of the clone dir is in, as git says; none where it is in none, or the
// kit is not installed in it.
func teamUnder(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	root := strings.TrimSpace(string(out))
	if !kit.Installed(root) {
		return ""
	}
	return filepath.Join(root, table.TeamDir)
}

// readTeam reads each stage's team presets from the team's folder, by the stages' names; a file that does not read is
// said in place of the keys, and the rest are read.
func (m *Model) readTeam(stages []string) {
	m.team = map[string][]table.Preset{}
	for _, name := range stages {
		ps, err := table.ReadTeam(m.teamDir, name)
		if err != nil {
			m.note.Text, m.note.Done = fmt.Sprintf("%s %v", Failed, err), false
		}
		m.team[name] = ps
	}
}

// shareView moves the person's view of the name into the team's folder, as the presets dialog asks of one of theirs,
// saying what it did, or why it could not.
func (m *Model) shareView(name string) {
	stage := m.stages[m.stage].name
	p, saved, err := m.viewsOf(m.stage).ShareAs(name)
	switch {
	case errors.Is(err, table.ErrTaken):
		m.note.Text, m.note.Done = Failed+" the team has a preset "+name+" already: rename yours first", false
		return
	case err == nil:
		err = table.WriteTeam(m.teamDir, stage, p)
	}
	if err != nil {
		m.note.Text, m.note.Done = fmt.Sprintf("%s the preset could not be shared: %v", Failed, err), false
		return
	}
	m.team[stage], _ = table.ReadTeam(m.teamDir, stage)
	if m.keep(saved, "shared") {
		m.done(Shared + " shared the preset " + name + " with the team, in " + table.TeamDir + "; commit it")
	}
}

// copyView copies the team's view of the name into the person's, under a name of its own, as the presets dialog asks.
func (m *Model) copyView(name string) {
	saved, copy := m.viewsOf(m.stage).CopyOf(name)
	if m.keep(saved, "copied") {
		m.done(Copied + " copied the team's preset " + name + " to yours as " + copy)
	}
}

// deleteTeamView takes the team's view of the name away, its file removed, the stage set as the view it opens on where
// the line was set to it.
func (m *Model) deleteTeamView(name string) {
	stage := m.stages[m.stage].name
	if err := table.RemoveTeam(m.teamDir, stage, name); err != nil {
		m.note.Text, m.note.Done = fmt.Sprintf("%s the preset could not be deleted: %v", Failed, err), false
		return
	}
	m.team[stage], _ = table.ReadTeam(m.teamDir, stage)
	if m.stages[m.stage].ViewName == name {
		m.apply(m.viewsOf(m.stage).Opening())
	}
	m.done(Deleted + " deleted the team's preset " + name + "; commit it")
}

// saveTeam keeps the selected stage's panel as the team's view of the name, in its file, as Save does on one of theirs.
func (m *Model) saveTeam(name string) {
	stage, s := m.stages[m.stage].name, &m.stages[m.stage]
	p := m.viewsOf(m.stage).Current(s.View)
	p.Name = name
	if err := table.WriteTeam(m.teamDir, stage, p); err != nil {
		m.note.Text, m.note.Done = fmt.Sprintf("%s the preset could not be saved: %v", Failed, err), false
		return
	}
	m.team[stage], _ = table.ReadTeam(m.teamDir, stage)
	m.done(Save + " saved the team's preset " + name + "; commit it")
}

// renameTeam renames the team's view of a name to another, its file written under the new name's slug and the old
// one's removed, its star following it, and says whether it did; a name refused, or a file that could not be written,
// says why.
func (m *Model) renameTeam(from, to string) bool {
	stage := m.stages[m.stage].name
	v := m.viewsOf(m.stage)
	p, err := v.RenameShared(from, to)
	if err != nil {
		m.refuseName(to, err)
		return false
	}
	if err = table.WriteTeam(m.teamDir, stage, p); err == nil && table.Slug(from) != table.Slug(to) {
		err = table.RemoveTeam(m.teamDir, stage, from)
	}
	if err == nil {
		err = table.KeepFavorites(m.presetsPath, stage, v.Renamed(from, to))
	}
	if err != nil {
		m.note.Text, m.note.Done = fmt.Sprintf("%s the preset could not be renamed: %v", Failed, err), false
		return false
	}
	m.team[stage], _ = table.ReadTeam(m.teamDir, stage)
	m.starred[stage] = v.Renamed(from, to)
	return true
}
