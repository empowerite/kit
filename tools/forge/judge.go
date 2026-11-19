package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
)

// Finding is one thing the gate leans on that the forge does not hold: what should hold, and what is wrong.
type Finding struct{ That, Wrong string }

// rule is a rule as GitHub gives it of a branch, and as an expectation names one. An expectation also says what the
// rule holds, in words that follow "that".
type rule struct {
	Type       string `json:"type"`
	That       string `json:"that"`
	Parameters any    `json:"parameters"`
}

// Judge holds the rules GitHub applies to a branch to those expected of it. It gives what does not hold, in the
// expectation's order, and how many rules were expected.
//
// rules is what GET /repos/{owner}/{repo}/rules/branches/{branch} answers: a list of rules, each with its type and
// its parameters. expect is {"rules": [...]}, each with its type, what it holds, and the parameters it must have. A
// rule may have more parameters than are expected of it, and a branch more rules.
func Judge(rules, expect io.Reader) (findings []Finding, expected int, err error) {
	var applied []rule
	if err := json.NewDecoder(rules).Decode(&applied); err != nil {
		return nil, 0, fmt.Errorf("the branch's rules cannot be read: %v", err)
	}
	var leaned struct{ Rules []rule }
	if err := json.NewDecoder(expect).Decode(&leaned); err != nil {
		return nil, 0, fmt.Errorf("what is expected cannot be read: %v", err)
	}
	if len(leaned.Rules) == 0 {
		return nil, 0, errors.New("what is expected names no rule")
	}
	for _, want := range leaned.Rules {
		if want.Type == "" || want.That == "" {
			return nil, 0, errors.New("an expected rule lacks its type, or what it holds")
		}
		wrong := "no rule of the kind " + want.Type + " applies"
		for _, got := range applied {
			if got.Type != want.Type {
				continue
			}
			if wrong = differs("", got.Parameters, want.Parameters); wrong == "" {
				break
			}
			wrong = "the rule " + want.Type + " has " + wrong
		}
		if wrong != "" {
			findings = append(findings, Finding{want.That, wrong})
		}
	}
	return findings, len(leaned.Rules), nil
}

// differs says how a value differs from the one wanted, or "" where it holds it. A set of named values holds one
// that is wanted where it has each name wanted, holding its value. A list of plain values holds one where it is the
// same list, in any order. A list of other things holds one where it has, for each thing wanted, a thing that holds
// it. path names the value, from the rule's parameters down.
func differs(path string, got, want any) string {
	switch want := want.(type) {
	case nil:
		return ""
	case map[string]any:
		has, named := got.(map[string]any)
		for _, name := range slices.Sorted(maps.Keys(want)) {
			value, inner := want[name], name
			if path != "" {
				inner = path + "." + name
			}
			if !named && got != nil {
				break
			}
			if _, there := has[name]; !there {
				return fmt.Sprintf("no %s, and not %s", inner, text(value))
			}
			if wrong := differs(inner, has[name], value); wrong != "" {
				return wrong
			}
		}
		if named || got == nil {
			return ""
		}
	case []any:
		has, listed := got.([]any)
		if !listed {
			break
		}
		if plain(want) {
			if len(has) == len(want) && within(has, want) {
				return ""
			}
			break
		}
		for _, thing := range want {
			if !within(has, []any{thing}) {
				return fmt.Sprintf("%s %s, which holds no %s", path, text(got), text(thing))
			}
		}
		return ""
	default:
		if got == want {
			return ""
		}
	}
	return fmt.Sprintf("%s %s, and not %s", path, text(got), text(want))
}

// plain says whether a list is of plain values: none of them a list or a set of named values.
func plain(list []any) bool {
	for _, value := range list {
		switch value.(type) {
		case map[string]any, []any:
			return false
		}
	}
	return true
}

// within says whether a list has, for each value wanted, a value that holds it.
func within(has, wanted []any) bool {
	for _, want := range wanted {
		found := false
		for _, got := range has {
			found = found || differs("", got, want) == ""
		}
		if !found {
			return false
		}
	}
	return true
}

// text writes a value as JSON.
func text(value any) string {
	written, _ := json.Marshal(value)
	return string(written)
}
