# Context

A rule is written once, in `.kit/rules/<kind>.md`, and copied nowhere. Before an agent creates or changes a file,
it asks which rules govern that file:

```sh
kit context "${path}"
```

The answer is the rule files to read, one per line. `--json` gives one object per path with its rules. Every ask
is appended to `.git/kit/context.log`, one line of epoch and path per ask.

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

- `paths:` are globs matched against the repository-relative path. `**` stands for any depth, and a lone `*`
  crosses directories too, so `**/*.sh` and `*.sh` both name every `.sh` file; a glob with no wildcard, such as
  `justfile`, names the one path.
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

It reads the hook's input on stdin, takes the file path from the tool's input, and answers in the hook's output,
naming the rules and never quoting them, so reading a rule stays an act the agent takes:

```json
{"hookSpecificOutput": {"hookEventName": "PreToolUse",
  "additionalContext": "kit: lib/run.sh is governed by .kit/rules/bash.md; read it before this change."}}
```

No state is kept between calls.

A harness with no hook, and a person, use the plain call. `.kit/1st.md` names it as the step before any file is
created or changed.
