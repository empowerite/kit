// The terms step drawn as a table, internal/table's sheet, as kit flow draws its stages: one row an entry, its fields
// in terms.json, the catalog. Each entry of the page is a `## <term>, <term>` heading, the terms it names, as many as
// share one meaning, comma-separated; the paragraphs under it, its meaning; and each line holding one backticked
// command, its examples, each a command that shows it. Its command, which the rows group by, is what its first example
// is run with, its tool and its verb, as kit flow or git remote. The terms are a list, as an issue's labels are, each a
// pill, so the rows can stand once per term, flattened, and sort by it. The text before the first heading is the
// step's lead, drawn over the table where the pane has room for it. A page with no such heading, as a house package
// may write, is drawn as a page, as every other step's.
package welcome

import (
	_ "embed"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

//go:embed terms.json
var termsJSON []byte

// termsTable is the name of the terms' table in the catalog, which keys the person's presets of it too.
const termsTable = "welcome/terms"

// termsCatalog is terms.json, read; one that does not read is kittui's own fault, which the tests catch before it
// ships.
var termsCatalog = readCatalog(termsJSON)

// readCatalog reads a catalog, as table.ReadCatalog does, and panics on one that does not read.
func readCatalog(data []byte) table.Catalog {
	c, err := table.ReadCatalog(data)
	if err != nil {
		panic(fmt.Sprintf("terms.json does not read: %v", err))
	}
	return c
}

// term is one entry of the page: the terms its heading names, its meaning, the paragraphs under its heading as one
// line, and its examples, the commands that show it, none where the page gives none.
type term struct {
	names    []string
	meaning  string
	examples []string
}

var (
	termHeading = regexp.MustCompile(`^##\s+(.*?)\s*#*$`)
	commandLine = regexp.MustCompile("^`([^`]+)`$")
	fence       = regexp.MustCompile("^(```|~~~)")
)

// parseTerms reads a page's entries, and the text before the first, its lead: a `## ` heading starts an entry, its
// terms the heading's comma-separated names; under it, each line that is one backticked command and nothing else is
// one of its examples, and every other line of text its meaning, a code block's lines among them, without the fence.
// A comment is dropped, and a ## inside one starts nothing. The meaning and the names are plain text, as plainText
// draws them.
func parseTerms(md string) (lead string, terms []term) {
	var before []string
	var meaning [][]string
	comment := false
	for _, raw := range strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n") {
		text := strings.TrimSpace(raw)
		last := len(terms) - 1
		switch {
		case comment || strings.HasPrefix(text, "<!--"):
			comment = !strings.Contains(text, "-->")
		case termHeading.MatchString(text):
			terms = append(terms, term{names: namesOf(plainText(termHeading.FindStringSubmatch(text)[1]))})
			meaning = append(meaning, nil)
		case last < 0:
			before = append(before, raw)
		case commandLine.MatchString(text):
			terms[last].examples = append(terms[last].examples, commandLine.FindStringSubmatch(text)[1])
		case text != "" && !fence.MatchString(text):
			meaning[last] = append(meaning[last], text)
		}
	}
	for k := range terms {
		terms[k].meaning = plainText(strings.Join(meaning[k], " "))
	}
	return strings.TrimSpace(strings.Join(before, "\n")), terms
}

// namesOf gives the terms a heading names, comma-separated, each trimmed, none empty.
func namesOf(heading string) []string {
	var out []string
	for _, n := range strings.Split(heading, ",") {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, n)
		}
	}
	return out
}

// plainText gives markdown's inline text as plain text: a link as its text, with its URL after it when it is a web
// address, code without its backticks, and strong text without its stars.
func plainText(text string) string {
	return strings.NewReplacer("`", "", "**", "").Replace(unlinked(text))
}

// verbOf gives what a command is run with: its tool and its verb, the first word after the tool that is no flag, as
// kit flow for kit flow --stages, or the tool alone; none for no command.
func verbOf(command string) string {
	words := strings.Fields(command)
	if len(words) == 0 {
		return ""
	}
	for _, w := range words[1:] {
		if !strings.HasPrefix(w, "-") {
			return words[0] + " " + w
		}
	}
	return words[0]
}

