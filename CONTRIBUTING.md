# How work is done here

This page says how work moves in this repository, who decides what, and where each decision is recorded. The steps
are taken with git and `gh`: kit does not run them.

## Who decides

- **The owner decides** what is worth doing, what the charter and a design say, and what lands on `main`.
- **An agent works.** It writes, tests, and proposes, on one issue, in a worktree of its own. It decides nothing: it
  never merges, and it never pushes to `main`.
- **A decision is recorded at GitHub**, where the issue or the pull request shows who made it and when.

## An issue

| It is | When | By |
| --- | --- | --- |
| proposed | it is open, and in no milestone | anyone, an agent too |
| accepted | it is in a milestone | the owner, or an agent at the owner's word |
| underway | it is assigned, and a draft pull request closes it | whoever does the work |
| in review | its pull request is ready | whoever does the work |
| done | its pull request is merged, which closes it | the owner |

- The assignee is the person who answers for the work. When an agent works for a person, that person is assigned.
- A part of an issue is its sub-issue, and an issue that must wait for another is blocked by it. Both are set at
  GitHub, and neither is written in a body.
- A lead issue is done when its sub-issues are.
- A milestone is one step of the order the owner set, and its title starts with its place in that order.

## A pull request

- One issue has one branch, one worktree, and one pull request, which starts as a draft.
- Its body is the report: what was done, the command that shows it and what the command printed, and what was not
  done.
- It is brought up to date by merging `main` into it. Nothing that was pushed is rewritten.
- Every commit is signed. The owner merges, and the merge squashes.

## The gate

`bin/gate` runs the gate, at a desk and in CI alike. It refuses when:

- `gofmt` would change a file, or `go vet` objects
- a test fails, or a package does not build
- a statement in a package is run by none of that package's tests
- a test takes over a second, or a package's tests over ten
- an import goes the wrong way through the layers of the design, or a package is in no layer
- the gate itself takes over 3 minutes

No test the gate runs starts a process or reaches the network. `go test ./...` is the quick loop while working.

## The owner's approval

The owner approves by merging. GitHub lets no one approve their own pull request, and the owner is the only person
here, so there is no other approval to give.

- To approve a change is to merge its pull request.
- To approve the charter or a design is to merge the pull request that brings it or changes it. Such a pull request
  changes nothing else, so the merge says exactly what was approved.
- To accept an issue is to give it a milestone.

Only the first commit reached `main` without a pull request, since an empty repository can take none. It holds the
README, the license, the charter, and this page, and it was pushed at the owner's word.
