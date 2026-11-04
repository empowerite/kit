// Package welcome is kit welcome on a terminal: a step per page as tabs along the top, the step's content below, and
// the keys bar last. The steps are the pages under .kit/welcome/, as kit welcome --text finds and orders them: each
// has an id, which kit's checks name as their step, a glyph, a name, and its page. A page kit knows keeps its glyph,
// its name, and its checks; any other, as a house package adds, is a plain page. A step with checks draws its page,
// then its checks, one row each. A failed check with a fix offers now and later on the cursor's row: now runs kit
// welcome --fix <id> with the terminal handed over, then reads the checks again; later leaves it for this session and
// moves to the step's next failed check. A fix kit only shows, as one that must run in the person's own shell, is
// drawn with its lines and no choices: n and enter do nothing there, and l still leaves it for later. The other steps
// are markdown pages drawn as text.
//
// The steps are a tab bar, internal/tabs. While it has the keyboard, left and right move between the steps; a step's
// number, or a click on the bar, selects it from anywhere and gives the bar the keyboard. Down, or enter, gives the
// pane the keyboard: a step's rows, where up on the first row gives it back, or its page, which up and down then
// scroll, and up at the top gives it back; a click on a check's line puts the cursor there. Inside a row with choices,
// left and right move between now and later. Esc gives the bar the keyboard back. Ctrl-c quits with 130 from anywhere,
// and alt-esc or super-esc, as a terminal that reports it sends command-esc, quits with 0. The screen opens on the
// intro while the checks run. When they answer, it moves to the first failed check, in the first step that has one,
// unless the person has moved already.
package welcome

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// tab is a step: the id its checks name, its tab's glyph and name, the file of its page, whether it has checks, and
// its page's markdown, or, when the page could not be read, where it was looked for. Each glyph is two cells wide with
// no variation selector.
type tab struct {
	id, glyph, name, file string
	checked               bool
	page, missing         string
}

// known gives the pages kit knows, in the kit package's order. A check whose step has no page shows under the last
// step with checks.
var known = []tab{
	{id: "intro", glyph: "👋", name: "intro", file: "1-intro.md"},
	{id: "before-toolchains", glyph: "🧱", name: "basics", file: "2-before-toolchains.md", checked: true},
	{id: "toolchains", glyph: "🧰", name: "toolchains", file: "3-toolchains.md", checked: true},
	{id: "setup", glyph: "🔧", name: "setup", file: "4-setup.md", checked: true},
	{id: "lay-of-the-land", glyph: "🧭", name: "lay of the land", file: "5-lay-of-the-land.md"},
	{id: "terms", glyph: "📖", name: "terms", file: "6-terms.md"},
	{id: "make-it-yours", glyph: "🎨", name: "make it yours", file: "7-make-it-yours.md"},
}

// pageGlyph is the glyph of a page kit does not know.
const pageGlyph = "📄"

// idOf gives the id of the step a page's file is for, as kit welcome reads it: the name without the order and the
// dash before it, and without .md. 2-before-toolchains.md is for before-toolchains.
func idOf(file string) string {
	name := strings.TrimSuffix(file, ".md")
	if _, id, found := strings.Cut(name, "-"); found {
		return id
	}
	return name
}

// tabOf gives the step a page's file makes: the known step with its id, with the file's name, so a house package's
// page in its place keeps its glyph, its name, and its checks; else a plain page, named for its id, dashes as spaces.
func tabOf(file string) tab {
	id := idOf(file)
	if i := indexOf(known, id); i >= 0 {
		t := known[i]
		t.file = file
		return t
	}
	return tab{id: id, glyph: pageGlyph, name: strings.ReplaceAll(id, "-", " "), file: file}
}

// indexOf gives the index of the first step with the id, or -1.
func indexOf(tabs []tab, id string) int {
	return slices.IndexFunc(tabs, func(t tab) bool { return t.id == id })
}

// stepOf gives the index of the step a check shows under: its own when that step has checks, else the last that has.
func (m Model) stepOf(c kit.Check) int {
	if i := indexOf(m.tabs, c.Step); i >= 0 && m.tabs[i].checked {
		return i
	}
	last := 0
	for i, t := range m.tabs {
		if t.checked {
			last = i
		}
	}
	return last
}

// The choices on a failed check's row.
const (
	now = iota
	later
)

