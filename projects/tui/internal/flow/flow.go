// Package flow is kit flow on a terminal: the stages as a tab bar, internal/tabs, the selected stage's rows under it,
// and a row opened in place. It draws from kit flow <stage> --json and never parses kit's text. While the bar has the
// keyboard, left and right move along it; a stage's number, or a click on the bar, selects it from anywhere and gives
// the bar the keyboard. Down enters a stage's rows and up on the first row leaves them; in the rows, right opens a row
// and left closes it, and a click on a row puts the cursor there, and on the row under it opens or closes it; r runs
// the stage's command again; ? shows the sidebar beside the rows, the stage's panel over its legend, and again takes it
// away, as a click anywhere on the rule beside the rows does; tab gives the panel the keyboard, and / or a click
// on the search, always on the screen's top right, gives the search the keyboard, the sidebar shown or not. A click on a linked glyph, as an issue's before its title, opens its page in the browser. Esc goes back
// one level: from the panel, the sidebar off, then from the rows to the bar. Ctrl-c quits with 130 from anywhere, and
// alt-esc or super-esc, as a terminal that reports it sends command-esc, quits with 0. A stage's command has two
// minutes; past them it is a refusal, and quitting kills the commands still running.
package flow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// Options is how the screen opens: the stages in the picture's order, origin for the title and the URLs when known,
// whether the fetch is skipped, and the filters backlog, underway, and changes take.
type Options struct {
	Stages  []string
	Origin  Origin
	NoFetch bool
	Filters []string
}

// loadedMsg brings a stage's JSON, or why kit did not print it, from the stage's load numbered gen.
type loadedMsg struct {
	stage string
	data  []byte
	err   error
	gen   int
}

// stage is one stage of the picture: its name, what kit printed for it, and where the person is in its rows.
type stage struct {
	name    string
	loading bool
	refusal string // kit's line, when the command failed
	issues  []issue
	under   []underway
	in      *incoming
	desk    *checkout
	trees   *worktrees
	pulls   []pull
	runs    []run
	rels    []release
	cursor  int                // the row under the cursor, kept while another stage is shown
	offset  int                // the first line of the pane
	open    map[string]bool    // the rows opened, by key
	view    grid.Panel         // the stage's panel: its search, the columns hidden, whose rows show, and its cursor
	gen     int                // the stage's latest load: an answer from an earlier one is dropped
	ctx     context.Context    // the latest load's, inside the screen's; nil for the first load, which has the screen's
	stop    context.CancelFunc // ends the latest load's command, when r starts another
}

// known names the stages kittui draws.
var known = map[string]bool{"backlog": true, "underway": true, "incoming": true, "desk": true, "worktrees": true,
	"changes": true, "builds": true, "releases": true}

// deadline is how long a stage's command may take: past it, the stage is a refusal, and r runs it again.
const deadline = 2 * time.Minute

// Model is the picture on the screen.
type Model struct {
	runner        kit.Runner
	ctx           context.Context    // ends with the screen, and every stage's command with it
	cancel        context.CancelFunc // ends ctx
	limit         time.Duration      // how long a stage's command may take: the deadline, shorter in a test
	origin        Origin
	noFetch       bool
	filters       []string
	stages        []stage
	stage         int                    // the selected stage
	inRows        bool                   // whether the rows have the keyboard, not the bar
	inPanel       bool                   // whether the panel has the keyboard, before the rows or the bar
	sidebar       bool                   // whether the sidebar, the panel and the legend, shows beside the rows
	sideOffset    int                    // the sidebar's first line, when it is taller than the pane
	widest        int                    // the cells of the widest stage's panel or legend
	me            string                 // the caller's login, once kit has said it, which the panels' mine and others match
	asked         bool                   // whether kit has been asked for the caller's login
	browse        func(url string) error // opens a page in the browser
	notice        string                 // why a page did not open, in place of the keys until the next key
	menu          bool                   // whether the selected tab's menu is open, which then has the keyboard
	menuAt        int                    // the menu's item under the keyboard
	width, height int
	interrupted   bool
	built         *built // each stage's rows as last built, shared by the model's copies until it is replaced
}

// built holds each stage's rows as they were last built, and the width they were laid out for. A stage that loads, a
// refresh, or a new width replaces it, since a stage's rows may draw from another's, as the desk's from changes'.
type built struct {
	width  int
	stages map[int]shown
}

