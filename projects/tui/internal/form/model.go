// The live form: a Bubble Tea model over the fields, the runner, and the keyboard. Up, down, and tab move between
// fields, left and right along a row or flip a toggle, space toggles or picks, backspace erases, Enter takes the next
// field or line and, on the command line, runs; shift-Enter runs from anywhere a terminal reports it, and the keys bar
// names it only where the terminal said it does; esc goes back
// to the palette and never quits; ctrl-q quits, and ctrl-c quits as interrupted. In a list, page down and page up move the cursor a page,
// and home and end to the first and last row. Where a window hides rows, a dim line says how many. The keys bar is
// the last line.
package form

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/syntax"
)

// BackMsg asks the palette to take the form back: esc on a form opened from the palette.
type BackMsg struct{}

// choicesMsg brings a list's rows; flipped names the value whose flip asked for them again, and asked holds the other
// answers the rows were asked with.
type choicesMsg struct {
	key     string
	rows    []kit.Choice
	set     bool
	err     error
	flipped string
	asked   []string
}

// listWindow is the lines a list's window takes: its rows, and the lines that say how many it hides.
const listWindow = 10

// live is how the form draws on a terminal.
var live = Styles{Chosen: style.Chosen, Off: style.Off, Label: style.Label, Dim: style.Off, Refusal: style.Refusal,
	Cursor: style.Cursor}

// Model is a verb's form on the screen.
type Model struct {
	spec    syntax.Spec
	runner  kit.Runner
	back    bool
	fields  []Field
	focus   int    // the field with the keyboard; len(fields) is the command line
	cursor  int    // the option in a row, or the row in a list
	input   string // the text being typed: a value, a line, or a list's filter
	note    string // the line under the focused field: a list's line about a row, or what kit refused
	refusal string // the line under the command line
	width   int
	height  int  // the terminal's lines; zero before it says, and nothing is clipped
	shift   bool // the terminal tells shift-Enter from Enter, so the keys bar names it

	lists  map[string][]kit.Choice
	sets   map[string]bool
	stale  map[string]bool
	seeded map[string]bool
	failed map[string]bool // the lists kit refused, which take what is typed instead

	extOrder []string
	ext      map[string]syntax.Argument
	extSeen  map[string]bool

	initCmd                tea.Cmd
	run, done, interrupted bool
	args                   []string
}

// New opens the form over the fields; back is true when esc returns to the palette, false when the form stands alone.
func New(spec syntax.Spec, fields []Field, r kit.Runner, back bool) Model {
	m := Model{spec: spec, runner: r, back: back, fields: fields, width: 80, lists: map[string][]kit.Choice{}, sets: map[string]bool{},
		stale: map[string]bool{}, seeded: map[string]bool{}, failed: map[string]bool{}, extSeen: map[string]bool{}}
	if spec.Extends {
		m.refreshExtension()
		m.rebuild(Collect(m.fields))
	}
	m.focus = FirstEmptyRequired(m.fields)
	if m.focus < 0 {
		m.focus = 0
	}
	m.initCmd = m.enter()
	return m
}

// Init fetches the focused list's rows when the form opens on one.
func (m Model) Init() tea.Cmd { return m.initCmd }

// Done says, after the program ends, whether the command line was accepted, and its words after kit <verb>.
func (m Model) Done() (bool, []string) {
	if !m.run {
		return false, nil
	}
	return true, m.args
}

// Interrupted says whether ctrl-c ended the form.
func (m Model) Interrupted() bool { return m.interrupted }

// Fields gives the fields as they stand, a line being typed included, for the transcript.
func (m Model) Fields() []Field { return m.viewFields() }

// CommandLine gives the words after kit <verb> the form builds now.
func (m Model) CommandLine() []string { return CommandLine(m.spec, m.viewFields()) }