// checksMsg brings the checks, or why kit did not print them.
type checksMsg struct {
	checks []kit.Check
	err    error
}

// pagesMsg brings the steps the pages make, each with its page's markdown, or the path it was looked for at.
type pagesMsg struct {
	tabs []tab
}

// fixedMsg says a check's fix has run, and how it ended.
type fixedMsg struct {
	id  string
	err error
}

// note is what the screen says under a check after its fix ran and the check still fails: a warning, or kit's
// refusal.
type note struct {
	text    string
	refusal bool
}

// Model is the welcome screen.
type Model struct {
	runner   kit.Runner
	ctx      context.Context                           // ends with the screen, and the checks still running with it
	cancel   context.CancelFunc                        // ends ctx
	exec     func(*exec.Cmd, tea.ExecCallback) tea.Cmd // hands the terminal to a fix; tea.ExecProcess but in tests
	repo     string                                    // owner/repo, for the title, empty when unknown
	tabs     []tab                                     // the steps, in order: the known ones until the pages are read
	step     int                                       // the selected step
	inPane   bool                                      // whether the pane has the keyboard, a check's row or the page, not the bar
	places   []string                                  // the check each step's cursor was on when it was left
	moved    bool                                      // whether the person has changed the step
	loaded   bool                                      // whether the checks have answered once
	checking bool                                      // whether kit is running the checks
	checks   []kit.Check
	refusal  string // kit's line, when the checks did not run
	cursor   int    // the check under the cursor, an index into checks, in the selected step
	choice   int    // now or later, on a row that offers them
	deferred map[string]bool
	notes    map[string]note
	offsets  []int // the first line of each step's pane
	width    int
	height   int
	appMenu  shell.Menu              // the app's menu, internal/shell's, which has the keyboard while it is open
	searches map[string]shell.Search // each step's search, internal/shell's, by the step's id

	interrupted bool
}

// Open opens the screen in the directory the runner names, with owner/repo read from git.
func Open(r kit.Runner) Model { return New(r, readRepo(r.Dir)) }

// New opens the screen on the first step, checking. Init runs the checks and reads the pages.
func New(r kit.Runner, repo string) Model {
	m := Model{runner: r, exec: tea.ExecProcess, repo: repo, checking: true, deferred: map[string]bool{},
		notes: map[string]note{}, width: 80, height: 24, tabs: slices.Clone(known),
		places: make([]string, len(known)), offsets: make([]int, len(known)), appMenu: shell.Menu{Here: shell.Welcome},
		searches: map[string]shell.Search{}}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	return m
}

// quit ends the screen, and the checks still running with it.
func (m Model) quit() tea.Cmd {
	m.cancel()
	return tea.Quit
}

// chose does what a choice from the app's menu asks: a screen asks kittui to show it in the guide's place, which stays
// as it is; and quit ends the screen.
func (m Model) chose(c shell.Choice) tea.Cmd {
	switch {
	case c.Quit:
		return m.quit()
	case c.Screen != "":
		return shell.Switch(c.Screen)
	}
	return nil
}

// Init runs the checks and reads the pages, at once.
func (m Model) Init() tea.Cmd { return tea.Batch(m.check(), m.read()) }

// Interrupted says whether ctrl-c ended the screen.
func (m Model) Interrupted() bool { return m.interrupted }

// Close ends the checks still running, as quitting does, for a guide kittui leaves hidden as it ends.
func (m Model) Close() { m.cancel() }

// check runs kit welcome --checks --json, for as long as the screen lasts and kit's limit lets it.
func (m Model) check() tea.Cmd {
	r, ctx := m.runner, m.ctx
	return func() tea.Msg {
		checks, err := r.WelcomeChecks(ctx)
		return checksMsg{checks: checks, err: err}
	}
}

// read reads the pages, as pages says.
func (m Model) read() tea.Cmd {
	r := m.runner
	return func() tea.Msg {
		return pagesMsg{tabs: readPages(r)}
	}
}

