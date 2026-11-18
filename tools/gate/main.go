// Command gate judges a run of this repository's tests: it reads what go test -json printed, the coverage profile
// the run wrote, and the status go test ended with, and refuses a failure, a skip, a statement no test runs, a slow
// test, a slow gate, and a run that stopped short. bin/gate runs it, once go test has ended.
package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// exit ends the program with a status. A test stands in for it.
var exit = os.Exit

func main() {
	exit(run(os.Args[1:], os.Stdin, os.Stdout, time.Now))
}

// run judges the events on stdin against the profile --profile names, for a gate that started at --since and a
// go test that ended with --status. It gives the exit status: 0 when the gate holds, 1 when it refuses, and 2 when
// it cannot judge.
func run(args []string, stdin io.Reader, stdout io.Writer, now func() time.Time) int {
	options := map[string]string{}
	for _, arg := range args {
		name, value, _ := strings.Cut(arg, "=")
		options[name] = value
	}
	started, errSince := strconv.ParseInt(options["--since"], 10, 64)
	status, errStatus := strconv.Atoi(options["--status"])
	if len(options) != 3 || options["--profile"] == "" || errSince != nil || errStatus != nil {
		fmt.Fprintln(stdout, "usage: gate --profile=<file> --since=<unix seconds> --status=<go test's exit status>")
		return 2
	}
	// A profile that cannot be opened is no profile: go test writes none when it stops before any package ends.
	var profile io.Reader
	if file, err := os.Open(options["--profile"]); err == nil {
		defer file.Close()
		profile = file
	}
	took := now().Sub(time.Unix(started, 0))
	verdict, err := Judge(stdin, profile, took, status, Limits{time.Second, 10 * time.Second, 3 * time.Minute})
	if err != nil {
		fmt.Fprintf(stdout, "gate: %v\n", err)
		return 2
	}
	for _, f := range verdict.Findings {
		fmt.Fprintf(stdout, "gate: %s: %s: %s\n", f.Rule, f.Where, strings.ReplaceAll(f.Detail, "\n", "\n  "))
	}
	if len(verdict.Findings) > 0 {
		fmt.Fprintf(stdout, "gate: refused: %d finding(s)\n", len(verdict.Findings))
		return 1
	}
	fmt.Fprintf(stdout, "gate: ok: %d package(s) tested, %d with nothing to test, %d test(s), %s\n",
		verdict.Tested, verdict.Bare, verdict.Tests, took.Round(time.Second))
	return 0
}
