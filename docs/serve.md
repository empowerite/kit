# Serve

`kit serve` is a local Model Context Protocol (MCP) server. It offers the kit's read-only verbs to an agent as
tools, and the kit's files as resources. It reads requests on stdin and answers on stdout, in the repository it
starts in, and does not open a network port.

A tool is its verb, with typed input. Each flag, option, and operand of the verb is a property of the tool's input
schema, built from the verb's own usage lines. So an agent calls a tool without knowing the verb's grammar:

```json
{"name": "context", "arguments": {"path": ["README.md"], "json": true}}
```

The server turns those arguments into the verb's words, here `kit context README.md --json`, and runs it.

## What it offers

The tools are the verbs that read and report, and do not change anything:

| Tool | Runs |
| --- | --- |
| `check` | `kit check` |
| `context` | `kit context` |
| `flow` | `kit flow` |
| `help` | `kit help` |
| `kinds` | `kit kinds` |
| `lint` | `kit lint` |
| `packages` | `kit packages` |
| `syntax` | `kit syntax` |
| `toolchains` | `kit toolchains` |
| `tools` | `kit tools` |
| `version` | `kit version` |

Each tool is described and typed from the verb's header, by the functions `kit help` and `kit syntax` use. So the
help, the syntax, and the server cannot drift apart.

- **The description** is the verb's purpose and the usage lines the tool offers. The verb's whole page is the
  resource `kit://help/<verb>`, and the description says so.
- **The input** has one property per argument:
  - A flag, such as `--json`, is a boolean.
  - An option, such as `--toolchain <name>`, is a string.
  - An operand, such as `<path>…`, is a string, or an array of strings when it repeats.
  - An argument with a closed set of choices has them as its `enum`, such as the package names for `packages`.
  - A line's leading word is one property too: `flow`'s `stage`, such as `backlog` or `desk`.
  - An argument every offered line needs is `required`, such as `context`'s `path`.
- **A call that does not fit** is refused with `-32602`, and the verb does not run. The error says what is wrong:
  - a name the tool does not take, with the names it does take
  - a value of the wrong type
  - an operand's value that starts with `-`, which the verb would read as a flag
  - arguments that fit none of the usage lines, which the error lists
- **A call runs the verb** in a new process with stdin at end of file, so the verb cannot read the protocol stream.
  The result's text is the verb's stdout, then its stderr. A verb that exits non-zero sets `isError`, and the text
  names its exit status.

Every tool carries `readOnlyHint`, and it holds:

- A call does not place anything under `.kit/`. The server places `.kit/` when it starts, and again before a
  message when `kit.yml` changed since.
- `flow` never fetches. The server gives it `--no-fetch`, so `incoming` shows the refs as they are. Run `kit flow`
  in a terminal to fetch.
- An action is never offered. `packages`, `toolchains`, and `tools` are listings only:
  - `packages` does not offer `select`, `add`, or `remove`.
  - `toolchains` does not offer `choose`.
  - `tools` does not offer `install`, `update`, `path`, `run`, or `--outdated`, which reaches the network.

  A call that names one of these is refused, and the error says to run the verb in a terminal.
- `context` does not offer `--claude`, the hook mode. It is the harness's own.

A tool's verb still writes the kit's own records, which git never sees: its line in `.kit/logs/`, the context log,
and a toolchain's detection cache under `.kit/cache/`.

`flow` asks the code forge over the network, and says so in its `openWorldHint`.

The verbs that change files, such as `new` and `generate`, are not tools. They change the working tree an agent is
already editing, which the agent can do by itself. The workflow verbs, `commit`, `issue`, and `submit`, are not tools
either; an agent runs them in its shell.

The resources are the kit's files under `.kit/`, each as the repository has it, and each tool's help page:

| Resource | File | Type |
| --- | --- | --- |
| `kit://1st.md` | `.kit/1st.md` | `text/markdown` |
| `kit://workflow.md` | `.kit/workflow.md`, when the `workflow` package is active | `text/markdown` |
| `kit://rules/<kind>` | `.kit/rules/<kind>.md` | `text/markdown` |
| `kit://styles/<kind>` | `.kit/styles/<kind>.md` | `text/markdown` |
| `kit://kinds/<file>` | `.kit/kinds/<file>.yml`, which defines one kind or several | `application/yaml` |
| `kit://help/<verb>` | none: the page `kit help <verb>` prints, for each tool | `text/plain` |

The server's instructions tell an agent to read `kit://1st.md` first, and to call `context` with a path before it
changes that file.

## The protocol

