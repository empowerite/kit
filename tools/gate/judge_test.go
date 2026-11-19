package main

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
	"time"
)

var limits = Limits{Test: time.Second, Package: 10 * time.Second, Whole: 3 * time.Minute}

// ok is a run of two packages: x/b, whose one test passes, and x/c, which has nothing to test.
const ok = `{"Action":"start","Package":"x/b"}
{"Action":"run","Package":"x/b","Test":"TestG"}
{"Action":"output","Package":"x/b","Test":"TestG","Output":"=== RUN   TestG\n","OutputType":"frame"}
{"Action":"pass","Package":"x/b","Test":"TestG","Elapsed":0}
{"Action":"output","Package":"x/b","Output":"ok  \tx/b\t0.2s\tcoverage: 100.0% of statements\n"}
{"Action":"pass","Package":"x/b","Elapsed":0.2}
{"Action":"start","Package":"x/c"}
{"Action":"skip","Package":"x/c","Elapsed":0}
`

const covered = "mode: set\nx/b/b.go:4.2,4.11 1 1\n"

// said is the line in which a package's tests say what they covered.
const said = `{"Action":"output","Package":"x/b","Output":"coverage: 100.0% of statements\n"}
`

// judge gives what Judge refuses in a run's events and profile, for a gate that took a second and a go test that
// ended with status 0, and fails the test on an error.
func judge(t *testing.T, events, profile string) []Finding {
	t.Helper()
	verdict, err := Judge(strings.NewReader(events), strings.NewReader(profile), time.Second, 0, limits)
	if err != nil {
		t.Fatalf("Judge gave the error %v", err)
	}
	return verdict.Findings
}

// want fails the test unless the findings are exactly those wanted, in their order.
func want(t *testing.T, got []Finding, wanted ...Finding) {
	t.Helper()
	if len(got) == 0 && len(wanted) == 0 {
		return
	}
	if !reflect.DeepEqual(got, wanted) {
		t.Errorf("findings:\n got  %q\n want %q", got, wanted)
	}
}

func TestAPassingRunHasNoFindings(t *testing.T) {
	verdict, err := Judge(strings.NewReader(ok), strings.NewReader(covered), time.Second, 0, limits)
	wanted := Verdict{Tested: 1, Bare: 1, Tests: 1}
	if err != nil || !reflect.DeepEqual(verdict, wanted) {
		t.Errorf("Judge = %+v, %v; want %+v and no error", verdict, err, wanted)
	}
}

func TestAFailedTestSaysWhatTheTestSaid(t *testing.T) {
	// A test's own lines are every line of its output that go test does not mark as a frame of its own.
	got := judge(t, `{"Action":"start","Package":"x/b"}
{"Action":"output","Package":"x/b","Test":"TestG","Output":"=== RUN   TestG\n","OutputType":"frame"}
{"Action":"output","Package":"x/b","Test":"TestG","Output":"    b_test.go:9: G(1) = 0; want 1\n"}
{"Action":"output","Package":"x/b","Test":"TestG","Output":"    b_test.go:12: G(0) = 1; want 0\n"}
{"Action":"output","Package":"x/b","Test":"TestG","Output":"--- FAIL: TestG (0.00s)\n","OutputType":"frame"}
{"Action":"fail","Package":"x/b","Test":"TestG","Elapsed":0}
{"Action":"output","Package":"x/b","Test":"TestH","Output":"    b_test.go:20: a line of a test that passes\n"}
{"Action":"pass","Package":"x/b","Test":"TestH","Elapsed":0}
{"Action":"fail","Package":"x/b","Elapsed":0.2}
`, covered)
	want(t, got, Finding{"test", "x/b.TestG", "b_test.go:9: G(1) = 0; want 1\nb_test.go:12: G(0) = 1; want 0"})
}