// Update takes a message.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.fit()
	case checksMsg:
		m.take(msg)
	case pagesMsg:
		m.setTabs(msg.tabs)
		m.fit()
	case fixedMsg:
		m.fixed(msg)
		m.checking = true
		return m, m.check()
	case tea.KeyPressMsg:
		return m.key(msg)
	case tea.MouseClickMsg:
		switch {
		case m.appMenu.Open && msg.Button == tea.MouseLeft:
			var c shell.Choice
			m.appMenu, c = m.appMenu.Click(msg.X, msg.Y, m.width)
			return m, m.chose(c)
		case msg.Button == tea.MouseLeft && shell.OnApp(msg.X, msg.Y):
			m.appMenu = m.appMenu.Opened()
			return m, nil
		case msg.Button == tea.MouseLeft && msg.Y == shell.Top && msg.X >= m.searchAt():
			m.focusSearch(true)
			return m, nil
		}
		if bar, ok := m.bar().Update(msg); ok {
			m.choose(bar)
		} else if msg.Button == tea.MouseLeft {
			m.click(msg.Y)
		}
		m.fit()
	}
	return m, nil
}

// take keeps the checks, or kit's refusal, and keeps the cursor on the check it was on. The first answer chooses where
// the screen opens, unless the person has moved: on the first failed check, in the first step that has one, or on
// the first step with checks when kit refused them.
func (m *Model) take(msg checksMsg) {
	m.checking = false
	if msg.err != nil {
		m.refusal = refusalOf(msg.err)
		if !m.loaded && !m.moved {
			for i, t := range m.tabs {
				if t.checked {
					m.step = i
					break
				}
			}
		}
		m.fit()
		return
	}
	id := m.cursorID()
	m.refusal, m.checks = "", msg.checks
	m.cursor = m.find(m.step, id)
	for _, c := range m.checks {
		if c.OK {
			delete(m.notes, c.ID)
			delete(m.deferred, c.ID)
		}
	}
	if !m.loaded && !m.moved {
		for i, c := range m.checks {
			if !c.OK {
				m.step, m.inPane, m.cursor = m.stepOf(c), true, i
				break
			}
		}
	}
	m.loaded = true
	m.fit()
}

// rows gives the indexes of the step's checks, in order, those its search lets through: whose id, line, or fix holds
// what is typed, without regard to case.
func (m Model) rows(step int) []int {
	query := m.searches[m.tabs[step].id].Query
	var out []int
	for i, c := range m.checks {
		if m.stepOf(c) == step && (matches(c.ID, query) || matches(c.Line, query) || matches(c.Fix, query)) {
			out = append(out, i)
		}
	}
	return out
}

// matches says whether a text holds what is typed in a search, without regard to case or escapes; anything holds
// nothing typed.
func matches(text, query string) bool {
	return strings.Contains(strings.ToLower(ansi.Strip(text)), strings.ToLower(query))
}

// search gives the selected step's search.
func (m Model) search() shell.Search { return m.searches[m.tabs[m.step].id] }

// setSearch keeps the selected step's search, the cursor on the check it was on when the search lets it through, else
// on the first it does, and the pane back at its top when what is typed changed.
func (m *Model) setSearch(s shell.Search) {
	id := m.tabs[m.step].id
	if s.Query != m.searches[id].Query {
		m.offsets[m.step] = 0
	}
	m.searches[id] = s
	if m.cursor < len(m.checks) {
		m.cursor = m.find(m.step, m.checks[m.cursor].ID)
	}
	m.fit()
}

// focusSearch gives the selected step's search the keyboard, or takes it away.
func (m *Model) focusSearch(focused bool) {
	s := m.search()
	s.Focused = focused
	m.setSearch(s)
}

// searchAt gives the column the search starts at on the title line, as internal/shell puts every screen's, over the
// steps' numbers.
func (m Model) searchAt() int { return shell.SearchAt(m.width, m.bar().Numbers(), m.search().InUse()) }

// find gives the index of the check with the id when it is in the step, else the step's first check, else 0.
func (m Model) find(step int, id string) int {
	rows := m.rows(step)
	for _, i := range rows {
		if m.checks[i].ID == id {
			return i
		}
	}
	if len(rows) > 0 {
		return rows[0]
	}
	return 0
}

// hasRows says whether the selected step shows checks.
func (m Model) hasRows() bool { return m.tabs[m.step].checked }

// setTabs takes the steps the pages make. The selected step, and each step's place and offset, go with its id; when
// the selected step's id has no page, the first step is selected.
func (m *Model) setTabs(tabs []tab) {
	if len(tabs) == 0 {
		return
	}
	old, offsets, places, selected := m.tabs, m.offsets, m.places, m.tabs[m.step].id
	m.tabs, m.offsets, m.places = tabs, make([]int, len(tabs)), make([]string, len(tabs))
	for i, t := range tabs {
		if j := indexOf(old, t.id); j >= 0 {
			m.offsets[i], m.places[i] = offsets[j], places[j]
		}
	}
	if m.step = indexOf(tabs, selected); m.step < 0 {
		m.step, m.inPane = 0, false
	}
}

