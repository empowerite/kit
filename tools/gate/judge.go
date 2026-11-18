package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Limits are the budgets the gate holds.
type Limits struct{ Test, Package, Whole time.Duration }

// Finding is one thing the gate refuses.
type Finding struct{ Rule, Where, Detail string }

// event is one line of what go test -json prints.
type event struct {
	Action, Package, Test, ImportPath, Output string
	Elapsed                                   float64
}

// Judge reads a run's events and its coverage profile, and gives what the gate refuses, in the order found, with how
// many packages and top-level tests the run held. took is how long the gate has run.
func Judge(events, profile io.Reader, took time.Duration, limits Limits) (findings []Finding, packages, tests int, err error) {
	said := map[string]string{} // what the compiler said of an import path that did not build
	failed := map[string]bool{} // the packages that a finding already names
	lines := bufio.NewScanner(events)
	lines.Buffer(nil, 16<<20)
	for n := 1; lines.Scan(); n++ {
		var e event
		if err := json.Unmarshal(lines.Bytes(), &e); err != nil {
			return nil, 0, 0, fmt.Errorf("go test's line %d is no event: %v", n, err)
		}
		path, _, _ := strings.Cut(e.ImportPath, " ")
		took := time.Duration(e.Elapsed * float64(time.Second))
		ended := e.Action == "pass" || e.Action == "fail" || e.Action == "skip"
		switch {
		case e.Action == "build-output" && !strings.HasPrefix(e.Output, "#"):
			said[path] += e.Output
		case e.Action == "build-fail":
			findings, failed[path] = append(findings, Finding{"build", path, said[path]}), true
		case ended && e.Test != "":
			if e.Action == "fail" {
				findings, failed[e.Package] = append(findings, Finding{"test", e.Package + "." + e.Test, ""}), true
			}
			if strings.Contains(e.Test, "/") {
				continue // a subtest's time is inside its test's
			}
			tests++
			if took > limits.Test {
				findings = append(findings, Finding{"slow", e.Package + "." + e.Test, over(took, limits.Test)})
			}
		case ended:
			packages++
			if e.Action == "fail" && !failed[e.Package] {
				findings = append(findings, Finding{"package", e.Package, "it failed outside a test"})
			}
			if took > limits.Package {
				findings = append(findings, Finding{"slow", e.Package, over(took, limits.Package)})
			}
		}
	}
	if packages == 0 {
		findings = append(findings, Finding{"empty", "go test", "it printed nothing"})
	}
	uncovered, err := unrun(profile)
	if err != nil {
		return nil, 0, 0, err
	}
	findings = append(findings, uncovered...)
	if took > limits.Whole {
		findings = append(findings, Finding{"budget", "the gate", over(took, limits.Whole)})
	}
	return findings, packages, tests, nil
}

// over says how long something took against its limit.
func over(took, limit time.Duration) string {
	return fmt.Sprintf("%s, over %s", took, limit)
}

// unrun gives a finding for each block of a coverage profile that no test ran. A profile is the line "mode: set",
// then a line for each block: <file>:<line>.<column>,<line>.<column> <statements> <count>.
func unrun(profile io.Reader) ([]Finding, error) {
	var findings []Finding
	lines := bufio.NewScanner(profile)
	for n := 1; lines.Scan(); n++ {
		line := lines.Text()
		if n == 1 && strings.HasPrefix(line, "mode:") {
			continue
		}
		file, block, found := cut(line)
		fields := strings.Fields(block)
		if !found || n == 1 || len(fields) != 3 {
			return nil, fmt.Errorf("the coverage profile's line %d cannot be read: %s", n, line)
		}
		first, _, _ := strings.Cut(fields[0], ".")
		_, errLine := strconv.Atoi(first)
		statements, errStatements := strconv.Atoi(fields[1])
		count, errCount := strconv.Atoi(fields[2])
		if errLine != nil || errStatements != nil || errCount != nil {
			return nil, fmt.Errorf("the coverage profile's line %d cannot be read: %s", n, line)
		}
		if count == 0 {
			findings = append(findings, Finding{"coverage", file + ":" + first,
				fmt.Sprintf("%d statement(s) no test runs", statements)})
		}
	}
	return findings, nil
}

// cut splits a profile's line at its last colon, into the file and the block.
func cut(line string) (file, block string, found bool) {
	at := strings.LastIndex(line, ":")
	if at < 0 {
		return "", "", false
	}
	return line[:at], line[at+1:], true
}
