// The values a row holds in a field, and the kinds of value a field can hold.
package table

import "strconv"

// ValueType is a kind of value a field holds, and whether a filter chooses among its values, as among an enum's known
// values and among people.
type ValueType struct {
	Filters bool
}

// Types gives each kind of value a field can hold, by the name the catalog gives it: a reference, as an issue's #45; a
// text; an enum, one of known values; a person's login; a time, in seconds; a number; and a forge's labels, which a
// row's column counts and its open row draws, each a pill.
var Types = map[string]ValueType{
	"ref":    {},
	"text":   {},
	"enum":   {Filters: true},
	"person": {Filters: true},
	"time":   {},
	"number": {},
	"label":  {Filters: true},
}

// Value is what a row holds in one field. Raw is what the row sorts, groups, and filters by, unstyled, one a Value, as
// a person field's logins, or none: a time's seconds, a number's digits, a reference as its tracker gives it. Of is the
// record, or the part of it, a renderer draws more from than the Value, as the checks' counts beside their state. Link
// is where the glyph the field draws leads; Else is what the field draws, dim, while it holds no Value, as … while
// changes loads.
type Value struct {
	Raw  []string
	Of   any
	Link string
	Else string
}

// One gives a value of One raw text.
func One(raw string) Value { return Value{Raw: []string{raw}} }

// Some gives a value of one raw text, or none when it is empty.
func Some(raw string) Value {
	if raw == "" {
		return Value{}
	}
	return One(raw)
}

// Seconds gives the value of a time, in Seconds, never fewer than none; nothing for a time that is missing.
func Seconds(secs *int64) Value {
	if secs == nil {
		return Value{}
	}
	return One(strconv.FormatInt(max(*secs, 0), 10))
}
