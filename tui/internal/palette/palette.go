// Package palette is what bare kit opens: every verb but tui with its purpose on the left, filtered as it is typed,
// and the highlighted verb's page on the right. Enter opens the verb's form; esc quits.
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
	width, height int
	cancelled     bool
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

// Init fetches the first verb's page.
func (m Model) Init() tea.Cmd { return m.page() }

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
		case "esc", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			if m.cursor < len(visible) {
				verb := visible[m.cursor].Name
				return m, func() tea.Msg { return OpenMsg{Verb: verb} }
			}
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

// page fetches the highlighted verb's page, once per verb.
func (m Model) page() tea.Cmd {
	visible := m.Visible()
	if m.cursor >= len(visible) {
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

// View draws the two panes and the keys, on the alternate screen.
func (m Model) View() tea.View {
	visible := m.Visible()
	leftWidth := m.width * 2 / 5
	if leftWidth > 48 {
		leftWidth = 48
	}
	var left strings.Builder
	left.WriteString(" " + style.Filter + " " + m.filter + "▏\n\n")
	if len(visible) == 0 {
		left.WriteString("   " + style.Off("no verb matches") + "\n")
	}
	widest := 0
	for _, v := range m.verbs {
		if len(v.Name) > widest {
			widest = len(v.Name)
		}
	}
	rows := m.height - 6
	start := 0
	if m.cursor >= rows && rows > 0 {
		start = m.cursor - rows + 1
	}
	for i := start; i < len(visible) && i-start < rows; i++ {
		v := visible[i]
		name := v.Name + strings.Repeat(" ", widest-len(v.Name))
		lead := "   "
		if i == m.cursor {
			lead, name = " "+style.Focus+" ", style.Chosen(v.Name)+strings.Repeat(" ", widest-len(v.Name))
		}
		left.WriteString(lead + name + "  " + style.Off(clip(v.Purpose, leftWidth-widest-6)) + "\n")
	}
	right := ""
	if m.cursor < len(visible) {
		right = m.pages[visible[m.cursor].Name]
	}
	rightWidth := m.width - leftWidth - 2
	panes := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(leftWidth).Render(left.String()),
		lipgloss.NewStyle().Width(rightWidth).MaxHeight(m.height-2).PaddingLeft(2).Render(page(right, rightWidth-2)))
	v := tea.NewView(panes + "\n " + style.Off("↑↓ move · type to filter · ⏎ open · esc quit"))
	v.AltScreen = true
	return v
}

// page wraps a help page to the width, keeping each line's indent.
func page(text string, width int) string {
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
	return strings.Join(out, "\n")
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
