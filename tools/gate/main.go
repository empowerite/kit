// Command gate judges a run of this repository's tests: it reads what go test -json prints, and the coverage profile
// the run wrote, and refuses a failure, a statement no test runs, a slow test, and a slow gate. bin/gate runs it.
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

// run judges the events on stdin against the profile --profile names, for a gate that started at --since, and gives
// the exit status: 0 when the gate holds, 1 when it refuses, and 2 when it cannot judge.
func run(args []string, stdin io.Reader, stdout io.Writer, now func() time.Time) int {
	var profile, since string
	unknown := false
	for _, arg := range args {
		switch name, value, _ := strings.Cut(arg, "="); name {
		case "--profile":
			profile = value
		case "--since":
			since = value
		default:
			unknown = true
		}
	}
	started, err := strconv.ParseInt(since, 10, 64)
	if unknown || profile == "" || err != nil {
		fmt.Fprintln(stdout, "usage: gate --profile=<file> --since=<unix seconds>")
		return 2
	}
	file, err := os.Open(profile)
	if err != nil {
		fmt.Fprintf(stdout, "gate: %v\n", err)
		return 2
	}
	defer file.Close()
	took := now().Sub(time.Unix(started, 0))
	findings, packages, tests, err := Judge(stdin, file, took, Limits{time.Second, 10 * time.Second, 3 * time.Minute})
	if err != nil {
		fmt.Fprintf(stdout, "gate: %v\n", err)
		return 2
	}
	for _, f := range findings {
		fmt.Fprintf(stdout, "gate: %s: %s: %s\n", f.Rule, f.Where, f.Detail)
	}
	if len(findings) > 0 {
		fmt.Fprintf(stdout, "gate: refused: %d finding(s)\n", len(findings))
		return 1
	}
	fmt.Fprintf(stdout, "gate: ok: %d package(s), %d test(s), %s\n", packages, tests, took.Round(time.Second))
	return 0
}
