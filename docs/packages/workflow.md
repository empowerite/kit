# The workflow package

Delivers `.kit/workflow.md`: how work moves from an issue to a merged pull request. `.kit/1st.md`, the entry file
the `kit` package delivers, sends an agent to it when the package is active, and to `AGENTS.md` when it is not. The
file holds:

- The workflow:
  - an issue first
  - `kit work <N>` for the worktree, the branch, and the draft pull request
  - `kit commit`, `kit submit`, and `kit rebase` in the worktree
  - the pull request marked ready for review, with the report in its body: what changed, what was checked and how,
    and what was left out and why
  - a human merges
- Plans are prose. A design, a spec, or a plan is the body of an issue, never a file.
- The worker contract. The ten steps whoever implements one issue follows, from `cd` into the worktree to the report.

The package is taste, not trust. The rules the `kit` package delivers hold in every repository: the integrity rules,
the desk that no agent changes, the kit commands that replace raw git commands, and the rules by kind of file. A
repository that works another way leaves this package out and writes its workflow in `AGENTS.md`. One that takes it
holds every agent to the same steps, in the same words, in every harness. The `empowerite` bundle includes it.

The file names a code forge's command where there is no kit verb for the step: `gh pr ready` on GitHub, `glab mr update
--ready` on GitLab.

To change a step, replace the file through a package of the repository's own; see "Changing what kit adds" in
`.kit/1st.md`.
