# Bash modules

The kit is written in bash. Its commands share code through bash modules, which a repository can extend with its
own. This page says how the module system works. The style every bash file follows is in the bash rule,
[.kit/rules/bash.md](../.kit/rules/bash.md).

A bash module is a sourced `.sh` file of functions. It is not a package: a package is what the kit delivers into a
repository, under `kit/packages/`, described in [packages.md](packages.md).

## Layout

| Directory | Holds |
| --- | --- |
| `bin/` | commands a person runs: `kit`, and each verb as `kit-<verb>`, with no extension |
| `bash/` | modules: `<module>.sh`, sourced, never run |

A verb runs the same through `kit <verb>` and called directly as `kit-<verb>`. In this repository both are under
`kit/`, the release.

## The module path

`use <module>` sources every `<module>.sh` on the module path, in order, once per process. The path is the
`bash_path` setting in `.kit/settings.yml`:

```yaml
bash_path:
  - kit:bash/        # the kit's own modules, always first
  - bash/            # this repository's, relative to its root
  - ~/lib/bash/      # the person's own, on this machine
  - /opt/team/bash/  # an absolute directory
```

| Entry | Directory |
| --- | --- |
| `kit:<dir>` | `<dir>` in the installed kit |
| `/<dir>` | that directory |
| `~/<dir>` | `<dir>` in the home directory |
| `<dir>` | `<dir>` relative to the repository root |

- Without the setting, the path is `kit:bash/` alone.
- The path starts with `kit:bash/`. A path that does not is refused.
- A directory that does not exist is skipped, as `PATH` skips one. A root other than `kit:` is refused.
- `use:path` prints the directories the path resolved to, one per line, which shows what was skipped.

A module only one directory has loads from there. A module several directories have loads from each, in path
order, so a later file adds functions to the module or replaces one an earlier file defined. A module that no
directory has is a bug in the caller: `use` names the path it searched and returns 2.

## What every verb and module can call

`bash/1st.sh` is the loader. Every verb sources it first. It defines:

| Function | Does |
| --- | --- |
| `use <module>…` | loads modules from the module path |
| `use:path` | prints the module path's directories |
| `use:modules <pattern>` | prints the modules on the path whose names match a glob: `use:modules 'forge-*'` |
| `kit:enter` | starts a verb's process: records where the command was typed, and moves to the repository root |
| `run <command> <argument>…` | runs a command in a new process: see [Processes](#processes) |
| `info`, `step`, `ok`, `section` | print a message to stderr |
| `warn`, `error` | print a warning or an error to stderr, and a workflow command on a GitHub runner |
| `summary <line>…` | appends lines to the GitHub step summary, when there is one |
| `guard:require <predicate>…` | refuses, with the remedy, when a signing key or a tool is missing |

Stdout carries only data, so a verb's output can be piped. Every message goes to stderr.

## Writing a module

A module is `<module>.sh` in a directory on the path. It:

- opens with `# shellcheck shell=bash` and the banner, and sets no shell options, because it runs in its caller's
  shell
- names each function `<module>:<name>`, so that every function says where it is defined
- loads what it needs with `use`, at the top
- returns non-zero to refuse, and leaves `exit` to the verb

A provider of a seam, such as `forge-<name>.sh` or `toolchain-<name>.sh`, counts wherever it is on the path. A
repository adds a provider by putting it in its own `bash/` directory.

## Overriding a function

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

## Processes

Within the modules, work is a function call. Across a process boundary it is `run`:

```bash
run kit check
run kit-lint --all-files
```

`run kit` and `run kit-<verb>` run this release's own commands, not whichever `kit` is first on `PATH`. The command
runs with `kit_depth` one more than the caller's, so a trace can nest it under the process that started it.

## Writing a verb

A verb is an executable `kit-<verb>` in `bin/`. Its header is the only place its help is written, and `kit help`
reads it:

```bash
#!/usr/bin/env bash
# kit-<verb> - <purpose> (kit verb)
# ---…
#
# kit <verb> [<argument>…]
#
# <description>
#
# see: bash/<module>.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=SCRIPTDIR/../bash/1st.sh
source "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")/../bash/1st.sh"
kit:enter
use <module>
```

- The banner's purpose is what `kit help` lists beside the verb.
- A usage line is a paragraph of one line that starts with `kit <verb>`. A verb can have several. A verb with none
  takes no arguments.
- Every other paragraph is the description, which `kit help <verb>` prints. `see:` lines point into the source, and
  are left out.
- The verb finds the loader beside its own real path, so it runs through a symlink, as Homebrew installs it.
