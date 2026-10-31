# The agents/gemini package

Adapts the kit for Gemini CLI. Gemini CLI reads `GEMINI.md`, not `AGENTS.md`, unless its settings say otherwise,
and it has no rule file loaded by path. So the package merges two entries into `.gemini/settings.json`, by key, and
leaves the repository's own entries alone:

- `context.fileName`: `GEMINI.md` and `AGENTS.md`, added to whatever list the repository has, so Gemini reads what
  every other agent reads and still reads a `GEMINI.md` the repository may keep.
- `tools.exclude`: the git commands `.kit/1st.md` forbids an agent, as `run_shell_command(<command>)` entries, the
  same commands as the `agents/claude` deny list. Gemini matches a shell command by prefix and checks the exclusions
  first, so `run_shell_command(git push)` refuses `git push --force` as well, and a chained command is split and
  each part checked.

Gemini's documentation marks `tools.exclude` as deprecated in favor of its policy engine, whose rules are TOML files.
Only that engine's user tier works today: its workspace tier, which would let a repository commit policies, is
documented as disabled. Until it works, `tools.exclude` is the one deny list a repository can commit, and the package
uses it. Rules are not mirrored: Gemini reaches the rules table in `.kit/1st.md` through `AGENTS.md`, and
`kit context <path>` names a file's rules.

It also merges `mcpServers.kit`, the kit's MCP server, `kit serve`; see [docs/serve.md](../../serve.md).

`kit check` holds the entries to the release, and `kit package remove agents/gemini` takes them out,
leaving the repository's own.

Verified on 2026-10-30 against Gemini CLI's [settings schema][gemini-schema], its
[configuration reference][gemini-config], dated 2026-10-15 and on the nested settings of v0.3.0, its
[shell tool page][gemini-shell], and its [policy engine page][gemini-policy]. The reference carries a notice that
for the unpaid tier and Google One users, Gemini CLI was replaced by Antigravity CLI on 2026-06-18.

[gemini-schema]: https://raw.githubusercontent.com/google-gemini/gemini-cli/main/schemas/settings.schema.json
[gemini-config]: https://geminicli.com/docs/reference/configuration/
[gemini-shell]: https://github.com/google-gemini/gemini-cli/blob/main/docs/tools/shell.md
[gemini-policy]: https://github.com/google-gemini/gemini-cli/blob/main/docs/reference/policy-engine.md
