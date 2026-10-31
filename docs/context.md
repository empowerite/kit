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
`.kit/styles/<kind>.md` for its kind, with the same frontmatter, and `kit context` prints it after the rule.

## Kinds, and how a rule names what it governs

A kind of file is defined once, in a kinds file under `.kit/kinds/`, and everything that asks what a file is reads
it there:

```yaml
name: just
paths:
  - "justfile"
  - "**/justfile"
  - "**/*.just"
content:
  - "^#!.*/(env +)?just( |$)"
embeds:
  - bash
```

| Key | Contents |
| --- | --- |
| `name:` | Required. The kind's name, which a rule or a style names under `kind:`. The file's own name plays no part. |
| `paths:` | Globs matched against the repository-relative path. |
| `content:` | Extended regular expressions matched against the first line of a file that already exists. |
| `embeds:` | Kinds whose text appears inside this kind's files: a justfile's recipe bodies and a workflow's `run:` blocks are bash. |
| `kinds:` | Kinds this kind is made of: a path of any of them is of this kind too. |

`kit kind <path>…` prints the kinds each path is of, and `--json` prints them as one object per path. The kinds
are compiled into one awk program the first time they are asked about after a change, kept in the repository's git
directory under `kit/`, and every path is classified in one process: `kit kind` over every tracked file of this
repository takes a fraction of a second. `kit context`, `kit lint`, and `kit new` classify through the same
program, so they cannot disagree about a file.

A kinds file holds one kind, or several as a YAML stream: one document per kind, separated by `---` lines. Two
kinds with the same name, in one file or in two, are refused, and so is a document without a name.

A path is of every kind whose glob or pattern matches it. A workflow is of kind `yaml` and of kind `workflows`,
and every path is of kind `any`, whose one glob matches everything. A kind made of others under `kinds:` gathers
them: `text` lists every kind the kit defines that holds prose, plus `**/*.txt`, so the prose style, which governs
`text`, applies to each of them and to no binary file.

- In a glob, `**` matches any depth, and a lone `*` also crosses directories. So `**/*.sh` and `*.sh` both match
  every `.sh` file.
- A glob with no wildcard, such as `justfile`, matches that one path.
- A `content:` pattern lets a bash script with no extension match by its shebang.
- A file that does not exist yet is matched by its path alone.

A rule file, or a style file, opens with a frontmatter block that names the kinds it governs and carries no glob of
its own. The templates belong to the style; the other keys appear in rules and styles alike:

```yaml
---
kind:
  - bash
lint: shellcheck --shell=bash --external-sources
templates:
  - variant: script
    executable: true
  - variant: module
---
```

| Key | Contents |
| --- | --- |
| `kind:` | The kinds the file governs. |
| `lint:` | The linter command for the files of those kinds. |
| `checks:` | The kit's own checks for the files of those kinds, by name. This release has `pinned`, which the workflows rule names: every `uses:` is a local action, a docker image, or an action pinned to a full commit SHA with its version as a trailing comment; and `jq-compiles`, which the jq rule names. |
| `templates:` | The shapes a file of this kind takes; see [templates.md](templates.md). |

`kit context <path>` prints the rules, then the styles, whose `kind:` meets the path's kinds, and then the rules of
the kinds those kinds embed. So opening a justfile names the just rule, the just style, and the bash rule, with no
sentence in one rule pointing at another. `--json` reports the kinds beside the rules.

`kit lint` runs each rule's `lint:` command once over the paths of that rule's kinds, and never follows `embeds:`,
so no linter runs over a file it cannot parse. The paths are the staged paths, the paths you name, or every tracked
path with `--all-files`. A linter that is not on `PATH` is a failure, not a skip. The kit assumes no hook manager;
the `hooks/lefthook` package adds a pre-commit hook that runs `kit lint`. `kit new file` chooses a template through
the same classification.

The kit package delivers the kinds its rules and styles name, one file each, named for its kind. A repository adds
a kind by delivering a kinds file from a package of its own, with a rule that names it. It widens a delivered kind,
say with one more suffix under `markdown`, by delivering a file at the same path from a package that depends on
`kit`: the later package in dependency order wins, as it does for any delivered file.

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

A harness that speaks the Model Context Protocol also has `context` as a tool of `kit serve`; see
[serve.md](serve.md). A harness with no hook, and a person, run `kit context "${path}"` directly. `.kit/1st.md`
lists it as the step before creating or changing any file.

Cursor and GitHub Copilot have no hook, but a rule file loaded by path. The `agents/cursor` and `agents/copilot`
packages deliver one per kind, attached by the kind's `paths`, whose body sends the agent to `.kit/rules/<kind>.md`,
so the agent reads the rule itself there too. A rule file says less than `kit context`, which also matches a file by
its content, such as a bash script with no suffix, and follows what a kind embeds; the body names
`kit context <path>` for those.