// refusalOf gives the line to show when the checks did not run: kit's line, or how kit died with nothing on stderr.
func refusalOf(err error) string {
	var exit *kit.ExitError
	if err.Error() != "" {
		return err.Error()
	}
	if errors.As(err, &exit) {
		return fmt.Sprintf("kit welcome --checks exited %d with nothing on stderr", exit.Code)
	}
	return "kit welcome --checks failed"
}

// fixed notes how a fix ended: nothing when it now holds, a warning when it still fails, kit's refusal on 2, and why
// it did not run when it could not.
func (m *Model) fixed(msg fixedMsg) {
	var exit *exec.ExitError
	switch {
	case msg.err == nil:
		delete(m.notes, msg.id)
	case errors.As(msg.err, &exit) && exit.ExitCode() == 1:
		m.notes[msg.id] = note{text: "the fix ran, and the check still fails"}
	case errors.As(msg.err, &exit) && exit.ExitCode() == 2:
		m.notes[msg.id] = note{text: fmt.Sprintf("kit refused the fix; quit to read what kit welcome --fix %s printed",
			msg.id), refusal: true}
	default:
		m.notes[msg.id] = note{text: fmt.Sprintf("kit welcome --fix %s: %v", msg.id, msg.err), refusal: true}
	}
}

// key takes a key press: the quitting keys, the app's menu's while it is open, = for it, the bar's, then the pane's.
func (m Model) key(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "ctrl+c":
		m.interrupted = true
		return m, m.quit()
	case "alt+esc", "super+esc":
		return m, m.quit()
	}
	if m.appMenu.Open {
		var c shell.Choice
		m.appMenu, c = m.appMenu.Key(k)
		return m, m.chose(c)
	}
	if s := m.search(); s.Focused {
		// the search has the keyboard: what it does not take, as the arrows, is nobody's
		s, _ = s.Key(k)
		m.setSearch(s)
		return m, nil
	}
	switch k.String() {
	case "=":
		m.appMenu = m.appMenu.Opened()
		return m, nil
	case "/":
		m.focusSearch(true)
		return m, nil
	}
	if bar, ok := m.bar().Update(k); ok {
		m.choose(bar)
		m.fit()
		return m, nil
	}
	choosing := m.choosing()
	switch k.String() {
	case "esc":
		m.back()
	case "left":
		if choosing {
			m.choice = now
		}
	case "right":
		if choosing {
			m.choice = later
		}
	case "up":
		m.up(1)
	case "down":
		m.down(1)
	case "pgup":
		m.page(-max(1, m.paneHeight()-1))
	case "pgdown":
		m.page(max(1, m.paneHeight()-1))
	case "enter":
		switch {
		case !m.inPane:
			m.down(1)
		case choosing && m.choice == now:
			return m, m.now()
		case choosing:
			m.later()
		}
	case "n":
		if choosing {
			return m, m.now()
		}
	case "l":
		if choosing || m.showOnly() {
			m.later()
		}
	}
	m.fit()
	return m, nil
}

// click takes a left click on the pane, on the screen line y: on a check's line, the cursor goes to that check and the
// pane has the keyboard; on a page, the page has it. A … line, or a line that is no check's, does nothing.
func (m *Model) click(y int) {
	r, h := y-paneTop, m.paneHeight()
	switch {
	case r < 0 || r >= h:
	case !m.hasRows():
		m.inPane = true
	default:
		lines := m.stepLines()
		if k := pane.At(len(lines), h, m.offsets[m.step], r); k >= 0 && lines[k].row >= 0 {
			m.cursor, m.inPane, m.choice = lines[k].row, true, now
		}
	}
}

// back gives the bar the keyboard, and a step's rows go back to their top; a page keeps where it was scrolled to.
func (m *Model) back() {
	if m.inPane && m.hasRows() {
		m.offsets[m.step] = 0
	}
	m.inPane = false
}

