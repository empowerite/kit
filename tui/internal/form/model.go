// The live form: a Bubble Tea model over the fields, the runner, and the keyboard. Up and down move between fields,
// left and right along a row, space toggles or picks, Enter takes the next field or line and, on the command line,
// runs; shift-Enter runs from anywhere a terminal reports it; esc goes back to the palette or cancels.
package form

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/tui/internal/kit"
	"github.com/empowerite/kit/tui/internal/style"
	"github.com/empowerite/kit/tui/internal/syntax"
)

// BackMsg asks the palette to take the form back: esc on a form opened from the palette.
type BackMsg struct{}

// choicesMsg brings a list's rows; flipped names the value whose flip asked for them again.
type choicesMsg struct {
	key     string
	rows    []kit.Choice
	err     error
	flipped string
}

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
	note    string // the line under the focused list
	refusal string // the line under the command line
	width   int

	lists  map[string][]kit.Choice
	stale  map[string]bool
	seeded map[string]bool

	extOrder []string
	ext      map[string]syntax.Argument
	extSeen  map[string]bool

	initCmd                tea.Cmd
	run, done, interrupted bool
	args                   []string
}

// New opens the form over the fields; back is true when esc returns to the palette, false when it cancels.
func New(spec syntax.Spec, fields []Field, r kit.Runner, back bool) Model {
	m := Model{spec: spec, runner: r, back: back, fields: fields, width: 80, lists: map[string][]kit.Choice{},
		stale: map[string]bool{}, seeded: map[string]bool{}, extSeen: map[string]bool{}}
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
		m.width = msg.Width
	case choicesMsg:
		m.choices(msg)
	case tea.KeyPressMsg:
		cmd := m.key(msg)
		return m, cmd
	}
	return m, nil
}

