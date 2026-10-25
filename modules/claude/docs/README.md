# The claude module

The adapter for Claude Code. Since 2.1.277, Claude Code reads `AGENTS.md` as a fallback: only when no `CLAUDE.md`,
`.claude/CLAUDE.md`, or `CLAUDE.local.md` is present in the working directory or above it, and only where its
configuration allows. A repository with a `CLAUDE.md` of its own would have `AGENTS.md` skipped. So the module
places `.claude/CLAUDE.md`, two lines, the second `@../AGENTS.md`: Claude then reads what every other agent reads
in every case, and a root `CLAUDE.md` of the repository's own stays its own. It also merges into
`.claude/settings.json`, by key, leaving the repository's own entries:

- `permissions.deny`: the seven spellings the rules forbid an agent, such as `git rebase` and `git push --force`,
  which `kit rebase` and the loop replace.
- `hooks.PreToolUse`: on `Write`, `Edit`, and `MultiEdit`, `kit context --claude`, which puts the rules governing
  the file before the agent, described in [docs/context.md](../../../docs/context.md).

It copies no rule. `.claude/rules/` is the repository's own.

Why not `@.kit/1st.md` in `AGENTS.md`: that is Claude's import syntax in a file every harness reads, and Claude,
having inlined the file, would then be told to read it again by the sentence beside it.
