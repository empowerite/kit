// Package markdown draws a page's markdown as text, for every screen that shows a page alike: headings bold, but a
// page's title, `# <name> — <about>`, has only its name bold, inline code in the accent, list items kept with a hanging
// indent, and paragraphs wrapped to the width. It is no markdown library: front matter at the top is dropped, a code
// block keeps its lines, a table its rows, a comment is dropped, and a link keeps its text, with its URL when it is a
// web address. A link to one of kittui's screens, kittui: and the screen's name, as [Flow](kittui:flow) is, or the
// screen's name, a slash, and one of its tabs, as [Issues](kittui:flow/issues) is, is drawn as a link, and Page says
// where each one stands, for a screen that follows it on a click. A line naming a verb, as `kit packages` does, is
// text: the screen opens no form.
package markdown

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

var (
	// Heading matches a heading's line, its text the first group.
	Heading = regexp.MustCompile(`^#{1,6}\s+(.*?)\s*#*$`)
	// Item matches a list item's line: its indent, its marker, and its text.
	Item = regexp.MustCompile(`^(\s*)([-*+]|\d+[.)])\s+(.*)$`)
	link = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
	// hyperlink matches an OSC 8 hyperlink's sequence, which takes no cell: its target, the first group, opens the link,
	// and none closes it. A word of a link to a screen is drawn between the two, so it wraps as any word does, and Page
	// then takes them out, keeping where the word landed.
	hyperlink = regexp.MustCompile("\x1b\\]8;;([^\x07]*)\x07")
)

// Scheme starts a link to one of kittui's screens.
const Scheme = "kittui:"

// A Link is where a link to one of kittui's screens stands among the lines Page gives: its line, the cell its text
// starts at, and its width in cells; and where it leads, the link's text after kittui:. A link that wraps is a Link
// for each word.
type Link struct {
	Line, Col, Width int
	To               string
}

// Page draws a page's markdown as Render does, and says where each link to one of kittui's screens stands in the
// lines.
func Page(md string, width int) ([]string, []Link) {
	lines := render(md, width)
	var links []Link
	for i, l := range lines {
		var at []Link
		lines[i], at = unmarked(l)
		for _, k := range at {
			k.Line = i
			links = append(links, k)
		}
	}
	return lines, links
}

// unmarked gives a line without its links' markers, and the links they marked, each where its text stands then.
func unmarked(line string) (string, []Link) {
	var out strings.Builder
	var links []Link
	open, prev := -1, 0
	end := func() {
		if open >= 0 {
			links[open].Width = ansi.StringWidth(out.String()) - links[open].Col
			open = -1
		}
	}
	for _, at := range hyperlink.FindAllStringSubmatchIndex(line, -1) {
		out.WriteString(line[prev:at[0]])
		prev = at[1]
		end()
		if to := line[at[2]:at[3]]; to != "" {
			links = append(links, Link{Col: ansi.StringWidth(out.String()), To: to})
			open = len(links) - 1
		}
	}
	out.WriteString(line[prev:])
	end()
	return out.String(), links
}

// marked gives a text with each link to one of kittui's screens in it drawn as a link, each of its words between its
// markers.
func marked(text string) string {
	return link.ReplaceAllStringFunc(text, func(s string) string {
		parts := link.FindStringSubmatch(s)
		to, ok := strings.CutPrefix(parts[2], Scheme)
		if !ok {
			return s
		}
		words := strings.Fields(parts[1])
		for i, w := range words {
			words[i] = "\x1b]8;;" + to + "\x07" + style.Link(w) + "\x1b]8;;\x07"
		}
		return strings.Join(words, " ")
	})
}

// Render draws a page's markdown in lines no wider than the width, in cells, its front matter dropped. Blank lines
// between blocks are kept, one at most, and none leads or trails.
func Render(md string, width int) []string {
	lines, _ := Page(md, width)
	return lines
}

