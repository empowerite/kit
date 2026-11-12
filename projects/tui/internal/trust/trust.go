// Package trust is the dialog kittui opens in a clone kit does not trust yet, before it reads anything there, as an
// editor asks before it trusts a folder: it names the clone, says what trusting it lets kit do, from what kit trust
// --asks prints, and offers Trust, which runs kit trust --yes on the clone, and Not now, which leaves it untrusted, so
// kittui opens on what kit runs there, the palette of kit help. A plain click takes either button, as does the
// keyboard.
package trust

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// The buttons, in the order they stand, and the line the dialog's keys bar gives.
const (
	TrustButton  = "Trust"
	NotNowButton = "Not now"
	keys         = "←→ choose · ⏎ press · esc not now"
)

// Model is the dialog: the clone's name, what it asks kit to run or why that could not be read, which button has the
// keyboard, the screen's size, what kit trust --yes said when it refused, whether it runs now, and how the dialog
// ended.
type Model struct {
	r             kit.Runner
	name          string
	asks          []string
	unread        string
	focus         int
	width, height int
	refused       string
	busy          bool
	trusted       bool
	interrupted   bool
}

// trustedMsg is kit trust --yes's answer: nil, or its refusal.
type trustedMsg struct{ err error }

// Open makes the dialog for the clone kit trusts by name, reading what it asks kit to run; a read kit refuses is said
// in the dialog in place of the list.
func Open(r kit.Runner, name string) Model {
	m := Model{r: r, name: name}
	asks, err := r.Asks()
	if err != nil {
		m.unread = err.Error()
	}
	m.asks = asks
	return m
}

// Trusted says whether the person trusted the clone, and kit recorded it.
func (m Model) Trusted() bool { return m.trusted }

// Interrupted says whether ctrl-c ended the dialog.
func (m Model) Interrupted() bool { return m.interrupted }

// Init starts nothing: the dialog has read what it shows.
func (m Model) Init() tea.Cmd { return nil }

// Update takes the screen's size, a key, a click on a button, and kit trust --yes's answer.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case trustedMsg:
		m.busy = false
		if msg.err != nil {
			m.refused = msg.err.Error()
			return m, nil
		}
		m.trusted = true
		return m, tea.Quit
	case tea.KeyPressMsg:
		return m.key(msg)
	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			return m, nil
		}
		if b, ok := m.at(msg.X, msg.Y); ok {
			m.focus = b
			return m.press()
		}
	}
	return m, nil
}

// key takes a key: ctrl-c ends the dialog; the arrows and tab move between the buttons; enter and space press the one
// with the keyboard; t presses Trust, and esc and n Not now.
func (m Model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.interrupted = true
		return m, tea.Quit
	case "left", "right", "tab", "shift+tab":
		m.focus = 1 - m.focus
	case "enter", "space":
		return m.press()
	case "t":
		m.focus = 0
		return m.press()
	case "esc", "n":
		m.focus = 1
		return m.press()
	}
	return m, nil
}

// press presses the button with the keyboard: Trust runs kit trust --yes on the clone, once at a time, and Not now
// ends the dialog with the clone as it was.
func (m Model) press() (tea.Model, tea.Cmd) {
	if m.focus == 1 {
		return m, tea.Quit
	}
	if m.busy {
		return m, nil
	}
	m.busy, m.refused = true, ""
	r, name := m.r, m.name
	return m, func() tea.Msg { return trustedMsg{r.TrustClone(name)} }
}

// width0 is the width the dialog wraps its prose to: the screen's, at most 100, and 80 before the screen's is known.
func (m Model) width0() int {
	switch {
	case m.width <= 0:
		return 80
	case m.width > 100:
		return 100
	}
	return m.width
}

// lines gives the dialog's lines, and the line the buttons stand on.
func (m Model) lines() ([]string, int) {
	w := m.width0() - 2
	out := []string{"", "  " + style.Label("Trust this clone?"), "", "  " + style.Repo(m.name), ""}
	out = append(out, wrap("kit runs nothing a clone names until you trust it. Trusting this one lets kit:", w)...)
	switch {
	case m.unread != "":
		out = append(out, wrap(style.Excluded+" what it asks could not be read: "+m.unread, w)...)
	case len(m.asks) == 0:
		out = append(out, wrap("• run nothing beyond the kit itself", w)...)
	}
	for _, a := range m.asks {
		out = append(out, wrap("• "+a, w)...)
	}
	out = append(out, "")
	out = append(out, wrap("You accept code from whoever can push to it. kit does not check each pull; review and "+
		"signed commits do. It asks again when a pull changes what the clone asks it to run.", w)...)
	out = append(out, "")
	row := len(out)
	out = append(out, "  "+m.button(0)+"   "+m.button(1))
	if m.busy {
		out = append(out, "", "  trusting "+m.name+"…")
	}
	if m.refused != "" {
		out = append(out, "")
		out = append(out, wrap(style.Refusal(style.Excluded+" "+m.refused), w)...)
	}
	return out, row
}

// button draws a button, reversed where it has the keyboard.
func (m Model) button(b int) string {
	label := "[ " + []string{TrustButton, NotNowButton}[b] + " ]"
	if b == m.focus {
		return style.Selected(label)
	}
	return label
}

// at says which button a cell of the screen is on: the button line's cells under Trust's label, or under Not now's.
func (m Model) at(x, y int) (int, bool) {
	_, row := m.lines()
	if y != row {
		return 0, false
	}
	first := 2 + ansi.StringWidth("[ "+TrustButton+" ]")
	second := first + 3
	switch {
	case x >= 2 && x < first:
		return 0, true
	case x >= second && x < second+ansi.StringWidth("[ "+NotNowButton+" ]"):
		return 1, true
	}
	return 0, false
}

// View draws the dialog over its keys bar, on the alternate screen, with the mouse on for clicks.
func (m Model) View() tea.View {
	lines, _ := m.lines()
	bar := style.BarDoes(keys, true)
	v := tea.NewView(strings.Join(style.Bottom(lines, m.height, bar), "\n"))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// wrap breaks a paragraph into lines of at most width cells, each indented two cells, a word too long for a line on
// one of its own.
func wrap(text string, width int) []string {
	var out []string
	line := ""
	for _, word := range strings.Fields(text) {
		if line != "" && ansi.StringWidth(line)+1+ansi.StringWidth(word) > width {
			out = append(out, "  "+line)
			line = ""
		}
		if line != "" {
			line += " "
		}
		line += word
	}
	return append(out, "  "+line)
}