// rebuild drops the rows built so far; each is built again when it is next drawn.
func (m *Model) rebuild() {
	m.built = &built{width: m.paneWidth(), stages: map[int]shown{}}
}

// Open reads the stages from kit and origin from git, and opens the screen on them. The words are kit flow's own:
// --no-fetch, and the filters backlog, underway, and changes take. Kit's refusal of --stages is the error.
func Open(r kit.Runner, words []string) (Model, error) {
	stages, err := r.Stages()
	if err != nil {
		return Model{}, err
	}
	o := Options{Stages: stages, Origin: ReadOrigin(r.Dir)}
	for _, word := range words {
		if word == "--no-fetch" {
			o.NoFetch = true
		} else {
			o.Filters = append(o.Filters, word)
		}
	}
	return New(r, o), nil
}

// New opens the screen on the options; every stage loads once Init starts it. A stage kittui does not know shows a
// line saying so, and runs nothing.
func New(r kit.Runner, o Options) Model {
	m := Model{runner: r, limit: deadline, origin: o.Origin, noFetch: o.NoFetch, filters: o.Filters, width: 80, height: 24,
		browse: browser}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.rebuild()
	for _, name := range o.Stages {
		s := stage{name: name, loading: true, open: map[string]bool{}}
		if !known[name] {
			s.loading, s.refusal = false, fmt.Sprintf("kittui does not draw the stage '%s'", name)
		}
		m.stages = append(m.stages, s)
		m.widest = max(m.widest, pane.Width(legendLines(name)), grid.Panel{Columns: columns[name], Owned: owned[name]}.Width())
	}
	return m
}

// Init starts the loading: every stage at once, so none waits on incoming's fetch, which can take as long as the
// remotes do; the stages the fetch moves load again once it has fetched.
func (m Model) Init() tea.Cmd { return m.all() }

// Interrupted says whether ctrl-c ended the screen.
func (m Model) Interrupted() bool { return m.interrupted }

// Update takes a message.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.rebuild()
		m.fit()
	case loadedMsg:
		if !m.take(msg) || msg.stage != "incoming" {
			return m, nil
		}
		if in := m.find("incoming").in; in != nil && in.Fetched {
			// the fetch, at the start or on r, moved what some stages read: they load again
			return m, m.reloads()
		}
	case meMsg:
		if msg.err == nil {
			m.me = msg.login
			m.rebuild()
		}
	case browsedMsg:
		if msg.err != nil {
			m.notice = fmt.Sprintf("%s could not open %s: %v", Failed, msg.url, msg.err)
		}
	case tea.KeyPressMsg:
		m.notice = ""
		return m.key(msg)
	case tea.MouseWheelMsg:
		d := 0
		switch msg.Button {
		case tea.MouseWheelUp:
			d = -1
		case tea.MouseWheelDown:
			d = 1
		}
		switch {
		case msg.X > m.paneWidth()+1 && m.sidebar:
			m.scrollSide(d)
		case msg.X < m.paneWidth() && msg.Y >= paneTop && msg.Y < paneTop+m.paneHeight():
			m.scrollRows(d * wheel)
		}
	case tea.MouseClickMsg:
		switch side := m.paneWidth() + 1; {
		case m.menu && msg.Button == tea.MouseLeft:
			m.menuClick(msg.X, msg.Y)
		case msg.Button == tea.MouseLeft && msg.Y == m.bar().Row+1 && msg.X == m.bar().MenuAt():
			m.openMenu()
		case msg.Button != tea.MouseLeft:
			if bar, ok := m.bar().Update(msg); ok {
				m.choose(bar)
			}
		case msg.X == side && msg.Y >= paneTop && msg.Y < m.sideHeight():
			return m, m.showSidebar(!m.sidebar)
		case msg.X >= m.width-m.sideWidth() && (m.sidebar || msg.Y == 0):
			return m, m.clickPanel(msg.Y)
		default:
			if bar, ok := m.bar().Update(msg); ok {
				m.choose(bar)
			} else {
				return m, m.click(msg.X, msg.Y)
			}
		}
	}
	return m, nil
}

// browsedMsg says whether the browser was asked to open the page at url: err is why not.
type browsedMsg struct {
	url string
	err error
}

// browser is how a screen opens a page, which New gives it: the browser, as browse opens it.
var browser = browse

