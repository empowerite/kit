# Logs

kit writes two logs in each worktree, one file a day each, under `.kit/logs/`:

| File | A line for |
| --- | --- |
| `kit-YYYY-MMDD.log` | each call of the kit CLI, from a terminal, a hook, a recipe, a tool call, or another kit call |
| `mcp-YYYY-MMDD.log` | each message `kit serve` answers, and its start and stop |

Neither needs setting up. `kit hydrate` lists `/logs/` in `.kit/.gitignore`, as it lists `/cache/`, so git never
sees a log. A directory where kit has not hydrated does not get any logs, and
neither does a directory outside a git worktree.

## A call

The kit CLI runs the verb as its child, and writes the line when the verb ends:

```text
2026-11-02T12:45:23-0800 seconds=0.238 pid=42234 depth=0 status=0 in=. kit version
2026-11-02T12:45:23-0800 seconds=0.007 pid=42272 depth=0 status=2 in=. kit nosuchverb
2026-11-02T12:45:24-0800 seconds=0.197 pid=42359 depth=1 status=2 in=kit/modules kit lint --nope a\ b
```

- The time is when the call started, and `seconds` is how long it ran.
- `depth` is 0 for a call typed or run by something other than kit, and one more for each kit call that made it.
- `status` is the verb's exit status. Ctrl-C ends the verb, and the call is logged with status 130.
- `in` is the directory the call was typed in, relative to the worktree's root.
- The command line comes last, quoted as bash quotes it, so it can be pasted back into a shell.

Lines are written when calls end, so a call that ran a shorter one inside it comes after it.

## An MCP message

```text
2026-11-02T12:45:23-0800 seconds=0.000 pid=42287 event=start
2026-11-02T12:45:23-0800 seconds=0.107 pid=42287 id=1 method=initialize outcome=ok
2026-11-02T12:45:24-0800 seconds=0.026 pid=42287 method=notifications/initialized outcome=none
2026-11-02T12:45:24-0800 seconds=0.436 pid=42287 id=2 method=tools/call outcome=status=2 kit lint --nope a\ b
2026-11-02T12:45:24-0800 seconds=0.256 pid=42287 id=3 method=resources/read outcome=ok uri=kit://1st.md
2026-11-02T12:45:23-0800 seconds=0.836 pid=42287 event=stop
```

- `outcome` is `ok`, `error=<code>` for a request the server refused, `status=<status>` for a tool whose verb
  failed, or `none` for a notification, which does not get an answer.
- A tool's line ends with the command it ran. The same call is in the kit log, one deeper.
- The stop line's `seconds` is how long the server ran.
- `event=reload` is the server starting itself again from a file that changed, and the new process logs its own
  `event=start`. `event=reload-refused` names a changed bash file that does not parse, which keeps the running code.
  [serve.md](serve.md) says when the server reloads.

The harness keeps its own log of the server's stderr. Claude Code, for one, keeps it under
`~/Library/Caches/claude-cli-nodejs/<project>/mcp-logs-kit/` on macOS.

## How long they are kept

`log_days` in `kit.yml` is how many days a log is kept, 14 when it is not set:

```yaml
log_days: 30
```

The first line of a day's log removes each log last written more than `log_days` days before. `log_days: 0` writes
no logs, and leaves the ones there to remove by hand. `kit uninstall` removes `.kit/logs/` with the rest of what kit
wrote.

see: [modules/bash/log.sh](../kit/modules/bash/log.sh), [serve.md](serve.md)
