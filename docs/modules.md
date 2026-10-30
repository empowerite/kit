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
`bash_path` setting in `.kit/settings.yml`:

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
- A directory that does not exist is skipped, as `PATH` skips one. A root other than `kit:` is refused.
- `use:path` prints the directories the path resolved to, one per line, which shows what was skipped.

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
| `use:modules <pattern>` | prints the modules on the path whose names match a glob: `use:modules 'forge-*'` |
| `kit:enter` | starts a verb's process: records where the command was typed, and moves to the repository root |
| `run <command> <argument>…` | runs a command in a new process: see [Processes](#processes) |
| `info`, `step`, `ok`, `section` | print a message to stderr |
| `warn`, `error` | print a warning or an error to stderr, and a workflow command on a GitHub runner |
| `summary <line>…` | appends lines to the GitHub step summary, when there is one |
| `guard:require <predicate>…` | refuses, with the remedy, when a signing key or a tool is missing |

Stdout carries only data, so a verb's output can be piped. Every message goes to stderr.

### Writing a bash module

A bash module is `<module>.sh` in a directory on the path. It:

- opens with `# shellcheck shell=bash` and the banner, and sets no shell options, because it runs in its caller's
  shell
- names each function `<module>:<name>`, so that every function says where it is defined
- loads what it needs with `use`, at the top
- returns non-zero to refuse, and leaves `exit` to the verb

A provider of a seam, such as `forge-<name>.sh` or `toolchain-<name>.sh`, counts wherever it is on the path. A
repository adds a provider by putting it in a directory of its own on the path, such as `modules/bash/`.

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
# see: modules/bash/<module>.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=SCRIPTDIR/../modules/bash/1st.sh
source "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")/../modules/bash/1st.sh"
kit:enter
use <module>
```

- The banner's purpose is what `kit help` lists beside the verb.
- A usage line is a paragraph of one line that starts with `kit <verb>`. A verb can have several. A verb with none
  takes no arguments.
- Every other paragraph is the description, which `kit help <verb>` prints. `see:` lines point into the source, and
  are left out.
- The verb finds the loader beside its own real path, so it runs through a symlink, as Homebrew installs it.

## jq modules

Much of the kit's logic is jq. A jq program that decides something, by folding, merging, matching, or selecting,
belongs in a jq module, where it is tested as jq, not quoted inside bash. Reading a field, or building an object
from arguments, stays inline where it is used.

| Module | Decides |
| --- | --- |
| `template.jq` | the expression that checks a file against a template, and the file a template makes |
| `forge.jq` | a pull request's checks and state, from either forge's answers |
| `merge.jq` | a merge by key: a release's JSON over a repository's, keeping what the repository added |
| `status.jq` | a checkout's document, and what of it fits on the screen |
| `toolchain.jq` | which of a tool manager's outdated packages the repository pins |

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
- jq compiles a program each time it starts, and has no compiled form to keep. Starting jq takes about 20 ms, and
  importing a module adds no measurable time to that, so speed comes from starting jq fewer times: one jq process
  for all the files a verb reads, not one per file.
- The jq modules come from the release alone. A repository has no jq module path of its own.
