// Package app is kittui's root model: the palette when kit help opened it, a verb's form when a verb handed off to it,
// and the slide between them. It reports, when the program ends, what to run.
package app

import (
	"errors"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/form"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/palette"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/syntax"
)

// Model is what is on the screen.
type Model struct {
	runner        kit.Runner
	palette       palette.Model
	form          form.Model
	hasPalette    bool
	inForm        bool
	verb          string
	width, height int
	keyboard      tea.KeyboardEnhancementsMsg // what the terminal said of its keyboard, once, for each form opened later
	workspace     shell.WorkspaceMsg          // what kittui said of the workspace last, for each form opened later
}

// Open opens the palette when no verb is named, else the verb's form filled from the words given, with no way back.
// An unknown verb, or words the verb refuses, give kit's refusal as the error.
func Open(r kit.Runner, verb string, words []string) (Model, error) {
	m := Model{runner: r}
	if verb == "" {
		verbs, err := r.List()
		if err != nil {
			return m, err
		}
		m.palette, m.hasPalette = palette.New(verbs, r).Remembering(shell.RecentFile()), true
		return m, nil
	}
	spec, err := r.Spec(verb)
	if err != nil {
		return m, err
	}
	match, err := r.Match(verb, words)
	if err != nil {
		return m, err
	}
	if match.Status == 2 {
		return m, &kit.ExitError{Code: 2, Stderr: match.Refusal}
	}
	fields, err := form.Fields(spec, form.FromMatch(match))
	if err != nil {
		return m, err
	}
	m.form, m.inForm, m.verb = form.New(spec, fields, r, false), true, verb
	return m, nil
}

// OpenHelp opens the palette as kit help does: on the named verb's page, or on a group's first verb, or, with no
// name, at the top. Enter on a verb opens its form, as Open's palette does.
func OpenHelp(r kit.Runner, name string) (Model, error) {
	m, err := Open(r, "", nil)
	if err != nil || name == "" {
		return m, err
	}
	m.palette = m.palette.At(name)
	return m, nil
}

// InForm says whether a form is on the screen.
func (m Model) InForm() bool { return m.inForm }

// Result says whether a command line was accepted, and its verb and words after kit <verb>.
func (m Model) Result() (bool, string, []string) {
	if !m.inForm {
		return false, "", nil
	}
	run, args := m.form.Done()
	return run, m.verb, args
}

// Interrupted says whether ctrl-c ended the program.
func (m Model) Interrupted() bool {
	if m.inForm {
		return m.form.Interrupted()
	}
	return m.palette.Interrupted()
}

// Init starts what is on the screen.
func (m Model) Init() tea.Cmd {
	if m.inForm {
		return m.form.Init()
	}
	return m.palette.Init()
}

// Update takes a message: the palette's request to open a verb, the form's to go back, or anything for the pane shown.
// A new size reaches the palette under an open form too, so it draws to the size the terminal has when the form goes.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.inForm && m.hasPalette {
			next, _ := m.palette.Update(msg)
			m.palette = next.(palette.Model)
		}
	case tea.KeyboardEnhancementsMsg:
		m.keyboard = msg
	case shell.WorkspaceMsg:
		// the workspace reaches the palette under an open form too, and the form, which names it as the palette does
		m.workspace = msg
		if m.inForm && m.hasPalette {
			m.palette = m.palette.Told(msg)
		}
	case palette.OpenMsg:
		return m.open(msg.Verb)
	case openedMsg:
		return m.opened(msg)
	case form.BackMsg:
		if m.hasPalette {
			m.inForm = false
			return m, nil
		}
	case form.FindMsg:
		return m.find()
	}
	if m.inForm {
		next, cmd := m.form.Update(msg)
		m.form = next.(form.Model)
		return m, cmd
	}
	next, cmd := m.palette.Update(msg)
	m.palette = next.(palette.Model)
	return m, cmd
}

// find gives the palette's search focus in the form's place, as the form's search field asks: the palette the form
// came from, or, for a form that stood alone, a palette made now, told the terminal's size and the workspace. A
// palette that cannot be made, kit not listing its verbs, leaves the form.
func (m Model) find() (tea.Model, tea.Cmd) {
	if !m.hasPalette {
		verbs, err := m.runner.List()
		if err != nil {
			return m, nil
		}
		m.palette, m.hasPalette = palette.New(verbs, m.runner).Remembering(shell.RecentFile()), true
		if m.width > 0 {
			next, _ := m.palette.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
			m.palette = next.(palette.Model)
		}
		m.palette = m.palette.Told(m.workspace)
	}
	m.inForm, m.palette = false, m.palette.Find()
	return m, nil
}

// openedMsg brings a verb's grammar and its empty fields, or why its form cannot open.
type openedMsg struct {
	verb   string
	spec   syntax.Spec
	fields []form.Field
	err    error
}

// open asks kit for the verb's grammar off the update loop; opened takes the answer.
func (m Model) open(verb string) (tea.Model, tea.Cmd) {
	r := m.runner
	return m, func() tea.Msg {
		spec, err := r.Spec(verb)
		if err != nil {
			return openedMsg{verb: verb, err: err}
		}
		fields, err := form.Fields(spec, form.Answers{})
		return openedMsg{verb: verb, spec: spec, fields: fields, err: err}
	}
}

// opened slides the palette away for the verb's form, empty, with the way back, telling it the terminal's size and
// keyboard, which came before it did; a form that cannot open leaves the palette, with kit's line under its filter.
func (m Model) opened(msg openedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		return m.refuse(msg.err)
	}
	verb := msg.verb
	m.form, m.inForm, m.verb = form.New(msg.spec, msg.fields, m.runner, true), true, verb
	init := m.form.Init()
	if m.width > 0 {
		next, _ := m.form.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		m.form = next.(form.Model)
	}
	next, _ := m.form.Update(m.keyboard)
	m.form = next.(form.Model)
	next, _ = m.form.Update(m.workspace)
	m.form = next.(form.Model)
	return m, init
}

// refuse keeps the palette on the screen, with kit's line about the verb that did not open under its filter.
func (m Model) refuse(err error) (tea.Model, tea.Cmd) {
	m.palette = m.palette.Refuse(err.Error())
	return m, nil
}

// View draws what is on the screen.
func (m Model) View() tea.View {
	if m.inForm {
		return m.form.View()
	}
	return m.palette.View()
}

// Refusal gives kit's line and exit code when an error is kit's refusal.
func Refusal(err error) (string, int, bool) {
	var refusal *kit.ExitError
	if errors.As(err, &refusal) {
		return refusal.Error(), refusal.Code, true
	}
	return "", 0, false
}