// browse opens a page in the browser: open on macOS, xdg-open elsewhere, without waiting for it.
func browse(url string) error {
	opener := "xdg-open"
	if runtime.GOOS == "darwin" {
		opener = "open"
	}
	return exec.Command(opener, url).Start()
}

// openPage asks the browser for the page at url.
func (m Model) openPage(url string) tea.Cmd {
	b := m.browse
	return func() tea.Msg { return browsedMsg{url, b(url)} }
}

// meMsg is kit's answer to the caller's login: the login, or why it could not be read.
type meMsg struct {
	login string
	err   error
}

// panel gives a stage's panel: what the person set in it, the columns it offers, whether its rows belong to people,
// the caller's login, and whether it has the keyboard.
func (m Model) panel(i int) grid.Panel {
	p := m.filter(i)
	p.Columns = m.offered(i)
	return p
}

// filter gives a stage's panel as its rows are let through: what the person set in it, whether its rows belong to
// people, and the caller's login; the columns it offers are drawn from the rows, so it has none.
func (m Model) filter(i int) grid.Panel {
	s := m.stages[i]
	p := s.view
	p.Owned, p.Me = owned[s.name], m.me
	p.Focused = m.inPanel && i == m.stage
	return p
}

// setPanel keeps what the selected stage's panel became; the rows are let through again, and the cursor stays within
// them.
func (m *Model) setPanel(p grid.Panel) {
	m.stages[m.stage].view = p
	m.rebuild()
	m.clamp(m.stage)
	m.fit()
}

// enterPanel gives the panel the keyboard: the search under it, as it is, shown or not, when find says so; else the
// panel's control under it, the sidebar shown, which asks kit for the caller's login the first time.
func (m *Model) enterPanel(find bool) tea.Cmd {
	if len(m.stages) == 0 {
		return nil
	}
	if find {
		m.inPanel = true
		m.stages[m.stage].view = m.panel(m.stage).Find()
		return nil
	}
	cmd := m.showSidebar(true)
	m.inPanel = true
	return cmd
}

// clickPanel takes a click on a line of the screen's right, from its first: on the search, the panel has the keyboard
// and the search under it, as / does; on one of the panel's other controls, the panel has the keyboard and acts on it
// as space does; on the … above or … more, the sidebar scrolls a page.
func (m *Model) clickPanel(line int) tea.Cmd {
	if len(m.stages) == 0 {
		return nil
	}
	if line == 0 {
		return m.enterPanel(true)
	}
	line--
	n, h := m.scrolled()
	start, end, above, below := pane.Window(n, h, m.sideOffset)
	k := pane.At(n, h, m.sideOffset, line)
	switch {
	case above && line == 0:
		m.scrollSide(-max(1, h-2))
	case below && line == end-start+btoi(above):
		m.scrollSide(max(1, h-2))
	case k >= 0:
		if p, ok := m.panel(m.stage).Click(k + 1); ok {
			m.inPanel = true
			m.setPanel(p)
		}
	}
	return nil
}

// scrolled gives how many of the sidebar's lines scroll, all but the search on the screen's first line, and the lines
// they have, all under it down to the keys bar.
func (m Model) scrolled() (n, h int) { return len(m.sideLines()) - 1, m.sideHeight() - 1 }

// btoi gives 1 for true, 0 for false.
func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// wheel is the lines a turn of the mouse wheel scrolls the rows.
const wheel = 3

// scrollRows scrolls the rows n lines, down when n is positive, never past their ends; the cursor stays where it is,
// until a key moves it and the pane follows it again.
func (m *Model) scrollRows(n int) {
	if len(m.stages) == 0 {
		return
	}
	s := &m.stages[m.stage]
	s.offset, _, _, _ = pane.Window(len(m.lines(m.stage)), m.paneHeight(), s.offset+n)
}

// scrollSide scrolls the sidebar under the search n lines, down when n is positive, never past its ends.
func (m *Model) scrollSide(n int) {
	lines, h := m.scrolled()
	m.sideOffset, _, _, _ = pane.Window(lines, h, m.sideOffset+n)
}

// fitSide scrolls the sidebar so the panel's control under the keyboard shows; the search always does, and on it the
// sidebar is back at its top.
func (m *Model) fitSide() {
	_, at := m.panel(m.stage).Lines()
	n, h := m.scrolled()
	line := slices.Index(at, m.stages[m.stage].view.Cursor) - 1
	if line < 0 {
		m.sideOffset = 0
	}
	for line >= 0 {
		start, end, _, _ := pane.Window(n, h, m.sideOffset)
		switch {
		case line < start:
			m.sideOffset = line
		case line >= end:
			m.sideOffset++
		default:
			return
		}
	}
}

