# Serve

`kit serve` is a local Model Context Protocol (MCP) server. It offers the kit's read-only verbs to an agent as
tools, and the kit's files as resources. It reads requests on stdin and answers on stdout, in the repository it
starts in, and opens no network port.

A tool is its verb, no more. It takes one argument, `args`: the strings that would follow `kit <verb>` on a command
line. Its description is the verb's help page, and the usage line there is what says what `args` may be. So an
agent still needs the verb's grammar; the server parses nothing for it.

## What it offers

The tools are the verbs that read and report, and change nothing:

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

- A tool's description is read from the verb's header by the functions `kit help` uses, so the help and the server
  cannot drift apart.
- A call runs the verb in a new process with stdin at end of file, so the verb cannot read the protocol stream.
  The result's text is the verb's stdout, then its stderr. A verb that exits non-zero sets `isError`, and the text
  names its exit status.
- Every tool carries `readOnlyHint`: none changes a file the repository tracks. The kit's own bookkeeping still
  happens: `kit serve` hydrates `.kit/` when it starts, and `context`, `kinds`, and `lint` hydrate before they
  read. `tools` hydrates too when no toolchain is chosen, to read the rules' linters. `context` appends to the
  context log.
- `flow` asks the forge over the network, and says so in its `openWorldHint`.
- `packages`, `toolchains`, and `tools` are offered as listings only. A call whose `args` hold an action is refused
  with status 2, and the verb never runs:
  - `packages`: `select`, `add`, `remove`
  - `toolchains`: `choose`
  - `tools`: `install`, `update`, `path`, `run`, and `--outdated`, which reaches the network

The verbs that change files, such as `new` and `generate`, are not tools. They change the working tree an agent is
already editing, which the agent can do by itself.

The resources are the kit's files under `.kit/`, each as the repository has it:

| Resource | File | Type |
| --- | --- | --- |
| `kit://1st.md` | `.kit/1st.md` | `text/markdown` |
| `kit://workflow.md` | `.kit/workflow.md`, when the `workflow` package is active | `text/markdown` |
| `kit://rules/<kind>` | `.kit/rules/<kind>.md` | `text/markdown` |
| `kit://styles/<kind>` | `.kit/styles/<kind>.md` | `text/markdown` |
| `kit://kinds/<file>` | `.kit/kinds/<file>.yml`, which defines one kind or several | `application/yaml` |

The server's instructions tell an agent to read `kit://1st.md` first, and to call `context` with a path before it
changes that file.

## The protocol

`kit serve` was checked against the [2026-07-28 revision](https://modelcontextprotocol.io/specification/2026-07-28)
of the specification. It answers two eras of the protocol, as the specification lets a dual-era server do:

- The 2026-07-28 revision has no session. A client may ask `server/discover`, and every other request names its
  version in `params._meta["io.modelcontextprotocol/protocolVersion"]`. A request that names none is refused with
  `-32602`, and one that names a revision the server does not answer gets `-32022` with the revisions it does.
- The revisions before it, `2025-11-25`, `2025-06-18`, and `2025-03-26`, open with `initialize`. The server answers
  with the revision the client asks for, or the newest of them, and from then on a request needs no `_meta` and
  `ping` is answered.

Either way:

- Stdout carries only protocol messages, one JSON object per line. Every log line goes to stderr.
- Requests are answered one at a time, in order.
- A message without an `id` is a notification, and gets no answer, even when it fails.
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
npx @modelcontextprotocol/inspector --cli kit serve --method tools/call --tool-name context --tool-args-json '{"args": ["README.md"]}'
```

see: [modules/bash/serve.sh](../kit/modules/bash/serve.sh), [context.md](context.md)
