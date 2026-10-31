# The agents/cursor package

Adapts the kit for Cursor. Cursor reads `AGENTS.md` on its own, so the neutral tree already reaches it. What the
package adds is Cursor's native delivery of the rules: one rule file per kind under `.cursor/rules/`, attached when
a file it names is in context.

Each `.cursor/rules/kit-<kind>.mdc` is a short rule with the frontmatter Cursor loads:

- `description`: one line naming the kind
- `globs`: the `paths` of the kind in `.kit/kinds/`, comma-separated, unquoted, with no space after a
  comma. That is the form Cursor's loader takes; it is not a YAML parser, and a quoted or bracketed value does not
  attach.
- `alwaysApply: false`, so the rule attaches by its globs

The body does not repeat the rule. It tells the agent to read `@.kit/rules/<kind>.md` and follow it, which is what
`kit context` tells Claude Code, so one rule text under `.kit/rules/` governs every harness and a mirror cannot drift
from it. A mirror says less than `kit context`: a rule that governs a file by its content, such as a bash script
with no suffix, cannot be a glob, so the body names `kit context <path>` for those.

The package also merges `mcpServers.kit` into `.cursor/mcp.json`, with `type: stdio`: the kit's MCP server,
`kit serve`; see [docs/serve.md](../../serve.md). Cursor asks before it runs each of the server's tools unless the
person allows it.

`.cursor/rules/` otherwise belongs to the repository. The package places only its seven files, `kit check` holds
them to the release, and `kit package remove agents/cursor` takes them out. Cursor offers no deny list a repository
can commit, so the git commands `.kit/1st.md` forbids are refused by the hooks and the ruleset alone.

Verified on 2026-10-30 against [Cursor's rules page][cursor-rules], which shows no version, and a Cursor staff
statement of the frontmatter form in [its forum][cursor-forum].

[cursor-rules]: https://cursor.com/docs/context/rules
[cursor-forum]: https://forum.cursor.com/t/rule-frontmatter-format/146274
