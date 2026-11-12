package table

import (
	"os"
	"testing"

	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// TestMain runs the tests with a home of their own, as testkit.Main gives them, so none reads what the person keeps.
func TestMain(m *testing.M) { os.Exit(testkit.Main(m)) }