// Update takes a message.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyboardEnhancementsMsg:
		m.shift = msg.SupportsKeyDisambiguation()
	case choicesMsg:
		m.choices(msg)
	case matchedMsg:
		cmd := m.matched(msg)
		return m, cmd
	case tea.KeyPressMsg:
		cmd := m.key(msg)
		return m, cmd
	}
	return m, nil
}

// View draws the form on the alternate screen, clipped to the terminal's lines so the way back and the focused field
// stay in sight, with the keys bar on the last line, which nothing above it moves.
func (m Model) View() tea.View {
	focus := m.focus
	if focus >= len(m.fields) {
		focus = -1
	}
	content, header, first, last, at := Layout(m.spec, m.viewFields(), Options{Width: m.width, Focus: focus,
		Cursor: m.cursor, Styles: live, Expand: m.expand, Note: m.note, CommandFocus: m.focus >= len(m.fields),
		CommandRefusal: m.refusal, Back: m.back})
	room := 0
	if m.height > 0 {
		room = max(1, m.height-2) // the gap and the keys bar under the form
	}
	lines := strings.Split(strings.TrimSuffix(window(content, room, header, first, last, at, live.Dim), "\n"), "\n")
	v := tea.NewView(strings.Join(style.Bottom(lines, m.height, m.keys().Line(m.width)), "\n"))
	v.AltScreen = true
	return v
}

// keys gives the path to the keyboard, which the keys bar is drawn from: the form, whose esc goes back to the palette,
// or, opened on its own, whose ctrl-q quits, as esc never does; up and down move between fields, and enter takes the next field; and the field under the keyboard, with
// the keys its kind takes: the arrows across a row of options, the space where it toggles, backspace where it erases
// or clears, and the page keys on a list. On the command line, enter runs. Tab, which moves as down and up do, is left
// unnamed. Shift-Enter is named only where the terminal answered Bubble Tea's ask for the kitty keyboard protocol;
// elsewhere, Terminal.app among them, it arrives as Enter, and the way to run is Enter on the command line.
func (m Model) keys() shell.Path {
	end := shell.Bind("⌃Q", "quit", shell.Quit)
	if m.back {
		end = shell.Bind("esc", "back")
	}
	form := shell.Segment{Takes: []string{"tab", "shift+tab"}, Bindings: []shell.Binding{
		end, shell.Bind("↑↓", "move", "up", "down"), shell.Bind("⏎", "next", "enter")}}
	if m.shift {
		form.Bindings = append(form.Bindings, shell.Bind("⇧⏎", "run", "shift+enter"))
	}
	if m.focus >= len(m.fields) {
		// the command line is the last: up moves back, down has nowhere to go, and enter runs
		return shell.Path{form, {Takes: []string{"down"}, Bindings: []shell.Binding{shell.Bind("↑", "move", "up"),
			shell.Bind("⏎", "run", "enter", "shift+enter")}}}
	}
	var field shell.Segment
	switch f := m.fields[m.focus]; f.Kind {
	case Toggle:
		if !f.Required {
			field.Bindings = append(field.Bindings, shell.Bind("␣", "toggle", "space"))
		}
	case AnyOf:
		field.Bindings = append(field.Bindings, shell.Bind("←→", "option", "left", "right"),
			shell.Bind("␣", "toggle", "space"))
	case OneOf:
		field.Bindings = append(field.Bindings, shell.Bind("←→", "choose", "left", "right"))
		if !f.Required {
			field.Bindings = append(field.Bindings, shell.Bind("⌫", "clear", "backspace"))
		}
	case Text, Lines, Wildcard:
		field.Types = true
		field.Bindings = append(field.Bindings, shell.Bind("⌫", "erase", "backspace"))
	case Pick:
		field.Types = true
		field.Bindings = append(field.Bindings, shell.Bind("⇞⇟", "page", "pgup", "pgdown"),
			shell.Bind("⌫", "erase", "backspace"))
	case PickMany:
		field.Types = true
		field.Bindings = append(field.Bindings, shell.Bind("⇞⇟", "page", "pgup", "pgdown"),
			shell.Bind("␣", "toggle", "space"), shell.Bind("⌫", "erase", "backspace"))
	}
	return shell.Path{form, field}
}

