package cli

import (
	"context"
	"fmt"

	"github.com/empowerite/kit/internal/plan"
	"github.com/empowerite/kit/internal/toolchain"
)

// Tools gives the handlers of the commands of the toolchain, on the desk that desk gives: cmd/kit's the real one, a
// test's its own.
func Tools(desk func() (toolchain.Desk, error)) map[string]Handler {
	return map[string]Handler{
		"tools add": func(ctx context.Context, call Call, out Out) int {
			return act(ctx, call, out, desk, func(d toolchain.Desk, r toolchain.Repo, list map[string]toolchain.Entry) (plan.Plan, error) {
				p, _, err := toolchain.Add(ctx, d, r, list, call.Args[0], call.Options["unconfirmed"] != nil)
				return p, err
			})
		},
		"tools lock": func(ctx context.Context, call Call, out Out) int {
			return act(ctx, call, out, desk, func(d toolchain.Desk, r toolchain.Repo, list map[string]toolchain.Entry) (plan.Plan, error) {
				return toolchain.Relock(ctx, d, r, list, call.Options["unconfirmed"] != nil)
			})
		},
	}
}

// fail says an error on stderr, and gives the status of a command that could not do what it was asked.
func fail(out Out, err error) int {
	fmt.Fprintf(out.Stderr, "kit: %v\n", err)
	return 1
}

// result is what apply did, as JSON says it.
type result struct {
	Done    []plan.Step `json:"done"`
	NotDone []plan.Step `json:"not_done"`
	Error   string      `json:"error"`
}

// act plans an act on the repository the desk is in, shows the plan, and, unless --plan, asks the person to approve
// it and applies it.
func act(ctx context.Context, call Call, out Out, desk func() (toolchain.Desk, error),
	planned func(toolchain.Desk, toolchain.Repo, map[string]toolchain.Entry) (plan.Plan, error)) int {
	d, err := desk()
	if err != nil {
		return fail(out, err)
	}
	r, err := toolchain.Open(ctx, d)
	if err != nil {
		return fail(out, err)
	}
	list, _ := toolchain.List()
	p, err := planned(d, r, list)
	if err != nil {
		return fail(out, err)
	}
	doc, _ := p.Doc(ctx)
	switch {
	case len(doc.Steps) == 0 && !out.JSON:
		fmt.Fprintln(out.Stdout, "kit: nothing to change: kit.yml and kit.lock hold it already")
		return 0
	case call.Options["plan"] != nil && out.JSON:
		return Print(out, "plan", map[string]any{"plan": doc})
	case !out.JSON:
		plan.Print(out.Stdout, doc)
	}
	if call.Options["plan"] != nil {
		return 0
	}
	var applied plan.Result
	approval, err := plan.Approve(ctx, p, d.Presence)
	if err != nil {
		applied = plan.Result{NotDone: doc.Steps, Err: err}
	} else {
		applied = plan.Apply(ctx, p, approval)
	}
	status := 0
	if applied.Err != nil {
		status = 1
	}
	if out.JSON {
		said := result{append([]plan.Step{}, applied.Done...), append([]plan.Step{}, applied.NotDone...), ""}
		if applied.Err != nil {
			said.Error = applied.Err.Error()
		}
		Print(out, "plan", map[string]any{"plan": doc, "result": said})
		return status
	}
	if err != nil {
		return fail(out, err)
	}
	plan.PrintResult(out.Stdout, applied)
	return status
}
