// A pill is its text on its color between half circles, a key and a value two segments; its text reads on its color,
// and a text without a color is drawn on the pastel its name picks.
package style

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// A plain pill is its text between two half circles, each a cell; a key and a value, the key, the round edge where the
// colors meet, then the value, the : or = dropped.
func TestAPillIsItsTextOnItsColorBetweenHalfCircles(t *testing.T) {
	for _, text := range []string{"bug", "Just starting", "UI 🎓"} {
		got := Pill(text, "#d73a4a")
		if ansi.Strip(got) != PillLeft+text+PillRight || ansi.StringWidth(got) != ansi.StringWidth(text)+2 {
			t.Errorf("%q: %q, %d cells", text, ansi.Strip(got), ansi.StringWidth(got))
		}
	}
	if got := ansi.Strip(Pill("priority=high", "")); got != PillLeft+"priority"+PillRight+"high"+PillRight {
		t.Errorf("a key and a value: %q", got)
	}
	if w := ansi.StringWidth(Pill("kind:fix", "")); w != len("kindfix")+3 {
		t.Errorf("a key and a value take %d cells", w)
	}
}

// The escapes are pinned: each half circle is its color on whatever the line is drawn on, the text white or black on
// the color, and every segment ends its own escapes, so no color runs past the pill.
func TestAPillsEscapesAreItsOwn(t *testing.T) {
	for _, c := range []struct{ text, color, want string }{
		{"bug", "#d73a4a", "\x1b[38;2;215;58;74m\ue0b6\x1b[m\x1b[38;2;255;255;255;48;2;215;58;74mbug\x1b[m" +
			"\x1b[38;2;215;58;74m\ue0b4\x1b[m"},
		{"priority=high", "", "\x1b[38;2;157;23;77m\ue0b6\x1b[m\x1b[38;2;255;255;255;48;2;157;23;77mpriority\x1b[m" +
			"\x1b[38;2;157;23;77;48;2;190;18;60m\ue0b4\x1b[m\x1b[38;2;255;255;255;48;2;190;18;60mhigh\x1b[m" +
			"\x1b[38;2;190;18;60m\ue0b4\x1b[m"},
		{"kind:fix", "#fbca04", "\x1b[38;2;161;98;7m\ue0b6\x1b[m\x1b[38;2;255;255;255;48;2;161;98;7mkind\x1b[m" +
			"\x1b[38;2;161;98;7;48;2;251;202;4m\ue0b4\x1b[m\x1b[38;2;0;0;0;48;2;251;202;4mfix\x1b[m" +
			"\x1b[38;2;251;202;4m\ue0b4\x1b[m"},
	} {
		if got := Pill(c.text, c.color); got != c.want {
			t.Errorf("%q on %q:\n%q\nwant\n%q", c.text, c.color, got, c.want)
		}
	}
	if got := PillEnd(PillLeft, "#123456"); got != "\x1b[38;2;18;52;86m\ue0b6\x1b[m" {
		t.Errorf("an end: %q", got)
	}
}

func TestALabelSplitsIntoAKeyAndAValueAtItsLastSeparator(t *testing.T) {
	for name, want := range map[string][3]string{
		"bug":                {"", "bug", ""},
		"priority=high":      {"priority", "high", "split"},
		"kind:fix":           {"kind", "fix", "split"},
		"area::tui":          {"area", "tui", "split"},
		"team::ux::research": {"team::ux", "research", "split"},
		"a:b=c":              {"a:b", "c", "split"},
		"a=b:c":              {"a=b", "c", "split"},
		"area::ui:dark":      {"area::ui", "dark", "split"},
		"area:ui::dark":      {"area:ui", "dark", "split"},
		":wip":               {"", ":wip", ""},
		"todo:":              {"", "todo:", ""},
		"::":                 {"", "::", ""},
		"area::":             {"", "area::", ""},
	} {
		key, val, ok := splitLabel(name)
		if got := [3]string{key, val, map[bool]string{true: "split"}[ok]}; got != want {
			t.Errorf("%q splits as %q, want %q", name, got, want)
		}
	}
}

// The text is black on a light color and white on a dark one, or on what is not #rrggbb; the key's color is its own,
// the same for every label with that key, and never its value's.
func TestAPillsTextReadsOnItsColorAndAKeyKeepsItsColor(t *testing.T) {
	light, dark := ink("#fbca04"), ink("#1d4ed8")
	if r, _, _, _ := light.RGBA(); r != 0 {
		t.Errorf("text on yellow is not black: %v", light)
	}
	if r, _, _, _ := dark.RGBA(); r == 0 {
		t.Errorf("text on blue is not white: %v", dark)
	}
	for _, odd := range []string{"12", "#zzzzzz"} {
		if r, _, _, _ := ink(odd).RGBA(); r == 0 {
			t.Errorf("text on %q is not white", odd)
		}
	}
	if keyColor("priority", "#ffffff") != keyColor("priority", "#000000") {
		t.Error("a key's color depends on its value's")
	}
	for _, c := range pastel {
		if keyColor("priority", c) == c {
			t.Errorf("the key priority is drawn on its value's color %s", c)
		}
	}
}

// A text without a color is drawn on the pastel its name picks, the same every time; a color given is the one drawn.
func TestAnEmptyColorPicksAPastelByName(t *testing.T) {
	if pillColor("x", "#123456") != "#123456" {
		t.Error("a color given is not drawn")
	}
	for _, name := range []string{"newcomer", "Just starting", "UI"} {
		got := pillColor(name, "")
		if got != pastel[picked(name)] || got != pillColor(name, "") {
			t.Errorf("%q is drawn on %q", name, got)
		}
		if Pill(name, "") != Pill(name, got) {
			t.Errorf("%q without a color is not drawn on its pastel", name)
		}
	}
}
