package os

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/empowerite/kit/internal/ports"
)

// terminal is a terminal of the test's making: what the person types, and what kit writes to it.
type terminal struct {
	io.Reader
	bytes.Buffer
	closed bool
}

func (t *terminal) Write(p []byte) (int, error) { return t.Buffer.Write(p) }
func (t *terminal) Read(p []byte) (int, error)  { return t.Reader.Read(p) }
func (t *terminal) Close() error                { t.closed = true; return nil }

func at(typed string) (*terminal, func() (io.ReadWriteCloser, error)) {
	tty := &terminal{Reader: strings.NewReader(typed)}
	return tty, func() (io.ReadWriteCloser, error) { return tty, nil }
}

const hash = "0d065e578f6a9b2c"

var _ ports.Presence = Presence{}

func TestTypedPresenceTakesTheHashsFirstCharacters(t *testing.T) {
	tty, open := at("0d065e\n")
	proof, err := Presence{open: open}.Prove(context.Background(), "approve kit tools add, plan 0d065e578f6a", hash)
	if err != nil || proof != (ports.Proof{Kind: "typed", Weaker: true}) {
		t.Errorf("Prove = %+v, %v; want a typed proof, marked weaker", proof, err)
	}
	if said := tty.String(); !strings.Contains(said, "approve kit tools add, plan 0d065e578f6a") ||
		!strings.Contains(said, "Type 0d065e to approve") || !strings.Contains(said, "weaker") || !tty.closed {
		t.Errorf("the terminal was told %q, and closed %v", said, tty.closed)
	}
}

func TestTypedPresenceRefusesOtherCharacters(t *testing.T) {
	for _, typed := range []string{"0d065f\n", "\n", ""} {
		_, open := at(typed)
		_, err := Presence{open: open}.Prove(context.Background(), "approve", hash)
		if err == nil || !strings.HasPrefix(err.Error(), "presence: ") {
			t.Errorf("Prove with %q typed gave %v; want a refusal", typed, err)
		}
	}
}

func TestPresenceRefusesWithNoTerminal(t *testing.T) {
	open := func() (io.ReadWriteCloser, error) { return nil, errors.New("device not configured") }
	_, err := Presence{open: open}.Prove(context.Background(), "approve", hash)
	if err == nil || err.Error() != "presence: nobody is here to approve: run this at a terminal" {
		t.Errorf("Prove with no terminal gave %v", err)
	}
}

func TestNewPresenceOpensTheMachinesTerminal(t *testing.T) {
	// Whether the test has a terminal depends on where it runs; either way it is the machine's own that is opened.
	if tty, err := NewPresence().open(); err == nil {
		tty.Close()
	}
}
