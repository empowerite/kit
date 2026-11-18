package main

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

var limits = Limits{Test: time.Second, Package: 10 * time.Second, Whole: 3 * time.Minute}

const ok = `{"Action":"start","Package":"x/b"}
{"Action":"pass","Package":"x/b","Test":"TestG","Elapsed":0}
{"Action":"pass","Package":"x/b","Elapsed":0.2}
{"Action":"skip","Package":"x/c","Elapsed":0}
`

const covered = "mode: set\nx/b/b.go:4.2,4.11 1 1\n"

// judge gives what Judge finds in a run's events and profile that took a second, and fails the test on an error.
func judge(t *testing.T, events, profile string) []Finding {
	t.Helper()
	findings, _, _, err := Judge(strings.NewReader(events), strings.NewReader(profile), time.Second, limits)
	if err != nil {
		t.Fatalf("Judge gave the error %v", err)
	}
	return findings
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
	findings, packages, tests, err := Judge(strings.NewReader(ok), strings.NewReader(covered), time.Second, limits)
	if err != nil || len(findings) != 0 || packages != 2 || tests != 1 {
		t.Errorf("Judge = %q, %d packages, %d tests, %v; want no findings, 2 packages, 1 test, no error",
			findings, packages, tests, err)
	}
}

func TestAFailedTestIsAFinding(t *testing.T) {
	got := judge(t, `{"Action":"fail","Package":"x/b","Test":"TestG","Elapsed":0}
{"Action":"fail","Package":"x/b","Elapsed":0.2}
`, covered)
	want(t, got, Finding{"test", "x/b.TestG", ""})
}

func TestAPackageThatDoesNotBuildSaysWhatTheCompilerSaid(t *testing.T) {
	got := judge(t, `{"ImportPath":"x/d [x/d.test]","Action":"build-output","Output":"# x/d [x/d.test]\n"}
{"ImportPath":"x/d [x/d.test]","Action":"build-output","Output":"d/d.go:3:17: undefined: missing\n"}
{"ImportPath":"x/d [x/d.test]","Action":"build-fail"}
{"Action":"fail","Package":"x/d","Elapsed":0,"FailedBuild":"x/d [x/d.test]"}
`, "mode: set\n")
	want(t, got, Finding{"build", "x/d", "d/d.go:3:17: undefined: missing\n"})
}

func TestAPackageThatFailsOutsideATestIsAFinding(t *testing.T) {
	got := judge(t, `{"Action":"pass","Package":"x/b","Test":"TestG","Elapsed":0}
{"Action":"fail","Package":"x/b","Elapsed":0.2}
`, covered)
	want(t, got, Finding{"package", "x/b", "it failed outside a test"})
}

func TestAStatementNoTestRunsIsAFinding(t *testing.T) {
	// x/a has code and no test file: go test passes it, and its statement stands in the profile with a count of 0.
	got := judge(t, `{"Action":"pass","Package":"x/a","Elapsed":0.5}
`, "mode: set\nx/a/a.go:4.16,4.26 1 0\n")
	want(t, got, Finding{"coverage", "x/a/a.go:4", "1 statement(s) no test runs"})
}

func TestATestOverASecondIsSlowAndItsSubtestIsNotCountedTwice(t *testing.T) {
	got := judge(t, `{"Action":"pass","Package":"x/b","Test":"TestSlow/sub","Elapsed":1.1}
{"Action":"pass","Package":"x/b","Test":"TestSlow","Elapsed":1.1}
{"Action":"pass","Package":"x/b","Elapsed":1.2}
`, covered)
	want(t, got, Finding{"slow", "x/b.TestSlow", "1.1s, over 1s"})
}

func TestAPackageOverTenSecondsIsSlow(t *testing.T) {
	got := judge(t, `{"Action":"pass","Package":"x/b","Elapsed":10.5}
`, covered)
	want(t, got, Finding{"slow", "x/b", "10.5s, over 10s"})
}

func TestAGateOverThreeMinutesIsOverBudget(t *testing.T) {
	got, _, _, err := Judge(strings.NewReader(ok), strings.NewReader(covered), 181*time.Second, limits)
	if err != nil {
		t.Fatal(err)
	}
	want(t, got, Finding{"budget", "the gate", "3m1s, over 3m0s"})
}

func TestARunThatPrintedNothingIsAFinding(t *testing.T) {
	// As when Go is missing: nothing ran, so nothing may pass. The profile is then empty too.
	got := judge(t, "", "")
	want(t, got, Finding{"empty", "go test", "it printed nothing"})
}

func TestALineThatIsNoEventIsAnError(t *testing.T) {
	_, _, _, err := Judge(strings.NewReader("FAIL\n"), strings.NewReader(covered), time.Second, limits)
	if err == nil || !strings.Contains(err.Error(), "line 1") {
		t.Errorf("Judge gave the error %v; want one that names line 1", err)
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
		_, _, _, err := Judge(strings.NewReader(ok), strings.NewReader(profile), time.Second, limits)
		if err == nil {
			t.Errorf("Judge took the profile %q; want an error", profile)
		}
	}
}
