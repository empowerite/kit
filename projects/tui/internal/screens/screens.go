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
	"github.com/empowerite/kit/projects/tui/internal/tutorials"
	"github.com/empowerite/kit/projects/tui/internal/welcome"
)

// Model is kittui's root: the screens made so far, by name, and the one shown; kittui's settings, and the dialog that
// changes them.
type Model struct {
	runner   kit.Runner
	here     shell.Screen
	made     map[shell.Screen]tea.Model
	size     *tea.WindowSizeMsg
	keyboard *tea.KeyboardEnhancementsMsg // what the terminal said of its keyboard, for a screen made after it said so
	refusal  string                       // kit's line, when a screen asked for could not open, until the next key
	settings shell.Settings
	dialog   shell.SettingsDialog
}

// New gives the root showing the screen first, made already, as kittui opened on it, with the settings kittui's file
// keeps.
func New(r kit.Runner, first shell.Screen, m tea.Model) Model {
	root := Model{runner: r, here: first, made: map[shell.Screen]tea.Model{first: m},
		settings: shell.LoadSettings(shell.SettingsFile())}
	root.made[first], _ = m.Update(shell.SettingsMsg{Settings: root.settings})
	return root
}

// told gives every screen made the settings, after the dialog changed one, and what they start.
func (m Model) told() (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	for to := range m.made {
		var cmd tea.Cmd
		_, cmd = m.deliver(to, shell.SettingsMsg{Settings: m.settings})
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
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

// Update takes a message: a screen's ask for another, shown in its place, or for the Settings dialog, opened over it;
// an answer to what a screen started, for that screen; while the dialog is open, a key or a click for it, which closes
// it outside it, and any other mouse event nowhere;
// and anything else for the screen shown, the terminal's size as the screen has it.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case shell.SwitchMsg:
		if msg.Screen == shell.SettingsScreen {
			m.dialog = shell.SettingsDialog{Open: true}
			return m, nil
		}
		return m.show(msg.Screen)
	case tagged:
		return m.deliver(msg.to, msg.msg)
	case tea.WindowSizeMsg:
		m.size = &msg
		return m.deliver(m.here, m.sized())
	case tea.KeyboardEnhancementsMsg:
		m.keyboard = &msg
	case tea.KeyPressMsg:
		m.refusal = ""
		if m.dialog.Open && msg.String() != "ctrl+c" {
			was := m.settings
			m.dialog, m.settings = m.dialog.Key(msg, m.settings)
			return m.changed(was)
		}
	case tea.MouseClickMsg:
		if m.dialog.Open {
			was := m.settings
			m.dialog, m.settings = m.dialog.Click(msg, m.settings, m.height(), m.width())
			return m.changed(was)
		}
	case tea.MouseMsg:
		if m.dialog.Open {
			return m, nil
		}
	}
	return m.deliver(m.here, msg)
}

// changed follows a setting the dialog changed from was: the keys bar shown or hidden, the screen shown has its line,
// or gives it back; another, every screen made is told.
func (m Model) changed(was shell.Settings) (tea.Model, tea.Cmd) {
	switch {
	case m.settings.KeysBar != was.KeysBar && m.size != nil:
		return m.deliver(m.here, m.sized())
	case m.settings != was:
		return m.told()
	}
	return m, nil
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
			return m.deliver(to, m.sized())
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
	_, told := m.deliver(to, shell.SettingsMsg{Settings: m.settings})
	cmds := []tea.Cmd{tag(to, s.Init()), told}
	if m.size != nil {
		_, cmd := m.deliver(to, m.sized())
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
	case shell.Tutorials:
		return tutorials.Open(m.runner, nil), nil
	case shell.Help:
		return app.OpenHelp(m.runner, "")
	}
	return nil, &kit.ExitError{Code: 1, Stderr: "kit has no " + string(to) + " yet"}
}

// sized gives the terminal's size as the screen shown has it: a line taller while the keys bar is hidden, as the screen
// draws its keys bar on the line under the terminal's last, which View drops.
func (m Model) sized() tea.WindowSizeMsg {
	size := *m.size
	if !m.settings.KeysBar && size.Height > 0 {
		size.Height++
	}
	return size
}

// width gives the terminal's width, or nothing before it says.
func (m Model) width() int {
	if m.size == nil {
		return 0
	}
	return m.size.Width
}

// height gives the lines View draws, which the Settings dialog is placed over: the terminal's, or nothing before it
// says.
func (m Model) height() int {
	if m.size == nil {
		return 0
	}
	return m.size.Height
}

// noticing is a screen that says, in place of its keys, what it did or why it did not.
type noticing interface{ Noticing() bool }

// View draws the screen shown. While the keys bar is hidden, the screen's last line, its keys, is dropped; a screen's
// notice in their place is kept on the last line, over the line above it. The Settings dialog, while open, is over the
// screen, its keys on the keys bar; kit's line is on the last line when a screen asked for could not open.
func (m Model) View() tea.View {
	v := m.made[m.here].View()
	lines := strings.Split(v.Content, "\n")
	if !m.settings.KeysBar && m.size != nil && m.size.Height > 0 && len(lines) > 1 {
		last := len(lines) - 1
		if n, ok := m.made[m.here].(noticing); ok && n.Noticing() {
			last--
		}
		lines = append(lines[:last], lines[last+1:]...)
	}
	width := m.width()
	if m.dialog.Open {
		lines = m.dialog.Overlay(lines, m.settings, width)
		if m.settings.KeysBar {
			lines[len(lines)-1] = shell.Path{m.dialog.Keys()}.Line(width)
		}
	}
	if m.refusal != "" {
		lines[len(lines)-1] = " " + style.Refusal(style.Excluded+" "+m.refusal)
	}
	v.Content = strings.Join(lines, "\n")
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
