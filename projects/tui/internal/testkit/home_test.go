package testkit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) { os.Exit(Main(m)) }

// Under Main, HOME is a directory of the run's own, which no person keeps anything in, and each XDG directory is under
// it.
func TestMainGivesTheTestsAHomeOfTheirOwn(t *testing.T) {
	home := os.Getenv("HOME")
	if !strings.HasPrefix(filepath.Base(home), "kittui-home-") {
		t.Fatalf("HOME is %q", home)
	}
	if entries, err := os.ReadDir(home); err != nil || len(entries) != 0 {
		t.Errorf("the tests' home holds %v, %v", entries, err)
	}
	for _, name := range Homes[1:] {
		if got := os.Getenv(name); got != filepath.Join(home, name) {
			t.Errorf("%s is %q", name, got)
		}
	}
}

// Where no directory can be made for a home, Main runs no test and gives 1.
func TestMainRunsNothingWithoutAHome(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "gone"))
	if got := Main(nil); got != 1 {
		t.Errorf("Main without a home gave %d", got)
	}
}
