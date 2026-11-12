// A page is drawn as text: the cases a page meets, each held to its lines, with and without their escapes.
package markdown

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// plain draws a page and strips its escapes.
func plain(md string, width int) string { return ansi.Strip(strings.Join(Render(md, width), "\n")) }

func TestHeadingsAreBoldAndSetApartByOneBlankLine(t *testing.T) {
	lines := Render("# The `kit` way\n\n\n## Next ##\ntext\n", 40)
	want := []string{style.Label("The kit way"), "", style.Label("Next"), "", "text"}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("got %q", lines)
	}
}

func TestATitleBoldsItsNameAndNotWhatFollowsItsDash(t *testing.T) {
	lines := Render("# Terms — These are the words `kit` uses.\ntext\n", 80)
	want := []string{style.Label("Terms") + " — These are the words " + style.Mark("kit") + " uses.", "", "text"}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("got %q\nwant %q", lines, want)
	}
	if got := plain("# Customize — kit is a starting point, and these are the parts teams change first.", 30); got !=
		"Customize — kit is a starting\npoint, and these are the parts\nteams change first." {
		t.Errorf("a long title wraps as a paragraph does: got %q", got)
	}
	if lines := Render("## Next — then", 40); len(lines) != 1 || lines[0] != style.Label("Next — then") {
		t.Errorf("only a page's title splits at its dash: got %q", lines)
	}
}

func TestInlineCodeIsInTheAccentAndStrongIsBold(t *testing.T) {
	lines := Render("Run `kit flow` **now**, then `kit work`.", 80)
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
	for _, l := range Render(strings.Repeat("`word` ", 40), 20) {
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

func TestFrontMatterIsDropped(t *testing.T) {
	if got := plain("---\ntitle: The tour\n---\n\n# Flow\n", 40); got != "Flow" {
		t.Errorf("got %q", got)
	}
	if got := plain("text\n---\nmore\n", 40); got != "text --- more" {
		t.Errorf("a rule under text: %q", got)
	}
}

func TestALinkToAScreenIsDrawnAsALinkAndPageSaysWhereItStands(t *testing.T) {
	md := "1. [Issues](kittui:flow/issues): new.\n\nOpen [the Flow app](kittui:flow), or see [the site](https://x.y)."
	lines, links := Page(md, 22)
	want := "1. Issues: new.\n\nOpen the Flow app, or\nsee the site\n(https://x.y)."
	if got := ansi.Strip(strings.Join(lines, "\n")); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	if !strings.Contains(lines[0], style.Link("Issues")) || strings.Contains(strings.Join(lines, ""), "\x1b]8") {
		t.Errorf("a link is drawn in its style, its markers taken out: %q", lines[0])
	}
	// a link of several words is one for each word, each where it stands on its line; a web link is no screen's
	wantLinks := []Link{{0, 3, 6, "flow/issues"}, {2, 5, 3, "flow"}, {2, 9, 4, "flow"}, {2, 14, 3, "flow"}}
	if len(links) != len(wantLinks) {
		t.Fatalf("got %+v", links)
	}
	for i, k := range links {
		if k != wantLinks[i] {
			t.Errorf("link %d: got %+v, want %+v", i, k, wantLinks[i])
		}
	}
	if Render("[Flow](kittui:flow)", 20)[0] != style.Link("Flow") {
		t.Errorf("Render draws a link as Page does: %q", Render("[Flow](kittui:flow)", 20))
	}
}
