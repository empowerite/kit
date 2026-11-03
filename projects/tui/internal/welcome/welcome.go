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
// Left and right move between the steps; inside a row with choices they move between now and later. Down enters a
// step's rows and up on the first row leaves them. Up and down scroll a page. Esc goes back one level: from the rows
// to the step line, and from there it quits; ctrl-c quits with 130 from anywhere. The screen opens on the intro while
// the checks run. When they answer, it moves to the first failed check, in the first step that has one, unless the
// person has moved already.
package welcome

import (
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/pane"
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
	exec     func(*exec.Cmd, tea.ExecCallback) tea.Cmd // hands the terminal to a fix; tea.ExecProcess but in tests
	repo     string                                    // owner/repo, for the title, empty when unknown
	tabs     []tab                                     // the steps, in order: the known ones until the pages are read
	step     int                                       // the selected step
	inRows   bool                                      // whether the cursor is on a check's row, not on the step line
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

	interrupted bool
}

// Open opens the screen in the directory the runner names, with owner/repo read from git.
func Open(r kit.Runner) Model { return New(r, readRepo(r.Dir)) }

// New opens the screen on the first step, checking. Init runs the checks and reads the pages.
func New(r kit.Runner, repo string) Model {
	return Model{runner: r, exec: tea.ExecProcess, repo: repo, checking: true, deferred: map[string]bool{},
		notes: map[string]note{}, width: 80, height: 24, tabs: slices.Clone(known),
		places: make([]string, len(known)), offsets: make([]int, len(known))}
}

// Init runs the checks and reads the pages, at once.
func (m Model) Init() tea.Cmd { return tea.Batch(m.check(), m.read()) }

// Interrupted says whether ctrl-c ended the screen.
func (m Model) Interrupted() bool { return m.interrupted }

// check runs kit welcome --checks --json.
func (m Model) check() tea.Cmd {
	r := m.runner
	return func() tea.Msg {
		checks, err := r.WelcomeChecks()
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
				m.step, m.inRows, m.cursor = m.stepOf(c), true, i
				break
			}
		}
	}
	m.loaded = true
	m.fit()
}

// rows gives the indexes of the step's checks, in order.
func (m Model) rows(step int) []int {
	var out []int
	for i, c := range m.checks {
		if m.stepOf(c) == step {
			out = append(out, i)
		}
	}
	return out
}

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
		m.step, m.inRows = 0, false
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

// key takes a key press.
func (m Model) key(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	choosing := m.choosing()
	switch k.String() {
	case "ctrl+c":
		m.interrupted = true
		return m, tea.Quit
	case "esc":
		if !m.inRows {
			return m, tea.Quit
		}
		m.inRows, m.offsets[m.step] = false, 0
	case "left":
		if choosing {
			m.choice = now
		} else {
			m.move(-1)
		}
	case "right":
		if choosing {
			m.choice = later
		} else {
			m.move(1)
		}
	case "up":
		m.up(1)
	case "down":
		m.down(1)
	case "pgup":
		m.up(max(1, m.paneHeight()-1))
	case "pgdown":
		m.down(max(1, m.paneHeight()-1))
	case "enter":
		switch {
		case m.hasRows() && !m.inRows:
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

// choosing says whether the cursor is on a row that offers now and later: a failed check with a fix kit runs, in a
// step with checks.
func (m Model) choosing() bool {
	if !m.hasRows() || !m.inRows || m.cursor >= len(m.checks) {
		return false
	}
	c := m.checks[m.cursor]
	return !c.OK && c.Fix != "" && c.Runs
}

// showOnly says whether the cursor is on a failed check whose fix kit shows and never runs.
func (m Model) showOnly() bool {
	if !m.hasRows() || !m.inRows || m.cursor >= len(m.checks) {
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

// move selects the step d along, round the ends, on the step line. The cursor goes back to the check it was on in
// that step, else to the step's first check.
func (m *Model) move(d int) {
	m.places[m.step] = m.cursorID()
	n := len(m.tabs)
	m.step = (m.step + d + n) % n
	m.cursor = m.find(m.step, m.places[m.step])
	m.inRows, m.moved, m.choice = false, true, now
}

// up moves the cursor n rows up in a step with checks, and past its first row back to the step line; on a page it
// scrolls.
func (m *Model) up(n int) {
	if !m.hasRows() {
		m.scroll(-n)
		return
	}
	if !m.inRows {
		return
	}
	rows := m.rows(m.step)
	at := slices.Index(rows, m.cursor)
	if at-n < 0 {
		m.inRows, m.offsets[m.step] = false, 0
		return
	}
	m.cursor = rows[at-n]
	m.choice = now
}

// down moves the cursor n rows down in a step with checks, entering the rows from the step line; on a page it
// scrolls.
func (m *Model) down(n int) {
	if !m.hasRows() {
		m.scroll(n)
		return
	}
	rows := m.rows(m.step)
	if len(rows) == 0 {
		return
	}
	at := max(0, slices.Index(rows, m.cursor))
	if !m.inRows {
		m.inRows = true
		n--
	}
	m.cursor = rows[min(at+n, len(rows)-1)]
	m.choice = now
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
