// The customize step as a table: the page read as parts, the settings each names, the table drawn under the lead
// in the page's order, its presets, its search, and its keys, as the screen hands them to it; its screens pinned as
// golden files; a page without parts stays a page.
package welcome

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

func TestAPageIsReadAsItsParts(t *testing.T) {
	md := "<!-- kit:generated -- customize -->\n# Customize\n\nA start.\n\n## Packages\n\n`packages:` in " +
		"`kit.yml` lists them.\n\n`kit packages`\n\n## Primary, and `forges`\n\n`primary:` names the remote. " +
		"`forges:` names a forge, and `primary:` again.\n\n`kit flow incoming --no-fetch`\n\n`kit flow`\n\n" +
		"## A house package\n\nIt holds rules.\n\n- To add a file, place it.\n- To replace one, depend\n  on it.\n" +
		"1. A step.\n\n## The docs\n\nAt <https://github.com/empowerite/kit>, and [the guide](../1st.md).\n"
	lead, sections := parseSections(md)
	if lead != "# Customize\n\nA start." || len(sections) != 4 {
		t.Fatalf("the lead %q and the sections %+v", lead, sections)
	}
	want := "It holds rules. To add a file, place it. To replace one, depend on it. A step."
	if sections[2].text != want {
		t.Errorf("a list's items are the text without their markers: %q", sections[2].text)
	}
	rs := partRecords(sections)
	values := func(k int, id string) []string { return rs[k].Values[id].Raw }
	if rs[1].Key != "1 Primary, and forges" || !reflect.DeepEqual(values(1, "part"), []string{"Primary, and forges"}) {
		t.Errorf("a heading is one part, a title, not names: %q %v", rs[1].Key, values(1, "part"))
	}
	for k, want := range [][]string{{"packages"}, {"primary", "forges"}, nil, nil} {
		if got := values(k, "settings"); !reflect.DeepEqual(got, want) {
			t.Errorf("the settings of %q: %v, want %v", rs[k].Key, got, want)
		}
	}
	for k, want := range []string{"1", "2", "3", "4"} {
		if got := values(k, "order"); !reflect.DeepEqual(got, []string{want}) {
			t.Errorf("the order of %q: %v", rs[k].Key, got)
		}
	}
	if !reflect.DeepEqual(values(1, "commands"), []string{"kit flow incoming --no-fetch", "kit flow"}) ||
		!reflect.DeepEqual(values(1, "command"), []string{"kit flow"}) || values(2, "commands") != nil ||
		values(2, "command") != nil {
		t.Errorf("the commands: %+v", rs)
	}
	want = "At https://github.com/empowerite/kit, and the guide."
	if got := values(3, "about"); !reflect.DeepEqual(got, []string{want}) {
		t.Errorf("an autolink and a link as plain text: %v", got)
	}
	if got := values(0, "about"); !reflect.DeepEqual(got, []string{"packages: in kit.yml lists them."}) {
		t.Errorf("the about of packages: %v", got)
	}
}

