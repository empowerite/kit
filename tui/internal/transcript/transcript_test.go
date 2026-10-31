// The transcript left in the scrollback: one ✓ line per field, then the command line, quoted as a shell needs.
package transcript

import (
	"testing"

	"github.com/empowerite/kit/tui/internal/form"
)

func TestTranscriptText(t *testing.T) {
	fields := []form.Field{
		{Key: "--all", Label: "--all", Kind: form.Toggle, Chosen: -1, Group: -1},
		{Key: "--message | --file", Label: "--message | --file", Kind: form.OneOf, Options: []string{"--message", "--file"}, Chosen: 0, Required: true, Group: -1},
		{Key: "--message", Label: "--message", Kind: form.Lines, Values: []string{"#1 tui: the first form"}, Chosen: -1, Group: 1},
		{Key: "<path>", Label: "<path>…", Kind: form.PickMany, Values: []string{".gitignore"}, Chosen: -1, Group: -1},
	}
	want := "  ✓ --all      no\n" +
		"  ✓ --message  #1 tui: the first form\n" +
		"  ✓ <path>…    .gitignore\n" +
		"\n" +
		"  kit commit --message '#1 tui: the first form' .gitignore\n" +
		"\n"
	got := Render("commit", fields, []string{"--message", "#1 tui: the first form", ".gitignore"})
	if got != want {
		t.Errorf("transcript:\n%s\nwant:\n%s", got, want)
	}
}

func TestARowWithoutFieldsOfItsOwnSaysItsChoice(t *testing.T) {
	fields := []form.Field{
		{Key: "--no-fetch --json", Label: "--no-fetch --json", Kind: form.AnyOf, Options: []string{"--no-fetch", "--json"}, Values: []string{"--json"}, Chosen: -1, Group: -1},
		{Key: "--all | --mine --others --owner", Label: "--all | --mine --others --owner", Kind: form.OneOf, Options: []string{"--all", "--mine --others --owner"}, Chosen: 0, Group: -1},
		{Key: "<x>", Label: "<x>", Kind: form.Text, Chosen: -1, Group: -1},
	}
	want := "  ✓ --no-fetch --json  --json\n" +
		"  ✓ --all              yes\n" +
		"  ✓ <x>                none\n" +
		"\n" +
		"  kit status --json --all\n" +
		"\n"
	if got := Render("status", fields, []string{"--json", "--all"}); got != want {
		t.Errorf("transcript:\n%s\nwant:\n%s", got, want)
	}
}