// askMe runs kit flow --me, once, for the limit at most.
func (m *Model) askMe() tea.Cmd {
	if m.asked {
		return nil
	}
	m.asked = true
	r, ctx, limit := m.runner, m.ctx, m.limit
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, limit)
		defer cancel()
		login, err := r.Me(ctx)
		return meMsg{login, err}
	}
}

// key takes a key press: the quitting keys, the menu's while it is open, the panel's while it has the keyboard, = for
// the menu, the bar's, then the rows' and the sidebar's.
func (m Model) key(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "ctrl+c":
		m.interrupted = true
		return m, m.quit()
	case "alt+esc", "super+esc":
		return m, m.quit()
	}
	if m.menu {
		return m.menuKey(k)
	}
	if m.inPanel {
		return m.panelKey(k)
	}
	if k.String() == "=" {
		m.openMenu()
		return m, nil
	}
	if bar, ok := m.bar().Update(k); ok {
		m.choose(bar)
		return m, nil
	}
	switch k.String() {
	case "tab":
		return m, m.enterPanel(false)
	case "/":
		return m, m.enterPanel(true)
	case "esc":
		switch {
		case m.sidebar:
			m.showSidebar(false)
		case m.inRows:
			s := &m.stages[m.stage]
			s.cursor, s.offset, m.inRows = 0, 0, false
		}
		m.fit()
	case "?":
		return m, m.showSidebar(!m.sidebar)
	case "up":
		m.up(1)
	case "down":
		m.down(1)
	case "pgup":
		m.page(-1)
	case "pgdown":
		m.page(1)
	case "right":
		m.open(true)
	case "left":
		m.open(false)
	case "r":
		return m, m.refresh()
	}
	return m, nil
}

// panelKey takes a key while the panel has the keyboard: tab and esc give it back to the rows or the bar, as they had
// it; page up and down scroll the sidebar, and so does down past the last control, so the legend under the panel can
// be read; the panel takes what it can, as typing in its search, the sidebar following its cursor; a number the bar
// takes selects a stage, and gives the bar the keyboard. While the sidebar is hidden the search stands alone: down and
// a page at a time have nothing to move to.
func (m Model) panelKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	h := max(1, m.sideHeight()-3)
	switch k.String() {
	case "tab", "esc":
		m.inPanel = false
		return m, nil
	case "down", "pgdown", "pgup":
		if !m.sidebar {
			return m, nil
		}
	}
	switch k.String() {
	case "pgdown":
		m.scrollSide(h)
		return m, nil
	case "pgup":
		m.scrollSide(-h)
		return m, nil
	}
	before := m.panel(m.stage)
	if k.String() == "=" && before.Cursor != 0 {
		// off the search, = drops the tab's menu down, as it does from the rows
		m.openMenu()
		return m, nil
	}
	if p, ok := before.Update(k); ok {
		m.setPanel(p)
		if k.String() == "down" && p.Cursor == before.Cursor {
			// past the last control, down scrolls on to the legend under the panel
			m.scrollSide(1)
		} else {
			m.fitSide()
		}
		return m, nil
	}
	m.inPanel = false
	if bar, ok := m.bar().Update(k); ok {
		m.choose(bar)
		return m, nil
	}
	m.inPanel = true
	return m, nil
}

// showSidebar shows the sidebar beside the rows, or takes it away; the rows are laid out again for the width left, and
// the panel loses the keyboard with it. Shown the first time, it asks kit for the caller's login.
func (m *Model) showSidebar(on bool) tea.Cmd {
	m.sidebar = on
	m.inPanel = m.inPanel && on
	m.rebuild()
	m.fit()
	if on {
		return m.askMe()
	}
	return nil
}

// choose takes what the bar chose: the stage it selected, where the cursor is as it was left, and the keyboard.
func (m *Model) choose(bar tabs.Bar) {
	if bar.Selected != m.stage {
		m.sideOffset = 0
	}
	m.stage, m.inRows, m.inPanel = bar.Selected, !bar.Focused, false
	m.fit()
}

