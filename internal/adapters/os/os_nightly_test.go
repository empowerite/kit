//go:build nightly

package os

import (
	"bytes"
	"context"
	"testing"

	"github.com/empowerite/kit/internal/ports"
)

func TestRunRunsAProgramAndGivesItsStatus(t *testing.T) {
	var out bytes.Buffer
	code, err := Processes{}.Run(context.Background(), ports.Cmd{Path: "sh", Args: []string{"-c", "echo said; exit 3"}, Stdout: &out})
	if code != 3 || err != nil || out.String() != "said\n" {
		t.Errorf("Run = %d, %v, %q; want 3 and what it said", code, err, out.String())
	}
}
