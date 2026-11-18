// Command kit keeps the work of AI coding agents trustworthy. For now it prints its version.
package main

import (
	"fmt"
	"io"
	"os"
)

// version is the release this kit was built as. A release sets it with -ldflags "-X main.version=<version>".
var version = "unreleased"

// exit ends the program with a status. A test stands in for it.
var exit = os.Exit

func main() {
	exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run does what the words after kit ask, and gives the exit status: 0, or 2 for words it cannot take.
func run(args []string, stdout, stderr io.Writer) int {
	switch {
	case len(args) == 1 && args[0] == "version":
		fmt.Fprintln(stdout, version)
		return 0
	case len(args) == 0 || args[0] == "version":
		fmt.Fprintln(stderr, "usage: kit version")
		return 2
	}
	fmt.Fprintf(stderr, "kit: no such command: %s\n", args[0])
	return 2
}

// unused is in no test, to see the gate refuse it.
func unused() int { return 1 }
