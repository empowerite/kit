// A notice stands in the keys bar's place: what was done, which goes after a while, unless a key or another notice
// takes its place first; or why something was not done, which stays until a key.
package shell

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// Notice is what stands in the keys bar's place, none while the text is empty: whether it says what was done, and the
// notices of what was done put up so far, the serial each one's fade carries.
type Notice struct {
	Text string
	Done bool
	put  int
}

// FadedMsg says a notice of what was done has been up its while: the serial of the one put up, and what it said.
type FadedMsg struct {
	put  int
	text string
}

// Did gives the notice saying what was done, which fades.
func (n Notice) Did(text string) Notice {
	n.Text, n.Done = text, true
	n.put++
	return n
}

// FadeMsg gives the message that fades the notice up now.
func (n Notice) FadeMsg() FadedMsg { return FadedMsg{n.put, n.Text} }

// Fading gives the command that fades the notice after a while, when it was put up since the notice was; nil when it
// was not.
func (n Notice) Fading(was Notice, after time.Duration) tea.Cmd {
	if n.put == was.put {
		return nil
	}
	msg := n.FadeMsg()
	return tea.Tick(after, func(time.Time) tea.Msg { return msg })
}

// Faded gives the notice once a fade comes: gone, while it is still the notice of what was done the fade is for; as it
// was, once another took its place.
func (n Notice) Faded(f FadedMsg) Notice {
	if f.put == n.put && n.Done && n.Text == f.text {
		n.Text = ""
	}
	return n
}
