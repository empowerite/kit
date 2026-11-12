package flow

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/fix"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// A stage whose command failed with a line that names a command, as kit's run: lines do, offers that command on its
// line, as internal/fix draws the pair: down, or a click on the line, selects it, and up or esc leaves it, as they
// leave a row; while it is selected, ⧉ or c copies the command, and run or r, where the command is plain words, hands
// the terminal to it, then loads the stage again.

// ranMsg says a stage's fix has run, and how it ended.
type ranMsg struct {
	stage, command string
	err            error
}

// remedy gives the command a stage's refusal names, or none.
func (m Model) remedy(i int) string {
	if s := m.stages[i]; !s.src.Loading && s.refusal != "" {
		return fix.Command(s.refusal)
	}
	return ""
}

// onRemedy says whether the selected stage's refusal line, which names a command, is selected: the rows have the
// keyboard there.
func (m Model) onRemedy() bool { return len(m.stages) > 0 && m.inRows && m.remedy(m.stage) != "" }

// refusalLines gives a stage's refusal, wrapped to the pane, ❌ before it: when it names a command and is selected, the
// pair after its last line, as fix.After draws it. One that names a command is wrapped short of the pair's room,
// selected or not, so its words never move when it is selected, nor are clipped.
func (m Model) refusalLines(i int) []table.Line {
	s, width, command := m.stages[i], m.paneWidth()-len(lead), m.remedy(i)
	selected := i == m.stage && m.onRemedy()
	if command != "" {
		// copied is the wider of ⧉'s two names
		width -= ansi.StringWidth(fix.Buttons(fix.Runs(command), true))
	}
	var out []table.Line
	for _, l := range pane.Wrap(Failed+" "+s.refusal, max(1, width)) {
		out = append(out, table.Line{Row: -1, Text: lead + style.Refusal(l)})
	}
	if selected && len(out) > 0 {
		last := &out[len(out)-1]
		last.Text = fix.After(last.Text, m.paneWidth(), fix.Runs(command), m.copied)
	}
	return out
}

// remedyKey takes a key that the refusal's line takes: down selects it, from the bar or the control line, and while it
// is selected, c copies its command and r runs it.
func (m *Model) remedyKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	if len(m.stages) == 0 || m.remedy(m.stage) == "" {
		return nil, false
	}
	command := m.remedy(m.stage)
	switch k.String() {
	case "down":
		m.onPreset, m.inRows = false, true
		return nil, true
	case fix.CopyKey:
		if m.inRows {
			return m.copyRemedy(command), true
		}
	case fix.RunKey:
		if m.inRows && fix.Runs(command) {
			return m.runRemedy(command), true
		}
	}
	return nil, false
}

// remedyClick takes a left click on the refusal's lines: it selects the line, and on ⧉ copies its command, on run runs
// it. A click elsewhere is not its.
func (m *Model) remedyClick(msg tea.MouseClickMsg) (tea.Cmd, bool) {
	if msg.Button != tea.MouseLeft || len(m.stages) == 0 || m.remedy(m.stage) == "" || msg.X >= m.paneWidth() {
		return nil, false
	}
	top := m.top()
	if m.controls() {
		top += 2
	}
	// the buttons are read where they were drawn, copied or not, before the click clears copied
	lines, r := m.refusalLines(m.stage), msg.Y-top
	if r < 0 || r >= len(lines) {
		return nil, false
	}
	m.onPreset, m.inPanel, m.inRows, m.copied = false, false, true, false
	if r != len(lines)-1 {
		return nil, true
	}
	command := m.remedy(m.stage)
	switch fix.On(lines[r].Text, msg.X) {
	case fix.Copy:
		return m.copyRemedy(command), true
	case fix.Run:
		if fix.Runs(command) {
			return m.runRemedy(command), true
		}
	}
	return nil, true
}

// copyRemedy puts a command on the clipboard, and ⧉ says copied until the next key or click.
func (m *Model) copyRemedy(command string) tea.Cmd {
	m.copied = true
	return fix.Copying(command)
}

// runRemedy hands the terminal to a command; when it ends, the stage loads again.
func (m *Model) runRemedy(command string) tea.Cmd {
	name := m.stages[m.stage].name
	return fix.Running(m.handOver, command, func(err error) tea.Msg { return ranMsg{stage: name, command: command, err: err} })
}

// ran takes how a fix ended: the stage loads again, and a fix that failed says so in place of the keys.
func (m *Model) ran(msg ranMsg) tea.Cmd {
	if msg.err != nil {
		m.note.Text, m.note.Done = fmt.Sprintf("%s %s: %v", Failed, msg.command, msg.err), false
	}
	return m.refresh(msg.stage)
}

// remedyKeys gives the segment of the refusal's line while it is selected: esc and up leave it, c copies, r runs.
func (m Model) remedyKeys() shell.Segment {
	return shell.Segment{Bindings: append([]shell.Binding{shell.Bind("esc", table.TabsHint), shell.Bind("↑", "back", "up")},
		fix.Bindings(fix.Runs(m.remedy(m.stage)))...)}
}