// choosing says whether the cursor is on a row that offers now and later: a failed check with a fix kit runs, in a
// step with checks.
func (m Model) choosing() bool {
	if !m.hasRows() || !m.inPane || m.cursor >= len(m.checks) {
		return false
	}
	c := m.checks[m.cursor]
	return !c.OK && c.Fix != "" && c.Runs
}

// showOnly says whether the cursor is on a failed check whose fix kit shows and never runs.
func (m Model) showOnly() bool {
	if !m.hasRows() || !m.inPane || m.cursor >= len(m.checks) {
		return false
	}
	c := m.checks[m.cursor]
	return !c.OK && c.Fix != "" && !c.Runs
}

// cursorID gives the id of the check under the cursor, or nothing when there is none.
func (m Model) cursorID() string {
	if m.cursor < len(m.checks) {
		return m.checks[m.cursor].ID
	}
	return ""
}

// choose takes what the bar chose: the step it selected, and the keyboard. The cursor goes back to the check it was on
// in that step, else to the step's first check.
func (m *Model) choose(bar tabs.Bar) {
	if bar.Selected != m.step {
		m.places[m.step] = m.cursorID()
		m.step = bar.Selected
		m.cursor = m.find(m.step, m.places[m.step])
		m.moved, m.choice = true, now
	}
	m.inPane = !bar.Focused
}

// up moves the cursor n rows up in a step with checks, and past its first row gives the bar the keyboard; on a page
// it scrolls, and at the top gives the bar the keyboard.
func (m *Model) up(n int) {
	if !m.inPane {
		return
	}
	if !m.hasRows() {
		if m.offsets[m.step] == 0 {
			m.inPane = false
		}
		m.scroll(-n)
		return
	}
	rows := m.rows(m.step)
	at := slices.Index(rows, m.cursor)
	if at-n < 0 {
		m.inPane, m.offsets[m.step] = false, 0
		return
	}
	m.cursor = rows[at-n]
	m.choice = now
}

// down moves the cursor n rows down in a step with checks, and on a page scrolls; from the bar it gives the pane the
// keyboard first, which takes one of the n.
func (m *Model) down(n int) {
	if !m.hasRows() {
		if !m.inPane {
			m.inPane = true
			n--
		}
		m.scroll(n)
		return
	}
	rows := m.rows(m.step)
	if len(rows) == 0 {
		return
	}
	at := max(0, slices.Index(rows, m.cursor))
	if !m.inPane {
		m.inPane = true
		n--
	}
	m.cursor = rows[min(at+n, len(rows)-1)]
	m.choice = now
}

// page moves the cursor n rows in a step with checks, or scrolls a page n lines, and never leaves the pane: it stops at
// the first row and the last, and at the page's ends; on the bar it does nothing.
func (m *Model) page(n int) {
	switch {
	case !m.inPane:
	case !m.hasRows():
		m.scroll(n)
	default:
		rows := m.rows(m.step)
		at := max(0, slices.Index(rows, m.cursor))
		m.cursor, m.choice = rows[min(max(at+n, 0), len(rows)-1)], now
	}
}

// scroll moves a page's pane n lines, never past its ends.
func (m *Model) scroll(n int) {
	start, _, _, _ := pane.Window(len(m.pageLines(m.step)), m.paneHeight(), m.offsets[m.step]+n)
	m.offsets[m.step] = start
}

// now hands the terminal to the fix of the check under the cursor; when it ends, the checks run again.
func (m *Model) now() tea.Cmd {
	c := m.checks[m.cursor]
	delete(m.deferred, c.ID)
	m.choice = now
	return m.exec(m.runner.WelcomeFix(c.ID), func(err error) tea.Msg { return fixedMsg{id: c.ID, err: err} })
}

// later leaves the check under the cursor for later and moves to the step's next failed check not left, if there is
// one.
func (m *Model) later() {
	m.deferred[m.checks[m.cursor].ID] = true
	if next := m.nextFailed(m.cursor); next >= 0 {
		m.cursor = next
	}
	m.choice = now
}

// nextFailed gives the step's first failed check after the check from, round the end, not left for later, or -1.
func (m Model) nextFailed(from int) int {
	rows := m.rows(m.step)
	n, at := len(rows), slices.Index(rows, from)
	for k := 1; k <= n; k++ {
		i := rows[(at+k+n)%n]
		if c := m.checks[i]; !c.OK && !m.deferred[c.ID] && i != from {
			return i
		}
	}
	return -1
}
