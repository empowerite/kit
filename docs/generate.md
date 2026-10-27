# Generate

A region is a stretch of a file that a command makes and a person does not edit: a table drawn from the
manifests, a list of verbs drawn from their help, the sentence the kit puts in `AGENTS.md`. The file itself says
how each region is made, with a directive in its own comment syntax, and `kit generate` makes every region what
its command makes it. A hand edit is then a stale region, and `kit check` fails on one wherever `kit ci` runs.
The mechanism is enchiridion's `:generate:` channel, carried into the kit so any repository has it.

## The directive

A comment line that is the directive and nothing else opens a region:

```text
:generate:<command>:until:"<pattern>"
:generate:<command>:while:"<pattern>"
```

The directive line stays. The command runs from the repository root, with the running kit's `bin/` first on
`PATH`, and its standard output replaces the lines after the directive. With `until`, the region runs up to the
first later line matching the pattern, an extended regular expression, and that line stays as the end marker.
With `while`, the region is every line matching the pattern, and the first line that does not match ends it. So
a markdown page carries:

```markdown
<!-- :generate:kit region AGENTS.md:until:"kit:end" -->
Read .kit/1st.md at the start of every session. It defines how any agent uses empowerite/kit to keep this
repository trustworthy.
<!-- kit:end -->
```

and a shell script or a `.gitignore` carries the same with `#`, a JavaScript file with `//`, a SQL file with
`--`. The comment openers the kit reads are `#`, `//`, `--`, `<!--`, `/*`, `%`, and `;`. A line that is not a
comment opened by one of them is not a directive, so a page may mention the syntax in prose or a code span, and a
directive inside a fenced code block is not live, so a page may show one whole.

The command is one command and its arguments, with no shell operator: no pipe, no `&&`, no redirection, no
substitution. A word is a name, a flag, a path, or a `key=value` pair. What makes a region is therefore always a
kit verb, a just recipe, a make target, or a script in the tree, where it is listed, read, and tested, and never
a one-liner hidden in a comment. A command outside that shape is refused by file and line.

## The verbs

`kit generate [<path>…]` expands the named files, taken from the directory the command was typed in, or with
none named every tracked file and every untracked one git does not ignore. Named files are expanded once, or
`--iterations=<n>` times. The whole repository is expanded until a pass changes nothing, at most twenty passes or
`--iterations=<n>`, since a region's output may depend on another region's. A run that is still changing at the
bound fails, naming the cause: a command whose output depends on its own output.

`kit generate --check [<path>…]` runs one pass without writing and fails naming every stale file. `kit check`
runs it over the whole repository, so the gate holds every generated region, the kit's and the repository's own,
and the lefthook module's pre-commit hook runs it over the staged paths.

`kit region [--home] <path>` prints the content the active modules deliver as the region at a path, the command
behind every region the kit delivers. One active module delivers a region at a path, so the answer is that
module's `regions/<base>/<path>`, and a path no active module delivers is refused.

## What is refused

A file is rewritten only on a clean expansion. Each of these leaves the file as it was and fails the run, naming
the file and the line:

- a command that fails;
- an `until` pattern no later line matches, so the region never closes;
- a command outside its shape;
- a conflict marker in the file, a line of seven `<` or `>`, since a conflicted file is not input a region
  expander can read;
- an expansion that produces nothing from a file that was not empty.

## The kit's regions

A file under a module's `regions/<base>/<path>` is the content of a region. What `kit install` writes into the
file at `<path>` is the directive naming `kit region <path>`, the content, and the end marker, in the file's
comment syntax: between the directive and the end marker where the file already carries them, appended after a
blank line where it does not, and as the whole file where there is none. The repository moves the block where it
likes and writes what it likes around it. `kit update` regenerates the region, and there is no three-way merge of
one, since its content is never the repository's to edit; `kit module remove` takes the block out, and removes a
file it leaves empty. A directive with no end marker is refused by name, and two modules delivering a region to
one file are refused where the modules are read.

The `kit` module places two: the sentence in `AGENTS.md` that sends every agent to `.kit/1st.md`, and the
`/worktrees/` entry in `.gitignore`. The `claude` module places one in `.claude/CLAUDE.md`, importing
`AGENTS.md`. [modules.md](modules.md) describes a module, and [architecture.md](architecture.md) the whole.
