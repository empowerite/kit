// The presets file shared: a picture keeps its stage's presets in the file as it stands when it saves, so what
// another screen saved since the picture opened stays.
package flow

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/table"
)

func TestAPictureAndAnotherScreenSavingPresetsKeepEachOthers(t *testing.T) {
	file := filepath.Join(t.TempDir(), "presets.json")
	// the picture reads the file as it opens; the welcome guide's terms save a preset of theirs after
	m := onDropdown(t, file)
	if _, err := table.KeepPresets(file, "welcome/terms", []table.Preset{{Name: "Glossary"}}); err != nil {
		t.Fatal(err)
	}
	m.choosePreset(table.ItemSaveAs)
	m.stages[0].Naming = "Mine"
	m = drive(m, press(tea.KeyEnter))
	// the terms save again, after the picture
	terms := []table.Preset{{Name: "Glossary"}, {Name: "Short"}}
	if _, err := table.KeepPresets(file, "welcome/terms", terms); err != nil {
		t.Fatal(err)
	}
	all, err := table.ReadPresets(file)
	if err != nil || len(all["backlog"]) != 1 || all["backlog"][0].Name != "Mine" || len(all["welcome/terms"]) != 2 {
		t.Fatalf("both writers' presets: %v %+v", err, all)
	}
	if m.saved["welcome/terms"] == nil {
		t.Errorf("the picture holds what it wrote: %+v", m.saved)
	}
}
