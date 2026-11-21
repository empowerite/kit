package os

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	goos "os"
	"strings"

	"github.com/empowerite/kit/internal/ports"
)

// Presence proves that the person is there by what they type at the machine's terminal: the first characters of the
// plan's hash. It reads the terminal itself, never kit's input, so nothing piped to kit can answer for the person.
// It is weaker than the machine's own proof, and says so.
type Presence struct {
	open func() (io.ReadWriteCloser, error)
}

// NewPresence gives the presence of the machine's terminal.
func NewPresence() Presence {
	return Presence{open: func() (io.ReadWriteCloser, error) { return goos.OpenFile("/dev/tty", goos.O_RDWR, 0) }}
}

// typed is how many of the hash's characters the person types.
const typed = 6

func (p Presence) Prove(_ context.Context, reason, hash string) (ports.Proof, error) {
	tty, err := p.open()
	if err != nil {
		return ports.Proof{}, errors.New("presence: nobody is here to approve: run this at a terminal")
	}
	defer tty.Close()
	want := hash[:typed]
	fmt.Fprintf(tty, "%s\nThis machine gives kit no proof of its own that you are here, so the approval is weaker.\n"+
		"Type %s to approve it, and press return: ", reason, want)
	line, _ := bufio.NewReader(tty).ReadString('\n')
	if strings.TrimSpace(line) != want {
		return ports.Proof{}, fmt.Errorf("presence: not approved: %s was not typed", want)
	}
	return ports.Proof{Kind: "typed", Weaker: true}, nil
}
