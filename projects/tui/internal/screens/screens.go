// Package screens is kittui's root: kit's screens, the picture of kit flow, the welcome's guide, and the help's
// palette, one shown at a time, which the app's menu on each switches between. Each is made the first time it is
// shown, and loads what it draws then, so kit help reads nothing of kit flow's until the picture is asked for; and each
// is kept as it was left while another is shown, nothing loaded again. What a screen started goes on while it is
// hidden: its answers reach it, not the screen shown. Keys, clicks, and the terminal's size reach the screen shown, and
// a screen shown again is told the size the terminal has now.
package screens

import (
	"reflect"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/app"
	"github.com/empowerite/kit/projects/tui/internal/flow"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/welcome"
)

// Model is kittui's root: the screens made so far, by name, and the one shown.
type Model struct {
	runner   kit.Runner
	here     shell.Screen
	made     map[shell.Screen]tea.Model
	size     *tea.WindowSizeMsg
	keyboard *tea.KeyboardEnhancementsMsg // what the terminal said of its keyboard, for a screen made after it said so
	refusal  string                       // kit's line, when a screen asked for could not open, until the next key
}

// New gives the root showing the screen first, made already, as kittui opened on it.
func New(r kit.Runner, first shell.Screen, m tea.Model) Model {
	return Model{runner: r, here: first, made: map[shell.Screen]tea.Model{first: m}}
}

// Shown gives the screen shown, as the program ended on it.
func (m Model) Shown() tea.Model { return m.made[m.here] }

// Close ends what every screen made started and still runs.
func (m Model) Close() {
	for _, s := range m.made {
		if c, ok := s.(interface{ Close() }); ok {
			c.Close()
		}
	}
}

// Init starts the screen shown.
func (m Model) Init() tea.Cmd { return tag(m.here, m.made[m.here].Init()) }

// Update takes a message: a screen's ask for another, shown in its place; an answer to what a screen started, for
// that screen; and anything else for the screen shown.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case shell.SwitchMsg:
		return m.show(msg.Screen)
	case tagged:
		return m.deliver(msg.to, msg.msg)
	case tea.WindowSizeMsg:
		m.size = &msg
	case tea.KeyboardEnhancementsMsg:
		m.keyboard = &msg
	case tea.KeyPressMsg:
		m.refusal = ""
	}
	return m.deliver(m.here, msg)
}

// deliver gives a message to a screen made, and tags what it starts as that screen's.
func (m Model) deliver(to shell.Screen, msg tea.Msg) (tea.Model, tea.Cmd) {
	s, ok := m.made[to]
	if !ok {
		return m, nil
	}
	next, cmd := s.Update(msg)
	m.made[to] = next
	return m, tag(to, cmd)
}

// show shows a screen in place of the one shown: made the first time, and loading then; else as it was left, told
// the terminal's size. A screen that cannot open leaves the one shown, with kit's line on its last line.
func (m Model) show(to shell.Screen) (tea.Model, tea.Cmd) {
	if _, ok := m.made[to]; ok {
		m.here = to
		if m.size != nil {
			return m.deliver(to, *m.size)
		}
		return m, nil
	}
	s, err := m.open(to)
	if err != nil {
		m.refusal = err.Error()
		if line, _, ok := app.Refusal(err); ok {
			m.refusal = line
		}
		return m, nil
	}
	m.made[to], m.here = s, to
	cmds := []tea.Cmd{tag(to, s.Init())}
	if m.size != nil {
		_, cmd := m.deliver(to, *m.size)
		cmds = append(cmds, cmd)
	}
	if m.keyboard != nil {
		_, cmd := m.deliver(to, *m.keyboard)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

// open makes a screen, on what kit and git say where kittui runs.
func (m Model) open(to shell.Screen) (tea.Model, error) {
	switch to {
	case shell.Flow:
		return flow.Open(m.runner, nil)
	case shell.Welcome:
		return welcome.Open(m.runner), nil
	case shell.Help:
		return app.OpenHelp(m.runner, "")
	}
	return nil, &kit.ExitError{Code: 1, Stderr: "kit has no " + string(to) + " yet"}
}

// View draws the screen shown, with kit's line on its last line when a screen asked for could not open.
func (m Model) View() tea.View {
	v := m.made[m.here].View()
	if m.refusal != "" {
		lines := strings.Split(v.Content, "\n")
		lines[len(lines)-1] = " " + style.Refusal(style.Excluded+" "+m.refusal)
		v.Content = strings.Join(lines, "\n")
	}
	return v
}

// tagged is an answer to what a screen started, for that screen.
type tagged struct {
	to  shell.Screen
	msg tea.Msg
}

// tag gives the command whose answer reaches the screen that started it: a batch's commands each tagged so; a
// screen's ask for another, and what the program itself takes, as quitting, as they are.
func tag(to shell.Screen, cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		switch msg := msg.(type) {
		case nil, shell.SwitchMsg:
			return msg
		case tea.BatchMsg:
			out := make(tea.BatchMsg, len(msg))
			for i, c := range msg {
				out[i] = tag(to, c)
			}
			return out
		}
		if strings.HasPrefix(reflect.TypeOf(msg).PkgPath(), "charm.land/bubbletea") {
			return msg
		}
		return tagged{to, msg}
	}
}
