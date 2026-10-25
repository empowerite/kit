# Context

A rule is written once, in `.kit/rules/<kind>.md`, and copied nowhere. Before an agent creates or changes a file,
it asks which rules govern that file:

```sh
kit context <path>…
```

The answer is the rule files to read, one per line. `--json` gives one object per path with its rules. Every ask
is appended to `.git/kit/context.log` as `<epoch>\t<path>`, which is the evidence a check can read to ask whether
every path a commit touched had its context asked for.

## How a rule declares what it governs

A rule file opens with a frontmatter of two optional lists:

```yaml
---
paths:
  - "**/*.sh"
content:
  - "^#!.*/(env +)?(ba)?sh( |$)"
---
```

- `paths:` are globs matched against the repository-relative path. `**` stands for any depth, so `**/*.sh` is
  every `.sh` file and `justfile` is the one at the root.
- `content:` are extended regular expressions matched against the first line of a file that already exists, so a
  bash script with no extension is bash by its shebang, and a file about to be created is governed by its path
  alone.

A rule governs a file when any glob or any pattern matches.

## Harness adapters

A harness that can run a command before a write runs `kit context` itself, so the rules reach the agent without
the agent remembering to ask. The `claude` module wires Claude Code's `PreToolUse` hook on `Write`, `Edit`, and
`MultiEdit` to:

```sh
kit context --claude
```

It reads the hook's input on stdin, takes the file path from the tool's input, and answers in the hook's output
with the rules as additional context: a rule's text the first time it is named in the session, its path after
that. The session is told apart by the hook input's `session_id`, and what was given is kept under `TMPDIR`.

A harness with no hook, and a person, use the plain call. `.kit/1st.md` names it as the step before any file is
created or changed.
