// What kit tui leaves in the scrollback when it runs a verb: the command line alone.
package transcript

import "testing"

func TestTheTranscriptIsTheCommandLineAlone(t *testing.T) {
	got := Render("commit", []string{"--message", "#1 tui: the first form", ".gitignore"})
	if want := "kit commit --message '#1 tui: the first form' .gitignore\n"; got != want {
		t.Errorf("transcript %q, want %q", got, want)
	}
	if got := Render("version", nil); got != "kit version\n" {
		t.Errorf("a verb with no words: %q", got)
	}
}
