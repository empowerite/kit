# The claude module

The adapter for Claude Code. It merges into `.claude/settings.json`, by key, leaving the repository's own entries:

- `permissions.deny`: the seven spellings the rules forbid an agent, such as `git rebase` and `git push --force`,
  which `kit rebase` and the loop replace.
- `hooks.PreToolUse`: on `Write`, `Edit`, and `MultiEdit`, `kit context --claude`, which puts the rules governing
  the file before the agent, described in [docs/context.md](../../../docs/context.md).

It copies no rule. `.claude/rules/` is the repository's own.
