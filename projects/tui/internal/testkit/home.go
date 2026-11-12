package testkit

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Homes are the variables that name where the person's own files are: their home, and the XDG directories kittui keeps
// its presets, panels, settings, and searches under, and reads its workspaces from.
var Homes = []string{"HOME", "XDG_CONFIG_HOME", "XDG_STATE_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME"}

// Main runs a package's tests, as its TestMain does, with HOME pointed at a directory of the run's own and each XDG
// directory under it, so no test reads or writes what the person keeps, and go test runs the same at any desk. The
// directory goes when the tests end. It gives what m.Run gives, for os.Exit, and 1, after a line that says why, where
// the directory cannot be made.
func Main(m *testing.M) int {
	home, err := os.MkdirTemp("", "kittui-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "testkit: no home for the tests:", err)
		return 1
	}
	defer os.RemoveAll(home)
	for _, name := range Homes {
		dir := home
		if name != "HOME" {
			dir = filepath.Join(home, name)
		}
		os.Setenv(name, dir)
	}
	return m.Run()
}
