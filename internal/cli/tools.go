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
		"tools": func(ctx context.Context, call Call, out Out) int {
			d, r, err := opened(ctx, desk)
			var listed []toolchain.Listed
			if err == nil {
				listed, err = toolchain.Tools(d, r)
			}
			if err != nil {
				return fail(out, err)
			}
			if out.JSON {
				return Print(out, "tools", map[string]any{"tools": append([]toolchain.Listed{}, listed...)})
			}
			widths := [3]int{}
			for _, l := range listed {
				widths = [3]int{max(widths[0], len(l.Name)), max(widths[1], len(l.Version)), max(widths[2], len(l.Where))}
			}
			for _, l := range listed {
				fmt.Fprintf(out.Stdout, "  %-*s  %-*s  %-*s  %s\n", widths[0], l.Name, widths[1], l.Version, widths[2], l.Where, l.State)
			}
			return 0
		},
		"tools fetch": func(ctx context.Context, call Call, out Out) int {
			d, r, err := opened(ctx, desk)
			var p plan.Plan
			if err == nil {
				p, err = toolchain.Fetch(ctx, d, r)
			}
			if err != nil {
				return fail(out, err)
			}
			doc, _ := p.Doc(ctx)
			applied := plan.Hydrate(ctx, p)
			status := 0
			if applied.Err != nil {
				status = 1
			}
			if out.JSON {
				Print(out, "plan", map[string]any{"plan": doc, "result": resultOf(applied)})
				return status
			}
			plan.PrintResult(out.Stdout, applied)
			return status
		},
		"tools path": func(ctx context.Context, call Call, out Out) int {
			d, r, err := opened(ctx, desk)
			var path string
			if err == nil {
				path, err = toolchain.Path(ctx, d, r, call.Args[0])
			}
			if err != nil {
				return fail(out, err)
			}
			if out.JSON {
				return Print(out, "path", map[string]string{"tool": call.Args[0], "path": path})
			}
			fmt.Fprintln(out.Stdout, path)
			return 0
		},
		"exec": func(ctx context.Context, call Call, out Out) int {
			d, r, err := opened(ctx, desk)
			if err != nil {
				return fail(out, err)
			}
			if err := toolchain.Exec(ctx, d, r, call.Args); err != nil {
				fail(out, err)
				return 127
			}
			return 0
		},
		"env": func(ctx context.Context, call Call, out Out) int {
			shell := "bash"
			if given := call.Options["shell"]; given != nil {
				shell = given[0]
			}
			d, r, err := opened(ctx, desk)
			if err != nil {
				return fail(out, err)
			}
			said, err := toolchain.Env(shell, r.Root)
			if err != nil {
				fail(out, err)
				return 2
			}
			p, err := toolchain.Fetch(ctx, d, r)
			if err == nil {
				err = plan.Hydrate(ctx, p).Err
			}
			if err != nil {
				return fail(out, err)
			}
			if out.JSON {
				return Print(out, "env", map[string]any{"path_add": []string{r.Root + "/.kit/bin"}, "watch": []string{"kit.yml", "kit.lock"}})
			}
			fmt.Fprint(out.Stdout, said)
			return 0
		},
	}
}

// opened gives the desk, and the repository it is in.
func opened(ctx context.Context, desk func() (toolchain.Desk, error)) (toolchain.Desk, toolchain.Repo, error) {
	d, err := desk()
	if err != nil {
		return d, toolchain.Repo{}, err
	}
	r, err := toolchain.Open(ctx, d)
	return d, r, err
}

// resultOf gives what apply did, as JSON says it.
func resultOf(applied plan.Result) result {
	said := result{append([]plan.Step{}, applied.Done...), append([]plan.Step{}, applied.NotDone...), ""}
	if applied.Err != nil {
		said.Error = applied.Err.Error()
	}
	return said
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
	d, r, err := opened(ctx, desk)
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
		Print(out, "plan", map[string]any{"plan": doc, "result": resultOf(applied)})
		return status
	}
	if err != nil {
		return fail(out, err)
	}
	plan.PrintResult(out.Stdout, applied)
	return status
}
