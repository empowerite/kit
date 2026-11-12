# Modules

A module is code the verbs share. Modules extend how kit works; packages extend what kit can do in a repository. A
module runs each time a verb does. A package is placed into a repository once, by `kit install`, and kept current by
`kit update`; [packages.md](packages.md) describes it.

| | Module | Package |
| --- | --- | --- |
| Extends | how kit works | what kit can do in a repository |
| Is | a file of functions, in one language | rules, styles, templates, and files to deliver |
| Lives in | `kit/modules/<language>/`, and a repository's directories on the module path | `kit/packages/`, and a repository's `.kit/packages/` |
| Reaches a repository | never: it runs from the release | `kit install`, `kit update`, and `kit hydrate` place it |
| Example | `modules/bash/template.sh`, `modules/jq/template.jq` | `style/bash`, `github/ci` |

## Layout

| Directory | Holds |
| --- | --- |
| `bin/` | commands a person runs: `kit`, and each verb as `kit-<verb>`, with no extension |
| `modules/bash/` | bash modules: `<module>.sh`, sourced, never run |
| `modules/jq/` | jq modules: `<module>.jq`, imported by the jq programs the bash modules run |
| `packages/` | packages |

Every module sits under `modules/`, in a directory named for its language, so each language's loader searches one
directory of its own. A verb runs the same through `kit <verb>` and called directly as `kit-<verb>`. In this
repository all of it is under `kit/`, the release.

## Bash modules

The verbs are written in bash, and share code through bash modules, which a repository can extend with its own. The
rule is [.kit/rules/bash.md](../.kit/rules/bash.md), the style [.kit/styles/bash.md](../.kit/styles/bash.md).

### The module path

`use <module>` sources every `<module>.sh` on the module path, in order, once per process. The path is the
`bash_path` setting in `kit.yml`:

```yaml
bash_path:
  - kit:modules/bash/  # the kit's own modules, always first
  - modules/bash/      # this repository's, relative to its root
  - ~/lib/bash/        # the person's own, on this machine
  - /opt/team/bash/    # an absolute directory
```

| Entry | Directory |
| --- | --- |
| `kit:<dir>` | `<dir>` in the installed kit |
| `/<dir>` | that directory |
| `~/<dir>` | `<dir>` in the home directory |
| `<dir>` | `<dir>` relative to the repository root |

- Without the setting, the path is `kit:modules/bash/` alone.
- The path starts with `kit:modules/bash/`. A path that does not is refused.
- A `kit.yml` that names `bash_path` is read through `yq`. One `yq` cannot parse, or one read where `yq` is
  missing, is refused, and `use` returns 2: the kit's own modules never stand in for a path it could not read.
- A directory that does not exist is skipped, as `PATH` skips one. A root other than `kit:` must be a source that
  `sources:` in `kit.yml` names; see [packages.md](packages.md). Any other is refused.
- `use:path` prints the directories the path resolved to, one per line, which shows what was skipped.
- The path is read once per process tree. A kit command that another one runs takes the path its parent read,
  unless the release, the home directory, the cache directory, the repository root, or the text of `kit.yml`
  differs, so an edit to the setting is read at once. In the same way, `yq --version` runs once, and not again
  while the `yq` on `PATH` is the one it ran.

A module only one directory has loads from there. A module several directories have loads from each, in path
order, so a later file adds functions to the module or replaces one an earlier file defined. A module that no
directory has is a bug in the caller: `use` names the path it searched and returns 2.

### What every verb and module can call

`modules/bash/1st.sh` is the loader. Every verb sources it first. It sets `kit_root`, the release, and `kit_jq`, its
jq module directory, and defines:

