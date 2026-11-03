// A page's markdown drawn as text: headings bold, inline code in the accent, list items kept with a hanging indent,
// and paragraphs wrapped to the width. It is no markdown library: a code block keeps its lines, a table its rows, a
// comment is dropped, and a link keeps its text, with its URL when it is a web address. A line naming a verb, as
// `kit packages` does, is text: the screen opens no form.
package welcome

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

var (
	heading = regexp.MustCompile(`^#{1,6}\s+(.*?)\s*#*$`)
	item    = regexp.MustCompile(`^(\s*)([-*+]|\d+[.)])\s+(.*)$`)
	link    = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
)

// render draws a page's markdown in lines no wider than the width, in cells. Blank lines between blocks are kept,
// one at most, and none leads or trails.
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
			out = append(out, wrapWords(inline(strings.Join(para, " ")), width, prefix, indent)...)
		}
		para, prefix, indent = nil, "", ""
	}
	fenced, comment := false, false
	for _, raw := range strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n") {
		text := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(text, "```") || strings.HasPrefix(text, "~~~"):
			flush()
			if fenced = !fenced; fenced {
				blank()
			}
		case fenced:
			out = append(out, clip("  "+style.Mark(raw), width))
		case comment || strings.HasPrefix(text, "<!--"):
			flush()
			comment = !strings.Contains(text, "-->")
		case text == "":
			flush()
			blank()
		case heading.MatchString(text):
			flush()
			blank()
			title := strings.ReplaceAll(heading.FindStringSubmatch(text)[1], "`", "")
			for _, l := range wrapWords(strings.Fields(title), width, "", "") {
				out = append(out, style.Label(l))
			}
			blank()
		case strings.HasPrefix(text, "|"):
			flush()
			out = append(out, clip(text, width))
		case item.MatchString(raw):
			flush()
			parts := item.FindStringSubmatch(raw)
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

// inline gives a paragraph's words, each drawn: `code` in the accent, **strong** bold, and a link as its text. A
// word is what lies between spaces, so a code span with spaces in it wraps as words do.
func inline(text string) []string {
	text = link.ReplaceAllStringFunc(text, func(s string) string {
		parts := link.FindStringSubmatch(s)
		if strings.HasPrefix(parts[2], "http://") || strings.HasPrefix(parts[2], "https://") {
			return parts[1] + " (" + parts[2] + ")"
		}
		return parts[1]
	})
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

// wrapWords lays words out in lines no wider than the width, in cells: the first line after first, the rest after
// indent. A word wider than its line stands alone, clipped.
func wrapWords(words []string, width int, first, indent string) []string {
	var lines []string
	current, lead := "", first
	for _, w := range words {
		switch {
		case current == "":
			current = w
		case ansi.StringWidth(lead)+ansi.StringWidth(current)+1+ansi.StringWidth(w) <= width:
			current += " " + w
		default:
			lines = append(lines, clip(lead+current, width))
			current, lead = w, indent
		}
	}
	if current != "" {
		lines = append(lines, clip(lead+current, width))
	}
	return lines
}

// clip shortens a text to n cells, with … at the cut.
func clip(s string, n int) string {
	if n < 1 {
		return ""
	}
	return ansi.Truncate(s, n, "…")
}
