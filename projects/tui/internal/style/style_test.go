// The styles draw their text whole, each with its own escapes, and the keys bar fits its line and ends the screen.
package style

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// Each style keeps its text as it is, and only adds the escapes that draw it; two styles draw it differently.
func TestEveryStyleDrawsItsTextWholeAndInItsOwnWay(t *testing.T) {
	styles := map[string]func(string) string{"Mark": Mark, "Gain": Gain, "Loss": Loss, "Warn": Warn, "Repo": Repo,
		"Chosen": Chosen, "Off": Off, "Label": Label, "Cursor": Cursor, "Refusal": Refusal}
	drawn := map[string]string{}
	for name, draw := range styles {
		got := draw("kit 🏭")
		if ansi.Strip(got) != "kit 🏭" {
			t.Errorf("%s changes the text: %q", name, ansi.Strip(got))
		}
		for other, before := range drawn {
			if got == before && other != name {
				t.Errorf("%s and %s draw alike: %q", name, other, got)
			}
		}
		drawn[name] = got
	}
}

// A band keeps the line's text and styles, pads it to the width, and opens the band again after each style ends.
func TestABandRunsTheWholeLine(t *testing.T) {
	line := "#45 " + Off("2h") + " title"
	got := Band(line, 20)
	if ansi.Strip(got) != "#45 2h title        " {
		t.Errorf("the band's text: %q", ansi.Strip(got))
	}
	open, _, _ := strings.Cut(band.Render("x"), "x")
	if !strings.Contains(got, "\x1b[m"+open+" title") || !strings.Contains(got, Off("2h")) {
		t.Errorf("the band does not run past the dim age: %q", got)
	}
}

// The bar is one dim line after a space, clipped to the width with …, and nothing at all on a screen of one column.
func TestTheBarFitsItsLine(t *testing.T) {
	keys := "↑↓ move · ⏎ next · esc back"
	if got := ansi.Strip(Bar(keys, 80)); got != " "+keys {
		t.Errorf("at 80: %q", got)
	}
	if got := ansi.Strip(Bar(keys, 12)); ansi.StringWidth(got) != 12 || !strings.HasSuffix(got, "…") {
		t.Errorf("at 12: %q", got)
	}
	if Bar(keys, 1) != "" || Bar(keys, 0) != "" {
		t.Errorf("at one column or none: %q %q", Bar(keys, 1), Bar(keys, 0))
	}
	if !strings.Contains(Bar(keys, 80), Off(keys)) {
		t.Error("the bar is not dim")
	}
}

// The bar is the last line of a screen of any height: the lines above are cut or padded to the rest, and a height not
// yet known draws them all, a gap, and the bar.
func TestTheBarIsTheLastLineOfTheScreen(t *testing.T) {
	lines := []string{"a", "b", "c"}
	for _, c := range []struct {
		height int
		want   []string
	}{
		{5, []string{"a", "b", "c", "", "bar"}},
		{3, []string{"a", "b", "bar"}},
		{1, []string{"bar"}},
		{0, []string{"a", "b", "c", "", "bar"}},
	} {
		if got := Bottom(slices.Clone(lines), c.height, "bar"); !slices.Equal(got, c.want) {
			t.Errorf("Bottom at %d = %q, want %q", c.height, got, c.want)
		}
	}
}

func TestTheDividerIsAGreyDarkerThanDim(t *testing.T) {
	if got := Divider("│"); !strings.Contains(got, "38;5;239") || strings.Contains(got, "\x1b[2m") {
		t.Errorf("the divider: %q", got)
	}
	if got := Tint("x"); strings.Contains(got, "1;") {
		t.Errorf("a tint is bold: %q", got)
	}
}
