# The agents/claude package

Adapts the kit for Claude Code. Claude Code reads `AGENTS.md` only as a fallback: when no `CLAUDE.md`,
`.claude/CLAUDE.md`, or `CLAUDE.local.md` is present in the working directory or above it, and where its
configuration allows. In a repository with its own `CLAUDE.md`, Claude Code would skip `AGENTS.md`.

So the package delivers a region in `.claude/CLAUDE.md` that contains `@../AGENTS.md`. The region starts at
a `kit:generate` directive, and `kit generate` fills it. As a result:

- Claude Code reads what every other agent reads, in every case.
- The rest of `.claude/CLAUDE.md`, and any root `CLAUDE.md`, belong to the repository.
- `kit check` checks only the region.

The package also merges two entries into `.claude/settings.json`, by key, and leaves the repository's own entries
alone:

- `permissions.deny`: the git commands that `.kit/1st.md` forbids an agent, because the kit's commands and the
  workflow replace them. They are `git commit` and `git push`, which `kit commit` and `kit submit` replace;
  `git rebase`, `git worktree add`, `git checkout -b`, `git switch -c`, `git worktree remove`, `git branch -D`,
  `git checkout main`, `git switch main`, `git reset --hard`, `git push --force`, and `git push -f`.
- `hooks.PreToolUse`: on `Write`, `Edit`, and `MultiEdit`, runs `kit context --claude`, which shows the agent the
  rules that govern the file. It is described in
  [docs/context.md](../../context.md).

The package merges one more entry, `mcpServers.kit`, into `.mcp.json`: the kit's MCP server, `kit serve`, which
offers the read-only verbs as tools and the rules as resources; see [docs/serve.md](../../serve.md). Claude Code
asks once, at the start of the next session, whether to use a project's servers, and the kit says so when it places
the entry. The hook stays: it fires before every edit, while a tool fires only when the agent calls it.

`.claude/rules/` belongs to the repository. `AGENTS.md` contains no `@` import for Claude Code, because every
harness reads `AGENTS.md`.
