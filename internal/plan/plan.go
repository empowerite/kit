// Package plan holds operations, plans, and apply. It may import model, lifecycle, and ports.
//
// Every act runs the same way. Its plan says each step: what changes, in which system, and whether it can be undone,
// and nothing has changed yet. A person approves the plan with proof of presence, bound to the plan's hash, which
// covers the facts each step depends on. Apply refuses when those facts changed since, and stops at a step that
// fails, saying which steps are done and which are not. A plan whose steps only place what kit owns, and change
// nothing that git sees, is hydrated with no approval.
package plan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/empowerite/kit/internal/ports"
)

// Step is what a plan shows of one operation.
type Step struct {
	System   string `json:"system"` // "files", ".kit", "cache", or "trust"
	Does     string `json:"does"`
	Command  string `json:"command,omitempty"` // the command line that does the same, where there is one
	Undoable bool   `json:"undoable"`
	Hydrates bool   `json:"hydrates,omitempty"` // it places only what kit owns, and changes nothing git sees
}

// Operation is one step of a plan. Facts is a digest of what the step depends on, as it stands now.
type Operation interface {
	Step() Step
	Facts(ctx context.Context) (string, error)
	Apply(ctx context.Context) error
}

// Plan is an act's steps, in the order they run.
type Plan struct {
	Act        string // the act's words, as "tools add"
	Operations []Operation
}

// Approval is a person's approval of a plan: the plan's hash, and the proof that the person was there.
type Approval struct {
	Hash  string
	Proof ports.Proof
}

// Result is what apply did: the steps done, those not done, and why it stopped.
type Result struct {
	Done, NotDone []Step
	Err           error
}

// Doc is a plan as kit prints it.
type Doc struct {
	Act   string `json:"act"`
	Hash  string `json:"hash"`
	Steps []Step `json:"steps"`
}

// ErrChanged is what apply gives when the facts a plan depends on changed after it was approved.
var ErrChanged = errors.New("the facts changed since the plan was made: make the plan again")

// Hash gives the plan's hash: of its act, and of each step and the facts it depends on.
func (p Plan) Hash(ctx context.Context) (string, error) {
	sum := sha256.New()
	fmt.Fprintf(sum, "act %q\n", p.Act)
	for _, o := range p.Operations {
		facts, err := o.Facts(ctx)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(sum, "step %+v\nfacts %q\n", o.Step(), facts)
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

// Doc gives the plan as kit prints it.
func (p Plan) Doc(ctx context.Context) (Doc, error) {
	h, err := p.Hash(ctx)
	if err != nil {
		return Doc{}, err
	}
	return Doc{p.Act, h, p.steps()}, nil
}

func (p Plan) steps() []Step {
	steps := []Step{}
	for _, o := range p.Operations {
		steps = append(steps, o.Step())
	}
	return steps
}

// Approve asks the person to approve the plan, with proof of presence, and binds the proof to the plan's hash.
func Approve(ctx context.Context, p Plan, presence ports.Presence) (Approval, error) {
	h, err := p.Hash(ctx)
	if err != nil {
		return Approval{}, err
	}
	proof, err := presence.Prove(ctx, fmt.Sprintf("approve kit %s, plan %s", p.Act, h[:12]), h)
	if err != nil {
		return Approval{}, err
	}
	return Approval{h, proof}, nil
}

// Apply runs an approved plan's steps in order. It refuses a plan whose facts changed since it was approved, and stops
// at the first step that fails.
func Apply(ctx context.Context, p Plan, a Approval) Result {
	h, err := p.Hash(ctx)
	switch {
	case err != nil:
		return Result{NotDone: p.steps(), Err: err}
	case h != a.Hash:
		return Result{NotDone: p.steps(), Err: ErrChanged}
	}
	return run(ctx, p)
}

// Hydrate runs a plan whose every step only places what kit owns, with no approval. A plan with any other step needs
// the person's.
func Hydrate(ctx context.Context, p Plan) Result {
	for _, s := range p.steps() {
		if !s.Hydrates {
			return Result{NotDone: p.steps(), Err: fmt.Errorf(
				"kit %s changes what git sees (%s), so it needs the person's approval", p.Act, s.Does)}
		}
	}
	if _, err := p.Hash(ctx); err != nil {
		return Result{NotDone: p.steps(), Err: err}
	}
	return run(ctx, p)
}

func run(ctx context.Context, p Plan) Result {
	var r Result
	for i, o := range p.Operations {
		if err := o.Apply(ctx); err != nil {
			r.NotDone, r.Err = p.steps()[i:], err
			return r
		}
		r.Done = append(r.Done, o.Step())
	}
	return r
}

// Print prints a plan as a person reads it.
func Print(w io.Writer, d Doc) {
	fmt.Fprintf(w, "Plan  kit %s, %s\n", d.Act, d.Hash[:12])
	width := 0
	for _, s := range d.Steps {
		width = max(width, len(s.System))
	}
	for _, s := range d.Steps {
		fmt.Fprintf(w, "  %-*s  %s\n", width, s.System, s.Does)
	}
}

// PrintResult prints what apply did.
func PrintResult(w io.Writer, r Result) {
	width := 0
	for _, s := range append(r.Done, r.NotDone...) {
		width = max(width, len(s.System))
	}
	for _, s := range r.Done {
		fmt.Fprintf(w, "  done      %-*s  %s\n", width, s.System, s.Does)
	}
	for _, s := range r.NotDone {
		fmt.Fprintf(w, "  not done  %-*s  %s\n", width, s.System, s.Does)
	}
	if r.Err != nil {
		fmt.Fprintf(w, "kit: %v\n", r.Err)
	}
}
