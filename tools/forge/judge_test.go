package main

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
)

// leaned is what a gate might lean on: a rule with no values, one whose values must be so, one that names a list
// of plain values, and one that names a list of things.
const leaned = `{"rules": [
  {"type": "required_linear_history", "that": "main's history is linear"},
  {"type": "pull_request", "that": "a change reaches main only by a squashed pull request",
   "parameters": {"allowed_merge_methods": ["squash"]}},
  {"type": "required_status_checks", "that": "a pull request passes the gate, and is up to date, before it merges",
   "parameters": {"strict_required_status_checks_policy": true, "required_status_checks": [{"context": "gate"}]}}
]}`

// held is what GitHub gives of a branch that holds all of it, with more than was asked beside.
const held = `[
  {"type": "deletion", "parameters": null, "ruleset_id": 7},
  {"type": "required_linear_history", "parameters": null, "ruleset_id": 7},
  {"type": "pull_request", "ruleset_id": 7,
   "parameters": {"allowed_merge_methods": ["squash"], "required_approving_review_count": 0}},
  {"type": "required_status_checks", "ruleset_id": 7,
   "parameters": {"strict_required_status_checks_policy": true, "do_not_enforce_on_create": false,
                  "required_status_checks": [{"context": "lint"}, {"context": "gate", "integration_id": 15368}]}}
]`

