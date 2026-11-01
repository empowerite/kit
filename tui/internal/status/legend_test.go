// Every glyph the picture draws is in kit help status's legend, the header of kit/bin/kit-status, so bash and Go
// draw the same picture.
package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// legendText reads the Legend of kit-status's header, from its heading to the exit codes.
func legendText(t *testing.T) string {
	t.Helper()
	page, err := os.ReadFile(filepath.Join("..", "..", "..", "kit", "bin", "kit-status"))
	if err != nil {
		t.Fatal(err)
	}
	_, after, ok := strings.Cut(string(page), "# Legend")
	if !ok {
		t.Fatal("kit-status has no Legend")
	}
	before, _, _ := strings.Cut(after, "# Exits")
	return before
}

func TestEveryGlyphIsInTheLegend(t *testing.T) {
	text := legendText(t)
	for _, glyph := range legend {
		if !strings.Contains(text, glyph) {
			t.Errorf("the legend has no %s", glyph)
		}
	}
	for state, glyph := range states {
		found := false
		for _, line := range strings.Split(text, "\n") {
			if strings.Contains(line, glyph) && strings.Contains(line, state) {
				found = true
			}
		}
		if !found {
			t.Errorf("the legend does not draw %s as %s", state, glyph)
		}
	}
	if stateGlyph("nonesuch") != Other || stateGlyph("ready") != "🛬" {
		t.Errorf("stateGlyph: %s %s", stateGlyph("nonesuch"), stateGlyph("ready"))
	}
	if kindGlyph("draft") != Draft || kindGlyph("pre-release") != Pre || kindGlyph("release") != Release {
		t.Errorf("kindGlyph: %s %s %s", kindGlyph("draft"), kindGlyph("pre-release"), kindGlyph("release"))
	}
}