// key takes a key press.
func (m *Model) key(k tea.KeyPressMsg) tea.Cmd {
	s := k.String()
	switch s {
	case "ctrl+c":
		m.interrupted, m.done = true, true
		return tea.Quit
	case shell.Quit:
		m.done = true
		return tea.Quit
	case "esc":
		// esc goes back to the palette, when the form came from it, and never quits
		if m.back {
			return func() tea.Msg { return BackMsg{} }
		}
		return nil
	case "shift+enter":
		return m.tryRun()
	}
	if m.focus >= len(m.fields) {
		switch s {
		case "enter":
			return m.tryRun()
		case "up", "shift+tab":
			return m.move(-1)
		}
		return nil
	}
	switch m.fields[m.focus].Kind {
	case Text, Lines, Wildcard:
		return m.text(k, s)
	case Pick, PickMany:
		return m.list(k, s)
	case OneOf, AnyOf:
		return m.row(s)
	}
	return m.toggle(s)
}

// move takes the keyboard to the field d away, the command line after the last.
func (m *Model) move(d int) tea.Cmd {
	m.commitLine()
	m.focus = min(max(m.focus+d, 0), len(m.fields))
	return m.enter()
}

// enter readies the focused field: its text, its cursor, its extension, and its list's rows.
func (m *Model) enter() tea.Cmd {
	m.input, m.note, m.cursor = "", "", 0
	if m.spec.Extends {
		m.refreshExtension()
		m.rebuild(Collect(m.fields))
	}
	if m.focus >= len(m.fields) {
		return nil
	}
	f := m.fields[m.focus]
	switch f.Kind {
	case Text:
		if len(f.Values) > 0 {
			m.input = f.Values[0]
		}
	case OneOf:
		m.cursor = f.Chosen
	case Pick, PickMany:
		if m.lists[f.Key] == nil || m.stale[f.Key] {
			return m.fetch(f.Key, "")
		}
	}
	return nil
}

func (m *Model) toggle(s string) tea.Cmd {
	f := &m.fields[m.focus]
	switch s {
	case "space", "left", "right":
		if !f.Required {
			if len(f.Values) > 0 {
				f.Values = nil
			} else {
				f.Values = []string{"1"}
			}
			m.changed(f.Key)
		}
	case "enter", "down", "tab":
		return m.move(1)
	case "up", "shift+tab":
		return m.move(-1)
	}
	return nil
}

func (m *Model) row(s string) tea.Cmd {
	f := &m.fields[m.focus]
	last := len(f.Options) - 1
	switch s {
	case "left", "right":
		if s == "left" && m.cursor > 0 {
			m.cursor--
		} else if s == "right" && m.cursor < last {
			m.cursor++
		} else if m.cursor < 0 {
			m.cursor = 0
		}
		if f.Kind == OneOf {
			f.Chosen = m.cursor
			m.changed(f.Key)
		}
	case "space":
		if f.Kind == AnyOf && m.cursor >= 0 {
			option := f.Options[m.cursor]
			if i := index(f.Values, option); i >= 0 {
				f.Values = append(append([]string{}, f.Values[:i]...), f.Values[i+1:]...)
			} else {
				f.Values = append(f.Values, option)
			}
			m.changed(f.Key)
		}
	case "backspace":
		if f.Kind == OneOf && !f.Required {
			f.Chosen, m.cursor = -1, -1
			m.changed(f.Key)
		}
	case "enter", "down", "tab":
		return m.move(1)
	case "up", "shift+tab":
		return m.move(-1)
	}
	return nil
}