// View draws the form, the keys it takes, on the alternate screen.
func (m Model) View() tea.View {
	focus := m.focus
	if focus >= len(m.fields) {
		focus = -1
	}
	content := Render(m.spec, m.viewFields(), Options{Width: m.width, Focus: focus, Cursor: m.cursor, Styles: live,
		Expand: m.expand, CommandFocus: m.focus >= len(m.fields), CommandRefusal: m.refusal, Back: m.back})
	end := "esc cancel"
	if m.back {
		end = "esc back"
	}
	hints := strings.Join([]string{"↑↓ field", "←→ along a row", "␣ toggle", "⏎ next", "⇧⏎ run", end}, " · ")
	for _, line := range wrap(hints, m.width-1) {
		content += "\n " + style.Off(line)
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// key takes a key press.
func (m *Model) key(k tea.KeyPressMsg) tea.Cmd {
	s := k.String()
	switch s {
	case "ctrl+c":
		m.interrupted, m.done = true, true
		return tea.Quit
	case "esc":
		if m.back {
			return func() tea.Msg { return BackMsg{} }
		}
		m.done = true
		return tea.Quit
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
	m.focus += d
	if m.focus < 0 {
		m.focus = 0
	}
	if m.focus > len(m.fields) {
		m.focus = len(m.fields)
	}
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
	case "enter":
		if f.Kind == Pick && m.cursor < len(rows) {
			f.Values = []string{rows[m.cursor].Value}
			m.changed(f.Key)
		}
		return m.move(1)
	case "ctrl+n":
		if f.Choices == "open" && m.input != "" {
			if f.Kind == Pick {
				f.Values = []string{m.input}
			} else {
				f.Values = append(append([]string{}, f.Values...), m.input)
			}
			m.input = ""
			m.changed(f.Key)
			if f.Kind == Pick {
				return m.move(1)
			}
		}
	case "tab":
		return m.move(1)
	case "shift+tab":
		return m.move(-1)
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

// choices takes a list's rows. A list of closed choices with state, empty, starts from the values active now, so the
// command line is the whole set; a row the verb has a line about, flipped off, is flipped back on with that line
// shown, since the verb keeps it anyway.
func (m *Model) choices(msg choicesMsg) {
	m.lists[msg.key] = msg.rows
	m.stale[msg.key] = false
	if msg.err != nil {
		m.note = msg.err.Error()
		return
	}
	i := m.fieldOf(msg.key)
	if i < 0 {
		return
	}
	f := &m.fields[i]
	if f.Kind == PickMany && f.Choices == "closed" && len(f.Values) == 0 && !m.seeded[msg.key] {
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

// fetch asks kit for a list's rows, with the answers as they stand.
func (m *Model) fetch(key, flipped string) tea.Cmd {
	r, verb, answers := m.runner, m.spec.Verb, answerList(Collect(m.viewFields()))
	return func() tea.Msg {
		rows, err := r.Choices(verb, key, answers)
		return choicesMsg{key: key, rows: rows, err: err, flipped: flipped}
	}
}

// tryRun asks kit whether the command line is complete, and runs it when it is; otherwise it shows kit's line under
// the command line and takes the keyboard to the first field that needs an answer.
func (m *Model) tryRun() tea.Cmd {
	m.commitLine()
	args := m.CommandLine()
	match, err := m.runner.Match(m.spec.Verb, args)
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

// rebuild lays the fields out again from the answers, with the extension in place of the wildcard.
func (m *Model) rebuild(a Answers) {
	fields, err := Fields(m.spec, a)
	if err != nil {
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

// refreshExtension asks kit for the arguments the answers add, once per set of answers.
func (m *Model) refreshExtension() {
	answers := answerList(Collect(m.fields))
	seen := strings.Join(answers, "\x00")
	if m.extSeen[seen] {
		return
	}
	m.extSeen[seen] = true
	order, args, err := m.runner.Extend(m.spec.Verb, answers)
	if err != nil {
		return
	}
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

// expand draws the focused field live: the text with its caret, the lines with the one being typed, or the list.
func (m Model) expand(i int, indent string) string {
	if i != m.focus || i >= len(m.fields) {
		return ""
	}
	f := m.fields[i]
	var b strings.Builder
	switch f.Kind {
	case Text:
		b.WriteString(indent + m.input + "▏\n")
	case Lines, Wildcard:
		for _, v := range f.Values {
			b.WriteString(indent + v + "\n")
		}
		b.WriteString(indent + m.input + "▏\n")
		b.WriteString(indent + style.Off("⏎ another line · an empty line ends the list") + "\n")
	case Pick, PickMany:
		m.drawList(&b, f, indent)
	default:
		return ""
	}
	return b.String()
}

// drawList draws a list: the filter, a window of rows around the cursor with their marks, the note, and the count.
func (m Model) drawList(b *strings.Builder, f Field, indent string) {
	b.WriteString(indent + style.Filter + " " + m.input + "▏\n")
	all := m.lists[f.Key]
	rows := m.filtered()
	if all == nil {
		b.WriteString(indent + style.Off("…") + "\n")
		return
	}
	if len(rows) == 0 {
		b.WriteString(indent + style.Off("nothing matches") + "\n")
	}
	start := m.cursor - 4
	if start < 0 {
		start = 0
	}
	end := start + 10
	if end > len(rows) {
		end = len(rows)
	}
	widest := 0
	for _, row := range rows[start:end] {
		if n := len([]rune(row.Value)); n > widest {
			widest = n
		}
	}
	planned := stateful(all)
	room := m.width - len(indent) - widest - 8
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
		lead := "  "
		value := row.Value + strings.Repeat(" ", widest-len([]rune(row.Value)))
		if k == m.cursor {
			lead, value = style.Focus+" ", style.Chosen(row.Value)+strings.Repeat(" ", widest-len([]rune(row.Value)))
		}
		b.WriteString(indent[:len(indent)-2] + lead + mark + value + "  " + style.Off(clip(row.Description, room)) + "\n")
	}
	if m.note != "" {
		b.WriteString(indent + style.Refusal(style.Excluded+" "+m.note) + "\n")
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
		b.WriteString(indent + style.Off(fmt.Sprintf("%d to add · %d to remove · ␣ flips · type to filter", add, remove)) + "\n")
	case f.Kind == PickMany:
		b.WriteString(indent + style.Off(fmt.Sprintf("%d of %d picked · ␣ pick · type to filter", len(f.Values), len(all))) + "\n")
	case f.Choices == "open":
		b.WriteString(indent + style.Off("⏎ takes it · ctrl-n takes what is typed · type to filter") + "\n")
	default:
		b.WriteString(indent + style.Off("⏎ takes it · type to filter") + "\n")
	}
}

// plan gives a row's mark in a list whose rows carry state: what it is now against what the command line will make it.
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

// clip shortens a text to n runes, with … at the cut.
func clip(s string, n int) string {
	r := []rune(s)
	if n < 1 {
		return ""
	}
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
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
