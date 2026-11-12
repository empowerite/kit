package shell

import "testing"

// A web address is a link the browser opens; a link to another place asks for the screen, or a tab of one with what it
// keeps in view, with the page as the crumb; none asks for nothing.
func TestALinkAsksForWhereItLeads(t *testing.T) {
	if !IsWeb("https://x.y") || !IsWeb("http://x.y") || IsWeb("flow") || IsWeb("kit.yml") {
		t.Error("IsWeb")
	}
	for _, c := range []struct {
		to   string
		want any
	}{
		{"flow", SwitchMsg{Screen: Flow, Crumb: true}},
		{"flow/issues", JumpMsg{Screen: Flow, To: "issues/"}},
		{"welcome/terms/workspace", JumpMsg{Screen: Welcome, To: "terms/workspace"}},
		{"tutorials/search", JumpMsg{Screen: Tutorials, To: "search/"}},
	} {
		if got := Link(c.to)(); got != c.want {
			t.Errorf("Link(%q) asked %#v, want %#v", c.to, got, c.want)
		}
	}
	if Link("") != nil {
		t.Error("no link asked for something")
	}
}