func (m *Model) text(k tea.KeyPressMsg, s string) tea.Cmd {
	f := &m.fields[m.focus]
	if k.Text != "" && k.Mod&(tea.ModCtrl|tea.ModAlt) == 0 {
		m.input += k.Text
		m.typedText()
		return nil
	}
	switch s {
	case "backspace":
		if m.input != "" {
			r := []rune(m.input)
			m.input = string(r[:len(r)-1])
			m.typedText()
		} else if f.Kind != Text && len(f.Values) > 0 {
			f.Values = f.Values[:len(f.Values)-1]
			m.changed(f.Key)
		}
	case "enter":
		if f.Kind != Text && m.input != "" {
			m.commitLine()
			return nil
		}
		return m.move(1)
	case "down", "tab":
		return m.move(1)
	case "up", "shift+tab":
		return m.move(-1)
	}
	return nil
}

// typedText applies the text typed: a text field's value changes as it is typed.
func (m *Model) typedText() {
	f := &m.fields[m.focus]
	if f.Kind != Text {
		return
	}
	f.Values = nil
	if m.input != "" {
		f.Values = []string{m.input}
	}
	m.markStale(f.Key)
}

// commitLine adds the line being typed to a field of lines.
func (m *Model) commitLine() {
	if m.focus >= len(m.fields) || m.input == "" {
		return
	}
	f := &m.fields[m.focus]
	if f.Kind != Lines && f.Kind != Wildcard {
		return
	}
	f.Values = append(append([]string{}, f.Values...), m.input)
	m.input = ""
	m.changed(f.Key)
}

func (m *Model) list(k tea.KeyPressMsg, s string) tea.Cmd {
	f := &m.fields[m.focus]
	rows := m.filtered()
	if s == "space" && f.Kind == PickMany {
		if m.cursor < len(rows) {
			return m.flip(rows[m.cursor].Value)
		}
		return nil
	}
	if k.Text != "" && k.Mod&(tea.ModCtrl|tea.ModAlt) == 0 {
		m.input += k.Text
		m.cursor = 0
		return nil
	}
	switch s {
	case "backspace":
		if r := []rune(m.input); len(r) > 0 {
			m.input = string(r[:len(r)-1])
			m.cursor = 0
		}
	case "up":
		if m.cursor > 0 {
			m.cursor--
			return nil
		}
		return m.move(-1)
	case "down":
		if m.cursor < len(rows)-1 {
			m.cursor++
			return nil
		}
		return m.move(1)
	case "pgup", "pgdown", "home", "end":
		m.cursor = paged(s, m.cursor, len(rows), listWindow-1)
	case "enter":
		if f.Kind == Pick && m.cursor < len(rows) {
			f.Values = []string{rows[m.cursor].Value}
			m.changed(f.Key)
		} else if m.failed[f.Key] && m.input != "" {
			return m.take()
		}
		return m.move(1)
	case "ctrl+n":
		if (f.Choices == "open" || m.failed[f.Key]) && m.input != "" {
			return m.take()
		}
	case "tab":
		return m.move(1)
	case "shift+tab":
		return m.move(-1)
	}
	return nil
}

// paged gives where a page key takes a cursor among n rows: page down and page up a page on, home and end to the first
// and the last row, never past either.
func paged(key string, cursor, n, page int) int {
	switch key {
	case "pgdown":
		cursor += page
	case "pgup":
		cursor -= page
	case "home":
		cursor = 0
	case "end":
		cursor = n - 1
	}
	return max(0, min(cursor, n-1))
}

// take makes what is typed the answer, as an open list does on ctrl-n and a list kit refused on Enter too: a pick's
// one value, and the keyboard moves on; one more value of a pick of many, and the keyboard stays.
func (m *Model) take() tea.Cmd {
	f := &m.fields[m.focus]
	kind := f.Kind
	if kind == Pick {
		f.Values = []string{m.input}
	} else {
		f.Values = append(append([]string{}, f.Values...), m.input)
	}
	m.input = ""
	m.changed(f.Key)
	if kind == Pick {
		return m.move(1)
	}
	return nil
}