// quit ends the screen, and the stages' commands still running with it.
func (m Model) quit() tea.Cmd {
	m.cancel()
	return tea.Quit
}

// find gives the stage by name, or nil when it is not in the picture.
func (m Model) find(name string) *stage {
	for i := range m.stages {
		if m.stages[i].name == name {
			return &m.stages[i]
		}
	}
	return nil
}

// args gives the words a stage's command takes after --json: --no-fetch for incoming, the filters for backlog,
// underway, and changes, nothing for the rest.
func (m Model) args(name string) []string {
	switch name {
	case "incoming":
		if m.noFetch {
			return []string{"--no-fetch"}
		}
	case "backlog", "underway", "changes":
		return m.filters
	}
	return nil
}

// load runs a stage's command: kit flow <stage> --json, with the stage's words, for the limit at most, as the stage's
// latest load, which r supersedes.
func (m Model) load(name string) tea.Cmd {
	r, args, ctx, limit, gen := m.runner, m.args(name), m.ctx, m.limit, 0
	if s := m.find(name); s != nil {
		gen = s.gen
		if s.ctx != nil {
			ctx = s.ctx
		}
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, limit)
		defer cancel()
		data, err := r.Status(ctx, name, args)
		return loadedMsg{stage: name, data: data, err: err, gen: gen}
	}
}

// all runs every stage kittui draws, at once.
func (m Model) all() tea.Cmd {
	var cmds []tea.Cmd
	for _, s := range m.stages {
		if known[s.name] {
			cmds = append(cmds, m.load(s.name))
		}
	}
	return tea.Batch(cmds...)
}

// refresh runs the selected stage's command again, and shows it loading until it answers.
func (m *Model) refresh() tea.Cmd {
	if len(m.stages) == 0 || !known[m.stages[m.stage].name] {
		return nil
	}
	return m.restart(m.stages[m.stage].name)
}

// restart runs a stage's command again as its latest load, ending the one before when it still runs, and shows the
// stage loading until it answers.
func (m *Model) restart(name string) tea.Cmd {
	cmd, s := m.again(name), m.find(name)
	s.loading, s.refusal = true, ""
	s.issues, s.under, s.in, s.desk, s.trees, s.pulls, s.runs, s.rels = nil, nil, nil, nil, nil, nil, nil, nil
	m.rebuild()
	return cmd
}

// again runs a stage's command again as its latest load, ending the one before when it still runs; the stage keeps
// what it shows until the answer replaces it.
func (m *Model) again(name string) tea.Cmd {
	s := m.find(name)
	if s.stop != nil {
		s.stop()
	}
	s.ctx, s.stop = context.WithCancel(m.ctx)
	s.gen++
	return m.load(name)
}

// moved names the stages whose rows the fetch moves: the desk's and the worktrees' counts against the default branch,
// the issues underway, which the pull requests the fetch brings close, and the backlog's branches, which it reads as
// the fetch left them.
var moved = []string{"backlog", "desk", "worktrees", "underway"}

// reloads runs again each stage the fetch moves, when it is in the picture, each keeping its rows on the screen until
// the new ones come.
func (m *Model) reloads() tea.Cmd {
	var cmds []tea.Cmd
	for _, name := range moved {
		if m.find(name) != nil {
			cmds = append(cmds, m.again(name))
		}
	}
	return tea.Batch(cmds...)
}

// take keeps what a stage's command printed, decoded by the stage, or the refusal when it failed, and says whether it
// did. The answer of a load a later one superseded is dropped. The cursor stays within the rows that came.
func (m *Model) take(msg loadedMsg) bool {
	s := m.find(msg.stage)
	if s == nil || msg.gen != s.gen {
		return false
	}
	s.loading = false
	err := msg.err
	if err == nil {
		err = s.decode(msg.data)
	}
	if err != nil {
		s.refusal = refusalOf(s.name, err)
	}
	m.rebuild()
	for i := range m.stages {
		m.clamp(i)
	}
	m.fit()
	return true
}

