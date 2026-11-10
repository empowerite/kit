# The agents/claude package

Adapts the kit for Claude Code. Claude Code reads `AGENTS.md` only as a fallback: when no `CLAUDE.md`,
`.claude/CLAUDE.md`, or `CLAUDE.local.md` is present in the working directory or above it, and where its
configuration allows. In a repository with its own `CLAUDE.md`, Claude Code would skip `AGENTS.md`.

So the package delivers a region in `.claude/CLAUDE.md` that contains `@../AGENTS.md`. The region starts at
a `kit:generate` directive, and `kit generate` fills it. As a result:

- Claude Code reads what every other agent reads, in every case.
- The rest of `.claude/CLAUDE.md`, and any root `CLAUDE.md`, belong to the repository. Its host template gives the
  file a title, the region, then the repository's own instructions, as `agents/generic` does for `AGENTS.md`.
- `kit check` checks only the region.

The package also merges two entries into `.claude/settings.json`, by key. The repository's own entries stay as they
are.

`permissions.deny` lists the git commands that `.kit/1st.md` forbids an agent. Each has a kit command in its place:

| Denied | Instead |
| --- | --- |
| `git commit` | `kit commit` |
| `git push` | `kit submit` |
| `git rebase` | `kit rebase` |
| `git worktree add`, `git checkout -b`, `git switch -c` | `kit work` |
| `git worktree remove`, `git branch -D` | `kit tidy` |
| `git checkout main`, `git switch main`, `git reset --hard` | none: an agent leaves the desk as it is |
| `git push --force`, `git push -f` | none: `kit submit` never forces |

`hooks.PreToolUse` runs `kit context --claude` before each `Write`, `Edit`, and `MultiEdit`. It shows the agent the
rules that govern the file; see [docs/context.md](../../context.md).

The package merges one more entry, `mcpServers.kit`, into `.mcp.json`: the kit's MCP server, `kit serve`, which
offers the rules as resources and the verbs as tools in three groups; see [docs/serve.md](../../serve.md):

- `context`, `flow`, `help`, `kinds`, `packages`, `syntax`, and `version` read only.
- `check`, `lint`, `toolchains`, and `tools` run what the repository's files name.
- `assign`, `close`, `gather`, `settle`, `unassign`, and `work` act on issues and pull requests.

Claude Code asks once, at the start of the next session, whether to use a project's servers, and the kit says so
when it places the entry. A yes to that question offers the tools that close issues too. The hook stays: it fires
before every edit, while a tool fires only when the agent calls it.

`.claude/rules/` belongs to the repository. `AGENTS.md` does not contain an `@` import for Claude Code, because every
harness reads `AGENTS.md`.
