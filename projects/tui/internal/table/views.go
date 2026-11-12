// A table's views: its own, the catalog's presets, which never change, then the person's, which they save, rename, and
// delete. The rules of which the table is set as, and of what a name can be saved or renamed as, are here; what the
// screen says of them, and where it keeps them, are the screen's.
package table

import (
	"errors"
	"reflect"
	"slices"
	"strconv"

	"github.com/empowerite/kit/projects/tui/internal/grid"
)

// Default is the name of the one view of its own a table has when its catalog gives it none, which it opens on.
const Default = "Default"

// The refusals of a name to save or rename a view as: none, one of the table's own, or another of the person's.
var (
	ErrUnnamed = errors.New("a preset needs a name")
	ErrBuiltin = errors.New("the name is the table's own")
	ErrTaken   = errors.New("the name is a preset already")
)

// Views is a table's views: its fields, its own views, the one it opens on the first, the person's, and the names of
// those the person starred, in their order; the team's, checked into the workspace, and whether the table keeps them,
// as a screen in a clone kit trusts does.
type Views struct {
	Fields  []Field
	Own     []Preset
	Saved   []Preset
	Starred []string
	Team    []Preset
	Shares  bool
}

// Favorites gives the names of the views the person starred, in their order, but those that name a view since gone,
// which stay hidden until the person's presets are saved, and are dropped then.
func (v Views) Favorites() []string {
	var out []string
	for _, name := range v.Starred {
		if slices.ContainsFunc(v.All(), func(p Preset) bool { return p.Name == name }) && !slices.Contains(out, name) {
			out = append(out, name)
		}
	}
	return out
}

// Favorite says whether the person starred the view of the name, and it is there.
func (v Views) Favorite(name string) bool { return slices.Contains(v.Favorites(), name) }

// Star gives the names of the views the person starred with the view of the name starred, last, or no longer, those
// since gone dropped.
func (v Views) Star(name string) []string {
	if v.Favorite(name) {
		return slices.DeleteFunc(v.Favorites(), func(n string) bool { return n == name })
	}
	return append(v.Favorites(), name)
}

// Favored says what favorites as given do to a view: starred where it was not a favorite, unstarred where it no longer
// is, and moved where it stays one.
func Favored(v Views, name string, starred []string) string {
	switch now := slices.Contains(starred, name); {
	case now && !v.Favorite(name):
		return "starred"
	case !now:
		return "unstarred"
	}
	return "moved"
}

// Renamed gives the names of the views the person starred, the view renamed from one name to another starred by its new
// name in its place.
func (v Views) Renamed(from, to string) []string {
	out := slices.Clone(v.Starred)
	for k, name := range out {
		if name == from {
			out[k] = to
		}
	}
	return out
}

// All gives the views, the table's own first, then the team's, then the person's.
func (v Views) All() []Preset { return append(append(slices.Clone(v.Own), v.Team...), v.Saved...) }

// Shared gives the team's view of the name, and whether there is one.
func (v Views) Shared(name string) (Preset, bool) {
	if k := slices.IndexFunc(v.Team, func(p Preset) bool { return p.Name == name }); k >= 0 {
		return v.Team[k], true
	}
	return Preset{}, false
}

// teams says whether a name is a team view's, but the one named from: the team's files go by their names' slugs, so a
// name of the same slug is one.
func (v Views) teams(name, from string) bool {
	return slices.ContainsFunc(v.Team, func(p Preset) bool { return p.Name != from && Slug(p.Name) == Slug(name) })
}

// ShareAs gives the person's view of the name as the team's, to be kept in the team's folder, and the person's views
// without it. A name the team has already, by its slug, is refused.
func (v Views) ShareAs(name string) (Preset, []Preset, error) {
	p, ok := v.Mine(name)
	switch {
	case !ok:
		return Preset{}, nil, ErrUnnamed
	case v.teams(name, ""):
		return Preset{}, nil, ErrTaken
	}
	return p, v.Delete(name), nil
}

// CopyOf gives the person's views with a copy of the team's view of the name, after theirs, named as it is, copy after
// it, and a number after that where that is taken too.
func (v Views) CopyOf(name string) ([]Preset, string) {
	p, _ := v.Shared(name)
	free := func(n string) bool {
		return !slices.ContainsFunc(v.All(), func(w Preset) bool { return w.Name == n }) && !v.teams(n, "")
	}
	p.Name = name + " copy"
	for k := 2; !free(p.Name); k++ {
		p.Name = name + " copy " + strconv.Itoa(k)
	}
	return append(slices.Clone(v.Saved), p), p.Name
}

