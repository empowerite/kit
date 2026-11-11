// A field its preset pins no rendering for is drawn by one automatic chooses: without a glyph every row shown shares,
// and a step down, from the rightmost column leftward, while the columns are short of width; the screen draws it so.
package flow

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// fieldIn gives a stage's field by its id.
func fieldIn(stage, id string) table.Field {
	fields := fieldsOf(stage)
	return fields[slices.IndexFunc(fields, func(f table.Field) bool { return f.ID == id })]
}

// columns gives a stage's fields as columns, in the order named, without names over them, none pinned.
func columns(stage string, ids ...string) []table.Placed {
	var out []table.Placed
	for _, id := range ids {
		out = append(out, table.Placed{Field: fieldIn(stage, id), Column: true})
	}
	return out
}

// asCatalog pins each of stage i's fields to its first rendering, so the rows draw each as the catalog first does, its
// glyph and all, whatever the rows share and the width.
func asCatalog(m *Model, i int) {
	p := m.stages[i].View
	for _, f := range fieldsOf(m.stages[i].name) {
		p = p.Rendered(f.ID, f.Renderings[0])
	}
	m.stages[i].View = p
	m.rebuild()
}

// sources gives the values of rows from these trackers.
func sources(names ...string) []table.Value {
	var out []table.Value
	for _, n := range names {
		out = append(out, table.Some(n))
	}
	return out
}

func TestAGlyphEveryRowSharesIsDropped(t *testing.T) {
	fields := columns("backlog", "source", "title")
	titles := []table.Value{table.One("Fix the thing"), table.One("Lint the modules")}
	got := table.Automatic(fields, map[string][]table.Value{"source": sources("github", "github"), "title": titles}, 200, false)
	if got["source"] != "text" || got["title"] != "text" {
		t.Errorf("every row on github, every title with its glyph: %v", got)
	}
	// the trackers differ, so the glyph says which: it stays
	if got := table.Automatic(fields, map[string][]table.Value{"source": sources("github", "gitlab")}, 200, false); got["source"] != "glyph+text" {
		t.Errorf("github and gitlab: %v", got)
	}
	// a row without a value draws no glyph, and does not count; one row alone shares its glyph with none, and keeps it;
	// no value at all leaves the field's first
	if got := table.Automatic(fields, map[string][]table.Value{"source": {{}, table.Some("gitlab"), table.Some("gitlab")}}, 200, false); got["source"] != "text" {
		t.Errorf("gitlab twice and a row without a source: %v", got)
	}
	if got := table.Automatic(fields, map[string][]table.Value{"source": {{}, table.Some("gitlab")}}, 200, false); got["source"] != "glyph+text" {
		t.Errorf("gitlab once and a row without a source: %v", got)
	}
	if got := table.Automatic(fields, map[string][]table.Value{"source": {{}, {}}}, 200, false); got["source"] != "glyph+text" {
		t.Errorf("no source: %v", got)
	}
	// an enum whose value has no glyph shares none
	event := columns("builds", "event")
	if got := table.Automatic(event, map[string][]table.Value{"event": sources("push", "push")}, 200, false); got["event"] != "text" {
		t.Errorf("events: %v", got)
	}
	// a field whose renderings all draw the glyph keeps its first
	latest := columns("releases", "latest")
	if got := table.Automatic(latest, map[string][]table.Value{"latest": sources("latest")}, 200, false); got["latest"] != "glyph+text" {
		t.Errorf("latest: %v", got)
	}
	// a checkout's name drops the glyph but keeps that it is detached
	branch := columns("workspace/repos", "branch")
	trees := []table.Value{{Raw: []string{"main"}, Of: checkout{}}, {Raw: []string{"spike"}, Of: checkout{Detached: true}}}
	if got := table.Automatic(branch, map[string][]table.Value{"branch": trees}, 200, false); got["branch"] != "name" ||
		branch[0].Draw(trees, got["branch"])[1].Values[0] != "spike "+Detached+" detached" {
		t.Errorf("a member's branch: %v", got)
	}
	// the panel showing the emoji every row shares, the glyph stays
	if got := table.Automatic(fields, map[string][]table.Value{"source": sources("github", "github")}, 200, true); got["source"] != "glyph+text" {
		t.Errorf("shared emoji shown: %v", got)
	}
	// a rendering pinned wins
	fields[0].Pinned = "glyph+text"
	if got := table.Automatic(fields, map[string][]table.Value{"source": sources("github", "github")}, 200, false); got["source"] != "glyph+text" {
		t.Errorf("source pinned: %v", got)
	}
}

