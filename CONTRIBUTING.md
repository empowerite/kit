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

- `go vet` objects, or `gofmt` would change a file
- a test fails, or a package does not build
- a test skips itself: a skip passes no gate
- a statement in a package is run by none of that package's tests
- a package's tests end without running, or `go test` stops short
- a test takes over a second, or a package's tests over ten
- the gate itself takes over 3 minutes
- an import goes the wrong way through the layers of the design, or a package is in no layer
- a package that is no adapter imports what starts a process, opens a connection, or touches a file
- a test imports what starts a process or opens a connection, unless its file begins `//go:build nightly`
- a file is in only some builds, or a `go.mod` stands below the root

The rule on imports sees what a file imports, and not what it calls: a test may import `os` for its files, and
nothing but a review can see a call of `os.StartProcess`.

`go test ./...` is the quick loop while working.

CI runs the gate on every pull request, and not on `main`. A pull request is up to date with `main` before it
merges, and the merge squashes, so the tree that lands is the tree the gate judged. A pull request's runs go one at
a time, and none is ended for a newer push: each starts from Go's build cache as the last one left it. Each run
keeps, as artifacts of the run:

- the gate's verdict, whether it held or refused, and with it `go test`'s events and its coverage profile
- kit, as `bin/build` builds it for macOS and Linux, where the gate held

Scripts are bash. A test in one is written `[[ … ]]`, and arithmetic `(( … ))`, never `[ … ]`.

## The owner's approval

The owner approves by merging. GitHub lets no one approve their own pull request, and the owner is the only person
here, so there is no other approval to give.

- A pull request that is ready asks for the owner's review. Whoever did the work makes it ready when the work is
  complete and its checks pass, and asks for the owner's review where GitHub takes the request.
- To approve a change is to merge its pull request.
- Not to accept a change is to make its pull request a draft again, and to say why in a comment on it. Whoever did
  the work changes it, answers the comment, and makes it ready again.
- What the owner says of a pull request is written on the pull request, so that the record stays with the change.
  Where the owner says it elsewhere, as in a session with an agent, the agent writes it on the pull request in the
  owner's words, and says that it writes for the owner.
- To approve the charter or a design is to merge the pull request that brings it or changes it. Such a pull request
  changes nothing else, so the merge says exactly what was approved.
- To accept an issue is to give it a milestone.

An agent's pull requests are opened under the owner's account until agents have an identity of their own. GitHub
then offers the owner no review of them, and takes no request for one, so these are all the signals there are:
ready, draft, the comments, and the merge.

Only the first commit reached `main` without a pull request, since an empty repository can take none. It holds the
README, the license, the charter, and this page, and it was pushed at the owner's word.
