// A page is drawn as text: the cases a page meets, each held to its lines, with and without their escapes.
package markdown

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
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

// Wide leaves a code block's and a table's lines whole, each a block of its own, which Render cuts at the width;
// prose wraps, in no block, and a block left open at the page's end ends there.
func TestWideKeepsABlocksLinesWholeAndSaysWhereItStands(t *testing.T) {
	code := strings.Repeat("x", 30)
	md := "Some prose that wraps.\n\n```\n" + code + "\nshort\n```\n| a | " + strings.Repeat("b", 20) +
		" |\n| c |\nafter\n\n```\nopen"
	lines, _, blocks := Wide(md, 12)
	want := []pane.Block{{ID: "0", From: 3, To: 5, Wide: 32}, {ID: "1", From: 5, To: 7, Wide: 28},
		{ID: "2", From: 9, To: 10, Wide: 6}}
	if !slices.Equal(blocks, want) {
		t.Fatalf("blocks %v, want %v\n%s", blocks, want, ansi.Strip(strings.Join(lines, "\n")))
	}
	if got := ansi.Strip(lines[3]); got != "  "+code {
		t.Errorf("a code block's line, whole: %q", got)
	}
	if got := ansi.Strip(Render(md, 12)[3]); got != "  xxxxxxxxx…" {
		t.Errorf("Render cuts it: %q", got)
	}
	if got := ansi.Strip(lines[7]); got != "after" {
		t.Errorf("the prose after the table: %q", got)
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
	// a link of several words is one for each word, each where it stands on its line; a web link's text and its address are links too
	wantLinks := []Link{{0, 3, 6, "flow/issues"}, {2, 5, 3, "flow"}, {2, 9, 4, "flow"}, {2, 14, 3, "flow"},
		{3, 4, 3, "https://x.y"}, {3, 8, 4, "https://x.y"}, {4, 1, 11, "https://x.y"}}
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

// A link to a tutorial or a term is a link as one to a screen is, its target whole after kittui:; a code span in a link's
// text is drawn as code, and a link to a file is its text.
func TestALinkToATutorialOrATermLeadsThereAndACodeSpanInALinkIsCode(t *testing.T) {
	lines, links := Page("Read [the menu bar](kittui:tutorials/menu-bar), the [workspace](kittui:welcome/terms/workspace),"+
		" `kit flow`, and [`kit flow`](kittui:flow) in [the file](kit.yml).", 120)
	if got := ansi.Strip(lines[0]); got != "Read the menu bar, the workspace, kit flow, and kit flow in the file." {
		t.Errorf("got %q", got)
	}
	if strings.Contains(lines[0], "\x1b[1;94m\x1b[4") || !strings.Contains(lines[0], style.Link("flow")) {
		t.Errorf("a code span in a link's text is drawn in the link's style: %q", lines[0])
	}
	var to []string
	for _, k := range links {
		to = append(to, k.To)
	}
	want := []string{"tutorials/menu-bar", "tutorials/menu-bar", "tutorials/menu-bar", "welcome/terms/workspace", "flow", "flow"}
	if !slices.Equal(to, want) {
		t.Errorf("links lead to %q, want %q", to, want)
	}
}

// Unlinked keeps a link's text, and a web address after it; any other link is its text.
func TestUnlinkedKeepsTheTextAndAWebAddress(t *testing.T) {
	got := Unlinked("[the docs](https://x.y/z), [Flow](kittui:flow), and [a file](kit.yml)")
	if want := "the docs (https://x.y/z), Flow, and a file"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
