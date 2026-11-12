package flow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// atTheOpener makes shell.Browse the screen's way to open a page, with an open and an xdg-open on PATH that record the
// links they are given in the file returned, and the environment of a desk with a display: SSH_CONNECTION and SSH_TTY
// are left for a test to set.
func atTheOpener(t *testing.T, m *Model) string {
	t.Helper()
	dir := t.TempDir()
	record := filepath.Join(dir, "record")
	for _, opener := range []string{"open", "xdg-open"} {
		script := "#!/bin/sh\nprintf '%s\\n' \"$1\" >> " + record + "\n"
		if err := os.WriteFile(filepath.Join(dir, opener), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	t.Setenv("DISPLAY", ":0")
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_TTY", "")
	m.browse = shell.Browse
	return record
}

// ran says whether the opener was run, waiting a moment for one that was started.
func ran(record string) bool {
	for range 100 {
		if _, err := os.Stat(record); err == nil {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// Over SSH, opening a page shows its link, to copy, until a key, and runs no opener; at a desk it runs the opener and
// says nothing.
func TestOverSSHOpeningAPageShowsTheLinkAndRunsNoOpener(t *testing.T) {
	const page = "https://github.com/empowerite/kit/pull/4"
	m := loaded(t, options())
	record := atTheOpener(t, &m)
	t.Setenv("SSH_CONNECTION", "1.2.3.4 5 6.7.8.9 22")
	m = drive(m, m.openPage(page)())
	if want := "Open " + page + " in a browser"; m.note.Text != want || m.note.Done {
		t.Errorf("the notice: %q (noted %v), want %q", m.note.Text, m.note.Done, want)
	}
	if v := view(m); !strings.Contains(v, "Open "+page+" in a browser") {
		t.Errorf("the link is not on the screen:\n%s", v)
	}
	if ran(record) {
		t.Error("an opener ran over SSH")
	}
	if m = drive(m, down); m.note.Text != "" {
		t.Errorf("a key kept the link: %q", m.note.Text)
	}
	// at a desk, the opener runs and the notice stays empty
	t.Setenv("SSH_CONNECTION", "")
	m = drive(m, m.openPage(page)())
	if !ran(record) || m.note.Text != "" {
		t.Errorf("at a desk: opener ran %v, notice %q", ran(record), m.note.Text)
	}
	if got, _ := os.ReadFile(record); string(got) != page+"\n" {
		t.Errorf("the opener was given %q", got)
	}
}

// Open Selected over SSH shows the first page's link and says how many more were asked, and runs no opener.
func TestOverSSHOpenSelectedShowsTheFirstLinkAndCountsTheRest(t *testing.T) {
	m := loaded(t, options(), down)
	record := atTheOpener(t, &m)
	t.Setenv("SSH_TTY", "/dev/pts/0")
	m.stages[0].Picked["#45"], m.stages[0].Picked["#46"] = true, true
	pages := m.pages()
	if len(pages) != 2 {
		t.Fatalf("pages: %q", pages)
	}
	m = choose(t, m, "Open Selected")
	if want := "Open " + pages[0] + " in a browser, and 1 more page"; m.note.Text != want || m.note.Done {
		t.Errorf("the notice: %q (noted %v), want %q", m.note.Text, m.note.Done, want)
	}
	if ran(record) {
		t.Error("an opener ran over SSH")
	}
	// three pages asked say two more
	m.stages[0].Picked["#7"] = true
	three := m.pages()
	if len(three) != 3 {
		t.Fatalf("pages: %q", three)
	}
	if m = choose(t, m, "Open Selected"); m.note.Text != "Open "+three[0]+" in a browser, and 2 more pages" {
		t.Errorf("three pages: %q", m.note.Text)
	}
	// one page asked says no more
	delete(m.stages[0].Picked, "#7")
	delete(m.stages[0].Picked, "#45")
	if m = choose(t, m, "Open Selected"); m.note.Text != "Open "+pages[0]+" in a browser" {
		t.Errorf("one page: %q", m.note.Text)
	}
}