| Function | Does |
| --- | --- |
| `use <module>…` | loads modules from the module path |
| `use:path` | prints the module path's directories |
| `use:modules <pattern>` | prints the modules on the path whose names match a glob: `use:modules 'forge/*'` |
| `kit:enter` | starts a verb's process: records where the command was typed, and moves to the repository root |
| `run <command> <argument>…` | runs a command in a new process: see [Processes](#processes) |
| `info`, `step`, `ok`, `section` | print a message to stderr |
| `warn`, `error` | print a warning or an error to stderr, and a workflow command on a GitHub runner; each remedy, the command after `run:`, gets `cd <folder> &&` in front, the folder kit runs in, so it runs from anywhere |
| `summary <line>…` | appends lines to the GitHub step summary, when there is one |
| `guard:require <predicate>…` | refuses, with the remedy, when a signing key or a tool is missing |
| `list:among <name> <name>…` | succeeds when the first name is one of the rest |
| `list:without <name> <name>…` | prints the rest without the first name, one per line |
| `list:read <array> <command> [<argument>…]` | returns the command's status, the array its lines, or none on failure |
| `list:read0 <array> <command> [<argument>…]` | does the same for items that each end in a NUL |
| `guard:arguments <verb> <argument>…` | judges a verb's arguments against its usage line and refuses in one line, pointing to `kit help <verb>`; at a terminal with kittui installed, a missing argument hands off to `kit tui -- <verb> <argument>…` instead |

Stdout carries only data, so a verb's output can be piped. Every message goes to stderr.

### The modules behind install and update

What a release delivers, and how, is split by concern into one module each, and a verb loads the ones it calls:

| Module | Functions | Decides |
| --- | --- | --- |
| `release.sh` | `release:` | a release: its version, where `kit update` fetches the previous one, and the bases it delivers to |
| `settings.sh` | `settings:` | `kit.yml`: every key read and validated, and the declaration written into it |
| `resolve.sh` | `resolve:` | the packages in reach, their metadata, and the active set from names, dependencies, and bundles |
| `render.sh` | `render:` | a package's templates and regions, rendered once per process with the repository's answers |
| `region.sh` | `region:` | the kit's region inside a file the repository owns: its markers, its host, written and removed |
| `place.sh` | `place:` | a delivered path: written, merged by key, compared with the release, and taken out again |
| `materialize.sh` | `materialize:` | what kit keeps under `.kit/`: a link or a read-only copy per path, and `.kit/.gitignore` |
| `manifest.sh` | `manifest:` | `.kit/manifest` and `~/.kit/manifest`: the digest of every delivered path |
| `file.sh` | `file:` | a file written whole or not at all, a file's digest, and the scratch space one EXIT trap sweeps |

### Writing a bash module

A bash module is `<module>.sh` in a directory on the path, or below one: the module `toolchain/brew` is
`toolchain/brew.sh`, and `use toolchain/brew` loads it. A name with an empty, `.`, or `..` part, or a leading `/`, is
refused, so no module loads from outside the path. A module:

- opens with `# shellcheck shell=bash` and the banner, and does not set any shell options, because it runs in its
  caller's shell
- names each function `<module>:<name>`, so that every function says where it is defined:
  `toolchain/brew:install` in `toolchain/brew.sh`
- loads what it needs with `use`, at the top
- reads a command's lines with `list:read` or `list:read0`, never through `< <(…)` alone, which loses the command's
  status: a failed read is never taken for an empty list
- returns non-zero to refuse, and leaves `exit` to the verb

A seam's providers sit in a directory named for the seam, beside it: `forge.sh` and `forge/github.sh`, and
`use:modules 'forge/*'` lists them. A provider counts wherever it is on the path. A repository adds one by putting it
in the seam's directory under a directory of its own on the path, such as `modules/bash/forge/local.sh`.

### Overriding a function

A later file on the path can replace a function an earlier file defined. Calls are bound when they run, so every
caller of the function, in any module, gets the replacement.

To wrap the function rather than replace it, load `super` and name the function before redefining it. `super` then
calls the definition that was replaced:

```bash
use super
super:override forge:issue-title
forge:issue-title() { super "$@" | sed 's/^\[WIP\] //'; }
```

Each override reaches the one before it, however many files override the same function. `super_origin` records the
file and line each replaced definition came from.

### Processes

Within the modules, work is a function call. Across a process boundary it is `run`:

```bash
run kit check
run kit-lint --all-files
```

`run kit` and `run kit-<verb>` run this release's own commands, not whichever `kit` is first on `PATH`. The command
runs with `kit_depth` one more than the caller's, so a trace can nest it under the process that started it.

### Writing a verb

A verb is an executable `kit-<verb>` in `bin/`. Its header is the only place its help is written, and it has the
shape of the verb template, `.kit/templates/bash/verb`, which the bash rule holds `kit/bin/kit-*` to: `kit lint`
refuses a header that leaves it, and `kit new verb <name> --purpose=<text> --group=<group>` makes one in it. `kit help`
reads the header through the template, every verb in one pass, and refuses one that leaves it, naming the template
line:

```bash
#!/usr/bin/env bash
# kit-<verb> - <purpose> (kit verb: <group>)
# ---…
# Usage:
#   kit <verb> [<argument>…]
#
# <description>
#
# Arguments:
#   <argument>    what it does
#
# Exits 0 when it did it, 1 when it could not, and 2 on a usage error.
#
# see: modules/bash/<module>.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=modules/bash/1st.sh
source "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")/../modules/bash/1st.sh"
kit:enter
use <module>
```

- The banner's purpose is what `kit help` lists beside the verb. Its group is the heading the verb is listed under:
  `workflow`, `files`, `checks`, `installation`, `interfaces`, `help`, or `plumbing`, in that order, as
  `help_groups` in `modules/bash/help.sh` names them. `plumbing` holds the verbs that scripts, the hooks, and the
  TUI call rather than a person; `kit help` sums them up in one line, and `kit help plumbing` lists them. A verb's
  page names its group and the verbs beside it.
- `kit usage <verb>` prints the usage lines alone, `kit help <verb>` the whole page, and `kit syntax <verb>` the
  grammar as JSON.
- The `Usage:` block holds a usage line for each way to call the verb, each indented three spaces after the `#`. It
  holds at least one: a verb that takes no arguments has `kit <verb>`. After `kit <verb>`, every word is one of:
  - a flag, `--name`
  - an option, `--name=<value>` or `--name <value>`
  - an operand, `<name>`
  - a lowercase literal

  They are grouped by `[ ]`, `( )`, and `|`, with `…` after what repeats. `kit syntax --check` refuses any other
  word, one with a dot or a capital, or `<a|b>`, naming the verb, the line, and the word.
- The description is every paragraph between the `Usage:` block and the `Arguments:` label, which `kit help <verb>`
  prints, reflowed to 80 columns. An indented list or a legend region stays in it. Its first line is not indented,
  since an indented line there would read as one more usage line.
- Every argument a usage line names has a line in the `Arguments:` block, indented three spaces after the `#`: the
  argument as the usage writes it, two or more spaces, and a fragment that says what it does, continued on lines
  indented further. A verb that takes none has the label over an empty block. `kit syntax <verb>` prints the usage
  and the block as JSON. `kit syntax --check`, which the gate and the pre-commit hook run, reports an argument that
  has no line.
- `Exits` says what each exit status means, and `see:` points into the source; both are left out of the usage, and
  `see:` out of the help page too.
- A verb exits:
  - 0 when it did what was asked
  - 1 when it could not, or when what it checks does not hold
  - 2 when the arguments are wrong, or a name given is not a verb, package, or path kit knows

  A verb that prints JSON, as `kit flow` and `kit workspace` do, exits 1 when a part cannot be read: with `--json`,
  what was read is still printed, and names in itself what could not, a remote's `error`, a member under `errors`,
  or a stage of `null`. Exit 1 with nothing on stdout says nothing could be read, and the last ❌ line on stderr says
  why. Exit 0 says the document is whole.

  A verb that runs another command may pass that command's status through, and says so. Every verb's header ends
  its description with one `Exits …` line stating its codes in those terms.
- The verb finds the loader beside its own real path, so it runs through a symlink, as Homebrew installs it.
- A verb has no `--help` of its own: `kit help <verb>` prints its header, and so does `kit <verb> --help` or `-h`.
  Every verb calls `guard:arguments <verb> "$@"` first. It judges the arguments against the usage line and refuses
  in one line that points to that page: an unknown flag or argument, a missing value, two alternatives together, or
  a required argument missing. The verb parses what the guard let through, and never restates the usage.
- A verb whose arguments have choices answers `kit syntax <verb> --choices` from a module named after it. The module
  defines two functions:
  - `<verb>:choices <key> --kind` prints `closed` or `open`.
  - `<verb>:choices <key> [<answer>…]` prints one row per value, tab-separated: the value, its description, its
    state (`active`, `excluded`, or none), and the line the verb would print about it given the answers so far,
    each `<key>=<value>`.

  It returns 1 for a key it does not answer. A closed set holds the only values the verb accepts, and
  `guard:arguments` refuses any other in the words the form uses. An open set only suggests; a value off it is
  never refused.
- A verb whose arguments grow with the answers, as `kit new` learns its template's holes, defines
  `<verb>:extend [<answer>…]`. It prints the arguments to add as an object of entries marked `extended`.
  `kit syntax <verb>` then carries `extends: true`, and `kit syntax <verb> --extend` prints them.

## jq modules

Much of the kit's logic is jq. A jq program that decides something, by folding, merging, matching, or selecting,
belongs in a jq module, where it is tested as jq, not quoted inside bash. Reading a field, or building an object
from arguments, stays inline where it is used.

| Module | Decides |
| --- | --- |
| `template.jq` | the expression that checks a file against a template, and the file a template makes |
| `forge.jq` | a pull request's checks and state, from either code forge's answers |
| `merge.jq` | a merge by key: a release's JSON over a repository's, keeping what the repository added |
| `flow.jq` | a checkout's document, what of it fits on the screen, and the changes the owner flags keep |
| `toolchain.jq` | what a tool manager pins and has installed, and which of its outdated packages the repository pins |
| `syntax.jq` | a syntax node's usage text, an argument's key, and how a command line matches a verb's syntax |
| `issues.jq` | every open issue, in the backlog or underway, with its change and its milestone, iteration, and swimlane |
| `backlog.jq` | the issues not closed by any open change, with the branch on the primary remote that claims each |
| `underway.jq` | the issues closed by an open change, one row per issue and change, and the rows the owner flags keep |
| `changes.jq` | the open changes that close one of the issues named |
| `completion.jq` | what the word being typed may be, from a verb's syntax: a flag, an option's value, a literal word, or an operand |

A jq module is `<module>.jq` in `modules/jq/`, a file of `def`s. A bash module runs it with `-L "${kit_jq}"` and
imports it under its own name, so the jq function reads as the bash function does, with `::` for `:`:

```bash
template:regex() {
  jq -L "${kit_jq}" --raw-input --slurp --raw-output 'import "template" as template; template::regex'
}
```

- A jq module opens with the banner, and a comment over each `def` says what it reads and what it gives.
- Its tests are `tests/jq/<module>.test`, in the format of `jq --run-tests`: a program, an input, and the outputs it
  gives. `tests/jq.bats` runs every module's tests, and fails a module that has none.
- jq compiles a program each time it starts, and has no compiled form to keep. Starting jq takes about 20 ms;
  importing a module does not add any measurable time. So speed comes from starting jq fewer times: one process for
  all the files a verb reads, not one per file.
- The jq modules come from the release alone. A repository has no jq module path of its own.

## YAML

The kit's files that are not JSON are YAML: `kit.yml`, each `package.yml`, the kinds files under `.kit/kinds/`, and
the frontmatter of a rule or style. `modules/bash/yaml.sh` reads them through `yq` 4, mikefarah's, one yq process
per read, with an expression in yq's own language. So a block scalar, a flow list, an anchor, or a CRLF line ending
reads as YAML says. `yaml:json` prints a file as JSON where the data enters a jq program. No awk or sed parses YAML.
A command reads the same file many times over, so each read's result is kept under
`${XDG_CACHE_HOME:-${HOME}/.cache}/kit/yaml/` beside the file's text, and replayed while the text is unchanged.