// render draws a page's markdown as Render does, each link to one of kittui's screens between its markers.
func render(md string, width int) []string {
	width = max(width, 10)
	var out, para []string
	prefix, indent := "", ""
	blank := func() {
		if len(out) > 0 && out[len(out)-1] != "" {
			out = append(out, "")
		}
	}
	flush := func() {
		if len(para) > 0 {
			out = append(out, WrapWords(inline(strings.Join(para, " ")), width, prefix, indent)...)
		}
		para, prefix, indent = nil, "", ""
	}
	fenced, comment := false, false
	for _, raw := range strings.Split(withoutFront(strings.ReplaceAll(md, "\r\n", "\n")), "\n") {
		text := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(text, "```") || strings.HasPrefix(text, "~~~"):
			flush()
			if fenced = !fenced; fenced {
				blank()
			}
		case fenced:
			out = append(out, pane.Clip("  "+style.Mark(raw), width))
		case comment || strings.HasPrefix(text, "<!--"):
			flush()
			comment = !strings.Contains(text, "-->")
		case text == "":
			flush()
			blank()
		case Heading.MatchString(text):
			flush()
			blank()
			title := Heading.FindStringSubmatch(text)[1]
			if name, about, ok := strings.Cut(title, " — "); ok && strings.HasPrefix(text, "# ") {
				out = append(out, WrapWords(titled(name, about), width, "", "")...)
			} else {
				for _, l := range WrapWords(strings.Fields(strings.ReplaceAll(title, "`", "")), width, "", "") {
					out = append(out, style.Label(l))
				}
			}
			blank()
		case strings.HasPrefix(text, "|"):
			flush()
			out = append(out, pane.Clip(text, width))
		case Item.MatchString(raw):
			flush()
			parts := Item.FindStringSubmatch(raw)
			marker := parts[2]
			if !strings.ContainsAny(marker[len(marker)-1:], ".)") {
				marker = "•"
			}
			nest := strings.Repeat("  ", len(parts[1])/2)
			prefix = nest + marker + " "
			indent = strings.Repeat(" ", ansi.StringWidth(prefix))
			para = []string{parts[3]}
		default:
			para = append(para, text)
		}
	}
	flush()
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

// titled gives the words of a page's title line, `# <name> — <about>`: its name bold, then the dash and what it says
// of the page, drawn as a paragraph's words are.
func titled(name, about string) []string {
	var words []string
	for _, w := range strings.Fields(strings.ReplaceAll(name, "`", "")) {
		words = append(words, style.Label(w))
	}
	return append(append(words, "—"), inline(about)...)
}

// withoutFront gives a page without the front matter at its top: the lines from a first line of --- to the next.
func withoutFront(md string) string {
	if rest, ok := strings.CutPrefix(md, "---\n"); ok {
		if _, body, found := strings.Cut(rest, "\n---\n"); found {
			return body
		}
	}
	return md
}

// Unlinked gives a text with each link in it as its text, with its URL after it when it is a web address.
func Unlinked(text string) string {
	return link.ReplaceAllStringFunc(text, func(s string) string {
		parts := link.FindStringSubmatch(s)
		if strings.HasPrefix(parts[2], "http://") || strings.HasPrefix(parts[2], "https://") {
			return parts[1] + " (" + parts[2] + ")"
		}
		return parts[1]
	})
}

// inline gives a paragraph's words, each drawn: `code` in the accent, **strong** bold, a link to one of kittui's
// screens as marked gives it, and any other link as its text, as Unlinked gives it. A word is what lies between
// spaces, so a code span with spaces in it wraps as words do.
func inline(text string) []string {
	text = Unlinked(marked(text))
	var words []string
	var word, frag strings.Builder
	code, strong := false, false
	draw := func() {
		s := frag.String()
		frag.Reset()
		switch {
		case s == "":
		case code:
			word.WriteString(style.Mark(s))
		case strong:
			word.WriteString(style.Label(s))
		default:
			word.WriteString(s)
		}
	}
	end := func() {
		draw()
		if word.Len() > 0 {
			words = append(words, word.String())
			word.Reset()
		}
	}
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		switch r := runes[i]; {
		case r == '`':
			draw()
			code = !code
		case !code && r == '*' && i+1 < len(runes) && runes[i+1] == '*':
			draw()
			strong = !strong
			i++
		case r == ' ' || r == '\t':
			end()
		default:
			frag.WriteRune(r)
		}
	}
	end()
	return words
}

// WrapWords lays words out in lines no wider than the width, in cells: the first line after first, the rest after
// indent. A word wider than its line stands alone, clipped.
func WrapWords(words []string, width int, first, indent string) []string {
	var lines []string
	current, lead := "", first
	for _, w := range words {
		switch {
		case current == "":
			current = w
		case ansi.StringWidth(lead)+ansi.StringWidth(current)+1+ansi.StringWidth(w) <= width:
			current += " " + w
		default:
			lines = append(lines, pane.Clip(lead+current, width))
			current, lead = w, indent
		}
	}
	if current != "" {
		lines = append(lines, pane.Clip(lead+current, width))
	}
	return lines
}
