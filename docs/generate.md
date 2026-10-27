# Generate

A region is a stretch of a file that a command makes and a person does not edit: a table drawn from the
manifests, a list of verbs drawn from their help, the sentence the kit puts in `AGENTS.md`. The file itself says
how each region is made, with a directive in its own comment syntax, and `kit generate` makes every region what
its command makes it. A hand edit is then a stale region, and `kit check` fails on one wherever `kit ci` runs.

## The directive

A region opens on a line that is the marker `kit:generate`, a name, and optionally a command in backticks, and
closes on a line that is `kit:end` and the same name. Nothing but punctuation may surround either: the comment
opener before, the comment closer after.

```text
<!-- kit:generate <name> -->
<!-- kit:generate <name> `<command>` -->
…
<!-- kit:end <name> -->
```

Both lines stay. The command's standard output replaces the lines between them. So a markdown page carries:

```markdown
<!-- kit:generate AGENTS.md -->
Read .kit/1st.md at the start of every session. It defines how any agent uses empowerite/kit to keep this
repository trustworthy.
<!-- kit:end AGENTS.md -->
```

and a shell script or a `.gitignore` carries the same with `#`, a JavaScript file with `//`, a SQL file with
`--`, a Visual Basic file with `'`. The reader knows no list of comment syntaxes. It reads a line as a directive
when only non-alphanumeric characters come before the marker and after the closing backtick, which every comment
frame satisfies and no line of prose does, since prose has letters before the marker. A line that opens with
the marker and is not a well-formed directive is an error, never a line silently skipped, so a misspelled
directive is found rather than left stale. A page that wants to show the syntax puts it in a fenced code block,
which is not live, or in prose, with a word before the marker.

The **name** is any run of characters without whitespace or a backtick, set off by whitespace. It pairs the
directive with its end marker, and it is unique in a file. A region the kit delivers is named by its path, so
the name says what the region is.

The **command** is a bash command line: pipes, redirections, and `$()` are all its own. It runs from the
repository root with the running kit's `bin/` first on `PATH`, standard input closed, `errexit` and `pipefail`
set, and two variables in its environment: `KIT_GENERATE_FILE`, the file, and `KIT_GENERATE_REGION`, the name.
The command is everything between the first backtick after the name and the last backtick before the closer, so
the one thing it cannot carry is a literal backtick; `$()` says the same thing. A region whose command needs
more than a line is a script in the tree, where it is read and tested.

With **no command**, the command is `kit region`, which reads the region's name from its environment as the path
of a region an active module delivers. A name no module delivers fails, naming the directive's line, so a
forgotten command is never a silent no-op. The directive may sit in any file; the name says only which region.

## Nesting and balance

Every unit is balanced on its own: the file as written, and each command's output. A directive opened in a unit
closes in that unit; an end marker closes the region most recently opened there. A command's output may open
regions of its own, each closed in that same output, and `kit generate` expands those in turn, depth first, so a
region nests and one pass settles a file. An output that closes a region it did not open, or opens one it does
not close, is refused, naming the file, the directive's line, and the command. A region that nests is therefore
always one the enclosing command wrote; a hand-written directive inside a region is replaced with the rest of the
region's content.

## The verbs

`kit generate [<path>…]` expands the named files, taken from the directory the command was typed in, or with
none named every tracked file and every untracked one git does not ignore. Named files are expanded once, or
`--iterations=<n>` times. The whole repository is expanded until a pass changes nothing, at most twenty passes or
`--iterations=<n>`, since a command may read another file's region. A run that is still changing at the bound
fails, naming the cause: a command whose output depends on its own output.

`kit generate --check [<path>…]` runs one pass without writing and fails naming every stale file. `kit check`
runs it over the whole repository, so the gate holds every generated region, the kit's and the repository's own,
and the lefthook module's pre-commit hook runs it over the staged paths.

`kit region [--home] [<path>]` prints the content the active modules deliver as the region at a path, rendered as
a template is, the command behind every region the kit delivers. The path defaults to `KIT_GENERATE_REGION`, so
a bare directive needs none. One active module delivers a region at a path, so the answer is that module's
`regions/<base>/<path>`, and a path no active module delivers is refused.

## What is refused

A file is rewritten only on a clean expansion. Each of these leaves the file as it was and fails the run, naming
the file and the line:

- a command that fails;
- a line that opens with the marker and is not a directive;
- an end marker with no open region, or naming a region other than the one open;
- a directive whose region never closes, in the file or in a command's output;
- a name opened twice in the expanded file;
- a fenced code block in a markdown file, or in a command's output, that opens and never closes, since the
  directives after it could not be read;
- a conflict marker in the file, a line of seven `<` or `>`, since a conflicted file is not input a region
  expander can read.

## The kit's regions

A file under a module's `regions/<base>/<path>` is a template for the content of a region: every `@key@` in it
is rendered from `.kit/settings.yml` and the module's defaults, as [modules.md](modules.md) describes. What
`kit install` writes into the file at `<path>` is the bare directive naming the path, the rendered content, and
the end marker, in the file's comment syntax: between the directive and the end marker where the file already
carries them, appended after a blank line where it does not, and as the whole file where there is none. The
repository moves the block where it likes and writes what it likes around it. `kit update` regenerates the
region, and there is no three-way merge of one, since its content is never the repository's to edit;
`kit module remove` takes the block out, and removes a file it leaves empty. A directive with no end marker is
refused by name, and two modules delivering a region to one file are refused where the modules are read. A region
under `home` spells its command, `kit region --home <path>`, since the bare form cannot.

The `kit` module places two: the sentence in `AGENTS.md` that sends every agent to `.kit/1st.md`, and the
`/worktrees/` entry in `.gitignore`. The `claude` module places one in `.claude/CLAUDE.md`, importing
`AGENTS.md`. [modules.md](modules.md) describes a module, and [architecture.md](architecture.md) the whole.