// RenameShared gives the team's view of a name renamed to another, to be kept in the team's folder in place of the
// first's file. None, a name of the table's own, the team's, or the person's are refused.
func (v Views) RenameShared(from, to string) (Preset, error) {
	p, _ := v.Shared(from)
	switch {
	case to == "":
		return Preset{}, ErrUnnamed
	case v.Builtin(to):
		return Preset{}, ErrBuiltin
	case v.teams(to, from) || slices.ContainsFunc(v.Saved, func(w Preset) bool { return w.Name == to }):
		return Preset{}, ErrTaken
	}
	p.Name = to
	return p, nil
}

// Builtin says whether a name is one of the table's own views.
func (v Views) Builtin(name string) bool {
	return slices.ContainsFunc(v.Own, func(p Preset) bool { return p.Name == name })
}

// Opening gives the view the table opens on: the first of its own, or Default, as it is, when it has none.
func (v Views) Opening() Preset {
	if len(v.Own) == 0 {
		return Preset{Name: Default}
	}
	return v.Own[0]
}

// Chosen gives the view of the name, the one a preset line was last set to, or the opening one when that view is gone.
func (v Views) Chosen(name string) Preset {
	all := v.All()
	if k := slices.IndexFunc(all, func(p Preset) bool { return p.Name == name }); k >= 0 {
		return all[k]
	}
	return v.Opening()
}

// Current gives a panel as a view would keep it.
func (v Views) Current(p grid.Panel) Preset { return PresetOf(p, v.Fields) }

// Same says whether two views set a table's panel alike, their names apart: each field placed, sorted, filtered, and
// drawn alike.
func (v Views) Same(a, b Preset) bool {
	normal := func(p Preset) Preset { return PresetOf(p.Panel(v.Fields, grid.Panel{}), v.Fields) }
	return reflect.DeepEqual(normal(a), normal(b))
}

// Is gives the name of the view a table is set as: the view chosen, when its panel is as it says, or as it grouped the
// rows as they loaded; else the first of its views it is as; none when it differs from every one.
func (v Views) Is(t Table) string {
	now, chosen := v.Current(t.View), v.Chosen(t.ViewName)
	if v.Same(now, chosen) || t.Grouped != nil && v.Same(now, *t.Grouped) {
		return chosen.Name
	}
	for _, p := range v.All() {
		if v.Same(now, p) {
			return p.Name
		}
	}
	return ""
}

// Mine gives the person's view of the name, and whether there is one: a table's own is not, nor is a view since
// deleted.
func (v Views) Mine(name string) (Preset, bool) {
	if k := slices.IndexFunc(v.Saved, func(p Preset) bool { return p.Name == name }); k >= 0 && !v.Builtin(name) {
		return v.Saved[k], true
	}
	return Preset{}, false
}

// SaveAs gives the person's views with a panel saved under a name: the person's view of that name overwritten, else a
// new one after theirs. None, and a name of the table's own, are refused, as those never change; and one the team has,
// by its slug, as the team's are its own to change.
func (v Views) SaveAs(name string, p grid.Panel) ([]Preset, error) {
	switch {
	case name == "":
		return nil, ErrUnnamed
	case v.Builtin(name):
		return nil, ErrBuiltin
	case v.teams(name, ""):
		return nil, ErrTaken
	}
	now := v.Current(p)
	now.Name = name
	saved := slices.Clone(v.Saved)
	if k := slices.IndexFunc(saved, func(w Preset) bool { return w.Name == name }); k >= 0 {
		saved[k] = now
	} else {
		saved = append(saved, now)
	}
	return saved, nil
}

// Rename gives the person's views with one renamed, from a name to another. None, a name of the table's own, another
// of the person's, and one the team has, by its slug, are refused; the same name gives them as they are.
func (v Views) Rename(from, to string) ([]Preset, error) {
	switch {
	case to == "":
		return nil, ErrUnnamed
	case v.Builtin(to):
		return nil, ErrBuiltin
	case to != from && slices.ContainsFunc(v.Saved, func(p Preset) bool { return p.Name == to }), v.teams(to, ""):
		return nil, ErrTaken
	}
	saved := slices.Clone(v.Saved)
	if k := slices.IndexFunc(saved, func(p Preset) bool { return p.Name == from }); k >= 0 {
		saved[k].Name = to
	}
	return saved, nil
}

// Delete gives the person's views without the one of the name.
func (v Views) Delete(name string) []Preset {
	return slices.DeleteFunc(slices.Clone(v.Saved), func(p Preset) bool { return p.Name == name })
}
