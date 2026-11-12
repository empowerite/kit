// In a clone kit does not trust yet, kittui asks before it reads anything there: Trust goes on to the screen asked
// for, Not now ends with kit's line and 1, ctrl-c with 130; a kit that cannot say is kit's refusal.
package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

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

func TestNotNowEndsWithKitsLine(t *testing.T) {
	record, shown := overUntrusted(t, testkit.Press("esc"))
	var stderr bytes.Buffer
	code := run([]string{"flow"}, &stderr)
	if code != 1 || !strings.HasPrefix(stderr.String(), "❌ kit runs nothing in /the/clone until you trust it") ||
		len(*shown) != 1 {
		t.Errorf("exit %d, stderr %q, %d screens", code, stderr.String(), len(*shown))
	}
	if _, err := os.Stat(record); err == nil {
		t.Error("kit trust ran")
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
	_, shown := overUntrusted(t, testkit.Press("esc"))
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
