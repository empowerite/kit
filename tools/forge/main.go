// Command forge judges whether GitHub holds what this repository's gate leans on: it reads the rules GitHub applies
// to a branch, and what is expected of them, and says which do not hold. bin/forge asks GitHub and runs it.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// exit ends the program with a status. A test stands in for it.
var exit = os.Exit

func main() {
	exit(run(os.Args[1:], os.Stdout))
}

// run judges the rules in the file --rules names, which GitHub gave of the branch --branch names, against the file
// --expect names. It gives the exit status: 0 when every rule holds, 1 when one does not, and 2 when it cannot judge.
func run(args []string, stdout io.Writer) int {
	options := map[string]string{}
	for _, arg := range args {
		name, value, _ := strings.Cut(arg, "=")
		options[name] = value
	}
	branch := options["--branch"]
	if len(options) != 3 || branch == "" || options["--rules"] == "" || options["--expect"] == "" {
		fmt.Fprintln(stdout, "usage: forge --branch=<branch> --rules=<file> --expect=<file>")
		return 2
	}
	rules, err := os.Open(options["--rules"])
	if err != nil {
		fmt.Fprintf(stdout, "forge: %v\n", err)
		return 2
	}
	defer rules.Close()
	expect, err := os.Open(options["--expect"])
	if err != nil {
		fmt.Fprintf(stdout, "forge: %v\n", err)
		return 2
	}
	defer expect.Close()
	findings, expected, err := Judge(rules, expect)
	if err != nil {
		fmt.Fprintf(stdout, "forge: %v\n", err)
		return 2
	}
	for _, f := range findings {
		fmt.Fprintf(stdout, "forge: %s: nothing holds that %s: %s\n", branch, f.That, f.Wrong)
	}
	if len(findings) > 0 {
		fmt.Fprintf(stdout, "forge: refused: GitHub does not hold %d of the %d rule(s) the gate leans on\n",
			len(findings), expected)
		return 1
	}
	fmt.Fprintf(stdout, "forge: ok: %s holds the %d rule(s) the gate leans on\n", branch, expected)
	return 0
}