// decode keeps what the stage's command printed. A document of another shape, as kit prints above repositories, is
// an error naming it, and JSON that does not decode is one too.
func (s *stage) decode(data []byte) error {
	if err := shape(s.name, data); err != nil {
		return err
	}
	var err error
	switch s.name {
	case "backlog":
		err = json.Unmarshal(data, &s.issues)
	case "underway":
		err = json.Unmarshal(data, &s.under)
	case "incoming":
		s.in = &incoming{}
		err = json.Unmarshal(data, s.in)
	case "desk":
		s.desk = &checkout{}
		err = json.Unmarshal(data, s.desk)
	case "worktrees":
		s.trees = &worktrees{}
		err = json.Unmarshal(data, s.trees)
	case "changes":
		err = json.Unmarshal(data, &s.pulls)
	case "builds":
		err = json.Unmarshal(data, &s.runs)
	case "releases":
		err = json.Unmarshal(data, &s.rels)
	}
	if err != nil {
		return fmt.Errorf("kit flow %s --json: %v", s.name, err)
	}
	return nil
}

// expected is each stage's document, as kit flow's page says: an array of objects for backlog, underway,
// changes, builds, and releases, each object with the stage's key fields; an object for incoming, desk, and
// worktrees, incoming's with its remotes.
var expected = map[string]struct {
	what  string
	array bool
	keys  []string
}{
	"backlog":   {"an array of issues", true, []string{"number"}},
	"underway":  {"an array of issues underway", true, []string{"number", "pr"}},
	"changes":   {"an array of pull requests", true, []string{"number"}},
	"builds":    {"an array of builds", true, []string{"id"}},
	"releases":  {"an array of releases", true, []string{"version"}},
	"incoming":  {"an object of remotes", false, []string{"remotes"}},
	"desk":      {"an object", false, nil},
	"worktrees": {"an object", false, nil},
}

// shape refuses a document that is not the stage's: an array where an object is expected, an object where an array
// is, an array whose objects lack one of the stage's key fields, as the array of desks kit prints above desks, or an
// object without its key field, as incoming's before every remote. Anything else is left to the decoder.
func shape(name string, data []byte) error {
	want, printed := expected[name], ""
	switch data = bytes.TrimSpace(data); {
	case len(data) == 0:
	case data[0] == '[' && !want.array:
		printed = "an array"
	case data[0] == '{' && want.array:
		printed = "an object"
	case data[0] == '[':
		var objects []map[string]json.RawMessage
		if json.Unmarshal(data, &objects) != nil {
			return nil
		}
		for _, o := range objects {
			if key := lacking(o, want.keys); key != "" {
				printed = fmt.Sprintf("an array of objects without %q", key)
				break
			}
		}
	case data[0] == '{':
		var object map[string]json.RawMessage
		if json.Unmarshal(data, &object) != nil {
			return nil
		}
		if key := lacking(object, want.keys); key != "" {
			printed = fmt.Sprintf("an object without %q", key)
		}
	}
	if printed == "" {
		return nil
	}
	return fmt.Errorf("kit flow %s --json printed %s where %s was expected; run kit flow inside a repository",
		name, printed, want.what)
}

// lacking gives the first of the keys an object lacks, or nothing.
func lacking(o map[string]json.RawMessage, keys []string) string {
	for _, key := range keys {
		if _, ok := o[key]; !ok {
			return key
		}
	}
	return ""
}

// refusalOf gives a stage's refusal from its error: that the stage took longer than the deadline and r runs it
// again; else kit's line; and when kit died with nothing on stderr, how: exited with its code, or killed.
func refusalOf(name string, err error) string {
	var exit *kit.ExitError
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Sprintf("kit flow %s took longer than %d minutes; r runs it again", name, int(deadline.Minutes()))
	case err.Error() != "":
		return err.Error()
	case errors.As(err, &exit) && exit.Code >= 0:
		return fmt.Sprintf("kit flow %s exited %d with nothing on stderr", name, exit.Code)
	}
	return fmt.Sprintf("kit flow %s was killed", name)
}

// clamp keeps a stage's cursor within its rows.
func (m *Model) clamp(i int) {
	s := &m.stages[i]
	if n := len(m.rows(i)); s.cursor >= n {
		s.cursor = max(0, n-1)
	}
}

// up moves the cursor n rows up; past the first row, it gives the bar the keyboard, and the pane goes to its top.
func (m *Model) up(n int) {
	if !m.inRows || len(m.stages) == 0 {
		return
	}
	s := &m.stages[m.stage]
	if s.cursor-n < 0 {
		s.cursor, s.offset, m.inRows = 0, 0, false
	} else {
		s.cursor -= n
	}
	m.fit()
}

