// Package app is kittui's root model: the palette when bare kit opened it, a verb's form when a verb handed off to it,
// and the slide between them. It reports, when the program ends, what to run.
package app

import (
	"errors"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/tui/internal/form"
	"github.com/empowerite/kit/tui/internal/kit"
	"github.com/empowerite/kit/tui/internal/palette"
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
		m.palette, m.hasPalette = palette.New(verbs, r), true
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

// Fields gives the form's fields as they stood at the end, for the transcript.
func (m Model) Fields() []form.Field { return m.form.Fields() }

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
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case palette.OpenMsg:
		return m.open(msg.Verb)
	case form.BackMsg:
		if m.hasPalette {
			m.inForm = false
			return m, nil
		}
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

// open slides the palette away for the verb's form, empty, with the way back.
func (m Model) open(verb string) (tea.Model, tea.Cmd) {
	spec, err := m.runner.Spec(verb)
	if err != nil {
		return m, nil
	}
	fields, err := form.Fields(spec, form.Answers{})
	if err != nil {
		return m, nil
	}
	m.form, m.inForm, m.verb = form.New(spec, fields, m.runner, true), true, verb
	init := m.form.Init()
	if m.width > 0 {
		next, _ := m.form.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		m.form = next.(form.Model)
	}
	return m, init
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