`kit serve` was checked against the [2026-07-28 revision](https://modelcontextprotocol.io/specification/2026-07-28)
of the specification. It answers two eras of the protocol, as the specification lets a dual-era server do:

- The 2026-07-28 revision has no session. A client may ask `server/discover`, and every other request names its
  version in `params._meta["io.modelcontextprotocol/protocolVersion"]`. A request that does not name one is refused with
  `-32602`, and one that names a revision the server does not answer gets `-32022` with the revisions it does.
- The revisions before it, `2025-11-25`, `2025-06-18`, and `2025-03-26`, open with `initialize`. The server answers
  with the revision the client asks for, or the newest of them, and from then on a request does not need `_meta` and
  `ping` is answered.

Either way:

- Stdout carries only protocol messages, one JSON object per line. Every log line goes to stderr.
- Requests are answered one at a time, in order.
- A message without an `id` is a notification, and does not get an answer, even when it fails.
- A list or a read carries `ttlMs` of 0 and `cacheScope` of `private`, because the files and the verbs' answers
  change as the agent works.

## How an agent finds it

Each `agents/<harness>` package merges the server into its harness's MCP configuration, by key, and owns only that
entry. `kit packages remove` takes the entry out and leaves the repository's own servers.

| Package | File | Entry |
| --- | --- | --- |
| [`agents/claude`](packages/agents/claude.md) | `.mcp.json` | `mcpServers.kit` |
| [`agents/cursor`](packages/agents/cursor.md) | `.cursor/mcp.json` | `mcpServers.kit`, with `type: stdio` |
| [`agents/copilot`](packages/agents/copilot.md) | `.vscode/mcp.json` | `servers.kit`, with `type: stdio` |
| [`agents/gemini`](packages/agents/gemini.md) | `.gemini/settings.json` | `mcpServers.kit` |

The command is `kit serve`, with `kit` found on `PATH`, as the hook in `.claude/settings.json` already assumes.
When the kit places an entry, `kit install`, `kit packages add`, and `kit update` say what the harness does next.
Claude Code, for one, asks once, at the start of the next session, whether to use a project's servers.

## When the kit changes

A harness starts the server once per session, with the `PATH` it started with. What the server runs stays current
on its own:

- In a checkout of the kit itself, whose `kit.yml` declares `version: unreleased`, the checkout's own
  `kit/bin/kit-serve` serves it, whichever kit the harness started. A worktree that changes the kit is served by its
  own code, not by the desk's or a release's.
- Before each message, the server looks for a file changed since it started, in its release's `bin/` and
  `modules/` and in the repository's own module directories. When one has, it starts itself again from the changed
  code over the same stdin, and the new process answers the message. An edit to a verb, a module, or the server
  takes effect on the next message, with no reconnect.
- After it starts again, it sends `notifications/tools/list_changed` and `notifications/resources/list_changed`,
  and the client asks for both lists again. A `.kit/.gitignore` that changed, as `kit hydrate` rewrites it when what
  it places changes, sends the second alone.
- A changed bash file that does not parse is kept out. The server goes on with the code it runs, says so on stderr,
  and tries again after the next change.
- A `kit.yml` changed since, as when the agent adds a package, is placed by the server before the next message.
  Then `.kit/.gitignore` changes too, and the client hears that the resources changed.

The harness's own configuration is read when its session starts, so a change to `.mcp.json` still needs a new
session, or `/mcp` and Reconnect in Claude Code.

`kit` on `PATH` is the harness's `PATH`, not a shell's. An app opened from the Dock may not have the `PATH` that mise
or direnv sets in a terminal. This repository's own `house` package therefore registers the command by path from the
project directory Claude Code names, as `${CLAUDE_PROJECT_DIR:-.}/kit/bin/kit`. A repository whose kit lives in the
tree can do the same with an `.mcp.json` entry in a package of its own, after `agents/claude`.

## Its log

Each message the server answers is a line in `.kit/logs/mcp-YYYY-MMDD.log`, with its method, its outcome, and how
long it took, and a tool's line ends with the command it ran. The server adds a line when it starts and one when it
stops. [logs.md](logs.md) shows the lines, and how long they are kept.

## Try it by hand

Feed it request lines. This asks for the server's revisions and its tools, in the 2026-07-28 shape:

```bash
version='{"_meta": {"io.modelcontextprotocol/protocolVersion": "2026-07-28"}}'
printf '%s\n' \
  '{"jsonrpc": "2.0", "id": 1, "method": "server/discover"}' \
  "{\"jsonrpc\": \"2.0\", \"id\": 2, \"method\": \"tools/list\", \"params\": ${version}}" \
  | kit serve | jq .
```

The [MCP Inspector](https://github.com/modelcontextprotocol/inspector) drives it the way a harness does:

```bash
npx @modelcontextprotocol/inspector --cli kit serve --method tools/list
npx @modelcontextprotocol/inspector --cli kit serve --method tools/call --tool-name context --tool-args-json '{"path": ["README.md"]}'
```

see: [modules/bash/serve.sh](../kit/modules/bash/serve.sh), [context.md](context.md)
