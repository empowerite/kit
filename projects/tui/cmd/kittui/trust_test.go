// In a clone kit does not trust yet, kittui asks before it reads anything there: Trust goes on to the screen asked
// for, Not now to the palette, which reads what an untrusted clone allows, ctrl-c ends with 130; a kit that cannot say
// is kittui's line.
package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/app"
	"github.com/empowerite/kit/projects/tui/internal/screens"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
	"github.com/empowerite/kit/projects/tui/internal/trust"
)

// overUntrusted makes kittui run on a terminal over the trust dialog's fake kit, in a clone it does not trust, with
// the program stood in for: the dialog driven by the key, and the screen after it a program that cannot run, so a
// test sees kittui go on past the dialog. It gives the file kit trust --yes writes its words to, and what was shown.
func overUntrusted(t *testing.T, key tea.Msg) (string, *[]tea.Model) {
	t.Helper()
	shown := &[]tea.Model{}
	onTerminal(t, func(m tea.Model) (tea.Model, error) {
		*shown = append(*shown, m)
		if d, ok := m.(trust.Model); ok {
			d, _ = testkit.Drive(d, nil, key)
			return d, nil
		}
		return nil, errors.New("past the dialog")
	})
	t.Setenv("KIT", testkit.Runner(t, "trust").Kit)
	record := filepath.Join(t.TempDir(), "record")
	t.Setenv("FAKE_TRUST_RECORD", record)
	return record, shown
}

func TestTrustGoesOnToTheScreenAskedFor(t *testing.T) {
	record, shown := overUntrusted(t, testkit.Press("t"))
	var stderr bytes.Buffer
	if code := run(nil, &stderr); code != 1 || stderr.String() != "❌ kittui unreleased: past the dialog\n" {
		t.Errorf("exit %d, stderr %q", code, stderr.String())
	}
	if said, err := os.ReadFile(record); err != nil || string(said) != "trust --yes /the/clone\n" || len(*shown) != 2 {
		t.Errorf("kit ran %q (%v), %d screens", said, err, len(*shown))
	}
}

// Not now opens what an untrusted clone allows, the palette, whatever screen was asked for, and runs no kit trust.
func TestNotNowOpensTheHelpPalette(t *testing.T) {
	for _, args := range [][]string{nil, {"flow"}, {"--", "commit"}} {
		record, shown := overUntrusted(t, testkit.Press("esc"))
		var stderr bytes.Buffer
		code := run(args, &stderr)
		if code != 1 || stderr.String() != "❌ kittui unreleased: past the dialog\n" || len(*shown) != 2 {
			t.Errorf("%q: exit %d, stderr %q, %d screens", args, code, stderr.String(), len(*shown))
			continue
		}
		if m, ok := (*shown)[1].(screens.Model); !ok || m.Shown() == nil {
			t.Errorf("%q: after Not now, %T", args, (*shown)[1])
		} else if _, ok := m.Shown().(app.Model); !ok {
			t.Errorf("%q: after Not now, %T, not the palette", args, m.Shown())
		}
		if _, err := os.Stat(record); err == nil {
			t.Errorf("%q: kit trust ran", args)
		}
	}
}

func TestCtrlCOnTheDialogEndsWith130(t *testing.T) {
	overUntrusted(t, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	var stderr bytes.Buffer
	if code := run(nil, &stderr); code != 130 || stderr.Len() != 0 {
		t.Errorf("exit %d, stderr %q", code, stderr.String())
	}
}

// A dialog the program cannot run is kittui's own line; a clone kit trusts opens no dialog; a kit that cannot answer
// kit trust --check is kittui's line too.
func TestTheDialogsProgramAndAKitThatCannotSay(t *testing.T) {
	onTerminal(t, func(tea.Model) (tea.Model, error) { return nil, errors.New("no tty") })
	t.Setenv("KIT", testkit.Runner(t, "trust").Kit)
	var stderr bytes.Buffer
	if code := run(nil, &stderr); code != 1 || stderr.String() != "❌ kittui unreleased: no tty\n" {
		t.Errorf("exit %d, stderr %q", code, stderr.String())
	}
	_, shown := overUntrusted(t, testkit.Press("t"))
	t.Setenv("FAKE_TRUST_TRUSTED", "1")
	stderr.Reset()
	if code := run(nil, &stderr); code != 1 || len(*shown) != 1 {
		t.Errorf("a trusted clone: exit %d, %d screens", code, len(*shown))
	} else if _, ok := (*shown)[0].(trust.Model); ok {
		t.Error("a trusted clone opened the dialog")
	}
	t.Setenv("KIT", filepath.Join(t.TempDir(), "no-kit"))
	stderr.Reset()
	if code := run(nil, &stderr); code != 1 || !strings.HasPrefix(stderr.String(), "❌ kittui unreleased: ") {
		t.Errorf("no kit: exit %d, stderr %q", code, stderr.String())
	}
}
