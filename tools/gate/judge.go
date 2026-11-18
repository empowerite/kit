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

// Verdict is what the judge makes of a run: what it refuses, and how much it judged.
type Verdict struct {
	Findings []Finding
	Tested   int // the packages whose tests ran
	Bare     int // the packages with nothing to test
	Tests    int // the top-level tests that ended
}

// event is one line of what go test -json prints. An output's type is "frame" for a line that is go test's own, as
// "=== RUN" and "--- FAIL" are.
type event struct {
	Action, Package, Test, ImportPath, Output, OutputType string
	Elapsed                                               float64
}

// pack is what the judge keeps of one package of a run.
type pack struct {
	ended, covered bool          // its own end came; its tests said what they covered
	said           []string      // its own lines, outside any test
	tests          time.Duration // how long its top-level tests took together
}

// Judge reads a run's events and its coverage profile, and gives what the gate refuses, in the order found. took is
// how long the gate has run, and status is what go test ended with. There is no profile when go test wrote none.
func Judge(events, profile io.Reader, took time.Duration, status int, limits Limits) (Verdict, error) {
	var verdict Verdict
	refuse := func(rule, where, detail string) {
		verdict.Findings = append(verdict.Findings, Finding{rule, where, detail})
	}
	compiler := map[string][]string{} // what the compiler said of an import path that did not build
	failed := map[string]bool{}       // the packages, and the tests, that a finding below them already explains
	told := map[string][]string{}     // each test's own lines
	packs := map[string]*pack{}
	var started []string // the packages, in the order they started
	of := func(name string) *pack {
		if packs[name] == nil {
			packs[name], started = &pack{}, append(started, name)
		}
		return packs[name]
	}
	lines := bufio.NewScanner(events)
	lines.Buffer(nil, 16<<20)
	for n := 1; lines.Scan(); n++ {
		var e event
		if err := json.Unmarshal(lines.Bytes(), &e); err != nil {
			return Verdict{}, fmt.Errorf("go test's line %d is no event: %v", n, err)
		}
		path, _, _ := strings.Cut(e.ImportPath, " ")
		test := e.Package + "." + e.Test
		line := strings.TrimSpace(e.Output)
		elapsed := time.Duration(e.Elapsed * float64(time.Second))
		ended := e.Action == "pass" || e.Action == "fail" || e.Action == "skip"
		switch {
		case e.Action == "build-output" && line != "" && !strings.HasPrefix(line, "#"):
			compiler[path] = append(compiler[path], line)
		case e.Action == "build-fail":
			refuse("build", path, strings.Join(compiler[path], "\n"))
			failed[path] = true
		case e.Action == "start":
			of(e.Package)
		case e.Action == "output" && e.OutputType != "frame" && line != "" && e.Test != "":
			told[test] = append(told[test], line)
		case e.Action == "output" && e.OutputType != "frame" && line != "":
			p := of(e.Package)
			p.covered = p.covered || strings.Contains(line, "coverage:")
			p.said = append(p.said, line)
		case ended && e.Test != "":
			above, _, sub := cutLast(e.Test, "/") // the test a subtest is in: a package's path has slashes of its own
			switch {
			case e.Action == "fail" && !failed[test]:
				refuse("test", test, first(told[test], ""))
			case e.Action == "skip":
				refuse("skip", test, first(told[test], "it skipped itself"))
			}
			if e.Action == "fail" {
				failed[e.Package], failed[e.Package+"."+above] = true, true
			}
			if sub {
				continue // a subtest's time is inside its test's
			}
			verdict.Tests++
			of(e.Package).tests += elapsed
			if elapsed > limits.Test {
				refuse("slow", test, over(elapsed, limits.Test))
			}
		case ended:
			p := of(e.Package)
			p.ended = true
			if e.Action == "skip" {
				verdict.Bare++
				continue
			}
			verdict.Tested++
			switch {
			case e.Action == "fail" && !failed[e.Package]:
				refuse("package", e.Package, first(p.said, "it failed outside a test"))
			case e.Action == "pass" && !p.covered:
				refuse("coverage", e.Package, "its tests ended without saying what they covered: is m.Run called?")
			}
			if whole := max(elapsed, p.tests); whole > limits.Package {
				refuse("slow", e.Package, over(whole, limits.Package))
			}
		}
	}
	if err := lines.Err(); err != nil {
		return Verdict{}, fmt.Errorf("go test's events broke off: %v", err)
	}
	for _, name := range started {
		if !packs[name].ended {
			refuse("package", name, "its tests never ended: go test stopped, or a test ended the program")
		}
	}
	if len(started) == 0 {
		refuse("empty", "go test", "it printed nothing")
	}
	switch {
	case profile != nil:
		unrun, err := uncovered(profile)
		if err != nil {
			return Verdict{}, err
		}
		verdict.Findings = append(verdict.Findings, unrun...)
	case verdict.Tested > 0:
		refuse("coverage", "the profile", "go test wrote none that can be read")
	}
	if took > limits.Whole {
		refuse("budget", "the gate", over(took, limits.Whole))
	}
	if status != 0 && len(verdict.Findings) == 0 {
		refuse("go test", "go test", fmt.Sprintf("it ended with status %d, and nothing above says why", status))
	}
	return verdict, nil
}

// over says how long something took against its limit.
func over(took, limit time.Duration) string {
	return fmt.Sprintf("%s, over %s", took, limit)
}

// first gives the first eight of some lines as one text, and how many more there are, or other words where there
// are none.
func first(lines []string, none string) string {
	const most = 8
	switch {
	case len(lines) == 0:
		return none
	case len(lines) > most:
		return strings.Join(lines[:most], "\n") + fmt.Sprintf("\n… and %d more line(s)", len(lines)-most)
	}
	return strings.Join(lines, "\n")
}

// cutLast cuts a text at the last of a separator, and says whether it was there.
func cutLast(text, separator string) (before, after string, found bool) {
	at := strings.LastIndex(text, separator)
	if at < 0 {
		return text, "", false
	}
	return text[:at], text[at+len(separator):], true
}

// uncovered gives a finding for each block of a coverage profile that no test ran. A profile is the line
// "mode: set", then a line for each block: <file>:<line>.<column>,<line>.<column> <statements> <count>.
func uncovered(profile io.Reader) ([]Finding, error) {
	var findings []Finding
	lines := bufio.NewScanner(profile)
	for n := 1; lines.Scan(); n++ {
		line := lines.Text()
		if n == 1 && strings.HasPrefix(line, "mode:") {
			continue
		}
		file, block, found := cutLast(line, ":")
		fields := strings.Fields(block)
		if !found || n == 1 || len(fields) != 3 {
			return nil, fmt.Errorf("the coverage profile's line %d cannot be read: %s", n, line)
		}
		start, _, _ := strings.Cut(fields[0], ".")
		_, errLine := strconv.Atoi(start)
		statements, errStatements := strconv.Atoi(fields[1])
		count, errCount := strconv.Atoi(fields[2])
		if errLine != nil || errStatements != nil || errCount != nil {
			return nil, fmt.Errorf("the coverage profile's line %d cannot be read: %s", n, line)
		}
		if count == 0 {
			findings = append(findings, Finding{"coverage", file + ":" + start,
				fmt.Sprintf("%d statement(s) no test runs", statements)})
		}
	}
	if err := lines.Err(); err != nil {
		return nil, fmt.Errorf("the coverage profile cannot be read: %v", err)
	}
	return findings, nil
}
