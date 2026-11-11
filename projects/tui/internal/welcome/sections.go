// A step drawn as a table, internal/table's sheet, as kit flow draws its stages: its page read as sections, each a
// record, one row, and its fields in the step's catalog, a JSON file beside the code. Each section of a page is a
// `## ` heading; the lines under it that hold one backticked command and nothing else, its commands; and the rest of
// its text, its paragraphs and its list items, its meaning, one line. The text before the first heading is the step's
// lead, drawn over the table where the pane has room for it. sheetSteps names the steps drawn so, each with its
// catalog's table and the records its sections make; a page with no such heading, as a house package may write, is
// drawn as a page, as every other step's.
package welcome

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// readCatalog reads a catalog, as table.ReadCatalog does, and panics on one that does not read: a catalog is
// kittui's own, and one that does not read is kittui's fault, which the tests catch before it ships.
func readCatalog(data []byte) table.Catalog {
	c, err := table.ReadCatalog(data)
	if err != nil {
		panic(fmt.Sprintf("a catalog of kit welcome's does not read: %v", err))
	}
	return c
}

// section is one section of a page: its heading's text, as the page writes it; its text, the lines under the heading
// joined into one, as markdown, a list item's without its marker; and its commands, none where the page gives none.
type section struct {
	heading  string
	text     string
	commands []string
}

var (
	sectionHeading = regexp.MustCompile(`^##\s+(.*?)\s*#*$`)
	commandLine    = regexp.MustCompile("^`([^`]+)`$")
	fence          = regexp.MustCompile("^(```|~~~)")
	autolink       = regexp.MustCompile(`<((?:https?|mailto):[^>\s]+)>`)
)

// parseSections reads a page's sections, and the text before the first, its lead: a `## ` heading starts a section;
// under it, each line that is one backticked command and nothing else is one of its commands, and every other line of
// text its text, a list item's without its marker and a code block's without the fence. A comment is dropped, and a
// ## inside one starts nothing.
func parseSections(md string) (lead string, sections []section) {
	var before []string
	var text [][]string
	comment := false
	for _, raw := range strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		last := len(sections) - 1
		switch {
		case comment || strings.HasPrefix(line, "<!--"):
			comment = !strings.Contains(line, "-->")
		case sectionHeading.MatchString(line):
			sections = append(sections, section{heading: sectionHeading.FindStringSubmatch(line)[1]})
			text = append(text, nil)
		case last < 0:
			before = append(before, raw)
		case commandLine.MatchString(line):
			sections[last].commands = append(sections[last].commands, commandLine.FindStringSubmatch(line)[1])
		case line != "" && !fence.MatchString(line):
			if m := item.FindStringSubmatch(line); m != nil {
				line = m[3]
			}
			text[last] = append(text[last], line)
		}
	}
	for k := range sections {
		sections[k].text = strings.Join(text[k], " ")
	}
	return strings.TrimSpace(strings.Join(before, "\n")), sections
}

// plainText gives markdown's inline text as plain text: a link as its text, with its URL after it when it is a web
// address, an autolink as its address, code without its backticks, and strong text without its stars.
func plainText(text string) string {
	return strings.NewReplacer("`", "", "**", "").Replace(autolink.ReplaceAllString(unlinked(text), "$1"))
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

// firstOf gives a list's first item, or none for no items.
func firstOf(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[0]
}

// listOf gives a list's value, its items as they stand, or none for no items.
func listOf(items []string) table.Value {
	if len(items) == 0 {
		return table.Value{}
	}
	return table.Value{Raw: items}
}

// sheetStep is a step drawn as a table where its page has sections: the step's id; the sheet's name, which titles its
// dialogs and is what the keys bar's down enters; its catalog's table; and the records its sections make.
type sheetStep struct {
	id, name string
	schema   *table.Schema
	records  func([]section) []table.Record
}

// sheetSteps are the steps drawn as tables: the terms, terms.go's, and the parts that make kit yours, yours.go's.
var sheetSteps = []sheetStep{
	{id: "terms", name: "terms", schema: termsCatalog[termsTable], records: termSections},
	{id: "make-it-yours", name: "parts", schema: yoursCatalog[yoursTable], records: partRecords},
}

// sheetColor is a step's table's color, the accent's, as the steps' bar draws the selected step.
const sheetColor = "12"

// gridded says whether the selected step is drawn as a table.
func (m Model) gridded() bool { return m.tabs[m.step].sheet != nil }

// sheet gives the selected step's table, none where the step is drawn as a page.
func (m Model) sheet() *table.Sheet { return m.tabs[m.step].sheet }

// takeSheets reads the sections of each page sheetSteps names, when it has some, into the step's table, the person's
// presets of it read from the file the screen keeps them in; a page without sections stays a page.
func (m *Model) takeSheets() {
	for k, t := range m.tabs {
		i := slices.IndexFunc(sheetSteps, func(s sheetStep) bool { return s.id == t.id })
		if i < 0 {
			continue
		}
		s := sheetSteps[i]
		lead, sections := parseSections(t.page)
		if len(sections) == 0 {
			continue
		}
		sheet := table.NewSheet(s.schema, s.name, sheetColor, m.presets)
		sheet.SetRecords(s.records(sections))
		sheet.Settings(m.settings.FoldShared, m.settings.SharedEmoji)
		m.tabs[k].sheet, m.tabs[k].lead = &sheet, lead
	}
	m.placeSheets()
}

// eachSheet calls do with every step's table.
func (m Model) eachSheet(do func(*table.Sheet)) {
	for _, t := range m.tabs {
		if t.sheet != nil {
			do(t.sheet)
		}
	}
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
	if len(out)+1+sheetRoom > m.paneHeight() {
		return nil
	}
	return append(out, "")
}

// sheetRoom is the lines a step's table keeps for itself under the lead: its control line and the blank line under
// it, its header and its rule, and three rows.
const sheetRoom = 7

// placeSheets places each step's table in the pane, under its lead where it shows.
func (m *Model) placeSheets() {
	for i, t := range m.tabs {
		if t.sheet != nil {
			n := len(m.leadLines(i))
			t.sheet.Place(paneTop+n, m.paneHeight()-n, m.width, m.height)
		}
	}
}

// sheetPane gives the pane of the selected step drawn as a table: the lead, then the table's lines, each clipped to
// the screen.
func (m Model) sheetPane() []string {
	out := m.leadLines(m.step)
	for _, l := range m.sheet().View() {
		out = append(out, pane.Clip(l, m.width-1))
	}
	return out
}

// sheetNotice gives the keys bar's line in place of the keys while the selected step's table says what it did, or why
// not; none while it says nothing, or the step is drawn as a page.
func (m Model) sheetNotice() string {
	if !m.gridded() {
		return ""
	}
	n := m.sheet().Notice()
	switch {
	case n.Text == "":
		return ""
	case n.Done:
		return " " + pane.Clip(n.Text, m.width-1)
	}
	return " " + style.Refusal(pane.Clip(n.Text, m.width-1))
}
