// Command kit keeps the work of AI coding agents trustworthy. For now it says which build it is.
package main

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
)

// build gives what Go recorded of this build. A test stands in for it.
var build = debug.ReadBuildInfo

// exit ends the program with a status. A test stands in for it.
var exit = os.Exit

func main() {
	exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run does what the words after kit ask, and gives the exit status: 0, or 2 for words it cannot take.
func run(args []string, stdout, stderr io.Writer) int {
	switch {
	case len(args) == 1 && args[0] == "version":
		fmt.Fprintln(stdout, which(build()))
		return 0
	case len(args) == 0 || args[0] == "version":
		fmt.Fprintln(stderr, "usage: kit version")
		return 2
	}
	fmt.Fprintf(stderr, "kit: no such command: %s\n", args[0])
	return 2
}

// which says which build this is: the commit it was built from, when that commit was made, and whether the tree
// held changes the commit does not. Go records these in every build that go build makes, with no flag.
//
// A build does not say its release, since it does not know one: a release is a tag put on a commit whose build
// exists already, and two releases can name one build. Which releases name a commit is read from the tags.
func which(info *debug.BuildInfo, recorded bool) string {
	of := map[string]string{}
	if recorded {
		for _, setting := range info.Settings {
			of[setting.Key] = setting.Value
		}
	}
	if of["vcs.revision"] == "" {
		return "kit, a build that names no commit"
	}
	said := fmt.Sprintf("kit, built from commit %s of %s", of["vcs.revision"], of["vcs.time"])
	if of["vcs.modified"] == "true" {
		said += ", and from changes it does not hold"
	}
	return said
}
