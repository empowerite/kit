<!-- kit:generated -- kit welcome's fourth step, the lay of the land, from the kit package -->
<!-- see: "Changing what kit adds" in .kit/1st.md, to replace it -->
# The lay of the land

Work moves from an issue to a release. `kit flow` draws the whole trip, one stage per heading. Each part below
ends with the command that shows it live.

`kit flow`

## The backlog

Work starts from an issue. An open issue with no pull request yet waits in the backlog.

`kit flow backlog`

## Underway

`kit work <N>` starts issue N. It makes a branch linked to the issue, a worktree for it, and a draft pull request.
The issue is then underway.

`kit flow underway`

## The worktree

A worktree is a checkout of its own, under `worktrees/`, on the issue's branch. The work happens there.

- `kit commit` commits what is staged. It runs the linters and checks the subject first.
- `kit submit` commits the same way, then pushes the branch.
- `kit rebase` brings the branch up to date with the default branch.

`kit flow worktrees`

## The pull request

`kit work` opened it as a draft. Mark it ready for review when the work is done. A person merges it. Nothing
reaches the default branch any other way.

`kit flow changes`

## The build

The code forge runs the delivered CI job on each pull request into the default branch. A pull request's checks show
on its row. The runs not owned by any pull request, such as those on the default branch, show as builds. `kit gate`
runs the same gate on your machine.

`kit flow builds`

## The release

`kit release <version>` tags the current commit as `release/<version>` and pushes the tag. The repository's
release workflow, where it has one, publishes it.

`kit flow releases`

## The desk

The desk is your root checkout, where you cloned the repository. It is yours: you read, review, and try things
there. No issue's work happens on it, so it stays as you left it.

Agents never change the desk. Each agent works in its own worktree, so two agents never step on each other, or on
you.

`kit flow desk`

## Incoming

A fetch lands on the desk. Incoming shows what it brought from each remote: new commits, and branches new and gone.

`kit flow incoming`
