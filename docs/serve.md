# Serve

`kit serve` is a local Model Context Protocol (MCP) server. It offers the kit's read-only verbs to an agent as
tools, and the kit's files as resources, so an agent whose harness speaks MCP reaches the kit through one entry
point without knowing the verbs or parsing their text. It reads requests on stdin and answers on stdout, in the
repository it starts in, and opens no network port.

## What it offers

The tools are the verbs that read and report, and change nothing:

| Tool | Runs |
| --- | --- |
| `backlog` | `kit backlog` |
| `changes` | `kit changes` |
| `check` | `kit check` |
| `context` | `kit context` |
| `help` | `kit help` |
| `lint` | `kit lint` |
| `package-list` | `kit package list` |
| `status` | `kit status` |
| `syntax` | `kit syntax` |
| `version` | `kit version` |

- A tool's description is its verb's help, read from the verb's header by the functions `kit help` uses, so the
  help and the server cannot drift apart.
- A tool takes one argument, `args`, an array of strings passed to the verb. The usage lines in the description
  say what they may be.
- A call runs the verb in a new process with stdin at end of file, so the verb cannot read the protocol stream.
  The result's text is the verb's stdout, then its stderr. A verb that exits non-zero sets `isError`, and the text
  names its exit status.
- `backlog`, `changes`, and `status` ask the forge over the network, and say so in their `openWorldHint`.

The verbs that change files, such as `new` and `generate`, are not tools. They change the working tree an agent is
already editing, which the agent can do by itself.

The resources are the kit's files under `.kit/`, each as the repository has it:

| Resource | File | Type |
| --- | --- | --- |
| `kit://1st.md` | `.kit/1st.md` | `text/markdown` |
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
entry. `kit package remove` takes the entry out and leaves the repository's own servers.

| Package | File | Entry |
| --- | --- | --- |
| [`agents/claude`](packages/agents/claude.md) | `.mcp.json` | `mcpServers.kit` |
| [`agents/cursor`](packages/agents/cursor.md) | `.cursor/mcp.json` | `mcpServers.kit`, with `type: stdio` |
| [`agents/copilot`](packages/agents/copilot.md) | `.vscode/mcp.json` | `servers.kit`, with `type: stdio` |
| [`agents/gemini`](packages/agents/gemini.md) | `.gemini/settings.json` | `mcpServers.kit` |

The command is `kit serve`, with `kit` found on `PATH`, as the hook in `.claude/settings.json` already assumes.
When the kit places an entry, `kit install`, `kit package add`, and `kit update` say what the harness does next:
Claude Code, for one, asks once, at the start of the next session, whether to use a project's servers.

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
