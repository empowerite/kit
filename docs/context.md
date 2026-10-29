# Context

`kit context` prints the rule files that govern a path. Run it before you create or change a file, then
read each rule it names.

```sh
kit context "${path}"
```

- The output is the rule files to read, one per line. Given several paths, it prints each rule once.
- `--json` prints one object per path, with that path's rules.
- Every call appends one line to `.git/kit/context.log`: the epoch time and the path.

Each rule is written once, in `.kit/rules/<kind>.md`, and copied nowhere else. A style package adds
`.kit/style/<kind>.md` for its kind, with the same frontmatter, and `kit context` prints it after the rule.

## How a rule declares the files it governs

A rule file, or a style file, opens with a frontmatter block. Every key in it is optional. In the kit's own files,
the rules carry `paths:`, `content:`, and `lint:`, and the templates belong to the style:

```yaml
---
paths:
  - "**/*.sh"
content:
  - "^#!.*/(env +)?(ba)?sh( |$)"
lint: shellcheck --shell=bash --external-sources
templates:
  - variant: script
    executable: true
  - variant: module
---
```

| Key | Contents |
| --- | --- |
| `paths:` | Globs matched against the repository-relative path. |
| `content:` | Extended regular expressions matched against the first line of a file that already exists. |
| `lint:` | The linter command for the files the rule governs. |
| `templates:` | The shapes a file of this kind takes; see [templates.md](templates.md). |

A rule governs a file when any of its globs or any of its patterns matches.

- In a glob, `**` matches any depth, and a lone `*` also crosses directories. So `**/*.sh` and `*.sh` both match
  every `.sh` file.
- A glob with no wildcard, such as `justfile`, matches that one path.
- A `content:` pattern lets a bash script with no extension match by its shebang.
- A file that does not exist yet is matched by its path alone.

`kit lint` runs each rule's `lint:` command once over the paths that rule governs. The paths are the staged paths,
the paths you name, or every tracked path with `--all-files`. A linter that is not on `PATH` is a failure, not a
skip. The kit assumes no hook manager; the `hooks/lefthook` package adds a pre-commit hook that runs `kit lint`.

[templates.md](templates.md) describes templates, `kit new file` (which makes a file from one), and the shape check
`kit lint` runs against them.

## Harness adapters

A harness that can run a command before a write can call `kit context` itself, so the agent gets the rules without
having to remember to ask. The `agents/claude` package adds a Claude Code `PreToolUse` hook on `Write`, `Edit`, and
`MultiEdit` that runs:

```sh
kit context --claude
```

This command:

1. Reads the hook's input on stdin.
2. Takes the file path from the tool's input.
3. Prints the hook's output, naming the rules without quoting them, so the agent still reads each rule itself.

```json
{"hookSpecificOutput": {"hookEventName": "PreToolUse",
  "additionalContext": "kit: lib/run.sh is governed by .kit/rules/bash.md; read it before this change."}}
```

It prints nothing when the tool input has no file path or no rule governs the file. Nothing is kept
between calls except the log line.

A harness with no hook, and a person, run `kit context "${path}"` directly. `.kit/1st.md` lists it as the step
before creating or changing any file.