// termRecords gives the entries as the table's records, one an entry, each keyed by its place and its terms, so two
// entries of one term stay two rows: its command, what its first example is run with; its terms and its examples,
// each a list; and its meaning.
func termRecords(terms []term) []table.Record {
	out := make([]table.Record, len(terms))
	for k, t := range terms {
		first := ""
		if len(t.examples) > 0 {
			first = t.examples[0]
		}
		out[k] = table.Record{Row: grid.Row{Key: strconv.Itoa(k) + " " + strings.Join(t.names, ", ")},
			Values: map[string]table.Value{"command": table.Some(verbOf(first)), "terms": listOf(t.names),
				"meaning": table.Some(t.meaning), "examples": listOf(t.examples)}}
	}
	return out
}

// listOf gives a list's value, its items as they stand, or none for no items.
func listOf(items []string) table.Value {
	if len(items) == 0 {
		return table.Value{}
	}
	return table.Value{Raw: items}
}

// termsColor is the terms' table's color, the accent's, as the steps' bar draws the selected step.
const termsColor = "12"

// gridded says whether the selected step is drawn as a table: the terms, where the page has terms.
func (m Model) gridded() bool { return m.tabs[m.step].gridded }

// takeTerms reads the terms of the terms step's page, when it has some, into the step's table, the person's presets
// of it read from the file the screen keeps them in; a page without terms stays a page.
func (m *Model) takeTerms() {
	for k, t := range m.tabs {
		if t.id != "terms" {
			continue
		}
		lead, terms := parseTerms(t.page)
		if len(terms) == 0 {
			continue
		}
		m.tabs[k].gridded, m.tabs[k].lead = true, lead
		m.terms = table.NewSheet(termsCatalog[termsTable], "terms", termsColor, m.presets)
		m.terms.SetRecords(termRecords(terms))
		m.terms.Settings(m.settings.FoldShared, m.settings.SharedEmoji)
	}
	m.placeTerms()
}

// leadLines gives the lead of the step i drawn as a page draws it, a blank line under it, where the pane has room for
// it and the table's control line, header, and a few rows; none where it does not.
func (m Model) leadLines(i int) []string {
	if m.tabs[i].lead == "" {
		return nil
	}
	var out []string
	for _, l := range render(m.tabs[i].lead, m.width-len(lead)-1) {
		out = append(out, pane.Clip(strings.TrimRight(lead+l, " "), m.width-1))
	}
	if len(out)+1+termsRoom > m.paneHeight() {
		return nil
	}
	return append(out, "")
}

// termsRoom is the lines the terms' table keeps for itself under the lead: its control line and the blank line under
// it, its header and its rule, and three rows.
const termsRoom = 7

// placeTerms places the terms' table in the pane, under the lead where it shows.
func (m *Model) placeTerms() {
	if i := indexOf(m.tabs, "terms"); i >= 0 && m.tabs[i].gridded {
		n := len(m.leadLines(i))
		m.terms.Place(paneTop+n, m.paneHeight()-n, m.width, m.height)
	}
}

// termsPane gives the terms step's pane: the lead, then the table's lines, each clipped to the screen.
func (m Model) termsPane() []string {
	out := m.leadLines(m.step)
	for _, l := range m.terms.View() {
		out = append(out, pane.Clip(l, m.width-1))
	}
	return out
}

// termsNotice gives the keys bar's line in place of the keys while the terms' table says what it did, or why not; none
// while it says nothing, or another step is selected.
func (m Model) termsNotice() string {
	n := m.terms.Notice()
	switch {
	case !m.gridded() || n.Text == "":
		return ""
	case n.Done:
		return " " + pane.Clip(n.Text, m.width-1)
	}
	return " " + style.Refusal(pane.Clip(n.Text, m.width-1))
}
