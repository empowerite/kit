// The customize step's table, drawn as sections.go draws a step's: one row a part of kit that teams change first, its
// fields in customize.json, the catalog. Each part is a section, its heading, `## <part>`, a title, whole; its text,
// what the part is about; and its commands, the commands that show it. Its settings are the kit.yml keys its text
// names, each backticked with its colon, as `packages:`, a list drawn as pills, as an issue's labels are; and its
// command, which the rows group by, is what its first command is run with, its tool and its verb, as kit packages. Its
// order, the section's place on the page, is hidden, and the rows stand in it, as the page gives them.
package welcome

import (
	_ "embed"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

//go:embed customize.json
var customizeJSON []byte

// customizeTable is the name of the parts' table in the catalog, which keys the person's presets of it too.
const customizeTable = "welcome/customize"

// customizeCatalog is customize.json, read.
var customizeCatalog = readCatalog(customizeJSON)

// whereLine is a part's line of where to change it: → and its links, each [text](target), a · between them.
var whereLine = regexp.MustCompile(`→(\s*·?\s*\[[^\]]+\]\([^)\s]+\))+`)

// mdLink is one markdown link, its text and its target.
var mdLink = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)

// whereOf gives a part's where, the links of its line of where to change it, each by its text leading to its target,
// none where it has no such line; and its text without that line.
func whereOf(text string) (table.Value, string) {
	line := whereLine.FindString(text)
	if line == "" {
		return table.Value{}, text
	}
	var v table.Value
	for _, m := range mdLink.FindAllStringSubmatch(line, -1) {
		v.Raw, v.Links = append(v.Raw, m[1]), append(v.Links, m[2])
	}
	return v, strings.TrimSpace(strings.Replace(text, line, "", 1))
}

// rooted gives the records with each link that is no page, as kit.yml, a path under the folder the welcome reads.
func rooted(records []table.Record, dir string) []table.Record {
	for _, r := range records {
		for id, v := range r.Values {
			for k, l := range v.Links {
				if !strings.Contains(l, "://") && !filepath.IsAbs(l) {
					v.Links[k] = filepath.Join(dir, l)
				}
			}
			r.Values[id] = v
		}
	}
	return records
}

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
// order, from 1; its part, the heading; its settings and its commands, each a list; its where, the file and the doc
// its line of where to change it leads to; what it is about, its text as plain text without that line; and its
// command, what its first command is run with.
func partRecords(sections []section) []table.Record {
	out := make([]table.Record, len(sections))
	for k, s := range sections {
		part := plainText(s.heading)
		where, text := whereOf(s.text)
		out[k] = table.Record{Row: grid.Row{Key: strconv.Itoa(k) + " " + part},
			Values: map[string]table.Value{"order": table.One(strconv.Itoa(k + 1)), "part": table.Some(part),
				"settings": listOf(settingsOf(text)), "about": table.Some(plainText(text)), "where": where,
				"commands": listOf(s.commands), "command": table.Some(verbOf(firstOf(s.commands)))}}
	}
	return out
}
