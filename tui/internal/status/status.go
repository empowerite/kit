// Package status is kit status on a terminal: the stages along one line, a tree hanging from each, the selected
// stage's rows under them, and a row opened in place. It draws from kit status <stage> --json and never parses kit's
// text. Left and right move along the line, even inside the rows; down enters a stage's rows and up on the first row
// leaves them; Enter opens or closes a row; r runs the stage's command again; ? hangs each stage's legend in its tree,
// and again takes it down. Esc goes back one level: the legend off, from the rows to the line, and from the line it
// quits; ctrl-c quits with 130 from anywhere. A stage's command has two minutes; past them it is a refusal, and
// quitting kills the commands still running.
package status

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/tui/internal/kit"
)

// Options is how the screen opens: the stages in the picture's order, origin for the title and the URLs when known,
// whether the fetch is skipped, and the filters backlog, underway, and changes take.
type Options struct {
	Stages  []string
	Origin  Origin
	NoFetch bool
	Filters []string
}

// loadedMsg brings a stage's JSON, or why kit did not print it.
type loadedMsg struct {
	stage string
	data  []byte
	err   error
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
	cursor  int             // the row under the cursor, kept while another stage is shown
	offset  int             // the first line of the pane
	open    map[string]bool // the rows opened, by key
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
	stage         int  // the selected stage
	inRows        bool // whether the cursor is on a row, not on the line
	legend        bool // whether each tree shows its stage's legend
	scroll        int  // the first cell of the canvas shown, when the line is wider than the screen
	started       bool // whether the stages after incoming have been started
	moving        bool // whether the dot is on its way along the line
	dot           dot  // where the dot is, and how far it has to go
	width, height int
	interrupted   bool
}

// Open reads the stages from kit and origin from git, and opens the screen on them. The words are kit status's own:
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
	m := Model{runner: r, limit: deadline, origin: o.Origin, noFetch: o.NoFetch, filters: o.Filters, width: 80, height: 24}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	for _, name := range o.Stages {
		s := stage{name: name, loading: true, open: map[string]bool{}}
		if !known[name] {
			s.loading, s.refusal = false, fmt.Sprintf("kittui does not draw the stage '%s'", name)
		}
		m.stages = append(m.stages, s)
	}
	return m
}

// Init starts the loading: incoming first, since it fetches, and the rest once it has answered; without incoming
// in the picture, every stage at once.
func (m Model) Init() tea.Cmd {
	if m.find("incoming") != nil {
		return m.load("incoming")
	}
	return m.others()
}

// Interrupted says whether ctrl-c ended the screen.
func (m Model) Interrupted() bool { return m.interrupted }

// Update takes a message.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.moving = false
		m.fit()
	case frameMsg:
		cmd := m.step()
		return m, cmd
	case loadedMsg:
		m.take(msg)
		if msg.stage == "incoming" && !m.started {
			m.started = true
			return m, m.others()
		}
	case tea.KeyPressMsg:
		return m.key(msg)
	}
	return m, nil
}

// key takes a key press: ctrl-c anywhere, and the line's and the rows' keys.
func (m Model) key(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "ctrl+c":
		m.interrupted = true
		return m, m.quit()
	case "esc":
		switch {
		case m.legend:
			m.legend, m.moving = false, false
		case !m.inRows:
			return m, m.quit()
		default:
			s := &m.stages[m.stage]
			s.cursor, s.offset, m.inRows = 0, 0, false
		}
		m.fit()
	case "?":
		m.legend, m.moving = !m.legend, false
		m.fit()
	case "left":
		cmd := m.move(-1)
		return m, cmd
	case "right":
		cmd := m.move(1)
		return m, cmd
	case "up":
		m.up(1)
	case "down":
		m.down(1)
	case "pgup":
		m.page(-1)
	case "pgdown":
		m.page(1)
	case "enter":
		m.toggle()
	case "r":
		return m, m.refresh()
	}
	return m, nil
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

// load runs a stage's command: kit status <stage> --json, with the stage's words, for the limit at most.
func (m Model) load(name string) tea.Cmd {
	r, args, ctx, limit := m.runner, m.args(name), m.ctx, m.limit
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, limit)
		defer cancel()
		data, err := r.Status(ctx, name, args)
		return loadedMsg{stage: name, data: data, err: err}
	}
}

// others runs every stage but incoming, at once.
func (m Model) others() tea.Cmd {
	var cmds []tea.Cmd
	for _, s := range m.stages {
		if s.name != "incoming" && known[s.name] {
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
	s := &m.stages[m.stage]
	s.loading, s.refusal = true, ""
	s.issues, s.under, s.in, s.desk, s.trees, s.pulls, s.runs, s.rels = nil, nil, nil, nil, nil, nil, nil, nil
	return m.load(s.name)
}

// take keeps what a stage's command printed, decoded by the stage, or the refusal when it failed. The cursor stays
// within the rows that came.
func (m *Model) take(msg loadedMsg) {
	s := m.find(msg.stage)
	if s == nil {
		return
	}
	s.loading = false
	err := msg.err
	if err == nil {
		err = s.decode(msg.data)
	}
	if err != nil {
		s.refusal = refusalOf(s.name, err)
	}
	for i := range m.stages {
		m.clamp(i)
	}
	m.fit()
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
		return fmt.Errorf("kit status %s --json: %v", s.name, err)
	}
	return nil
}

// expected is each stage's document, as kit status's page says: an array of objects for backlog, underway,
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
	return fmt.Errorf("kit status %s --json printed %s where %s was expected; run kit status inside a repository",
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
		return fmt.Sprintf("kit status %s took longer than %d minutes; r runs it again", name, int(deadline.Minutes()))
	case err.Error() != "":
		return err.Error()
	case errors.As(err, &exit) && exit.Code >= 0:
		return fmt.Sprintf("kit status %s exited %d with nothing on stderr", name, exit.Code)
	}
	return fmt.Sprintf("kit status %s was killed", name)
}

