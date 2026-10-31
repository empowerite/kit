<!-- kit:generated -- the workflow from an issue to a merged change, from the workflow package -->
<!-- see: "Changing what kit adds" in .kit/1st.md, to replace it -->
# The workflow

This file is the kit's workflow: how work moves from an issue to a merged change. Every agent follows it in every
session, in any repository whose `.kit/` directory holds it, after [1st.md](1st.md), the entry file. The repository's
own rules in `AGENTS.md` apply as well. Where the two differ, `AGENTS.md` wins.

## From an issue to a merged change

1. Start from an issue. A change without an issue has not started.
2. `kit issue <N>` creates the following, and never switches the checkout you run it from:
   - the worktree `worktrees/<N>-<slug>`, hydrated: what the kit delivers under `.kit/` is in place before your
     first command there
   - a branch, linked to the issue under Development
   - a draft pull request
3. Work in that worktree. Commit with `kit commit`, with the subject `#<N> <area>: <subject>`; it supplies `#<N>`
   from the branch when the subject lacks it. `kit commit` warns when a commit is unsigned and `kit.yml` sets
   `require_signing`. Push with `kit submit`. Run `kit rebase` to bring the branch up to date with the default branch
   and push it.
4. Mark the pull request ready for review when the work is done: `gh pr ready` on GitHub, `glab mr update --ready`
   on GitLab. The pull request body is the report. It states:
   - what changed
   - what was checked, and how
   - what was left out, and why
5. A human merges. Nothing reaches `main` any other way. Do not call anything merged until you see the merge.

## Plans are prose

Write a design, a spec, or a plan as prose in the body of an issue. Never commit it as a file. When the design is
written, the issue is the deliverable, and the workflow above starts from it.

## The worker contract

A worker is whoever implements one issue, in the worktree named for that issue.

1. `cd` into the worktree named in the task. Confirm that `git rev-parse --abbrev-ref HEAD` prints a branch name
   that begins with `<N>-`.
2. Run every command from there. Never touch the root checkout.
3. Never run `kit issue`, create a branch, or create a worktree. These were done before the handoff.
4. Implement what the issue asks and nothing else.
5. Push as you go.
6. Before you call anything done, run `kit gate`: the repository's gate (its own CI command, such as `just ci`)
   through the pinned toolchain.
7. Finish by marking the pull request ready for review: `gh pr ready` on GitHub, `glab mr update --ready` on
   GitLab, with the report in the pull request body.
8. If the gate fails or the work is partial, leave the pull request as a draft and say why.
9. Never merge.
10. Report what ran and what it printed. Nothing more.
