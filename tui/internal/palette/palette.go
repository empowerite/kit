// Package palette is what bare kit opens: every verb but tui with its whole purpose, filtered as it is typed, under
// the keys. The right arrow trades the purposes for the highlighted verb's page beside the names, and the left arrow
// brings them back. Enter opens the verb's form; esc quits.
package palette

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/empowerite/kit/tui/internal/kit"
	"github.com/empowerite/kit/tui/internal/style"
)

// OpenMsg asks for the verb's form.
type OpenMsg struct{ Verb string }

// pageMsg brings a verb's page.
type pageMsg struct {
	verb, page string
	err        error
}

// Model is the palette on the screen.
type Model struct {
	verbs         []kit.Verb
	runner        kit.Runner
	filter        string
	cursor        int
	pages         map[string]string
	showPages     bool
	width, height int
	cancelled     bool
	interrupted   bool
}

// New opens the palette on the verbs, leaving out tui itself.
func New(verbs []kit.Verb, r kit.Runner) Model {
	m := Model{runner: r, pages: map[string]string{}, width: 100, height: 30}
	for _, v := range verbs {
		if v.Name != "tui" {
			m.verbs = append(m.verbs, v)
		}
	}
	return m
}

// Init starts nothing: the list needs no page until the right arrow asks for one.
func (m Model) Init() tea.Cmd { return nil }

// Interrupted says whether ctrl-c ended the palette.
func (m Model) Interrupted() bool { return m.interrupted }

// Cancelled says whether esc ended the palette.
func (m Model) Cancelled() bool { return m.cancelled }

// Visible gives the verbs the filter leaves: those whose name holds the text typed.
func (m Model) Visible() []kit.Verb {
	var out []kit.Verb
	for _, v := range m.verbs {
		if strings.Contains(v.Name, strings.ToLower(m.filter)) {
			out = append(out, v)
		}
	}
	return out
}

// Update takes a message.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case pageMsg:
		if msg.err != nil {
			m.pages[msg.verb] = msg.err.Error()
		} else {
			m.pages[msg.verb] = msg.page
		}
	case tea.KeyPressMsg:
		visible := m.Visible()
		switch msg.String() {
		case "esc":
			m.cancelled = true
			return m, tea.Quit
		case "ctrl+c":
			m.interrupted = true
			return m, tea.Quit
		case "enter":
			if m.cursor < len(visible) {
				verb := visible[m.cursor].Name
				return m, func() tea.Msg { return OpenMsg{Verb: verb} }
			}
		case "right":
			m.showPages = true
		case "left":
			m.showPages = false
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(visible)-1 {
				m.cursor++
			}
		case "backspace":
			if r := []rune(m.filter); len(r) > 0 {
				m.filter = string(r[:len(r)-1])
				m.cursor = 0
			}
		default:
			if msg.Text != "" && msg.Mod&(tea.ModCtrl|tea.ModAlt) == 0 {
				m.filter += msg.Text
				m.cursor = 0
			}
		}
		return m, m.page()
	}
	return m, nil
}

// page fetches the highlighted verb's page, once per verb, when the pages are shown.
func (m Model) page() tea.Cmd {
	visible := m.Visible()
	if !m.showPages || m.cursor >= len(visible) {
		return nil
	}
	verb := visible[m.cursor].Name
	if _, ok := m.pages[verb]; ok {
		return nil
	}
	r := m.runner
	return func() tea.Msg {
		page, err := r.Help(verb)
		return pageMsg{verb: verb, page: page, err: err}
	}
}

// View draws the keys on top, so filtering never moves them, then the filter and the verbs: each with its whole
// purpose, clipped at the screen's edge, or, after the right arrow, the names beside the highlighted verb's page.
func (m Model) View() tea.View {
	visible := m.Visible()
	keys := "↑↓ move · type to filter · → pages · ⏎ open · esc quit"
	if m.showPages {
		keys = "↑↓ move · type to filter · ← purposes · ⏎ open · esc quit"
	}
	var top strings.Builder
	for _, line := range page(keys, m.width-2) {
		top.WriteString(" " + style.Off(line) + "\n")
	}
	top.WriteString("\n " + style.Filter + " " + m.filter + "▏\n\n")
	widest := 0
	for _, v := range m.verbs {
		if len(v.Name) > widest {
			widest = len(v.Name)
		}
	}
	used := strings.Count(top.String(), "\n")
	rows := m.height - used - 1
	if rows < 1 {
		rows = 1
	}
	start := 0
	if m.cursor >= rows {
		start = m.cursor - rows + 1
	}
	var list strings.Builder
	if len(visible) == 0 {
		list.WriteString("   " + style.Off("no verb matches") + "\n")
	}
	for i := start; i < len(visible) && i-start < rows; i++ {
		v := visible[i]
		lead, name := "   ", v.Name
		if i == m.cursor {
			lead, name = " "+style.Focus+" ", style.Chosen(v.Name)
		}
		line := lead + name + strings.Repeat(" ", widest-len(v.Name))
		if !m.showPages {
			line += "  " + style.Off(clip(v.Purpose, m.width-3-widest-2))
		}
		list.WriteString(line + "\n")
	}
	content := top.String() + list.String()
	if m.showPages {
		right := ""
		if m.cursor < len(visible) {
			right = m.pages[visible[m.cursor].Name]
		}
		leftWidth := 3 + widest + 2
		rightWidth := m.width - leftWidth
		content = top.String() + lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(leftWidth).Render(strings.TrimSuffix(list.String(), "\n")),
			lipgloss.NewStyle().Width(rightWidth).MaxHeight(rows).Render(strings.Join(page(right, rightWidth-1), "\n")))
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// page wraps a text to the width, keeping each line's indent, and gives its lines.
func page(text string, width int) []string {
	var out []string
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
		words := strings.Fields(line)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		current := indent + words[0]
		for _, word := range words[1:] {
			if len([]rune(current))+1+len([]rune(word)) > width {
				out = append(out, current)
				current = indent + word
				continue
			}
			current += " " + word
		}
		out = append(out, current)
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