// clamp keeps a stage's cursor within its rows.
func (m *Model) clamp(i int) {
	s := &m.stages[i]
	if n := len(m.rows(i)); s.cursor >= n {
		s.cursor = max(0, n-1)
	}
}

// move selects the stage d along the line, round its ends, staying in the rows or on the line as before, and sends
// the dot there over the canvas, the way the arrow points; the view follows it.
func (m *Model) move(d int) tea.Cmd {
	n := len(m.stages)
	if n == 0 {
		return nil
	}
	from := m.center(m.stage)
	if m.moving {
		from = m.dot.at
	}
	m.stage = (m.stage + d + n) % n
	l := m.layout()
	w := l.width
	// the cells to go, always the way the arrow points, round the end of the line when it must
	far := ((l.at[m.stage]-from)*d%w + w) % w
	already := m.moving
	m.moving = far != 0
	if m.moving {
		m.dot = dot{at: from, left: far, dir: d, step: max(1, (far+frames-1)/frames)}
	}
	m.fit()
	if !m.moving || already {
		return nil
	}
	return frame()
}

// dot is the selected stage's marker on its way along the line: the cell it is at, the cells it has left to go, the
// way it goes, and how far it goes each frame.
type dot struct {
	at, left, dir, step int
}

// frames is how many frames the dot takes to reach its stage, however far.
const frames = 8

// frameEvery is how long a frame lasts; the tests shorten it.
var frameEvery = 16 * time.Millisecond

// frameMsg asks the dot to take its next step.
type frameMsg struct{}

// frame asks for the next step after a frame.
func frame() tea.Cmd {
	return tea.Tick(frameEvery, func(time.Time) tea.Msg { return frameMsg{} })
}

// step moves the dot one frame along, round the end of the canvas, and stops it on its stage; the view follows it.
func (m *Model) step() tea.Cmd {
	if !m.moving {
		return nil
	}
	w := m.layout().width
	stride := min(m.dot.step, m.dot.left)
	m.dot.at = ((m.dot.at+m.dot.dir*stride)%w + w) % w
	m.dot.left -= stride
	m.moving = m.dot.left > 0
	m.follow()
	if !m.moving {
		return nil
	}
	return frame()
}

// up moves the cursor n rows up; past the first row, it returns to the line, and the pane to its top.
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

// down moves the cursor n rows down; from the line, it enters the rows, when the stage has any.
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

// toggle opens the row under the cursor, or closes it; a leaf stays as it is.
func (m *Model) toggle() {
	if !m.inRows || len(m.stages) == 0 {
		return
	}
	s := &m.stages[m.stage]
	rows := m.rows(m.stage)
	if s.cursor >= len(rows) || len(rows[s.cursor].detail) == 0 {
		return
	}
	key := rows[s.cursor].key
	s.open[key] = !s.open[key]
	m.fit()
}

// page moves a page down or up. Inside a row taller than the pane, as an open pull request with many issues is, it
// first scrolls through that row, so every line it hides can be reached; at the row's end it moves the cursor a page
// of rows. A page up stops at the first row, and from there leaves the rows for the line.
func (m *Model) page(d int) {
	h := max(1, m.paneHeight()-1)
	if !m.inRows || len(m.stages) == 0 {
		if d > 0 {
			m.down(h)
		}
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
	start, end, _, _ := window(len(all), m.paneHeight(), s.offset)
	switch {
	case first >= 0 && d > 0 && last >= end:
		s.offset, _, _, _ = window(len(all), m.paneHeight(), s.offset+h)
	case first >= 0 && d < 0 && first < start:
		s.offset, _, _, _ = window(len(all), m.paneHeight(), max(first, s.offset-h))
	case d > 0:
		m.down(h)
	case s.cursor > 0:
		// a page up stops at the first row; only from there does it leave the rows for the line
		s.cursor = max(0, s.cursor-h)
		m.fit()
	default:
		m.up(h)
	}
}

// fit scrolls the view sideways to the selected stage's tree, and its pane so the row under the cursor shows, with
// what it opens to when that fits, and never past the last line.
func (m *Model) fit() {
	if len(m.stages) == 0 {
		return
	}
	m.follow()
	s := &m.stages[m.stage]
	lines := m.lines(m.stage)
	h := m.paneHeight()
	s.offset, _, _, _ = window(len(lines), h, s.offset)
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
		start, end, _, _ := window(len(lines), h, s.offset)
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
