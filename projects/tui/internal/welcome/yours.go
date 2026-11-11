// The make-it-yours step's table, drawn as sections.go draws a step's: one row a part of kit that teams change first,
// its fields in yours.json, the catalog. Each part is a section, its heading, `## <part>`, a title, whole; its text,
// what the part is about; and its commands, the commands that show it. Its settings are the kit.yml keys its text
// names, each backticked with its colon, as `packages:`, a list drawn as pills, as an issue's labels are; and its
// command, which the rows group by, is what its first command is run with, its tool and its verb, as kit packages. Its
// order, the section's place on the page, is hidden, and the rows stand in it, as the page gives them.
package welcome

import (
	_ "embed"
	"regexp"
	"slices"
	"strconv"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

//go:embed yours.json
var yoursJSON []byte

// yoursTable is the name of the parts' table in the catalog, which keys the person's presets of it too.
const yoursTable = "welcome/make-it-yours"

// yoursCatalog is yours.json, read.
var yoursCatalog = readCatalog(yoursJSON)

// settingKey is a kit.yml key as a page names it: backticked, with its colon, as `packages:`.
var settingKey = regexp.MustCompile("`([A-Za-z][A-Za-z0-9_-]*):`")

// settingsOf gives the kit.yml keys a section's text names, each once, in the order it names them.
func settingsOf(text string) []string {
	var out []string
	for _, m := range settingKey.FindAllStringSubmatch(text, -1) {
		if !slices.Contains(out, m[1]) {
			out = append(out, m[1])
		}
	}
	return out
}

// partRecords gives the sections as the table's records, one a part, each keyed by its place and its heading: its
// order, from 1; its part, the heading; its settings and its commands, each a list; what it is about, its text as
// plain text; and its command, what its first command is run with.
func partRecords(sections []section) []table.Record {
	out := make([]table.Record, len(sections))
	for k, s := range sections {
		part := plainText(s.heading)
		out[k] = table.Record{Row: grid.Row{Key: strconv.Itoa(k) + " " + part},
			Values: map[string]table.Value{"order": table.One(strconv.Itoa(k + 1)), "part": table.Some(part),
				"settings": listOf(settingsOf(s.text)), "about": table.Some(plainText(s.text)),
				"commands": listOf(s.commands), "command": table.Some(verbOf(firstOf(s.commands)))}}
	}
	return out
}
