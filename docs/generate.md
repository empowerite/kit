# Generate

A region is a part of a file that a command writes and a person does not edit. Mark a region with a directive in
the file's own comment syntax, then run `kit generate` to replace the region's content with its command's output.

Examples of regions:

- a table built from the manifests
- a list of commands built from their help text
- the sentence the `agents/generic` package puts in `AGENTS.md`

A hand edit makes a region stale. `kit check` fails on a stale region, so `kit ci` fails wherever it runs.

## The directive

A region opens on a line with the marker `kit:generate`, a name, and optionally a command in backticks. It closes on
a line with `kit:end` and the same name. On either line, only punctuation may come before or after the marker: the
comment opener before, and the comment closer after.

```text
<!-- kit:generate <name> -->
<!-- kit:generate <name> `<command>` -->
…
<!-- kit:end <name> -->
```

Both marker lines stay. The command's standard output replaces the lines between them. A markdown page contains:

```markdown
<!-- kit:generate AGENTS.md -->
Read .kit/1st.md at the start of every session. It defines how any agent uses empowerite/kit to keep this
repository trustworthy.
<!-- kit:end AGENTS.md -->
```

Other files use their own comment syntax:

- `#` in a shell script or a `.gitignore`
- `//` in a JavaScript file
- `--` in a SQL file
- `'` in a Visual Basic file

`kit generate` has no list of comment syntaxes. It reads a line as a directive when only non-alphanumeric
characters come before the marker and after the closing backtick. Every comment syntax passes this test. A line of
prose does not, because it has letters before the marker.

A line that starts with the marker but is not a well-formed directive is an error. It is never skipped, so a
misspelled directive is reported instead of left stale. To show the syntax on a markdown page, put it in a fenced
code block, which `kit generate` does not read, or in prose with a word before the marker.

### The name

The name is any run of characters with no whitespace and no backtick, separated from the marker by
whitespace. It pairs the directive with its end marker, and it must be unique in the file. A region the kit
delivers is named by its path.

### The command

The command is a bash command line, so pipes, redirections, and `$()` all work. It runs:

- from the repository root
- with the running kit's `bin/` first on `PATH`
- with standard input closed
- with `errexit` and `pipefail` set
- with `KIT_GENERATE_FILE` set to the file and `KIT_GENERATE_REGION` set to the region's name

The command is everything between the first backtick after the name and the last backtick before the comment
closer. It cannot contain a literal backtick; use `$()` instead. A command that needs more than one line belongs
in a script in the tree, where it can be read and tested.

### No command

A directive with no command runs `kit region`. `kit region` reads the region's name from its environment and
treats it as the path of a region an active package delivers. A name no package delivers fails, and the error names
the directive's line, so a forgotten command never passes silently. The directive can be in any file; the name
only selects the region.

## Nesting and balance

Each unit must be balanced on its own. A unit is either the file as written or one command's output.

- A region opened in a unit must close in that unit.
- An end marker closes the region most recently opened in that unit.
- A command's output may open regions of its own, as long as it closes them in the same output. `kit generate`
  expands those too, depth first, so regions can nest and one pass settles a file.
- An output that closes a region it did not open, or opens one it does not close, is refused. The error names the
  file, the directive's line, and the command.

So a nested region is always one the enclosing command wrote. A hand-written directive inside a region is
replaced along with the rest of the region's content.

## Commands

`kit generate [<path>…]` expands the regions in files.

- Named paths are relative to the directory you typed the command in. Each named file gets one pass, or up to
  `--iterations=<n>` passes.
- With no paths, it expands every tracked file and every untracked file git does not ignore. It repeats passes
  until a pass changes nothing, up to twenty passes or `--iterations=<n>`, because one command may read another
  file's region.
- A run over the whole repository that is still changing at the limit fails. The cause is a command whose output
  depends on its own output.

`kit generate --check [<path>…]` runs one pass, writes nothing, and fails naming every stale file. It runs
in two places:

- `kit check`, over the whole repository, so `kit ci` checks every generated region, the kit's and the
  repository's own.
- The `hooks/lefthook` package's pre-commit hook, over the staged paths.

`kit region [--home] [<path>]` prints the content the active packages deliver as the region at a path, rendered
the way a template is. It is the command behind every region the kit delivers.

- The path defaults to `KIT_GENERATE_REGION`, so a bare directive needs no argument.
- Only one active package can deliver a region at a given path. `kit region` prints that package's
  `regions/<base>/<path>`.
- A path no active package delivers is refused.

## What is refused

`kit generate` rewrites a file only when the whole expansion succeeds. Each of these leaves the file unchanged and
fails the run, naming the file and the line:

- a command that fails
- a line that starts with the marker and is not a directive
- an end marker with no open region, or one that names a region other than the open one
- a directive whose region never closes, in the file or in a command's output
- a name opened twice in the expanded file
- in a markdown file or a command's output, a fenced code block that never closes, since the directives after it
  could not be read
- a conflict marker in the file (a line of seven `<` or `>`), since a conflicted file cannot be expanded

## The kit's regions

A file under a package's `regions/<base>/<path>` is a template for a region's content. Every `@key@` in it is
rendered from `.kit/settings.yml` and the package's defaults. `kit install` writes the region into the file at
`<path>`: the bare directive naming the path, the rendered content, and the end marker, in the file's comment
syntax. [packages.md](packages.md#regions) says where the block goes, and what `kit update` and
`kit package remove` do with it. In addition:

- A directive with no end marker is refused, by name.
- A region under `home` must spell out its command, `kit region --home <path>`, because the bare directive cannot
  select the home directory.

| Package | File | Region |
| --- | --- | --- |
| `agents/generic` | `AGENTS.md` | the sentence that sends every agent to `.kit/1st.md` |
| `kit` | `.gitignore` | the `/worktrees/` entry |
| `agents/claude` | `.claude/CLAUDE.md` | an import of `AGENTS.md` |

[packages.md](packages.md) describes packages, and [architecture.md](architecture.md) describes the whole kit.
