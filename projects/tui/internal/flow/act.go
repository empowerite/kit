// The backlog's actions: what the tab's menu offers on the issues selected in backlog, each a kit verb kittui runs and
// never does itself, so a terminal, an agent over kit serve, and the menu do the same thing. An action that changes
// what others see first shows the issues it will change and asks; one that wants an answer, a title, a login, an issue,
// or one of the caller's pull requests, asks for it. While kit runs, the keys bar says so; then it says what kit did,
// or why it refused, the selection is cleared when kit did it, and the stages that read the forge's issues and pull
// requests read them afresh.
package flow

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// The glyphs beside the backlog's actions that the picture's own glyphs do not name already: the issues handed to
// someone, the caller walking away from them, gathered in a basket, closed as not planned, and closed as a twin.
const (
	Hand   = "👥"
	Leave  = "🚶"
	Gather = "🧺"
	Drop   = "🚫"
	Twin   = "♊"
)

// actions gives the backlog's actions, in the tab's menu's order, on backlog, and none elsewhere. They are there with
// nothing selected too, dim, so the menu shows what selecting issues makes possible.
func (m Model) actions() []shell.Item {
	if len(m.stages) == 0 || m.stages[m.stage].name != "backlog" {
		return nil
	}
	return []shell.Item{
		{Glyph: Sprout, Name: "Start Local Work for Each"},
		{Glyph: Branch, Name: "Start Local Work Together…"},
		{Glyph: Change, Name: "Add to Pull Request…"},
		{Glyph: Owner, Name: "Assign to Me"},
		{Glyph: Hand, Name: "Assign to…"},
		{Glyph: Leave, Name: "Unassign"},
		{Glyph: Gather, Name: "Gather Under a Parent…"},
		{Glyph: Drop, Name: "Close as Not Planned"},
		{Glyph: Twin, Name: "Close as Duplicate of…"},
		{Glyph: ToPage, Name: "Copy Links"},
	}
}

// ask is what an action asks before kit runs: a title, and either a line to type, the issues it will change to
// confirm, or the caller's pull requests to pick one of, by number; and the kit commands the answer makes.
type ask struct {
	title  string
	typing bool     // whether the answer is typed
	typed  string   // what is typed so far
	lines  []string // the issues to confirm, or the pull requests to pick from
	pick   []int    // the pull requests' numbers, when one is picked
	at     int      // the pull request under the keyboard
	run    func(answer string) [][]string
}

// actedMsg is what kit printed for an action's commands, and its refusal when one failed: those after it did not run.
type actedMsg struct {
	out string
	err error
}

// refs gives the references of the issues selected in backlog, in the rows' order: #45.
func (m Model) refs() []string {
	var out []string
	for _, r := range m.selectedRows() {
		out = append(out, r.Key)
	}
	return out
}

// mine gives the caller's open pull requests, as the changes stage read them: each one's number and line.
func (m Model) mine() (numbers []int, lines []string) {
	for _, s := range m.stages {
		if s.name != "changes" {
			continue
		}
		for _, p := range s.pulls {
			if m.me != "" && p.Author == m.me {
				numbers = append(numbers, p.Number)
				lines = append(lines, fmt.Sprintf("%s #%d %s", Change, p.Number, p.Title))
			}
		}
	}
	return numbers, lines
}

// acting says whether an item of the tab's menu is one of the backlog's actions, and whether it does anything now:
// none while no issue is selected, and Add to Pull Request only while the caller has an open pull request to add to.
func (m Model) acting(name string) (action, enabled bool) {
	if !slices.ContainsFunc(m.actions(), func(i shell.Item) bool { return i.Name == name }) {
		return false, false
	}
	if m.selected() == 0 {
		return true, false
	}
	if name == "Add to Pull Request…" {
		numbers, _ := m.mine()
		return true, len(numbers) > 0
	}
	return true, true
}

// act does what the backlog's action named asks: it runs kit at once, or asks first.
func (m *Model) act(name string) tea.Cmd {
	refs := m.refs()
	each := func(words ...string) func(string) [][]string {
		return func(string) [][]string { return [][]string{append(words, refs...)} }
	}
	confirm := func(title string, words ...string) {
		m.ask = &ask{title: title, lines: m.listed(refs), run: each(words...)}
	}
	question := func(title string, run func(answer string) [][]string) {
		m.ask = &ask{title: title, typing: true, run: run}
	}
	switch name {
	case "Start Local Work for Each":
		var cmds [][]string
		for _, r := range refs {
			cmds = append(cmds, []string{"work", r})
		}
		return m.run(cmds)
	case "Start Local Work Together…":
		question("Title of the parent issue", func(t string) [][]string {
			return [][]string{append([]string{"work", "--title", t}, refs...)}
		})
	case "Add to Pull Request…":
		numbers, lines := m.mine()
		m.ask = &ask{title: "Add to which of your pull requests?", lines: lines, pick: numbers,
			run: func(n string) [][]string { return [][]string{append([]string{"work", "--pr", n}, refs...)} }}
	case "Assign to Me":
		return m.run(each("assign")(""))
	case "Assign to…":
		question("Assign to which login?", func(login string) [][]string {
			return [][]string{append([]string{"assign", "--to", login}, refs...)}
		})
	case "Unassign":
		confirm("Take yourself off these issues?", "unassign")
	case "Gather Under a Parent…":
		question("Title of the parent issue", func(t string) [][]string {
			return [][]string{append([]string{"gather", "--title", t}, refs...)}
		})
	case "Close as Not Planned":
		confirm("Close these issues as not planned?", "close", "--not-planned")
	case "Close as Duplicate of…":
		question("Duplicate of which issue?", func(original string) [][]string {
			return [][]string{append([]string{"close", "--duplicate-of", original}, refs...)}
		})
	case "Copy Links":
		m.notice, m.noted = fmt.Sprintf("%s copied %d links", ToPage, len(refs)), true
		return tea.SetClipboard(m.links())
	}
	return nil
}