// flip picks a value of a list of many, or unpicks it; a list whose rows carry state is asked again with the plan.
func (m *Model) flip(value string) tea.Cmd {
	f := &m.fields[m.focus]
	if i := index(f.Values, value); i >= 0 {
		f.Values = append(append([]string{}, f.Values[:i]...), f.Values[i+1:]...)
	} else {
		f.Values = append(append([]string{}, f.Values...), value)
	}
	key := f.Key
	m.changed(key)
	if stateful(m.lists[key]) {
		return m.fetch(key, value)
	}
	return nil
}

// choices takes a list's rows. A list kit calls a set, empty, starts from the values active now, so the command line
// is the whole set; a row the verb has a line about, flipped off, is flipped back on with that line
// shown, since the verb keeps it anyway. A list kit refused shows kit's line as the note, and takes what is typed.
// Rows asked for before another answer changed are kept, and stay stale, so the list asks again when it is next
// entered.
func (m *Model) choices(msg choicesMsg) {
	m.lists[msg.key] = msg.rows
	m.sets[msg.key] = msg.set
	m.stale[msg.key] = !slices.Equal(msg.asked, m.others(msg.key))
	m.failed[msg.key] = msg.err != nil
	if msg.err != nil {
		m.note = msg.err.Error()
		return
	}
	i := m.fieldOf(msg.key)
	if i < 0 {
		return
	}
	f := &m.fields[i]
	if f.Kind == PickMany && msg.set && len(f.Values) == 0 && !m.seeded[msg.key] {
		for _, row := range msg.rows {
			if row.State == "active" {
				f.Values = append(f.Values, row.Value)
			}
		}
		m.rebuild(Collect(m.fields))
	}
	m.seeded[msg.key] = true
	if msg.flipped == "" {
		return
	}
	m.note = ""
	f = &m.fields[m.fieldOf(msg.key)]
	for _, row := range msg.rows {
		if row.Value == msg.flipped && row.Refusal != "" && index(f.Values, row.Value) < 0 {
			f.Values = append(append([]string{}, f.Values...), row.Value)
			m.note = row.Refusal
			m.rebuild(Collect(m.fields))
		}
	}
}

// fetch asks kit for a list's rows, with the answers as they stand, and, for closed choices, whether they are a set.
func (m *Model) fetch(key, flipped string) tea.Cmd {
	r, verb, answers, asked := m.runner, m.spec.Verb, m.answers(), m.others(key)
	closed := false
	if i := m.fieldOf(key); i >= 0 {
		closed = m.fields[i].Choices == "closed"
	}
	return func() tea.Msg {
		rows, err := r.Choices(verb, key, answers)
		set := false
		if err == nil && closed {
			kind, kindErr := r.Kind(verb, key, answers)
			set = kindErr == nil && kind == "set"
		}
		return choicesMsg{key: key, rows: rows, set: set, err: err, flipped: flipped, asked: asked}
	}
}

// others gives the answers but a list's own: what its rows may depend on.
func (m Model) others(key string) []string {
	var out []string
	for _, a := range m.answers() {
		if !strings.HasPrefix(a, key+"=") {
			out = append(out, a)
		}
	}
	return out
}

// answers gives the answers kit syntax takes after a --: each <key>=<value>, and the subcommand words chosen, which a
// verb's choices may depend on, as kit packages select's list does.
func (m Model) answers() []string {
	fields := m.viewFields()
	out := answerList(Collect(fields))
	for _, f := range fields {
		if f.Kind == OneOf && f.Chosen >= 0 {
			if word := f.Options[f.Chosen]; !strings.ContainsAny(word[:1], "-<") && !strings.Contains(word, " ") {
				out = append(out, word)
			}
		}
	}
	return out
}

// matchedMsg brings kit's judgment of the command line it was asked about.
type matchedMsg struct {
	args  []string
	match syntax.Match
	err   error
}