func TestATestThatPanicsSaysItsFirstLines(t *testing.T) {
	events := `{"Action":"start","Package":"x/b"}
`
	for _, line := range []string{"panic: assignment to entry in nil map", "", "goroutine 19 [running]:", "three", "four",
		"five", "six", "seven", "eight", "nine", "ten"} {
		events += `{"Action":"output","Package":"x/b","Test":"TestG","Output":"` + line + `\n"}` + "\n"
	}
	events += `{"Action":"fail","Package":"x/b","Test":"TestG","Elapsed":0}
{"Action":"fail","Package":"x/b","Elapsed":0.2}
`
	want(t, judge(t, events, covered), Finding{"test", "x/b.TestG",
		"panic: assignment to entry in nil map\ngoroutine 19 [running]:\nthree\nfour\nfive\nsix\nseven\neight\n… and 2 more line(s)"})
}

func TestAFailedSubtestIsReportedAndTheTestsAboveItAreNot(t *testing.T) {
	got := judge(t, `{"Action":"start","Package":"x/b"}
{"Action":"output","Package":"x/b","Test":"TestG/a/deep","Output":"    b_test.go:9: wrong\n"}
{"Action":"fail","Package":"x/b","Test":"TestG/a/deep","Elapsed":0}
{"Action":"fail","Package":"x/b","Test":"TestG/a","Elapsed":0}
{"Action":"pass","Package":"x/b","Test":"TestG/b","Elapsed":0}
{"Action":"fail","Package":"x/b","Test":"TestG","Elapsed":0}
{"Action":"fail","Package":"x/b","Elapsed":0.2}
`, covered)
	want(t, got, Finding{"test", "x/b.TestG/a/deep", "b_test.go:9: wrong"})
}

func TestATestThatSkipsItselfIsAFinding(t *testing.T) {
	got := judge(t, `{"Action":"start","Package":"x/b"}
{"Action":"output","Package":"x/b","Test":"TestSkips","Output":"    b_test.go:5: not today\n"}
{"Action":"output","Package":"x/b","Test":"TestSkips","Output":"--- SKIP: TestSkips (0.00s)\n","OutputType":"frame"}
{"Action":"skip","Package":"x/b","Test":"TestSkips","Elapsed":0}
{"Action":"skip","Package":"x/b","Test":"TestQuiet","Elapsed":0}
`+said+`{"Action":"pass","Package":"x/b","Elapsed":0.2}
`, covered)
	want(t, got, Finding{"skip", "x/b.TestSkips", "b_test.go:5: not today"}, Finding{"skip", "x/b.TestQuiet", "it skipped itself"})
}

func TestAPackageThatDoesNotBuildSaysWhatTheCompilerSaid(t *testing.T) {
	got := judge(t, `{"ImportPath":"x/d [x/d.test]","Action":"build-output","Output":"# x/d [x/d.test]\n"}
{"ImportPath":"x/d [x/d.test]","Action":"build-output","Output":"d/d.go:3:17: undefined: missing\n"}
{"ImportPath":"x/d [x/d.test]","Action":"build-fail"}
{"Action":"start","Package":"x/d"}
{"Action":"fail","Package":"x/d","Elapsed":0,"FailedBuild":"x/d [x/d.test]"}
`, "mode: set\n")
	want(t, got, Finding{"build", "x/d", "d/d.go:3:17: undefined: missing"})
}

func TestAPackageThatFailsOutsideATestSaysWhatItSaid(t *testing.T) {
	// As when TestMain panics: the package's own lines say why.
	got := judge(t, `{"Action":"start","Package":"x/b"}
{"Action":"output","Package":"x/b","Output":"panic: in TestMain\n"}
{"Action":"output","Package":"x/b","Output":"\n"}
{"Action":"output","Package":"x/b","Output":"goroutine 1 [running]:\n"}
{"Action":"output","Package":"x/b","Output":"FAIL\tx/b\t0.012s\n","OutputType":"frame"}
{"Action":"fail","Package":"x/b","Elapsed":0.2}
`, covered)
	want(t, got, Finding{"package", "x/b", "panic: in TestMain\ngoroutine 1 [running]:"})
}