// judge gives what Judge finds, and fails the test on an error.
func judge(t *testing.T, rules, expect string) []Finding {
	t.Helper()
	findings, _, err := Judge(strings.NewReader(rules), strings.NewReader(expect))
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

func TestABranchThatHoldsEveryRuleHasNoFindings(t *testing.T) {
	findings, rules, err := Judge(strings.NewReader(held), strings.NewReader(leaned))
	if err != nil || len(findings) != 0 || rules != 3 {
		t.Errorf("Judge = %q, %d rules, %v; want no findings, 3 rules, no error", findings, rules, err)
	}
}

func TestARuleThatDoesNotApplyIsAFinding(t *testing.T) {
	got := judge(t, `[{"type": "deletion", "parameters": null}]`,
		`{"rules": [{"type": "required_linear_history", "that": "main's history is linear"}]}`)
	want(t, got, Finding{"main's history is linear", "no rule of the kind required_linear_history applies"})
}

func TestAListOfPlainValuesMustBeTheSameList(t *testing.T) {
	// Squash among three ways to merge is not squash alone.
	rules := `[{"type": "pull_request", "parameters": {"allowed_merge_methods": ["merge", "squash", "rebase"]}}]`
	expect := `{"rules": [{"type": "pull_request", "that": "a change is squashed",
		"parameters": {"allowed_merge_methods": ["squash"]}}]}`
	want(t, judge(t, rules, expect), Finding{"a change is squashed",
		`the rule pull_request has allowed_merge_methods ["merge","squash","rebase"], and not ["squash"]`})
	inAnotherOrder := `[{"type": "x", "parameters": {"list": ["b", "a"]}}]`
	want(t, judge(t, inAnotherOrder, `{"rules": [{"type": "x", "that": "t", "parameters": {"list": ["a", "b"]}}]}`))
}

func TestAListOfThingsMustHoldEachThingAsked(t *testing.T) {
	rules := `[{"type": "required_status_checks", "parameters": {"required_status_checks": [{"context": "lint"}]}}]`
	expect := `{"rules": [{"type": "required_status_checks", "that": "the gate is required",
		"parameters": {"required_status_checks": [{"context": "gate"}]}}]}`
	want(t, judge(t, rules, expect), Finding{"the gate is required",
		`the rule required_status_checks has required_status_checks [{"context":"lint"}], which holds no {"context":"gate"}`})
}

func TestAValueThatDiffersIsAFinding(t *testing.T) {
	for _, c := range []struct{ parameters, wrong string }{
		{`{"strict": false}`, `the rule r has strict false, and not true`},
		{`{}`, `the rule r has no strict, and not true`},
		{`null`, `the rule r has no strict, and not true`},
		{`{"strict": "true"}`, `the rule r has strict "true", and not true`},
	} {
		got := judge(t, `[{"type": "r", "parameters": `+c.parameters+`}]`,
			`{"rules": [{"type": "r", "that": "t", "parameters": {"strict": true}}]}`)
		want(t, got, Finding{"t", c.wrong})
	}
}

func TestOfSeveralValuesThatDifferTheFirstByNameIsSaid(t *testing.T) {
	// So that the same rules always give the same words.
	got := judge(t, `[{"type": "r", "parameters": {"a": 1, "b": 1, "c": 1}}]`,
		`{"rules": [{"type": "r", "that": "t", "parameters": {"c": 2, "a": 2, "b": 2}}]}`)
	want(t, got, Finding{"t", "the rule r has a 1, and not 2"})
}

func TestAValueIsFoundInsideOtherValues(t *testing.T) {
	rules := `[{"type": "r", "parameters": {"outer": {"inner": 1, "more": 2}}}]`
	want(t, judge(t, rules, `{"rules": [{"type": "r", "that": "t", "parameters": {"outer": {"inner": 1}}}]}`))
	want(t, judge(t, rules, `{"rules": [{"type": "r", "that": "t", "parameters": {"outer": {"inner": 2}}}]}`),
		Finding{"t", "the rule r has outer.inner 1, and not 2"})
	want(t, judge(t, `[{"type": "r", "parameters": {"outer": 3}}]`,
		`{"rules": [{"type": "r", "that": "t", "parameters": {"outer": {"inner": 2}}}]}`),
		Finding{"t", `the rule r has outer 3, and not {"inner":2}`})
	want(t, judge(t, `[{"type": "r", "parameters": {"list": 3}}]`,
		`{"rules": [{"type": "r", "that": "t", "parameters": {"list": ["a"]}}]}`),
		Finding{"t", `the rule r has list 3, and not ["a"]`})
}

func TestOneRuleOfTheKindThatHoldsIsEnough(t *testing.T) {
	// Two rulesets can each bring a rule of one kind: the branch holds what either brings.
	rules := `[{"type": "pull_request", "parameters": {"allowed_merge_methods": ["merge", "squash"]}},
	           {"type": "pull_request", "parameters": {"allowed_merge_methods": ["squash"]}}]`
	expect := `{"rules": [{"type": "pull_request", "that": "t", "parameters": {"allowed_merge_methods": ["squash"]}}]}`
	want(t, judge(t, rules, expect))
}

func TestWhatGitHubGivesOfKitTodayIsJudged(t *testing.T) {
	// testdata/today.json is what GitHub gave of empowerite/kit's main on 2026-11-18, before its gate was required
	// and while its rule on pull requests named three ways to merge. expect.json is what this repository leans on.
	today, err := os.Open("testdata/today.json")
	if err != nil {
		t.Fatal(err)
	}
	defer today.Close()
	expect, err := os.Open("expect.json")
	if err != nil {
		t.Fatal(err)
	}
	defer expect.Close()
	findings, rules, err := Judge(today, expect)
	if err != nil {
		t.Fatal(err)
	}
	if rules != 6 {
		t.Errorf("expect.json has %d rules; want 6", rules)
	}
	want(t, findings,
		Finding{"a change reaches main only by a pull request, and only squashed",
			`the rule pull_request has allowed_merge_methods ["merge","squash","rebase"], and not ["squash"]`},
		Finding{"a pull request passes the gate, and is up to date with main, before it merges",
			"no rule of the kind required_status_checks applies"})
}

func TestWhatCannotBeReadIsAnError(t *testing.T) {
	broken := iotest.ErrReader(errors.New("the pipe broke"))
	for _, c := range []struct {
		name          string
		rules, expect string
	}{
		{"rules that are no JSON", "Not Found", leaned},
		{"rules that are no list", `{"message": "Not Found"}`, leaned},
		{"an expectation that is no JSON", held, "{"},
		{"an expectation with no rules", held, `{"rules": []}`},
		{"an expected rule with no kind", held, `{"rules": [{"that": "t"}]}`},
		{"an expected rule that does not say what it holds", held, `{"rules": [{"type": "deletion"}]}`},
	} {
		if _, _, err := Judge(strings.NewReader(c.rules), strings.NewReader(c.expect)); err == nil {
			t.Errorf("%s: Judge gave no error", c.name)
		}
	}
	_, _, err := Judge(broken, strings.NewReader(leaned))
	if err == nil || !strings.Contains(err.Error(), "the pipe broke") {
		t.Errorf("rules that break off: Judge gave the error %v", err)
	}
	_, _, err = Judge(strings.NewReader(held), broken)
	if err == nil || !strings.Contains(err.Error(), "the pipe broke") {
		t.Errorf("an expectation that breaks off: Judge gave the error %v", err)
	}
}
