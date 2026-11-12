// The links on a step's page, as internal/markdown says where they stand: a plain click on one follows it, as one on a
// tutorial's page does. A link to a web address opens the browser, one to a tab of this screen, or to a term of the
// Terms, as welcome/terms/ and the term, shows it here, and any other goes where shell.Link says, a screen, a tab of
// one, or a tutorial.
package welcome

import (
	"fmt"
	"net/url"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/markdown"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// linkedMsg says whether the browser was asked to open a link: err is why not.
type linkedMsg struct {
	url string
	err error
}

// linkIn gives where the link at the cell x of the line leads, or "" where there is none.
func linkIn(links []markdown.Link, x int) string {
	for _, l := range links {
		if x >= l.Col && x < l.Col+l.Width {
			return l.To
		}
	}
	return ""
}

// leadLinkAt gives where the link at a cell of the screen leads, over the lead of the selected step drawn as a table,
// or "" where there is none.
func (m Model) leadLinkAt(x, y int) string {
	if len(m.leadLines(m.step)) == 0 {
		return ""
	}
	_, links := markdown.Page(m.tabs[m.step].lead, m.width-len(lead)-1)
	var on []markdown.Link
	for _, l := range links {
		if l.Line == y-m.paneTop() {
			l.Col += len(lead)
			on = append(on, l)
		}
	}
	return linkIn(on, x)
}

// follow does what a link leads to: the command that asks for it, none for a link to a place on this screen, which it
// shows at once.
func (m *Model) follow(to string) tea.Cmd {
	if rest, here := strings.CutPrefix(to, string(shell.Welcome)); here && (rest == "" || rest[0] == '/') {
		m.jump(strings.TrimPrefix(rest, "/"))
		return nil
	}
	if shell.IsWeb(to) {
		browse := m.browse
		return func() tea.Msg { return linkedMsg{to, browse(to)} }
	}
	return shell.Link(to)
}

// jump shows a tab of the screen, by its id, and where the tab keeps it in view, after a slash: a term of the Terms is
// what its search narrows the table to. A tab the screen has not, and no tab, leave it as it is.
func (m *Model) jump(to string) {
	id, rest, _ := strings.Cut(to, "/")
	if term, err := url.PathUnescape(rest); err == nil {
		rest = term // a term of several words, as pull%20request
	}
	i := indexOf(m.tabs, id)
	if i < 0 {
		return
	}
	b := m.bar()
	b.Selected = i
	m.choose(b)
	if rest != "" {
		s := m.search()
		s.Query = rest
		m.setSearch(s)
	}
}

// linked says what a link's opening came to, in place of the keys until a key: the link to open by hand where the
// browser cannot be asked, or why it could not.
func (m *Model) linked(msg linkedMsg) {
	switch {
	case msg.err == nil:
	case msg.err == shell.ErrNoDisplay:
		m.link = "Open " + msg.url + " in a browser"
	default:
		m.link = fmt.Sprintf("%s could not open %s: %v", Failed, msg.url, msg.err)
	}
}

// linkNotice gives the keys bar's line in place of the keys while a link's opening has something to say, none else.
func (m Model) linkNotice() string {
	if m.link == "" {
		return ""
	}
	return " " + style.Refusal(m.link)
}