// listed gives a line for each issue, its reference and title, as the dialog that confirms an action lists them.
func (m Model) listed(refs []string) []string {
	var out []string
	for _, r := range refs {
		out = append(out, r+" "+m.issueTitle(r))
	}
	return out
}

// issueTitle gives a backlog issue's title, by its reference, or nothing.
func (m Model) issueTitle(ref string) string {
	n, _ := strconv.Atoi(strings.TrimPrefix(ref, "#"))
	for _, s := range m.stages {
		if s.name == "backlog" {
			for _, i := range s.issues {
				if i.Number == n {
					return i.Title
				}
			}
		}
	}
	return ""
}

// links gives the selected issues as Markdown, one a line: a link to each issue's page, its reference and title the
// link's text, or the text alone for an issue with no page.
func (m Model) links() string {
	var out []string
	for _, r := range m.selectedRows() {
		text := r.Key + " " + m.issueTitle(r.Key)
		page := ""
		for _, c := range r.Cells {
			if strings.Contains(c.Link, "://") {
				page = c.Link
				break
			}
		}
		if page == "" {
			out = append(out, "- "+text)
		} else {
			out = append(out, fmt.Sprintf("- [%s](%s)", text, page))
		}
	}
	return strings.Join(out, "\n")
}

// run runs kit's commands, one after another, the keys bar saying so until kit answers; a refusal stops the rest.
func (m *Model) run(cmds [][]string) tea.Cmd {
	if len(cmds) == 0 {
		return nil
	}
	m.notice, m.noted = "→ kit "+strings.Join(cmds[0], " ")+" …", true
	r, ctx := m.runner, m.ctx
	return func() tea.Msg {
		var out strings.Builder
		for _, c := range cmds {
			o, err := r.Act(ctx, c...)
			out.WriteString(o)
			if err != nil {
				return actedMsg{out.String(), err}
			}
		}
		return actedMsg{out.String(), nil}
	}
}

// acted takes what kit said for an action: its last line in place of the keys, or its refusal; when kit did what
// was asked, the selection is cleared; either way, what reads the forge's issues reads them afresh.
func (m *Model) acted(msg actedMsg) tea.Cmd {
	lines := strings.Split(strings.TrimSpace(ansi.Strip(msg.out)), "\n")
	m.notice, m.noted = lines[len(lines)-1], true
	if msg.err != nil {
		m.notice, m.noted = Failed+" "+msg.err.Error(), false
	} else if s := m.find("backlog"); s != nil {
		clear(s.picked)
	}
	return m.refresh("backlog")
}

// askKey takes a key while an action asks: esc does not run it; enter runs it with the answer, which a typed one
// needs; up and down move along the pull requests; anything else is typed, when the answer is.
func (m Model) askKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	a := m.ask
	switch key := k.String(); {
	case key == "esc":
		m.ask = nil
	case key == "enter":
		answer := strings.TrimSpace(a.typed)
		switch {
		case a.pick != nil:
			answer = strconv.Itoa(a.pick[a.at])
		case a.typing && answer == "":
			return m, nil
		}
		m.ask = nil
		return m, m.run(a.run(answer))
	case a.pick != nil && key == "up":
		a.at = max(a.at-1, 0)
	case a.pick != nil && key == "down":
		a.at = min(a.at+1, len(a.pick)-1)
	case a.typing:
		a.typed, _ = shell.Type(a.typed, k)
	}
	return m, nil
}

// askKeys gives what the dialog an action asks in shows, and its keys: the line typed so far, with its caret; the
// pull requests, the one under the keyboard inverse; or the issues to confirm.
func (m Model) askKeys() (body []string, keys string) {
	a := m.ask
	switch {
	case a.typing:
		return []string{a.typed + style.Mark("▏")}, "esc cancel · ⏎ ok"
	case a.pick != nil:
		body = slices.Clone(a.lines)
		body[a.at] = style.Cursor(body[a.at])
		return body, "esc cancel · ↑↓ move · ⏎ add"
	}
	return a.lines, "esc no · ⏎ yes"
}

// askBox gives the dialog an action asks in, centered over the rows near their top, and the column it is drawn at.
func (m Model) askBox() (lines []string, left int) {
	body, keys := m.askKeys()
	lines = shell.Dialog(m.ask.title, body, keys, m.paneWidth()-2)
	return lines, max(0, (m.paneWidth()-ansi.StringWidth(lines[0]))/2)
}