// tryRun asks kit, off the update loop, whether the command line is complete; matched takes the answer.
func (m *Model) tryRun() tea.Cmd {
	m.commitLine()
	r, verb, args := m.runner, m.spec.Verb, m.CommandLine()
	return func() tea.Msg {
		match, err := r.Match(verb, args)
		return matchedMsg{args: args, match: match, err: err}
	}
}

// matched runs the command line when kit judged it complete; otherwise it shows kit's line under the command line and
// takes the keyboard to the first field that needs an answer. An answer about a line that changed since is dropped:
// the person asks again.
func (m *Model) matched(msg matchedMsg) tea.Cmd {
	args, match, err := msg.args, msg.match, msg.err
	if !slices.Equal(args, m.CommandLine()) {
		return nil
	}
	if err != nil {
		m.refusal = err.Error()
		return nil
	}
	switch match.Status {
	case 0:
		m.run, m.done, m.args, m.refusal = true, true, args, ""
		return tea.Quit
	case 1:
		literals := ""
		if len(match.Literals) > 0 {
			literals = " " + strings.Join(match.Literals, " ")
		}
		m.refusal = fmt.Sprintf("kit %s%s needs %s; run: kit help %s", m.spec.Verb, literals, match.Missing, m.spec.Verb)
		if i := FirstEmptyRequired(m.fields); i >= 0 {
			m.focus = i
			return m.enter()
		}
		return nil
	}
	m.refusal = match.Refusal
	return nil
}

// changed rebuilds the fields from the answers after one of them changed.
func (m *Model) changed(key string) {
	m.markStale(key)
	m.rebuild(Collect(m.fields))
}

// markStale marks every other list to be asked again: its rows may depend on the answer that changed.
func (m *Model) markStale(changed string) {
	for key := range m.lists {
		if key != changed {
			m.stale[key] = true
		}
	}
}

// rebuild lays the fields out again from the answers, with the extension in place of the wildcard; a grammar that
// no longer lays out leaves the fields as they stand, with why as the note.
func (m *Model) rebuild(a Answers) {
	fields, err := Fields(m.spec, a)
	if err != nil {
		m.note = err.Error()
		return
	}
	if len(m.extOrder) > 0 {
		kept := fields[:0:0]
		for _, f := range fields {
			if f.Kind != Wildcard {
				kept = append(kept, f)
			}
		}
		for _, key := range m.extOrder {
			kept = append(kept, Field{Key: key, Label: "«" + strings.TrimPrefix(key, "--") + "»",
				Description: m.ext[key].Description, Kind: Text, Required: true, Chosen: -1, Group: -1,
				Values: a.Options[key]})
		}
		fields = kept
	}
	m.fields = fields
	if m.focus > len(m.fields) {
		m.focus = len(m.fields)
	}
}

// refreshExtension asks kit for the arguments the answers add, once per set of answers kit answered; kit's refusal is
// the focused field's note, and the next field asks again.
func (m *Model) refreshExtension() {
	answers := m.answers()
	seen := strings.Join(answers, "\x00")
	if m.extSeen[seen] {
		return
	}
	order, args, err := m.runner.Extend(m.spec.Verb, answers)
	if err != nil {
		m.note = err.Error()
		return
	}
	m.extSeen[seen] = true
	m.extOrder, m.ext = order, args
}

// viewFields gives the fields with the line being typed counted, as the command line shows it.
func (m Model) viewFields() []Field {
	if m.focus >= len(m.fields) || m.input == "" {
		return m.fields
	}
	f := m.fields[m.focus]
	if f.Kind != Lines && f.Kind != Wildcard {
		return m.fields
	}
	fields := append([]Field{}, m.fields...)
	fields[m.focus].Values = append(append([]string{}, f.Values...), m.input)
	return fields
}

// fieldOf gives the index of the field with the key, or -1.
func (m *Model) fieldOf(key string) int {
	for i, f := range m.fields {
		if f.Key == key {
			return i
		}
	}
	return -1
}

