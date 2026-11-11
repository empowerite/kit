// The terms step's table, drawn as sections.go draws a step's: one row an entry, its fields in terms.json, the
// catalog. Each entry of the page is a section, its heading `### <term>, <term>` under a `## <topic>`, or, on a page
// with no topics, `## <term>, <term>`: the terms it names, as many as share one meaning, comma-separated; its text, its
// meaning; and its commands, its examples, each a command that shows it, a list, which By command groups by, an entry
// under each of its commands. Its topic, which the rows group by, stands in the page's order, and so do the rows under
// it, by their order, hidden. The terms stand first, a list, as an issue's labels are, each a pill, so the rows can
// stand once per term, flattened, and sort by it.
package welcome

import (
	_ "embed"
	"strconv"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

//go:embed terms.json
var termsJSON []byte

// termsTable is the name of the terms' table in the catalog, which keys the person's presets of it too.
const termsTable = "welcome/terms"

// termsCatalog is terms.json, read.
var termsCatalog = readCatalog(termsJSON)

// term is one entry of the page: the terms its heading names, its meaning, the paragraphs under its heading as one
// line, its examples, the commands that show it, none where the page gives none, and its topic, none on a page with
// no topics.
type term struct {
	names    []string
	meaning  string
	examples []string
	topic    string
}

// parseTerms reads a page's entries, and the text before the first, its lead, as pageSections reads its sections:
// an entry's terms are its heading's comma-separated names, and its meaning and its names are plain text, as
// plainText draws them.
func parseTerms(md string) (lead string, terms []term) {
	lead, sections := pageSections(md)
	return lead, termsOf(sections)
}

// termsOf gives the entries the sections are, none for no sections.
func termsOf(sections []section) []term {
	var out []term
	for _, s := range sections {
		out = append(out, term{names: namesOf(plainText(s.heading)), meaning: plainText(s.text), examples: s.commands,
			topic: plainText(s.topic)})
	}
	return out
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

// termSections gives the records of the entries the sections are, as termRecords gives them.
func termSections(sections []section) []table.Record { return termRecords(termsOf(sections)) }

// termRecords gives the entries as the table's records, one an entry, each keyed by its place and its terms, so two
// entries of one term stay two rows: its topic, none where the page has none; its order, from 1; its terms and its
// commands, its examples as written, each a list; and its meaning.
func termRecords(terms []term) []table.Record {
	out := make([]table.Record, len(terms))
	for k, t := range terms {
		out[k] = table.Record{Row: grid.Row{Key: strconv.Itoa(k) + " " + strings.Join(t.names, ", ")},
			Values: map[string]table.Value{"topic": table.Some(t.topic), "order": table.One(strconv.Itoa(k + 1)),
				"terms": listOf(t.names), "meaning": table.Some(t.meaning), "commands": listOf(t.examples)}}
	}
	return out
}
