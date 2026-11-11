// Package screens is kittui's root: kit's screens, the apps over kit flow's and kit workspace's pictures, the welcome's
// guide, and the help's palette, one shown at a time, which the app's menu on each switches between. Each is made the first time it is
// shown, and loads what it draws then, so kit help reads nothing of kit flow's until the picture is asked for; and each
// is kept as it was left while another is shown, nothing loaded again. What a screen started goes on while it is
// hidden: its answers reach it, not the screen shown. Keys, clicks, and the terminal's size reach the screen shown, and
// a screen shown again is told the size the terminal has now.
//
// The root reads the workspace from kit once kittui starts, and tells every screen made, as it tells the settings. A
// screen's workspace menu asks for another; the root asks kit for it, keeping a group as the last used, and tells every
// screen again. When the place the screens read moves, to another repository or onto or off a group, the screens made
// are made anew there, all but the tutorials, the help, and the workspaces' screen, which read kit alone. Only the latest ask lands. The menu's
// Open… opens a dialog over the screen shown, where a repository's folder is typed: kit keeps it among the recent and
// says the workspace, which is switched to as a pick's is, or says why not, in the dialog.
package screens

import (
	"context"
	"os"
	"path/filepath"
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
	"github.com/empowerite/kit/projects/tui/internal/workspaces"
)

// Model is kittui's root: the screens made so far, by name, and the one shown; kittui's settings, and the dialog that
// changes them.
type Model struct {
	runner    kit.Runner
	here      shell.Screen
	made      map[shell.Screen]tea.Model
	size      *tea.WindowSizeMsg
	keyboard  *tea.KeyboardEnhancementsMsg // what the terminal said of its keyboard, for a screen made after it said so
	refusal   string                       // kit's line, when a screen asked for could not open, until the next key
	settings  shell.Settings
	dialog    shell.SettingsDialog
	opener    shell.OpenDialog     // the Open… dialog, which the workspace menu's Open… opens over the screen shown
	home      string               // where kittui was started, which the screens read over anything but a repository
	start     string               // the group the first screen was made over, as kittui started with --workspace
	workspace kit.Workspace        // the workspace kit said last, zero until it says
	refused   string               // kit's line, when kit could not say the workspace last asked for
	asked     int                  // the latest ask, 0 at kittui's start: an older ask's answer is dropped
	epochs    map[shell.Screen]int // how many times each screen was made again: an answer to one closed is dropped
}

// workspaceMsg is kit's answer to an ask for the workspace: the ask's number, and the workspace, or why there is none.
type workspaceMsg struct {
	gen int
	ws  kit.Workspace
	err error
}

// openedMsg is kit's answer to the Open… dialog's ask for a repository by its folder, as workspaceMsg is to a pick.
type openedMsg workspaceMsg

