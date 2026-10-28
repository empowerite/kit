# Architecture

kit has three parts: a release that a repository installs, commands that agents and people both run, and checks that
verify the work follows the rules. This page covers:

- what a release contains and how it reaches a repository,
- how kit finds the rules for a file,
- how a change moves from an issue to `main`, and
- which check verifies which claim.

[modules.md](modules.md) and [context.md](context.md) describe the module format and the rule lookup in detail.

## The release and its delivery

![Delivery: a release's modules, through the verbs, into a repository and a home](delivery.svg)

A release is one tarball, `kit-<version>.tar.gz`, with the same layout as this checkout:

- `bin/kit` is the command. Each verb is an executable `bin/kit-<verb>.sh` beside it. The verbs share bash modules,
  loaded with `use`.
- `modules/` holds everything a release delivers.
- `version` holds the release version. The release workflow writes it from the tag. A checkout has no `version` file
  and reports `unreleased`.

### Modules

A module is a directory under `modules/`, named after the module. It contains:

| Path | Purpose |
| --- | --- |
| `module.yml` | whether the module is on by default, which modules it depends on, and default answers for its templates |
| `files/<base>/` | files copied as they are to the same path under the base |
| `templates/<base>/` | files copied after each `@key@` is replaced by a setting |
| `regions/<base>/` | content kit owns inside a file the repository owns, rendered like a template |
| `docs/` | documentation, packed with the release and never installed |

There are two bases: `repo` is the repository root, and `home` is the user's home directory.

### The verbs that move content

Content reaches a repository only through these verbs:

- `kit install` resolves the active modules (`kit`, the ones named, and their dependencies), renders them, places the
  files, and writes the manifests.
- `kit update` moves to a newly pinned release. It replaces each file the repository never edited and merges each
  edited file three ways, with the previous release as the base.
- `kit check` fails when a delivered file differs from the release.
- `kit module` changes the set of active modules.

How a file is placed depends on its destination:

- A `.json` file, other than a schema, is merged by key. kit owns only the entries it writes.
- A region lives inside a file the repository owns. It starts at a `kit:generate` line naming the file's path and
  ends at a matching end line. `kit generate` replaces the region with the output of `kit region`. It does the same
  for every region any file in the repository declares; see [generate.md](generate.md).
- Every other file is copied.

### Manifests

Each base has a manifest:

- `.kit/manifest`, committed with the repository.
- `~/.kit/manifest` on a developer machine, when a module delivers to the home directory.

A manifest records the version, the active modules, and the release's digest of every delivered path. `kit update`
compares a file against its digest to tell whether the repository edited it.

## Finding a file's rules

![Context: how any agent, and Claude Code's hook, find the rules that govern a file](context.svg)

Each rule lives in one file, `.kit/rules/<kind>.md`, and is not copied into any agent tool's own directory. Its
frontmatter says which files it covers:

- `paths:` lists globs matched against the repository-relative path.
- `content:` lists patterns matched against the first line of an existing file.

`kit context <path>` prints the rules that cover a path and appends a line to `.git/kit/context.log`. Every caller
gets the same answer:

- an agent in any harness, following the instruction in `.kit/1st.md`,
- a person at the shell,
- a tool reading `--json`, and
- Claude Code's `PreToolUse` hook, which the `claude` module wires to `kit context --claude`, so the rules are shown
  before every write.

The log records which rules were shown, and when.

A rule also names its linter. `kit lint` runs each rule's linter over the files the rule covers, so a hook manager
only has to call `kit lint`. A rule's templates are skeleton files of its kind: `kit new file <path>` creates a file
from one, and `kit lint` checks that every file the rule covers keeps the shape of one; see
[templates.md](templates.md).

## The workflow and the gate

![The workflow: an issue becomes a worktree and a draft pull request, the gate checks it, a human merges](loop.svg)

The desk is the root checkout on the human's machine, on whatever branch it currently has, and it belongs to the
human. It usually has `main` checked out, and it may have any branch that no worktree is using. No agent changes it
in place.

Work starts from an issue:

1. `kit issue <N>` creates `worktrees/<N>-<slug>`, a branch linked to the issue, a start commit, and a draft pull
   request. If the desk has uncommitted work, `kit issue` moves it into the new worktree and says so. That is the
   only change it makes to the desk.
2. In the worktree:
   - `kit context` before changing a file,
   - `kit lint` to run the pre-commit linters by hand,
   - `kit rebase` to rebase the branch onto the default branch at origin and push it.
3. `gh pr ready` once the pull request body holds the report.
4. A human merges, and `kit tidy` removes the worktree and the branch.

### Forges, interfaces, and providers

A forge is the service that hosts a repository with its issues and pull requests, such as GitHub or GitLab. kit says
forge rather than GitHub because the workflow runs on more than one.

Four files each define an interface: a list of calls, and the rule that picks which provider implements them. Shell
has no types, so the choice is made at run time: a setting, or origin's host, becomes part of a function name. In
Michael Feathers' terms (*Working Effectively with Legacy Code*), each is a seam.

| Interface | Providers | Chosen by |
| --- | --- | --- |
| `bin/forge.sh` | `forge-github.sh` over `gh`, `forge-gitlab.sh` over `glab` | the `forge` setting, else origin's host |
| `bin/tracker.sh` | `tracker-forge.sh`, `tracker-ado.sh` | the reference's prefix, `#7` or `AB#4521`; see [trackers.md](trackers.md) |
| `bin/toolchain.sh` | `toolchain-mise.sh`, `toolchain-brew.sh`, `toolchain-docker.sh` | the `toolchains` setting, else the files present; see [toolchains.md](toolchains.md) |
| `bin/policy.sh` | `policy-patterns.sh`, `policy-opa.sh` | `policy_engine: opa` or a `.rego` file under `.kit/policy/`, else patterns; see [policy.md](policy.md) |

Every forge call the workflow makes goes through `forge:` in `bin/forge.sh`: an issue's title, the branch linked to
it, a pull request and its state, checks, review threads, and linked issues. A self-hosted forge must be named in the
`forge` setting.

- Every provider returns the same shape, so `kit status` shows the same picture on either forge.
- Each provider uses its forge's own name for a change: pull request `#3` on GitHub, merge request `!3` on GitLab.
- A repository with one origin has one forge. A repository on two forges is a mirror or a migration, and the setting
  picks the one the workflow uses.
- `tests/forge-conformance.bats` runs every provider against a fake of its tool and checks that each returns the
  same shape.
- The core verbs never call the forge, so a repository with no forge can still install, update, check, and run
  `kit context`. Only a provider needs its forge's tool.

### The checks

`kit ci` is the delivered check. The delivered GitHub Actions job runs it on every pull request into the default
branch, and any other pipeline can run it too. It runs four checks, each a kit command:

- `kit gate` runs the repository's own gate: the `gate` setting, or a detected justfile recipe, Makefile target, or
  `package.json` script. It fails when there is no gate, because a gate that cannot run verifies nothing. The gate
  runs with the tools the repository's toolchains manage (mise, Homebrew, or a container image); see
  [toolchains.md](toolchains.md).
- `kit check` fails when a delivered file differs from the release.
- `kit linked` fails when a pull request is not linked to an open issue.
- `kit policy` fails when a branch name, commit subject, or pull request title breaks the repository's policy. The
  policy is a set of patterns or an Open Policy Agent policy; see [policy.md](policy.md).

`kit status` shows everything in flight:

- what the last fetch brought in,
- the desk and every worktree, with its uncommitted paths and its commits since the branch point, and
- every open pull request, with its state, age, and checks.

It queries the forge in the background and draws the local checkouts while it waits.

## What verifies what

| Claim | Verified by |
| --- | --- |
| A delivered file matches the release | `kit check`, in the gate and locally, against the release's files and the manifest's version |
| A change has an issue | `kit linked` in the gate; `kit issue` creates the link when it starts the work |
| The repository's own gate ran | `kit gate`, which fails when there is no gate to run |
| Branch names, commit subjects, and titles follow the policy | `kit policy`, with patterns or Open Policy Agent |
| The gate ran with the pinned tools | `kit toolchain install`, then the gate run through the toolchain, locally and in CI |
| All four held, wherever CI runs | `kit ci`, which runs all four and reports each one |
| An agent was shown a file's rules | the Claude Code hook; `kit context` in other harnesses |
| An agent asked for a file's rules | `.git/kit/context.log`, one line per request |
| A file follows its rule's mechanical checks | `kit lint`, each rule's linter over the files it covers |
| A file keeps its rule's shape | `kit lint`, against the rule's templates; `kit new file` creates it in that shape |
| The forbidden git commands were not used | the harness's deny list, delivered by its adapter module |
| The desk was not changed | `kit rebase` refusing the root checkout; `kit issue` never switching its branch |

No check verifies that an agent followed a rule after reading it. The linters cover the mechanical part for the kinds
of file they handle. The rest is the report in the pull request body, which the integrity rules in `.kit/1st.md`
govern.