func TestCustomizeIsATableOfItsPartsInThePagesOrder(t *testing.T) {
	m, _ := loaded(t, press('6'))
	if !m.gridded() || keysAt(m) != "= menu - workspace / search │ ⇥·⇧⇥ tab ↓ parts 1-6 jump" {
		t.Fatalf("the customize step: gridded %v, keys %q", m.gridded(), keysAt(m))
	}
	got := view(m)
	for _, want := range []string{"   Customize\n", "kit is a starting point.", "▾ 6 ", "Default ▼", "part ",
		"settings", "about", "commands", "▸ Primary and forges", pill("packages"), "kit flow incoming --no-fetch"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
	at := -1
	for _, part := range []string{"Packages", "Policy", "The stages of kit flow", "Primary and forges",
		"A house package", "The docs"} {
		k := strings.Index(got, "▸ "+part)
		if k < at {
			t.Errorf("%s stands out of the page's order:\n%s", part, got)
		}
		at = k
	}
	// down gives the table the keyboard, and down again its rows; the terms' table holds none
	if m = drive(m, down, down); !m.inPane || !m.sheet().InRows() || termsSheet(m).Focused() {
		t.Errorf("down, down: in the pane %v, keys %q", m.inPane, keysAt(m))
	}
	// the terms step, and back, leaves each table its own
	if m = drive(m, esc, press('5'), press('6')); m.inPane || m.sheet().Focused() || m.sheet().Name() != "parts" {
		t.Errorf("6 then 7: in the pane %v", m.inPane)
	}
}

func TestGreaterAndLessShowALevelMoreAndLessOfTheParts(t *testing.T) {
	m, _ := loaded(t, press('6'))
	// from the step bar, as from a stage's tab bar in flow: > opens each part, < closes them
	if m = drive(m, press('>')); !strings.Contains(view(m), "▾ Packages") || m.inPane {
		t.Errorf("> from the bar: in the pane %v\n%s", m.inPane, view(m))
	}
	if m = drive(m, press('<')); !strings.Contains(view(m), "▸ Packages") {
		t.Errorf("< from the bar:\n%s", view(m))
	}
	// and from the rows
	if m = drive(m, down, down, press('>')); !strings.Contains(view(m), "▾ Packages") || !m.sheet().InRows() {
		t.Errorf("> in the rows:\n%s", view(m))
	}
}

func TestThePartsPresetsGroupThemByCommandAndCompactThem(t *testing.T) {
	m, _ := loaded(t, press('6'))
	m = drive(m, clickOn(t, m, "Default ▼"))
	m = drive(m, clickOn(t, m, "By command"))
	got := view(m)
	for _, want := range []string{"▾ command ▾ 6", "By command ▼", "▾ kit flow 2", "▾ kit help 1", "▾ kit packages 2",
		"▾ kit policy 1", "    ▸ The stages of kit flow", "    ▸ A house package"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
	// the groups keep the page's order within them
	if strings.Index(got, "▸ Packages") > strings.Index(got, "▸ A house package") {
		t.Errorf("Packages after A house package:\n%s", got)
	}
	m = drive(m, clickOn(t, m, "By command ▼"))
	m = drive(m, clickOn(t, m, "Compact"))
	header := strings.Split(view(m), "\n")[paneTop+6]
	if !strings.Contains(header, "part ") || !strings.Contains(header, "about") ||
		strings.Contains(header, "settings") || strings.Contains(header, "commands") {
		t.Errorf("Compact hides the settings and the commands:\n%s", view(m))
	}
}

func TestTheSearchNarrowsTheParts(t *testing.T) {
	m, _ := loaded(t, press('6'), press('/'))
	m = drive(m, testkit.Typed("forge")...)
	got := view(m)
	if !strings.Contains(got, "▸ Primary and forges") || strings.Contains(got, "▸ Policy") ||
		!strings.Contains(got, "▾ 1 ") {
		t.Errorf("searched:\n%s", got)
	}
	// each step keeps its own search: the terms are not narrowed by the parts'
	if m = drive(m, enter, esc, press('5')); !strings.Contains(view(m), "▾ 12 ") {
		t.Errorf("the terms, after the parts' search:\n%s", view(m))
	}
}

func TestThePartsFollowTheSettings(t *testing.T) {
	m, _ := loaded(t, press('6'))
	m = drive(m, shell.SettingsMsg{Settings: shell.Settings{FoldShared: false, SharedEmoji: true}})
	for _, s := range []*table.Sheet{m.sheet(), termsSheet(m)} {
		if p := s.Panel(); p.Fold || !p.Glyphs {
			t.Errorf("the settings of %s", s.Name())
		}
	}
}

func TestACustomizePageWithoutPartsIsAPage(t *testing.T) {
	r := repoWith(t, map[string]string{"6-customize.md": "# Ours\n\nWe change `policy:` first.\n"})
	m := New(r, "")
	m = drive(m, append(testkit.Start(m, at120), press('6'))...)
	if m.gridded() || !strings.Contains(view(m), "We change policy: first.") || !strings.Contains(keysAt(m), "↓ page") {
		t.Errorf("a page without parts:\n%s", view(m))
	}
}

// The customize step's screens: at 120×30, as every step's is pinned; at 120×40, as it opens, grouped by command,
// and searched for kit flow with the rows given the keyboard; and at 80×40.
func TestCustomizeGoldens(t *testing.T) {
	at40 := tea.WindowSizeMsg{Width: 120, Height: 40}
	byCommand := func(m Model) Model {
		m = drive(m, clickOn(t, m, "Default ▼"))
		return drive(m, clickOn(t, m, "By command"))
	}
	for file, c := range map[string]struct {
		keys []tea.Msg
		then func(Model) Model
	}{
		"customize.golden":         {keys: []tea.Msg{press('6')}},
		"customize-40.golden":      {keys: []tea.Msg{at40, press('6')}},
		"customize-grouped.golden": {keys: []tea.Msg{at40, press('6')}, then: byCommand},
		"customize-search.golden": {keys: append([]tea.Msg{at40, press('6'), press('/')},
			append(testkit.Typed("kit flow"), enter)...)},
		"customize-80.golden": {keys: []tea.Msg{tea.WindowSizeMsg{Width: 80, Height: 40}, press('6')}},
	} {
		m, _ := loaded(t, c.keys...)
		if c.then != nil {
			m = c.then(m)
		}
		testkit.Golden(t, filepath.Join("testdata", file), view(m))
	}
}