func TestAPackageThatFailsAndSaysNothingIsAFinding(t *testing.T) {
	got := judge(t, `{"Action":"start","Package":"x/b"}
{"Action":"fail","Package":"x/b","Elapsed":0.2}
`, covered)
	want(t, got, Finding{"package", "x/b", "it failed outside a test"})
}

func TestAPackageWhoseTestsNeverSaidWhatTheyCoveredIsAFinding(t *testing.T) {
	// As when TestMain exits without calling m.Run: go test says ok, runs no test, and leaves the package out of
	// the profile, so nothing else would show that its statements were never run.
	got := judge(t, `{"Action":"start","Package":"x/b"}
{"Action":"output","Package":"x/b","Output":"ok  \tx/b\t0.009s\n"}
{"Action":"pass","Package":"x/b","Elapsed":0.01}
`, "mode: set\n")
	want(t, got, Finding{"coverage", "x/b", "its tests ended without saying what they covered: is m.Run called?"})
}

func TestAPackageThatNeverEndedIsAFinding(t *testing.T) {
	// As when a test ends the program, or go test is stopped: the package started, and has no end of its own.
	got := judge(t, `{"Action":"start","Package":"x/b"}
{"Action":"run","Package":"x/b","Test":"TestExits"}
{"Action":"output","Package":"x/b","Test":"TestExits","Output":"ok  \tx/b\t0.012s\n"}
{"Action":"pass","Package":"x/b","Test":"TestExits","Elapsed":0.012}
`, "mode: set\n")
	want(t, got, Finding{"package", "x/b", "its tests never ended: go test stopped, or a test ended the program"})
}

func TestAStatementNoTestRunsIsAFinding(t *testing.T) {
	// x/a has code and no test file: go test passes it, and its statement stands in the profile with a count of 0.
	got := judge(t, `{"Action":"start","Package":"x/a"}
{"Action":"output","Package":"x/a","Output":"\tx/a\t\tcoverage: 0.0% of statements\n"}
{"Action":"pass","Package":"x/a","Elapsed":0.5}
`, "mode: set\nx/a/a.go:4.16,4.26 1 0\n")
	want(t, got, Finding{"coverage", "x/a/a.go:4", "1 statement(s) no test runs"})
}

func TestARunWithNoProfileIsAFinding(t *testing.T) {
	verdict, err := Judge(strings.NewReader(ok), nil, time.Second, 0, limits)
	if err != nil {
		t.Fatal(err)
	}
	want(t, verdict.Findings, Finding{"coverage", "the profile", "go test wrote none that can be read"})
}

func TestATestOverASecondIsSlowAndItsSubtestIsNotCountedTwice(t *testing.T) {
	got := judge(t, `{"Action":"start","Package":"x/b"}
{"Action":"pass","Package":"x/b","Test":"TestSlow/sub","Elapsed":1.1}
{"Action":"pass","Package":"x/b","Test":"TestSlow","Elapsed":1.1}
`+said+`{"Action":"pass","Package":"x/b","Elapsed":1.2}
`, covered)
	want(t, got, Finding{"slow", "x/b.TestSlow", "1.1s, over 1s"})
}

func TestAPackageOverTenSecondsIsSlow(t *testing.T) {
	got := judge(t, `{"Action":"start","Package":"x/b"}
`+said+`{"Action":"pass","Package":"x/b","Elapsed":10.5}
`, covered)
	want(t, got, Finding{"slow", "x/b", "10.5s, over 10s"})
}

func TestAPackageIsSlowByItsTestsWhenTheCacheGivesItNoTime(t *testing.T) {
	// A package go test answers from its cache has no time of its own, and its tests keep theirs.
	events := `{"Action":"start","Package":"x/b"}
`
	for _, name := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"} {
		events += `{"Action":"pass","Package":"x/b","Test":"Test` + name + `","Elapsed":0.9}` + "\n"
	}
	events += said + `{"Action":"pass","Package":"x/b","Elapsed":0}
`
	want(t, judge(t, events, covered), Finding{"slow", "x/b", "10.8s, over 10s"})
}

