# The claude module

The adapter for Claude Code. Claude Code reads `AGENTS.md` only as a fallback: when no `CLAUDE.md`,
`.claude/CLAUDE.md`, or `CLAUDE.local.md` is present in the working directory or above it, and where its
configuration allows. A repository with a `CLAUDE.md` of its own would have `AGENTS.md` skipped. So the module
places a region in `.claude/CLAUDE.md` holding `@../AGENTS.md`: Claude then reads what every other agent reads in
every case, the rest of that file and a root `CLAUDE.md` are the repository's own, and `kit check` holds only
the region. It also merges into
`.claude/settings.json`, by key, leaving the repository's own entries:

- `permissions.deny`: the spellings `.kit/1st.md` forbids an agent, `git rebase`, `git worktree add`,
  `git checkout -b`, `git switch -c`, `git worktree remove`, `git branch -D`, `git checkout main`,
  `git switch main`, `git reset --hard`, and `git push --force`, which the kit's verbs and the loop replace.
- `hooks.PreToolUse`: on `Write`, `Edit`, and `MultiEdit`, `kit context --claude`, which puts the rules governing
  the file before the agent, described in [docs/context.md](../../../docs/context.md).

It copies no rule. `.claude/rules/` is the repository's own. `AGENTS.md` carries no `@` import of Claude's, since
every harness reads it.