// filtered gives the focused list's rows that hold the text typed, ignoring case.
func (m Model) filtered() []kit.Choice {
	if m.focus >= len(m.fields) {
		return nil
	}
	rows := m.lists[m.fields[m.focus].Key]
	if m.input == "" {
		return rows
	}
	needle := strings.ToLower(m.input)
	var out []kit.Choice
	for _, row := range rows {
		if strings.Contains(strings.ToLower(row.Value), needle) {
			out = append(out, row)
		}
	}
	return out
}

// expand draws the focused field live: the text with its caret, the lines with the one being typed, or the list, and
// gives the line of the list's cursor, -1 for none. Text wider than the screen shows its end, after a …, so the caret
// stays in sight.
func (m Model) expand(i int, indent string) (string, int) {
	if i != m.focus || i >= len(m.fields) {
		return "", -1
	}
	f := m.fields[i]
	at := -1
	room := m.width - ansi.StringWidth(indent) - 2 // the caret, and the cell the screen keeps at its edge
	var b strings.Builder
	switch f.Kind {
	case Text:
		b.WriteString(indent + pane.Tail(m.input, room) + "▏\n")
	case Lines, Wildcard:
		for _, v := range f.Values {
			b.WriteString(indent + pane.Clip(v, room+1) + "\n")
		}
		b.WriteString(indent + pane.Tail(m.input, room) + "▏\n")
		m.hint(&b, indent, "⏎ another line · an empty line ends the list")
	case Pick, PickMany:
		at = m.drawList(&b, f, indent)
	default:
		return "", -1
	}
	return b.String(), at
}

// drawList draws a list: the filter, a window of rows around the cursor with their marks, a dim line for the rows it
// hides above and one for those below, and the count; the note follows, drawn under the field. A list kit refused
// draws the values taken so far and says it takes what is typed. It gives the line of the cursor's row, -1 for none.
func (m Model) drawList(b *strings.Builder, f Field, indent string) int {
	room := m.width - ansi.StringWidth(indent) - 4 // the filter glyph, its space, the caret, and the screen's edge
	b.WriteString(indent + style.Filter + " " + pane.Tail(m.input, room) + "▏\n")
	all := m.lists[f.Key]
	rows := m.filtered()
	if all == nil {
		switch {
		case m.failed[f.Key] && f.Kind == PickMany:
			for _, v := range f.Values {
				b.WriteString(indent + style.Picked + " " + v + "\n")
			}
			m.hint(b, indent, "⏎ takes what is typed · an empty line ends the list")
		case m.failed[f.Key]:
			for _, v := range f.Values {
				b.WriteString(indent + v + "\n")
			}
			m.hint(b, indent, "⏎ takes what is typed")
		default:
			b.WriteString(indent + style.Off("…") + "\n")
		}
		return -1
	}
	if len(rows) == 0 {
		b.WriteString(indent + style.Off("nothing matches") + "\n")
	}
	span := pane.Frame(len(rows), listWindow, m.cursor-4, m.cursor, m.cursor)
	start, end, at := span.Start, span.End, -1
	if span.Above {
		b.WriteString(indent + style.Off(pane.Above(start)) + "\n")
	}
	widest := 0
	for _, row := range rows[start:end] {
		if n := ansi.StringWidth(row.Value); n > widest {
			widest = n
		}
	}
	planned := m.sets[f.Key]
	states := stateful(all)
	room = m.width - len(indent) - widest - 8
	for k := start; k < end; k++ {
		row := rows[k]
		picked := index(f.Values, row.Value) >= 0
		mark := ""
		switch {
		case planned:
			mark = plan(row, picked) + " "
		case f.Kind == PickMany && picked:
			mark = style.Picked + " "
		case f.Kind == PickMany:
			mark = style.Unpicked + " "
		}
		if states && !planned {
			mark += state(row) + " "
		}
		lead := "  "
		value := row.Value + strings.Repeat(" ", widest-ansi.StringWidth(row.Value))
		if k == m.cursor {
			lead, value = style.Focus+" ", style.Chosen(row.Value)+strings.Repeat(" ", widest-ansi.StringWidth(row.Value))
			at = strings.Count(b.String(), "\n")
		}
		b.WriteString(indent[:len(indent)-2] + lead + mark + value + "  " + style.Off(pane.Clip(row.Description, room)) + "\n")
	}
	if span.Below {
		b.WriteString(indent + style.Off(pane.More(len(rows)-end)) + "\n")
	}
	glyphs := ""
	if named := legend(all, f.Values, planned); states && named != "" {
		glyphs = named + " · "
	}
	switch {
	case f.Kind == PickMany && planned:
		add, remove := 0, 0
		for _, row := range all {
			picked := index(f.Values, row.Value) >= 0
			if picked && row.State != "active" {
				add++
			} else if !picked && row.State == "active" {
				remove++
			}
		}
		m.hint(b, indent, fmt.Sprintf("%s %d to add · %s %d to remove · %s␣ flips · type to filter", style.Add, add,
			style.Remove, remove, glyphs))
	case f.Kind == PickMany:
		m.hint(b, indent, fmt.Sprintf("%d of %d picked · %s␣ pick · type to filter", len(f.Values), len(all), glyphs))
	case f.Choices == "open":
		m.hint(b, indent, glyphs+"⏎ takes it · ctrl-n takes what is typed · type to filter")
	default:
		m.hint(b, indent, glyphs+"⏎ takes it · type to filter")
	}
	return at
}