func TestAGateOverThreeMinutesIsOverBudget(t *testing.T) {
	verdict, err := Judge(strings.NewReader(ok), strings.NewReader(covered), 181*time.Second, 0, limits)
	if err != nil {
		t.Fatal(err)
	}
	want(t, verdict.Findings, Finding{"budget", "the gate", "3m1s, over 3m0s"})
}

func TestARunThatPrintedNothingIsAFinding(t *testing.T) {
	// As when Go is missing: nothing ran, so nothing may pass. There is no profile then either.
	verdict, err := Judge(strings.NewReader(""), nil, time.Second, 0, limits)
	if err != nil {
		t.Fatal(err)
	}
	want(t, verdict.Findings, Finding{"empty", "go test", "it printed nothing"})
}

func TestGoTestsStatusIsAFindingWhenNothingElseSaysWhy(t *testing.T) {
	// As when go test is stopped between two packages: every package it reached ended well, and it did not.
	verdict, err := Judge(strings.NewReader(ok), strings.NewReader(covered), time.Second, 1, limits)
	if err != nil {
		t.Fatal(err)
	}
	want(t, verdict.Findings, Finding{"go test", "go test", "it ended with status 1, and nothing above says why"})
}

func TestGoTestsStatusIsNoFindingBesideAnother(t *testing.T) {
	events := `{"Action":"start","Package":"x/b"}
{"Action":"fail","Package":"x/b","Test":"TestG","Elapsed":0}
{"Action":"fail","Package":"x/b","Elapsed":0.2}
`
	verdict, err := Judge(strings.NewReader(events), strings.NewReader(covered), time.Second, 1, limits)
	if err != nil {
		t.Fatal(err)
	}
	want(t, verdict.Findings, Finding{"test", "x/b.TestG", ""})
}

func TestALineThatIsNoEventIsAnError(t *testing.T) {
	_, err := Judge(strings.NewReader("FAIL\n"), strings.NewReader(covered), time.Second, 0, limits)
	if err == nil || !strings.Contains(err.Error(), "line 1") {
		t.Errorf("Judge gave the error %v; want one that names line 1", err)
	}
}

func TestEventsThatBreakOffAreAnError(t *testing.T) {
	broken := io.MultiReader(strings.NewReader(ok), iotest.ErrReader(errors.New("the pipe broke")))
	_, err := Judge(broken, strings.NewReader(covered), time.Second, 0, limits)
	if err == nil || !strings.Contains(err.Error(), "the pipe broke") {
		t.Errorf("Judge gave the error %v; want the reader's own", err)
	}
}

func TestAProfileThatBreaksOffIsAnError(t *testing.T) {
	broken := io.MultiReader(strings.NewReader(covered), iotest.ErrReader(errors.New("the disk broke")))
	_, err := Judge(strings.NewReader(ok), broken, time.Second, 0, limits)
	if err == nil || !strings.Contains(err.Error(), "the disk broke") {
		t.Errorf("Judge gave the error %v; want the reader's own", err)
	}
}

func TestAProfileLineThatCannotBeReadIsAnError(t *testing.T) {
	for _, profile := range []string{
		"mode: set\nnonsense\n",
		"mode: set\nx/b/b.go:4.2,4.11 one 1\n",
		"mode: set\nx/b/b.go:4.2,4.11 1 many\n",
		"mode: set\nx/b/b.go:four.2,4.11 1 1\n",
		"x/b/b.go:4.2,4.11 1 1\n",
	} {
		_, err := Judge(strings.NewReader(ok), strings.NewReader(profile), time.Second, 0, limits)
		if err == nil {
			t.Errorf("Judge took the profile %q; want an error", profile)
		}
	}
}