func TestATableShortOfWidthStepsDownFromTheRightmostColumn(t *testing.T) {
	// changes' state and checks, both of several values, and the title that wraps
	fields := columns("changes", "state", "checks", "title")
	vs := map[string][]table.Value{
		"state":  sources("ready", "draft"),
		"checks": {{Raw: []string{"SUCCESS"}, Of: &checks{State: "SUCCESS", Total: 3}}, {Raw: []string{"FAILURE"}, Of: &checks{State: "FAILURE", Failing: "lint", Passed: 1, Total: 3}}},
		"title":  {table.One("A title long enough to want the width")},
	}
	need := func(id, rendering string) int {
		f := fields[slices.IndexFunc(fields, func(f table.Placed) bool { return f.ID == id })]
		return grid.Need(f.Draw(vs[id], rendering), "")
	}
	state, checks := need("state", "glyph+text"), need("checks", "checks")
	wide := state + 2 + checks + 2 + 12
	if got := table.Automatic(fields, vs, wide, false); got["state"] != "glyph+text" || got["checks"] != "checks" || got["title"] != "text" {
		t.Errorf("wide enough, at %d: %v", wide, got)
	}
	// a cell short, the rightmost column that can steps down first
	if got := table.Automatic(fields, vs, wide-1, false); got["state"] != "glyph+text" || got["checks"] != "glyph" {
		t.Errorf("a cell short, at %d: %v", wide-1, got)
	}
	// shorter than that, the next one leftward steps too, a step at a time: its text, without the glyph, then its glyph
	short := state + 2 + need("checks", "glyph") + 2 + 12 - 1
	if got := table.Automatic(fields, vs, short, false); got["state"] != "text" || got["checks"] != "glyph" || got["title"] != "text" {
		t.Errorf("shorter, at %d: %v", short, got)
	}
	shorter := need("state", "text") + 2 + need("checks", "glyph") + 2 + 12 - 1
	if got := table.Automatic(fields, vs, shorter, false); got["state"] != "glyph" || got["checks"] != "glyph" {
		t.Errorf("shorter still, at %d: %v", shorter, got)
	}
	// none fits: each steps as far as it can, and the title stays
	if got := table.Automatic(fields, vs, 10, false); got["state"] != "glyph" || got["checks"] != "glyph" || got["title"] != "text" {
		t.Errorf("at 10: %v", got)
	}
	// a column pinned never steps, and the one left of it steps in its place
	pinned := slices.Clone(fields)
	pinned[1].Pinned = "checks"
	if got := table.Automatic(pinned, vs, wide-1, false); got["state"] != "text" || got["checks"] != "checks" {
		t.Errorf("checks pinned, at %d: %v", wide-1, got)
	}
	// a field not a column takes no width, and never steps
	grouped := slices.Clone(fields)
	grouped[1].Column = false
	if got := table.Automatic(grouped, vs, state+2+12, false); got["state"] != "glyph+text" || got["checks"] != "checks" {
		t.Errorf("checks grouped: %v", got)
	}
	// the name the header stands over a column widens it
	named := slices.Clone(fields)
	named[0].Name = strings.Repeat("s", state+4)
	if got := table.Automatic(named, vs, wide, false); got["checks"] != "glyph" {
		t.Errorf("a wide name over state, at %d: %v", wide, got)
	}
}

// The screen draws the choice: a glyph backlog's rows all share is gone from them, and from the group's heading, but
// the title's, which leads to the issue's page, stays; a search that matched a glyph's word still lets its rows
// through once the field is drawn without it; and changes at 80 columns draws its checks by their glyph, while at 160
// it has the width for their counts. A rendering pinned in the popover draws the glyph again, and so does the panel's
// box that shows the emoji every row shares, for every field.
func TestTheScreenDrawsTheRenderingsAutomaticChooses(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 120, Height: 24})
	var pane []string
	for _, l := range m.allLines(0) {
		pane = append(pane, ansi.Strip(l.Text))
	}
	text := strings.Join(pane, "\n")
	if !strings.Contains(text, Issue+" Fix the thing") || strings.Contains(text, GitHub) ||
		!strings.Contains(text, "▾ github") {
		t.Errorf("backlog keeps a glyph every row shares, or drops the title's:\n%s", text)
	}
	if got := m.shown(0).drawn; got["source"] != "text" || got["title"] != "glyph+text" || got["id"] != "text" {
		t.Errorf("backlog's renderings: %v", got)
	}
	// a search for 🐙, which the sources no longer draw, still lets every row through
	p := m.panel(0)
	p.Query = GitHub
	m.setPanel(p)
	if n := table.Counted(m.every(0)); n != 3 {
		t.Errorf("a search for the sources' glyph lets %d rows through", n)
	}
	// pinned, the source draws its glyph again
	p.Query = ""
	m.setPanel(p.Rendered("source", "glyph+text"))
	if !strings.Contains(raw(m), GitHub+" github") {
		t.Errorf("source pinned to glyph+text:\n%s", raw(m))
	}
	// kittui's setting to show the emoji every row shares draws each again, in every view: the table is still as its
	// preset sets it
	setting := shell.Settings{FoldShared: true, SharedEmoji: true}
	n, _ := m.Update(shell.SettingsMsg{Settings: setting})
	m = n.(Model)
	if got := m.shown(0).drawn; got["source"] != "glyph+text" || got["assignees"] != "glyph+text" || !m.panel(0).Glyphs {
		t.Errorf("shared emoji shown: %v", got)
	}
	setting.SharedEmoji = false
	n, _ = m.Update(shell.SettingsMsg{Settings: setting})
	m = n.(Model)
	changes := slices.Index(six, "changes")
	at := func(width int) map[string]string {
		m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: width, Height: 24})
		return m.shown(changes).drawn
	}
	if narrow, wide := at(80), at(160); narrow["checks"] != "glyph" || wide["checks"] != "checks" {
		t.Errorf("changes' checks at 80: %q, at 160: %q", narrow["checks"], wide["checks"])
	}
}