// legend names, once, each state glyph the list's rows draw, active then excluded: in a list kit calls a set, among
// the marks plan gives, whose ➕ and ➖ the counts name; else among the states state gives.
func legend(all []kit.Choice, values []string, planned bool) string {
	drawn := map[string]bool{}
	for _, row := range all {
		if planned {
			drawn[plan(row, index(values, row.Value) >= 0)] = true
		} else {
			drawn[state(row)] = true
		}
	}
	var out []string
	for _, g := range []struct{ glyph, meaning string }{{style.Active, "active"}, {style.Excluded, "excluded"}} {
		if drawn[g.glyph] {
			out = append(out, g.glyph+" "+g.meaning)
		}
	}
	return strings.Join(out, " · ")
}

// hint writes a line of keys under a field, dim, wrapped to the width left beside the indent.
func (m Model) hint(b *strings.Builder, indent, text string) {
	for _, line := range pane.Wrap(text, m.width-len(indent)) {
		b.WriteString(indent + style.Off(line) + "\n")
	}
}

// state gives a row's state as kit packages draws it: ✅ active, ❌ excluded, or blank.
func state(row kit.Choice) string {
	switch row.State {
	case "active":
		return style.Active
	case "excluded":
		return style.Excluded
	}
	return "  "
}

// plan gives a row's mark in a list kit calls a set: what it is now against what the command line will make it.
func plan(row kit.Choice, picked bool) string {
	switch {
	case picked && row.State == "active":
		return style.Active
	case picked:
		return style.Add
	case row.State == "active":
		return style.Remove
	case row.State == "excluded":
		return style.Excluded
	}
	return "  "
}

// stateful says whether any row carries a state or a line.
func stateful(rows []kit.Choice) bool {
	for _, row := range rows {
		if row.State != "" || row.Refusal != "" {
			return true
		}
	}
	return false
}

// answerList gives the answers as kit syntax takes them after a --: each <key>=<value>, in key order.
func answerList(a Answers) []string {
	var out []string
	for _, source := range []map[string][]string{a.Options, a.Values} {
		keys := make([]string, 0, len(source))
		for key := range source {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			for _, value := range source[key] {
				out = append(out, key+"="+value)
			}
		}
	}
	return out
}

// index gives the position of a value among values, or -1.
func index(values []string, value string) int {
	for i, v := range values {
		if v == value {
			return i
		}
	}
	return -1
}
