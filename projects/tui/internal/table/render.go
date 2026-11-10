// The renderers that draw a field down its column. Those every table has are here; a screen registers its own by name,
// so its catalog names them as it names these, and the field draws by whichever its preset pins, or its first.
package table

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Renderer draws a field down its column: from the values the rows hold in it, one a row, the cell each row shows.
// It draws nothing for a value that holds none, styles a value its drawing gives a style of its own, and leaves the
// rest, the field's dim, alignment, link, and sort, to draw.
type Renderer func(f Field, vs []Value) []grid.Cell

// Each gives a renderer that draws every row's value alone, as texts, without regard to the others.
func Each(draw func(f Field, v Value) []string) Renderer {
	return func(f Field, vs []Value) []grid.Cell {
		out := make([]grid.Cell, len(vs))
		for k, v := range vs {
			if len(v.Raw) > 0 {
				out[k].Values = draw(f, v)
			}
		}
		return out
	}
}

// Mapped gives each raw text read as seconds, drawn.
func Mapped(raw []string, draw func(int64) string) []string {
	var out []string
	for _, r := range raw {
		secs, _ := strconv.ParseInt(r, 10, 64)
		out = append(out, draw(secs))
	}
	return out
}

// GlyphOf gives the glyph of one of a field's values: an enum's known value's own, else the enum's for a value it does
// not know; a field that is not an enum draws its own before every value.
func (f Field) GlyphOf(raw string) string {
	if f.Type != "enum" {
		return f.Glyph
	}
	if k := slices.IndexFunc(f.Values, func(e EnumValue) bool { return e.Value == raw }); k >= 0 {
		return f.Values[k].Glyph
	}
	return f.Other
}

// Glyphed gives each raw text after its glyph, with the text when text says so, or the glyph alone; a text without a
// glyph stands alone.
func (f Field) Glyphed(raw []string, text bool) []string {
	var out []string
	for _, r := range raw {
		g := f.GlyphOf(r)
		switch {
		case g == "":
			out = append(out, r)
		case text:
			out = append(out, g+" "+r)
		default:
			out = append(out, g)
		}
	}
	return out
}

// DrawnBy gives the rendering a field is drawn by: the one pinned, when it is one of the field's, else its first.
func (f Field) DrawnBy(pinned string) string {
	if slices.Contains(f.Renderings, pinned) {
		return pinned
	}
	return f.Renderings[0]
}

// Draw gives a field's cells from the values the rows hold in it, as the rendering pinned draws them, when it is one of
// the field's, else its first: each dim where the field is, the field's alignment, each row's link, and what each
// sorts, groups, and filters by, its raw values; what a field holding no value draws instead, dim. A link leads from
// the glyph, so a field drawn without its glyph, which another of its renderings draws, has none: its text is no
// button. A field none of whose renderings draws a glyph, as a reference, leads from its text. A field of labels, or a
// list with a glyph for many, as people, says how many a row has, dim, in its column, where it has more than one,
// after that glyph where the rendering draws glyphs, and draws the one where it has one; a group's heading says the
// one label its rows share, as it draws it.
func (f Field) Draw(vs []Value, pinned string) []grid.Cell {
	rendering := f.DrawnBy(pinned)
	links := glyphed[rendering] || !slices.ContainsFunc(f.Renderings, func(r string) bool { return glyphed[r] })
	out := renderers[rendering](f, vs)
	for k, v := range vs {
		c := &out[k]
		if f.Dim {
			for j, s := range c.Values {
				if s != "" {
					c.Values[j] = style.Off(s)
				}
			}
		}
		if len(v.Raw) == 0 && v.Else != "" {
			c.Values = []string{style.Off(v.Else)}
		}
		c.Right, c.Wrap, c.Raw, c.Hang = f.Right, f.Wrap, v.Raw, glyphed[rendering]
		if f.Type == "label" && len(c.Values) > 0 {
			c.Head = c.Values[0]
		}
		if (f.Type == "label" || f.Many != "") && len(c.Values) > 1 {
			c.Summary = strconv.Itoa(len(c.Values))
			if f.Many != "" && glyphed[rendering] {
				c.Summary = f.Many + " " + c.Summary
			}
			c.Summary = style.Off(c.Summary)
		}
		if links {
			c.Link = v.Link
		}
	}
	return out
}

// renderers gives each renderer by the name a catalog gives it: those every table has, and those a screen registers.
var renderers = map[string]Renderer{
	"text":       Each(func(f Field, v Value) []string { return slices.Clone(v.Raw) }),
	"glyph+text": Each(func(f Field, v Value) []string { return f.Glyphed(v.Raw, true) }),
	"glyph":      Each(func(f Field, v Value) []string { return f.Glyphed(v.Raw, false) }),
	"age":        Each(func(f Field, v Value) []string { return Mapped(v.Raw, Age) }),
	"took":       Each(func(f Field, v Value) []string { return Mapped(v.Raw, func(s int64) string { return Took(&s) }) }),
	"pills":      Each(Pills),
}

// glyphed names the renderers that draw a glyph before a value, the field's own or its value's.
var glyphed = map[string]bool{"glyph+text": true, "glyph": true}

// Register adds a screen's renderer by the name its catalog gives it, and says whether it draws a glyph before a value.
// A name registered twice is the program's own fault, which panics before any table is drawn.
func Register(name string, r Renderer, glyph bool) {
	if _, ok := renderers[name]; ok {
		panic(fmt.Sprintf("the renderer %q is registered twice", name))
	}
	renderers[name] = r
	if glyph {
		glyphed[name] = true
	}
}

// Registered says whether a renderer of the name is registered, as every name a catalog gives must be.
func Registered(name string) bool { return renderers[name] != nil }

// DrawsGlyph says whether the renderer of the name draws a glyph before a value.
func DrawsGlyph(name string) bool { return glyphed[name] }