// down moves the cursor n rows down; from the bar, it enters the rows, when the stage has any.
func (m *Model) down(n int) {
	if len(m.stages) == 0 {
		return
	}
	s := &m.stages[m.stage]
	rows := len(m.rows(m.stage))
	if rows == 0 {
		return
	}
	if !m.inRows {
		m.inRows = true
		n--
	}
	s.cursor = min(s.cursor+n, rows-1)
	m.fit()
}

// open opens the row under the cursor, or closes it; a leaf stays as it is.
func (m *Model) open(open bool) {
	if !m.inRows || len(m.stages) == 0 {
		return
	}
	s := &m.stages[m.stage]
	rows := m.rows(m.stage)
	if s.cursor >= len(rows) || !grid.Opens(rows, s.cursor, m.paneWidth()) {
		return
	}
	s.open[rows[s.cursor].Key] = open
	m.fit()
}

// click takes a left click on the pane's rows: the cursor goes to the row clicked, and the rows have the keyboard; a
// click on a linked glyph of a row's first line, as an issue's before its title, opens its page in the browser; one
// on the first line of the row already under the cursor, or on the gutter of a row's first line, its ▸ or ▾, opens
// the row, or closes it. A click on a … line, or on a line about the stage, does nothing.
func (m *Model) click(x, y int) tea.Cmd {
	if len(m.stages) == 0 || x >= m.paneWidth() {
		return nil
	}
	lines, s := m.lines(m.stage), &m.stages[m.stage]
	k := pane.At(len(lines), m.paneHeight(), s.offset, y-paneTop)
	if k < 0 || lines[k].row < 0 {
		return nil
	}
	rows := m.rows(m.stage)
	row, first := lines[k].row, k == 0 || lines[k-1].row != lines[k].row
	link := ""
	if first {
		link = grid.LinkAt(rows, row, m.paneWidth(), x)
	}
	toggle := first && link == "" && (x < 3 || m.inRows && row == s.cursor)
	s.cursor, m.inRows, m.inPanel = row, true, false
	switch {
	case link != "":
		m.fit()
		return m.openPage(link)
	case toggle:
		m.open(!s.open[rows[row].Key])
	default:
		m.fit()
	}
	return nil
}

// page moves a page down or up. Inside a row taller than the pane, as an open pull request with many issues is, it
// first scrolls through that row, so every line it hides can be reached; at the row's end it moves the cursor a page
// of rows. A page never leaves the rows: up stops at the first row, and down at the last; on the bar it does nothing.
func (m *Model) page(d int) {
	h := max(1, m.paneHeight()-1)
	if !m.inRows || len(m.stages) == 0 {
		return
	}
	s := &m.stages[m.stage]
	all := m.lines(m.stage)
	first, last := -1, -1
	for k, l := range all {
		if l.row == s.cursor {
			if first < 0 {
				first = k
			}
			last = k
		}
	}
	start, end, _, _ := pane.Window(len(all), m.paneHeight(), s.offset)
	switch {
	case first >= 0 && d > 0 && last >= end:
		s.offset, _, _, _ = pane.Window(len(all), m.paneHeight(), s.offset+h)
	case first >= 0 && d < 0 && first < start:
		s.offset, _, _, _ = pane.Window(len(all), m.paneHeight(), max(first, s.offset-h))
	case d > 0:
		m.down(h)
	default:
		s.cursor = max(0, s.cursor-h)
		m.fit()
	}
}

// fit scrolls the pane so the row under the cursor shows, with what it opens to when that fits, and never past the
// last line.
func (m *Model) fit() {
	if len(m.stages) == 0 {
		return
	}
	s := &m.stages[m.stage]
	lines := m.lines(m.stage)
	h := m.paneHeight()
	s.offset, _, _, _ = pane.Window(len(lines), h, s.offset)
	if !m.inRows {
		return
	}
	first, last := -1, -1
	for k, l := range lines {
		if l.row == s.cursor {
			if first < 0 {
				first = k
			}
			last = k
		}
	}
	if first < 0 {
		return
	}
	// Scroll until the cursor's row shows whole, or its first line and as much as the pane holds; a … line takes the
	// place of a row at either edge, so the window is asked rather than counted.
	for {
		start, end, _, _ := pane.Window(len(lines), h, s.offset)
		switch {
		case first < start:
			s.offset = first
		case last >= end && first > start:
			s.offset++
		default:
			return
		}
	}
}
