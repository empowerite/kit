// A page is drawn as text: the cases a page meets, each held to its lines, with and without their escapes.
package welcome

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// plain draws a page and strips its escapes.
func plain(md string, width int) string { return ansi.Strip(strings.Join(render(md, width), "\n")) }

func TestHeadingsAreBoldAndSetApartByOneBlankLine(t *testing.T) {
	lines := render("# The `kit` way\n\n\n## Next ##\ntext\n", 40)
	want := []string{style.Label("The kit way"), "", style.Label("Next"), "", "text"}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("got %q", lines)
	}
}

func TestInlineCodeIsInTheAccentAndStrongIsBold(t *testing.T) {
	lines := render("Run `kit flow` **now**, then `kit work`.", 80)
	want := "Run " + style.Mark("kit") + " " + style.Mark("flow") + " " + style.Label("now") + ", then " +
		style.Mark("kit") + " " + style.Mark("work") + "."
	if len(lines) != 1 || lines[0] != want {
		t.Errorf("got %q\nwant %q", lines, want)
	}
}

func TestParagraphsWrapToTheWidth(t *testing.T) {
	got := plain("one two three\nfour five six seven", 14)
	if got != "one two three\nfour five six\nseven" {
		t.Errorf("got %q", got)
	}
	for _, l := range render(strings.Repeat("`word` ", 40), 20) {
		if ansi.StringWidth(l) > 20 {
			t.Errorf("%q is wider than 20", ansi.Strip(l))
		}
	}
}

func TestListItemsKeepTheirMarkerAndHangTheirLines(t *testing.T) {
	md := "- first item that wraps\n  onto two lines\n* second\n  - nested\n1. one\n2) two\n"
	want := "• first item\n  that wraps\n  onto two\n  lines\n• second\n  • nested\n1. one\n2) two"
	if got := plain(md, 14); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestCodeBlocksKeepTheirLinesAndCommentsAreDropped(t *testing.T) {
	md := "<!-- a note\nfor the linter -->\ntext\n```sh\nkit flow   desk\n```\n<!-- one line -->\n| a | b |\n"
	if got := plain(md, 40); got != "text\n\n  kit flow   desk\n| a | b |" {
		t.Errorf("got %q", got)
	}
}

func TestALinkKeepsItsTextAndAWebAddress(t *testing.T) {
	got := plain("See [the docs](https://example.com/docs) and [the rule](../rules/x.md).", 80)
	if got != "See the docs (https://example.com/docs) and the rule." {
		t.Errorf("got %q", got)
	}
}