// New gives the root showing the screen first, made already, as kittui opened on it, with the settings kittui's file
// keeps.
func New(r kit.Runner, first shell.Screen, m tea.Model) Model {
	root := Model{runner: r, here: first, made: map[shell.Screen]tea.Model{first: m},
		settings: shell.LoadSettings(shell.SettingsFile()), home: r.Dir, start: r.Group,
		epochs: map[shell.Screen]int{}}
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

// Init starts the screen shown, and asks kit for the workspace: the group the runner spans, as kittui started with
// --workspace, else the one kit starts on.
func (m Model) Init() tea.Cmd { return tea.Batch(m.tag(m.here, m.made[m.here].Init()), m.ask(m.start)) }

// ask gives the command that asks kit for the workspace given, "" for the one kit starts on, as the latest ask: a
// group given is kept as the last used first, and a repository given by its folder first among the recent. A
// repository kit says by another way, as the one kittui starts in, is kept first among the recent too, unless it is
// already, so the workspace menu offers it again once another is switched to; kit is asked again for what it then
// keeps. A recent kit could not keep leaves the workspace as kit said it.
func (m Model) ask(given string) tea.Cmd {
	r, gen := m.runner, m.asked
	return func() tea.Msg {
		ctx := context.Background()
		if given != "" && (!strings.ContainsAny(given, "/:") || strings.HasPrefix(given, "/")) {
			if err := r.UseWorkspace(ctx, given); err != nil {
				return workspaceMsg{gen: gen, err: err}
			}
		}
		ws, err := r.Workspace(ctx, given)
		if err == nil && ws.Kind == "repository" && ws.Checkout != "" &&
			(len(ws.Recent) == 0 || !sameRepository(ws.Recent[0], ws)) && r.UseWorkspace(ctx, ws.Checkout) == nil {
			if again, err := r.Workspace(ctx, given); err == nil {
				ws = again
			}
		}
		return workspaceMsg{gen: gen, ws: ws, err: err}
	}
}

// sameRepository says whether a recent repository is the one a workspace is: its folder, or, as a worktree belongs to
// its repository, kept among the recent by its desk, its member.
func sameRepository(r kit.WorkspaceRecent, ws kit.Workspace) bool {
	return r.Checkout == ws.Checkout || r.Member != "" && r.Member == ws.Member
}

// opening opens the Open… dialog over the screen shown, the folder the repository shown stands in typed, else the
// folder kittui started in.
func (m Model) opening() Model {
	home, _ := os.UserHomeDir()
	start := m.home
	if m.workspace.Kind == "repository" && m.workspace.Checkout != "" {
		start = filepath.Dir(m.workspace.Checkout)
	}
	m.opener = shell.NewOpen(start, home)
	return m
}

// openFolder asks kit for the repository whose working tree holds the folder, as the latest ask, the Open… dialog's.
func (m Model) openFolder(folder string) (tea.Model, tea.Cmd) {
	m.asked++
	ask := m.ask(folder)
	return m, func() tea.Msg { return openedMsg(ask().(workspaceMsg)) }
}

// opened takes kit's answer to the Open… dialog: its line, when kit refused, stands in the dialog, which stays open;
// else the dialog closes, and the repository is switched to, as a pick's answer is.
func (m Model) opened(msg openedMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.asked {
		return m, nil
	}
	if msg.err != nil {
		m.opener = m.opener.Refused(refusalLine(msg.err))
		return m, nil
	}
	m.opener.Open = false
	return m.answered(workspaceMsg(msg))
}

// answered takes kit's answer to the latest ask for the workspace, dropping an earlier one's: kit's line, when it could
// not say, told to every screen made, the workspace as it was; else the workspace, which the runner then reads, the
// screens made anew when it moved the place they read, then told it. At kittui's start, the screen open is made anew
// only over a group, since inside a repository, or outside every one, it already reads where kit says.
func (m Model) answered(msg workspaceMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.asked {
		return m, nil
	}
	if msg.err != nil {
		m.refused = refusalLine(msg.err)
		return m.tellWorkspace()
	}
	moved := msg.ws.Kind == "group" && msg.ws.Name != m.start
	if msg.gen > 0 {
		moved = msg.ws.Kind != m.workspace.Kind || msg.ws.Name != m.workspace.Name
	}
	m.workspace, m.refused = msg.ws, ""
	m.runner.Dir, m.runner.Group, m.runner.NoGroup = m.home, "", msg.ws.Kind != "group"
	switch {
	case msg.ws.Kind == "group":
		m.runner.Group = msg.ws.Name
	case msg.ws.Kind == "repository" && msg.ws.Checkout != "":
		m.runner.Dir = msg.ws.Checkout
	}
	if !moved {
		return m.tellWorkspace()
	}
	// the screen shown is made anew first, so one that cannot open there gives way to the welcome, which always opens,
	// kit's line on the last line, before anything is closed
	here, remade := m.here, !readsKitAlone(m.here)
	var shown tea.Model
	if remade {
		var err error
		if shown, err = m.open(here); err != nil {
			m.refusal, here = refusalLine(err), shell.Welcome
			shown, _ = m.open(here)
		}
	}
	for to, s := range m.made {
		if readsKitAlone(to) {
			continue
		}
		if c, ok := s.(interface{ Close() }); ok {
			c.Close()
		}
		delete(m.made, to)
		m.epochs[to]++
	}
	if !remade {
		return m.tellWorkspace()
	}
	next, made := m.place(here, shown)
	next, told := next.(Model).tellWorkspace()
	return next, tea.Batch(made, told)
}

// readsKitAlone says whether a screen reads kit alone, not the place the screens read, so a move leaves it as it was:
// the tutorials, the help, and the workspaces' screen.
func readsKitAlone(s shell.Screen) bool {
	return s == shell.Tutorials || s == shell.Help || s == shell.Manage
}

// regrouped asks kit for the workspace again once a screen had it change the groups: the group shown, or the one it was
// renamed to, else, with it deleted or none shown, the one kit starts on.
func (m Model) regrouped(msg shell.ChangedMsg) (tea.Model, tea.Cmd) {
	given := ""
	if m.workspace.Kind == "group" {
		given = m.workspace.Name
		if given == msg.From {
			given = msg.To
		}
	}
	m.asked++
	return m, m.ask(given)
}

// refusalLine gives kit's line for a screen that could not open, or the error's.
func refusalLine(err error) string {
	if line, _, ok := app.Refusal(err); ok {
		return line
	}
	return err.Error()
}

// tellWorkspace tells every screen made the workspace, or kit's line, and gives what they start.
func (m Model) tellWorkspace() (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	for to := range m.made {
		var cmd tea.Cmd
		_, cmd = m.deliver(to, m.workspaceMsg())
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

// workspaceMsg gives what a screen is told of the workspace.
func (m Model) workspaceMsg() shell.WorkspaceMsg {
	return shell.WorkspaceMsg{Workspace: m.workspace, Refusal: m.refused}
}

// Update takes a message: a screen's ask for another, shown in its place, or for a row of another, which is shown and
// then given the ask; or for the Settings dialog, or the Open… dialog, opened over it, and a walk's clones for the
// Open… dialog;
// an answer to what a screen started, for that screen; while a dialog is open, a key or a click for it, which closes
// it outside it, a turn of the wheel for the Open… dialog's list, and any other mouse event nowhere;
// and anything else for the screen shown, the terminal's size as the screen has it.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case shell.SwitchMsg:
		if msg.Screen == shell.SettingsScreen {
			m.dialog = shell.SettingsDialog{Open: true}
			return m, nil
		}
		return m.show(msg.Screen)
	case shell.JumpMsg:
		next, shown := m.show(msg.Screen)
		if next.(Model).here != msg.Screen {
			return next, shown
		}
		next, landed := next.(Model).deliver(msg.Screen, msg)
		return next, tea.Batch(shown, landed)
	case tagged:
		if msg.epoch != m.epochs[msg.to] {
			return m, nil
		}
		return m.deliver(msg.to, msg.msg)
	case workspaceMsg:
		return m.answered(msg)
	case openedMsg:
		return m.opened(msg)
	case shell.OpenMsg:
		return m.opening(), nil
	case shell.ClonesMsg:
		m.opener = m.opener.Found(msg)
		return m, nil
	case shell.ChangedMsg:
		return m.regrouped(msg)
	case shell.PickMsg:
		m.asked++
		return m, m.ask(msg.Workspace)
	case tea.WindowSizeMsg:
		m.size = &msg
		return m.deliver(m.here, m.sized())
	case tea.KeyboardEnhancementsMsg:
		m.keyboard = &msg
	case tea.KeyPressMsg:
		m.refusal = ""
		if m.opener.Open && msg.String() != "ctrl+c" {
			was, folder := m.opener, ""
			if m.opener, folder = m.opener.Key(msg); folder != "" {
				return m.openFolder(folder)
			}
			return m, m.opener.Walk(was)
		}
		if m.dialog.Open && msg.String() != "ctrl+c" {
			was := m.settings
			m.dialog, m.settings = m.dialog.Key(msg, m.settings)
			return m.changed(was)
		}
	case tea.MouseClickMsg:
		if m.opener.Open {
			was, folder := m.opener, ""
			if m.opener, folder = m.opener.Click(msg, m.height(), m.width()); folder != "" {
				return m.openFolder(folder)
			}
			return m, m.opener.Walk(was)
		}
		if m.dialog.Open {
			was := m.settings
			m.dialog, m.settings = m.dialog.Click(msg, m.settings, m.height(), m.width())
			return m.changed(was)
		}
	case tea.MouseWheelMsg:
		if m.opener.Open {
			m.opener = m.opener.Wheel(msg)
			return m, nil
		}
		if m.dialog.Open {
			return m, nil
		}
	case tea.MouseMsg:
		if m.dialog.Open || m.opener.Open {
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
	return m, m.tag(to, cmd)
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
		m.refusal = refusalLine(err)
		return m, nil
	}
	return m.place(to, s)
}

// place shows a screen just made in place of the one shown: told the settings and the workspace, then started, and
// told the terminal's size and its keyboard.
func (m Model) place(to shell.Screen, s tea.Model) (tea.Model, tea.Cmd) {
	m.made[to], m.here = s, to
	_, told := m.deliver(to, shell.SettingsMsg{Settings: m.settings})
	cmds := []tea.Cmd{told}
	if m.workspace.Kind != "" || m.refused != "" {
		_, cmd := m.deliver(to, m.workspaceMsg())
		cmds = append(cmds, cmd)
	}
	cmds = append(cmds, m.tag(to, m.made[to].Init()))
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
	switch {
	case flow.IsApp(to):
		return flow.OpenApp(m.runner, to, nil)
	}
	switch to {
	case shell.Welcome:
		return welcome.Open(m.runner, m.workspace), nil
	case shell.Tutorials:
		return tutorials.Open(m.runner, nil), nil
	case shell.Help:
		return app.OpenHelp(m.runner, "")
	case shell.Manage:
		return workspaces.Open(m.runner), nil
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
	if m.opener.Open {
		lines = m.opener.Overlay(lines, width)
		if m.settings.KeysBar {
			lines[len(lines)-1] = shell.Path{m.opener.Keys()}.Line(width)
		}
	}
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

// tagged is an answer to what a screen started, for that screen, as it was made the epoch-th time over.
type tagged struct {
	to    shell.Screen
	epoch int
	msg   tea.Msg
}

// tag tags the command as the screen's, as it is made now.
func (m Model) tag(to shell.Screen, cmd tea.Cmd) tea.Cmd { return tag(to, m.epochs[to], cmd) }

// tag gives the command whose answer reaches the screen that started it: a batch's commands each tagged so; a
// screen's ask for another, or for a row of another, and what the program itself takes, as quitting, as they are.
func tag(to shell.Screen, epoch int, cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		switch msg := msg.(type) {
		case nil, shell.SwitchMsg, shell.JumpMsg, shell.PickMsg, shell.OpenMsg, shell.ChangedMsg:
			return msg
		case tea.BatchMsg:
			out := make(tea.BatchMsg, len(msg))
			for i, c := range msg {
				out[i] = tag(to, epoch, c)
			}
			return out
		}
		if strings.HasPrefix(reflect.TypeOf(msg).PkgPath(), "charm.land/bubbletea") {
			return msg
		}
		return tagged{to, epoch, msg}
	}
}
