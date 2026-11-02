// Package welcome is kit welcome on a terminal: four steps as tabs along the top, the step's content below, and the
// keys bar last. Setup lists kit's checks, one row each. A failed check with a fix offers now and later on the cursor's
// row: now runs kit welcome --fix <id> with the terminal handed over, then reads the checks again; later leaves it for
// this session and moves to the next failed check. The other three steps are markdown pages drawn as text.
//
// Left and right move between the steps; inside a row with choices they move between now and later. Down enters the
// setup rows and up on the first row leaves them. Up and down scroll a page. Esc goes back one level: from the rows to
// the step line, and from there it quits; ctrl-c quits with 130 from anywhere. The screen opens on Setup while the
// checks run, and moves to the lay of the land when every check passes, unless the person has moved already.
package welcome

import (
	"errors"
	"fmt"
	"os/exec"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/tui/internal/kit"
)

// The steps, in order.
const (
	setup = iota
	land
	terms
	yours
	steps
)

// tab is a step's tab: its glyph, its name, and the file of its page. Each glyph is two cells wide with no variation
// selector.
type tab struct {
	glyph, name, file string
}

// tabs gives the four steps' tabs.
var tabs = [steps]tab{
	{"🔧", "setup", "1-setup.md"},
	{"🧭", "lay of the land", "2-lay-of-the-land.md"},
	{"📖", "terms", "3-terms.md"},
	{"🎨", "make it yours", "4-make-it-yours.md"},
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

// pagesMsg brings the pages' markdown, each step's, and for a page not found the path it was looked for at.
type pagesMsg struct {
	pages, missing [steps]string
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
	step     int                                       // the selected step
	inRows   bool                                      // whether the cursor is on a check's row, not on the step line
	moved    bool                                      // whether the person has changed the step
	loaded   bool                                      // whether the checks have answered once
	checking bool                                      // whether kit is running the checks
	checks   []kit.Check
	refusal  string // kit's line, when the checks did not run
	cursor   int    // the check under the cursor
	choice   int    // now or later, on a row that offers them
	deferred map[string]bool
	notes    map[string]note
	pages    [steps]string
	missing  [steps]string
	offsets  [steps]int // the first line of each step's pane
	width    int
	height   int

	interrupted bool
}

// Open opens the screen in the directory the runner names, with owner/repo read from git.
func Open(r kit.Runner) Model { return New(r, readRepo(r.Dir)) }

// New opens the screen on Setup, checking. Init runs the checks and reads the pages.
func New(r kit.Runner, repo string) Model {
	return Model{runner: r, exec: tea.ExecProcess, repo: repo, checking: true, deferred: map[string]bool{},
		notes: map[string]note{}, width: 80, height: 24}
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
		var msg pagesMsg
		msg.pages, msg.missing = readPages(r)
		return msg
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
		m.pages, m.missing = msg.pages, msg.missing
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
// the screen opens: on the first failed check's row, or on the lay of the land when none failed, unless the person
// has moved.
func (m *Model) take(msg checksMsg) {
	m.checking = false
	if msg.err != nil {
		m.refusal = refusalOf(msg.err)
		m.fit()
		return
	}
	id := m.cursorID()
	m.refusal, m.checks = "", msg.checks
	m.cursor = 0
	for i, c := range m.checks {
		if c.ID == id {
			m.cursor = i
		}
		if c.OK {
			delete(m.notes, c.ID)
			delete(m.deferred, c.ID)
		}
	}
	if !m.loaded && !m.moved {
		if first := m.nextFailed(-1); first >= 0 {
			m.step, m.inRows, m.cursor = setup, true, first
		} else if len(m.checks) > 0 {
			m.step = land
		}
	}
	m.loaded = true
	m.fit()
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
		m.inRows, m.offsets[setup] = false, 0
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
		case m.step == setup && !m.inRows:
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
		if choosing {
			m.later()
		}
	}
	m.fit()
	return m, nil
}

// choosing says whether the cursor is on a row that offers now and later: a failed check with a fix, in Setup.
func (m Model) choosing() bool {
	if m.step != setup || !m.inRows || m.cursor >= len(m.checks) {
		return false
	}
	c := m.checks[m.cursor]
	return !c.OK && c.Fix != ""
}

// cursorID gives the id of the check under the cursor, or nothing when there is none.
func (m Model) cursorID() string {
	if m.cursor < len(m.checks) {
		return m.checks[m.cursor].ID
	}
	return ""
}

// move selects the step d along, round the ends, on the step line.
func (m *Model) move(d int) {
	m.step = (m.step + d + steps) % steps
	m.inRows, m.moved, m.choice = false, true, now
}

// up moves the cursor n rows up in Setup, and past the first row back to the step line; on a page it scrolls.
func (m *Model) up(n int) {
	if m.step != setup {
		m.scroll(-n)
		return
	}
	if !m.inRows {
		return
	}
	if m.cursor-n < 0 {
		m.inRows, m.offsets[setup] = false, 0
		return
	}
	m.cursor -= n
	m.choice = now
}

// down moves the cursor n rows down in Setup, entering the rows from the step line; on a page it scrolls.
func (m *Model) down(n int) {
	if m.step != setup {
		m.scroll(n)
		return
	}
	if len(m.checks) == 0 {
		return
	}
	if !m.inRows {
		m.inRows = true
		n--
	}
	m.cursor = min(m.cursor+n, len(m.checks)-1)
	m.choice = now
}

// scroll moves a page's pane n lines, never past its ends.
func (m *Model) scroll(n int) {
	start, _, _, _ := window(len(m.pageLines(m.step)), m.paneHeight(), m.offsets[m.step]+n)
	m.offsets[m.step] = start
}

// now hands the terminal to the fix of the check under the cursor; when it ends, the checks run again.
func (m *Model) now() tea.Cmd {
	c := m.checks[m.cursor]
	delete(m.deferred, c.ID)
	m.choice = now
	return m.exec(m.runner.WelcomeFix(c.ID), func(err error) tea.Msg { return fixedMsg{id: c.ID, err: err} })
}

// later leaves the check under the cursor for later and moves to the next failed check not left, if there is one.
func (m *Model) later() {
	m.deferred[m.checks[m.cursor].ID] = true
	if next := m.nextFailed(m.cursor); next >= 0 {
		m.cursor = next
	}
	m.choice = now
}

// nextFailed gives the first failed check after the row from, round the end, not left for later, or -1.
func (m Model) nextFailed(from int) int {
	n := len(m.checks)
	for k := 1; k <= n; k++ {
		i := (from + k + n) % n
		if c := m.checks[i]; !c.OK && !m.deferred[c.ID] && i != from {
			return i
		}
	}
	return -1
}
