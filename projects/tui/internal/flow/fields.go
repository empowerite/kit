// The field catalog, fields.json: for each stage, the kit command that prints its rows, the words it takes beside its
// own, whether its verb's --stages and --marks name it, where its records are in what the command prints, the paths of
// each record's key, the line it shows empty, its fields, and its own presets. A field is declared once: a stable id,
// which presets and the code use, and a label, which only the header shows, with, for a field that holds a list, the
// name of one of its values, which sorts by size until the rows stand once per value, then by value; the type of value
// it holds; whether its rows can sort, group, and be filtered by it; an enum's known values, each with its glyph; its
// renderings, each a renderer by name, in order of preference; the record path its value is read from; and the
// placement it takes where a preset says none. The catalog is the data's description, read as internal/table reads
// one; the value types and the renderers, its behavior, are Go's: table's own, and kit flow's, registered here by
// name, and a test fails on a name the catalog gives that is not registered.
package flow

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// caller is the person a person's filter names by the caller's login.
const caller = grid.Caller

//go:embed fields.json
var fieldsJSON []byte

// catalog is fields.json, each stage's entry by the stage's name. A stage it does not name kittui does not draw.
var catalog = readCatalog(fieldsJSON)

// readCatalog reads the catalog, as table.ReadCatalog does; one that does not read is kittui's own fault, which the
// tests catch before it ships.
func readCatalog(data []byte) table.Catalog {
	c, err := table.ReadCatalog(data)
	if err != nil {
		panic(fmt.Sprintf("fields.json does not read: %v", err))
	}
	return c
}

// fieldsOf gives a stage's fields, in the order its columns stand, as table.Schema's Of says; none for a stage without
// columns.
func fieldsOf(stage string) []table.Field { return catalog[stage].Of() }

// columnsOf gives the names the panel knows a stage's columns by, in the order they stand, as table.Schema's Columns
// says.
func columnsOf(stage string) []string { return catalog[stage].Columns() }

// sortableOf names a stage's columns its rows can sort by, as table.Schema's Sortable says.
func sortableOf(stage string) map[string]bool { return catalog[stage].Sortable() }

// onesOf names one value of each of a stage's fields that holds a list, as table.Schema's Ones says.
func onesOf(stage string) map[string]string { return catalog[stage].Ones() }

// groupableOf names a stage's columns its rows can group by, as table.Schema's Groupable says.
func groupableOf(stage string) map[string]bool { return catalog[stage].Groupable() }

// ownerOf gives the id of the person field that names the people a stage's rows belong to, as table.Schema's Owner
// says; none for a stage whose rows belong to nobody.
func ownerOf(stage string) string { return catalog[stage].Owner() }

// The renderers of kit flow's own values, registered with the table's by the names fields.json gives them, and whether
// each draws a glyph before a value.
func init() {
	for name, r := range map[string]struct {
		draw  table.Renderer
		glyph bool
	}{
		"glyph+age": {pushedAges, true},
		"checkout":  {table.Each(checkoutName), true},
		"name":      {table.Each(plainName), false},
		"counts":    {aheadBehind, false},
		"checks":    {checksCells, true},
		"threads":   {table.Each(threadsText), true},
		"where":     {whereCells, true},
		"member":    {table.Each(memberName), false},
		"path":      {pathCells, false},
		"worktree":  {table.Each(worktreeNames), true},
	} {
		table.Register(name, r.draw, r.glyph)
	}
}

// pushedAges draws when a branch was last pushed: the field's glyph and the age, right-aligned to the widest age in
// the column, so the glyphs stand in a line.
func pushedAges(f table.Field, vs []table.Value) []cell {
	widest := 0
	for _, v := range vs {
		for _, a := range table.Mapped(v.Raw, table.Age) {
			widest = max(widest, len(a))
		}
	}
	return table.Each(func(f table.Field, v table.Value) []string {
		return table.Mapped(v.Raw, func(s int64) string { return f.Glyph + " " + fmt.Sprintf("%*s", widest, table.Age(s)) })
	})(f, vs)
}

// checkoutName draws a checkout's name after the field's glyph, and, for a checkout on no branch, that it is detached.
func checkoutName(f table.Field, v table.Value) []string {
	label := f.Glyph + " " + v.Raw[0]
	if c, ok := v.Of.(checkout); ok && c.Detached {
		label += " " + Detached + " detached"
	}
	return []string{label}
}

// aheadBehind draws a checkout's commits ahead and behind, as counts does, ahead right-aligned to the widest in the
// column.
func aheadBehind(f table.Field, vs []table.Value) []cell {
	widest := 0
	for _, v := range vs {
		if c, ok := v.Of.(checkout); ok {
			widest = max(widest, len("↑"+strconv.Itoa(c.Ahead)))
		}
	}
	return table.Each(func(f table.Field, v table.Value) []string { return []string{counts(v.Of.(checkout), widest)} })(f, vs)
}

// checksCells draws a pull request's checks, as their cell does; a heading over rows grouped by them says their state
// after its glyph, as the counts and the time differ from row to row.
func checksCells(f table.Field, vs []table.Value) []cell {
	out := make([]cell, len(vs))
	for k, v := range vs {
		c, ok := v.Of.(*checks)
		if len(v.Raw) == 0 || !ok {
			continue
		}
		out[k] = text(c.cell())
		out[k].Head = f.Glyphed(v.Raw, true)[0]
	}
	return out
}

// memberName draws a row's member by its short name, as its value's Of holds it.
func memberName(f table.Field, v table.Value) []string {
	if name, ok := v.Of.(string); ok {
		return []string{name}
	}
	return v.Raw
}

// plainName draws a checkout's name as checkoutName does, without the field's glyph.
func plainName(f table.Field, v table.Value) []string {
	return []string{strings.TrimPrefix(checkoutName(f, v)[0], f.Glyph+" ")}
}

// threadsText draws a pull request's review threads unresolved after the field's glyph, or nothing for none.
func threadsText(f table.Field, v table.Value) []string {
	if n, _ := strconv.Atoi(v.Raw[0]); n > 0 {
		return []string{fmt.Sprintf("%s %d unresolved", f.Glyph, n)}
	}
	return nil
}

// whereCells draws where the work of an issue underway is, as its value's Of says: here, after the desk's glyph or a
// worktree's, for a checkout on this desk; local for a branch this desk has without a checkout; else the pull
// request's author, dim.
func whereCells(f table.Field, vs []table.Value) []cell {
	out := make([]cell, len(vs))
	for k, v := range vs {
		if len(v.Raw) == 0 {
			continue
		}
		switch v.Of {
		case "desk":
			out[k] = text(Desk + " here")
		case "here":
			out[k] = text(Branch + " here")
		case "local":
			out[k] = text(Sprout + " local")
		default:
			out[k] = dim(Owner + " " + v.Raw[0])
		}
	}
	return out
}
