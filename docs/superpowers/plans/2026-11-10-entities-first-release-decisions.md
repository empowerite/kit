# Entities First-Release Decisions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Settle the entities spec's four first-release decisions: a full reference on every row kit prints as JSON,
a person's file laid out by owner and reach, each seam's contract in one document, and a verdict a CI run can sign.

**Architecture:** Four parts, each shippable alone, in the order they lean on each other. kit builds a reference in
jq and puts it on every row, and kittui keys and links rows by it; a `config` module reads the person's file by
reach; one contract file lists every seam's calls, held by a suite; and `kit ci --json` prints the statement.

**Tech Stack:** bash, jq, yq, git, bats; Go and Bubble Tea for kittui.

**Spec:** `docs/superpowers/specs/2026-11-09-entities-design.md`, § What the first release settles (§ settles
below), with § Simplifications, § Code, § Configuration, and § Trust.

None of the four is met (Gaps, Entities spec). What the missing reference costs today is Apps
8, 9, and 28, and Flow 5 and 27. Not here: Apps 10 and Flow 13.

**Review:** [the findings](../reviews/2026-11-10-code-review-findings.md) of the review of 2026-11-10, cited
here by area and number: Serve 8 is item 8 under its Serve heading.

## Decisions for the owner

1. Names the spec leaves open: it fixes `…#7`, `…!3`, `…@<sha>`, and `sha256:…` (§ Simplifications 1), and says only
   "the repository's, and the branch" (§ Code). Default: git's full ref names after `@`; `:run/<id>` and
   `:remote/<name>`; a remote's branch as the clone tracks it; a release by its tag. Option: `:branch/main`.
2. The fields. Default: `reference` beside today's short `ref`, and `url`, the page; `kit workspace`'s rows lose
   `key`, and their `links` carry references, not `row:<stage>/<key>`. Option: keep `key`, two names for one row.
3. The person's keys: § Configuration names `all` and `selected`, and leaves "under the repository" unnamed.
   Default: `repositories`, by member, and `system_toolchain` under `all`; the spec's `toolchain: nix` is a project's.
4. The preferences. Default: `flow` alone; every other key is a setting, `log_days` too. The mark is the property's
   `"$comment"`: ajv, as `bin/ci:29` runs it, refuses a keyword of kit's own (tried). Option: `--strict=false`.
5. What v0.1.0 reads of the file. Default: every reach, a selection by `workspace` alone, any other selector
   refused by name. Option: validate the file, and read `system_toolchain` alone.
6. The move of `~/.kit/settings.yml`. Default: once, at the first read of the system choice, with one `→` line, as
   `kit:move-settings` does for a repository (`modules/bash/1st.sh:55-62`). Option: read the old file for ever.
7. The command provider. § settles 3 asks only that one be "possible for every seam the day one is needed".
   Default: the contract ships, and no `provider: command`. A command prints one JSON value and exits 0; 1 with the
   remedy on stderr; 2 for a call it lacks. Option: an error object on stdout.
8. The statement. Default: `predicateType` `https://github.com/empowerite/kit/verdict/v1`; the subject is HEAD, on
   a pull request's run the merge commit the checks ran on, not the change's head; the policy's version is a digest.

## Global Constraints

- kit commit only, never kit submit or push; every subject starts with `#135`, as `#135 kit: …`.
- `export PATH=$PWD/kit/bin:$PATH` before `kit commit`.
- Peers share the worktree: stage named paths only, guard that the index holds only them, never leave anything staged.
- Everything under `kit/` is written for a consumer, and a path in it is written from `kit/`: `modules/bash/x.sh`.
- Lines are 120 cells at most.
- A verb's header comment is its help page and its usage line is its grammar (`kit syntax --check` holds them): a new
  flag is added to the header's usage line and described there, and `docs/` says the same.
- kit builds a reference from what git and the forge say; kittui never does. Where kit cannot name a row's place,
  `reference` and `url` are null, and kittui keys the row as today and links it nowhere.
- Without `config.yml` and `--json`, every verb prints what it prints today; an existing test changes only where a
  task says so; no golden of kittui's changes, and its coverage stays at 98% or more.
- Bats runs as `KIT_TOOLCHAIN=mise kit/bin/kit tools run bats <files>`; Go as `cd projects/tui && go test -cover
  ./...`. A commit stages its task's Files and ends `--message "Co-Authored-By: Claude <noreply@anthropic.com>"`.

## Review Focus

1. A repository no service hosts, as after `git init`: its references stand at its desk's path (Task 1).
2. A change that closes `other/repo#9` beside an open local `#9`: the local one stays in the backlog (Task 1).
3. A document from before kit printed references, as `--cached` gives: kittui keys it as today (Task 2).
4. The old file and the new both name a system toolchain: the new stands, and a warning names the old (Task 3).
5. `kit ci --json` with no commit refuses before any check; on a desk it prints `"run": null` (Task 5).

---

## Part 1: References

A reference is a place, then a sigil and a key for what the place holds. A place is a member, host and path, or,
for what only this machine holds, a path; an artifact is `sha256:<64 hex>`. `…` is `github.com/empowerite/kit`.

| Subject | Reference | Example, and the rows that carry it |
| --- | --- | --- |
| repository | `<host>/<path>` | `github.com/empowerite/kit`; `kit workspace repos` |
| issue, GitHub pull request | `<place>#<number>` | `…#7`, `…#3`; backlog, underway, changes |
| GitLab merge request | `<place>!<number>` | `gitlab.com/group/sub/repo!3`; changes |
| Azure DevOps work item | `<host>/<organization>/<project>#<id>` | `dev.azure.com/example/Claims#4521`, short `AB#4521`; none until #77 |
| commit | `<place>@<40 hex>` | `…@2dad35529c403673effe27426a583a9f6bd0c199`; commits |
| branch, and a remote's | `<place>@refs/heads/<branch>`, `<place>@refs/remotes/<remote>/<branch>` | `…@refs/heads/scratch`, of kind `local`; `…@refs/remotes/origin/main`, of kind `remote` or `gone`; branches |
| tag, and its release | `<place>@refs/tags/<tag>` | `…@refs/tags/release/0.1.0`; tags, releases |
| remote | `<place>:remote/<name>` | `…:remote/upstream`; remotes |
| run | `<place>:run/<id>` | `…:run/42778922819`; builds |
| checkout, worktree | its absolute path | `/Users/ada/code/kit/worktrees/7-fix`; worktrees |

### Task 1: Every row kit prints carries its reference, and a forge's row its page

**Files:**

- Create: `kit/modules/jq/reference.jq`, `tests/jq/reference.test`
- Modify: `kit/modules/jq/{forge,workspace,underway,backlog}.jq`, `kit/modules/bash/forge.sh:288-291,329-389`,
  `kit/modules/bash/{workspace,across}.sh`, `kit/bin/kit-flow:95-106`, `kit/bin/kit-workspace:53-54,63`,
  `docs/architecture.md`, `tests/jq/*.test`, `tests/forge-conformance.bats`, `tests/workspace.bats`

**Interfaces:**

- Produces:
  - `reference::of($place; $sigil; $key)`: the three joined; null when the place or the key is null or empty.
  - `reference::page($forge; $place; $kind; $key)`: `https://<place>/`, then GitHub's `issues/`, `pull/`, or
    `releases/tag/`, or GitLab's `-/issues/`, `-/merge_requests/`, or `-/releases/`, for an `issue`, a `change`, or a
    `release`, then the key; null for another forge, or a place that is a path.
  - `forge::referenced($kind; $forge; $place)` adds `reference` and `url` to a document: a `change` by its `ref`; an
    `issue`, and each of a change's `closes`, by `#` and its number, at the place's host and its own `organization`
    and `repository`; a `release` by its tag; a `run` by `:run/` and its `id`, its `url` kept.
  - `forge:place` prints the primary's member. `forge:documents <call>` pipes `forge:call <call>` through
    `forge::referenced`; it replaces `forge:call` at `forge.sh` lines 333, 362, and 386, and in `forge:releases`
    and `forge:runs`. `underway::rows` keeps the issue's fields, and its `pr` gains `ref`, `reference`, and `url`;
    `backlog::unclaimed` matches a closed issue by `reference`, by `number` where either has none.
  - `workspace:place` prints `workspaces:member-of <desk>`, else the desk's path; each stage's function in
    `workspace.jq` takes it last. A row has `reference`, as the table gives its stage, and no `key`; `row` (line 32)
    goes, and a link to a row is its reference: a worktree's `branch`, its upstream's where it has one.

- [ ] **Step 1: Write the failing tests.** `reference.test`: `of("gitlab.com/g/s/r"; "!"; 3)` is
  `"gitlab.com/g/s/r!3"`, and null for an empty place or a null key. `forge.test`: change `#3` at `github.com/o/r`,
  closing number 9 of organization `other`, repository `repo`, has `reference` `github.com/o/r#3`, `url`
  `https://github.com/o/r/pull/3`, and `closes[0].reference` `github.com/other/repo#9`; `backlog.test`: issue
  `github.com/o/r#9` stays when a change closes that one; `underway.test`: a change `!3` gives `pr.ref` `!3`.
  `workspace.test` and `tests/workspace.bats:69-118` expect `reference` for `key` and for `row:`; after `git init`,
  `main`'s is `$(pwd -P)@refs/heads/main`. `tests/forge-conformance.bats:260,421,525,561` assert each `.reference`,
  and delete the new fields before they compare documents. And in `tests/workspace.bats`:

```bash
@test "every row of kit workspace carries its full reference, and a link to a row is that row's reference" {
  history-at "${BATS_TEST_TMPDIR}/r" github.com/o/r && cd "${BATS_TEST_TMPDIR}/r" && p=github.com/o/r
  run "${kit_root}/bin/kit" workspace remotes --json
  [ "$(jq -c '.rows | map([.reference, .links.default, has("key")])' <<<"$output")" \
    = "[[\"$p:remote/origin\",\"$p@refs/remotes/origin/main\",false]]" ]
  run "${kit_root}/bin/kit" workspace branches --json
  [ "$(jq -c '[.rows[] | select(.kind != "remote" or .branch == "main") | .reference]' <<<"$output")" \
    = "[\"$p@refs/remotes/origin/main\",\"$p@refs/remotes/origin/old\",\"$p@refs/heads/scratch\"]" ]
}
```

- [ ] **Step 2: Run** `tests/jq.bats tests/forge-conformance.bats tests/workspace.bats`. Expected: they fail.
- [ ] **Step 3: Write the functions above,** `### References` in `docs/architecture.md`, and the two headers.
- [ ] **Step 4: Run** them with `tests/{flow,backlog,underway,changes,builds,releases,across}.bats`. Expected: pass.
- [ ] **Step 5: Commit**, subject `#135 kit: every row of kit flow and kit workspace carries its full reference`.

### Task 2: kittui keys and links rows by kit's references

**Files:**

- Create: `projects/tui/internal/reference/reference.go`, `reference_test.go`
- Modify: in `projects/tui/internal/flow/`, `rows.go:18-199,496-525,551-649,843-975`, `workspace.go:17-94`,
  `origin.go:46-95`, `flow.go:125,211`, their tests, and the stages' fixtures under `testdata/`

**Interfaces:**

- Consumes: Task 1's fields.
- Produces:
  - `reference.Parse(text string) (Reference, bool)`, `type Reference struct { Kind Kind; Place, Key string }`, the
    kinds `Repository`, `Issue`, `Change`, `Commit`, `Branch`, `Tag`, `Remote`, `Run`, `Checkout`, `Artifact`. A
    place on a service holds a slash and ends at the first of `#`, `!`, `@`, `:`; `#` is `Issue`, as GitHub writes a
    pull request too. A text that starts with `/` is a `Checkout`, unless it holds `@refs/`, split at the first,
    ends in `@` and 40 hex, or holds `:remote/` or `:run/`.
  - `func (m Model) keyed(member, reference, key string) string`: the reference, else `m.keyOf(member, key)`. Every
    stage keys its rows by it; underway by the issue's reference, a space, and the change's.
  - A page is kit's `url`, or none: `originOf`, `pullKey`, `Origin`'s `issue`, `release`, `pull`, `gitlab`, and
    `infix`, `Model.origin`, and `underway.PR.Source` go; `Options.Origin` stays, for the 📚 menu's fallback.
  - `func (m Model) workspaceLink(link string) string`: `http://` or `https://`, as it is; a reference,
    `jumpTo("workspace/<stage>", link)`, the stage by its kind; else nothing. `here` links a worktree by its path.

- [ ] **Step 1: Write the failing tests.** `TestParse` is a table of Part 1's rows, each with its kind, place, and
  key, then the two lines below; the third's texts do not parse. The fixtures gain `reference` and `url` for
  `github.com/empowerite/kit`, in Task 1's shape. `TestRowsAreKeyedByKitsReferences`: after `loaded(t, options())`,
  `backlog` holds a row keyed `…#45`, `underway` `…#1 …#2`, `changes` `…#2`, `builds` `…:run/41309409501`, and
  `releases` `…@refs/tags/release/0.2.0`; `backlogRows` of issue 7 of member `github.com/o/r`, with no reference,
  is keyed `github.com/o/r #7`, and its title links nowhere; an underway row links its change to `jumpTo("changes",
  "gitlab.com/g/s/r!3")`, its `pr.reference`. `workspaceLink` gives `workspace/branches` of
  `github.com/o/r@refs/remotes/origin/main`, `workspace/worktrees` of `/src/r`, and nothing of `row:tags/v1`.

```go
"github.com/o/r@refs/remotes/my/fork/topic": {Branch, "github.com/o/r", "refs/remotes/my/fork/topic"},
"/src/a@b/r@refs/heads/x@y":                 {Branch, "/src/a@b/r", "refs/heads/x@y"},
"", "#7", "kit", "github.com/o/r#x", "https://github.com/o/r", "github.com/o/r:deploy/1", "sha256:abc", "row:tags/v1"
```

- [ ] **Step 2: Run** `go test ./internal/reference/ ./internal/flow/`. Expected: no build, `Parse` undefined.
- [ ] **Step 3: Write the package and the changes above;** `TestTheURLsFollowTheHost` goes with what it tests.
- [ ] **Step 4: Run** `go test -cover ./...` and `tests/tui.bats`. Expected: pass, 98.0% or more, no golden rewritten.
- [ ] **Step 5: Commit**, subject `#135 kittui: a row is keyed by kit's reference and opens kit's page`.

## Part 2: Owner and reach

§ Configuration puts a person's values in "`~/.config/kit/`, under `all`", "under a selection", and "under the
repository, never checked in", and says "Each key declares in the schema whether it is a setting or a preference".
kit's files say `${XDG_CONFIG_HOME:-~/.config}/kit/`, and `config.yml` follows them. A person's machine holds today:

- the system toolchain, `${KIT_HOME:-~}/.kit/settings.yml` (`modules/bash/toolchain.sh:344,368,386-388`), which moves;
- beside `config.yml`: `workspaces.yml` (`workspaces.sh:20`), membership; kittui's `kittui.json` and `presets.json`
  (`internal/shell/settings.go:30`, `internal/table/store.go:21`), for `all.kittui` later; and its
  `search-history.json` (`internal/shell/recent.go:32`), local state; under `${XDG_CACHE_HOME:-~/.cache}/kit/`,
  local state (`workspaces.sh:23`, `sources.sh:34`, `yaml.sh:94`, `tui.sh:43`, `release.sh:88`); and under
  `${KIT_HOME:-~}/.kit/`, a home install's files (`release.sh:48`). All of these stay.

### Task 3: The person's file, by reach, with each key's kind and the system toolchain

**Files:**

- Create: `kit/modules/bash/config.sh`, `tests/config.bats`, `docs/configuration.md`
- Modify: `kit/packages/kit/files/repo/.kit/kit.schema.json` and `.kit/kit.schema.json`, the copy `bin/refresh`
  writes; `kit/modules/bash/settings.sh:244-285`; `kit/modules/bash/toolchain.sh:26-30,319-350,359-370,386-389`;
  `docs/toolchains.md:87`; `tests/toolchains.bats:421-452,521`

**Interfaces:**

- Produces:
  - `config:file` prints `${XDG_CONFIG_HOME:-${HOME}/.config}/kit/config.yml`: `all`, a map of keys; `selected`, a
    list, each `repos: {workspace: <group>}` and keys; `repositories`, a map of keys by member.
  - `config:kind <key>` prints `setting` or `preference`, the property's `$comment` in the schema, `preference` for
    `system_toolchain`, the person's own key, which `all` alone takes, and returns 1 for any other.
  - `config:validate` returns 1 after a line for each of: `<file>: '<key>' is no reach; the reaches are all,
    selected, and repositories`; `<file>: <key> is a setting, which kit.yml and the organization decide`; `<file>:
    kit knows no key '<key>'`; `<file>: a selection selects by workspace for now, not by '<selector>'`.
  - `config:preference <key>` validates, then prints the file that gives the key, a tab, and its value as one line
    of JSON, or nothing: the first of `repositories.<member>`, `kit.yml`, the last entry of `selected` whose group's
    `workspaces:fixed` holds the member, and `all`. `settings:stages` reads `flow` through it.
  - `config:moved` runs where `${KIT_HOME:-${HOME}}/.kit/settings.yml` has a `toolchain`: with no
    `all.system_toolchain`, it writes it, removes the old key, and the old file once empty, and says `→ <file>: the
    system toolchain moved here from <old>`; with one, it warns `<old> still names a system toolchain, which kit
    no longer reads; remove it`. `toolchain:recorded` and `toolchain:choose` call it, then read and write that key.

- [ ] **Step 1: Write the failing tests.** In `tests/config.bats`, `setup` runs `hide-system-toolchains` and
  `fresh-repo`, adds origin `https://github.com/o/a.git`, and sets `kit_bash_modules='config settings'`. One test
  asserts that `flow` alone has the `$comment` `preference`, and `config:kind`'s answers; one gives `all: {policy:
  …}`, a reach `every`, and a selection by `topic`, and asserts status 1 and `config:validate`'s first, second, and
  fourth lines; one, that with `all: {flow: [desk]}` and a selection of `w`, which holds `github.com/o/a`, with
  `flow: [builds]`, `settings:stages` prints `builds`, then `changes` once `kit.yml` says so, then `backlog` once
  `repositories` does. The test below replaces `toolchains.bats:421`; line 521 reads `.all.system_toolchain`.

```bash
@test "the system choice is in the person's file, under all, and one in ~/.kit/settings.yml moves there once" {
  KIT_HOME=$(mktemp -d) && export KIT_HOME && mkdir "${KIT_HOME}/.kit"
  old=${KIT_HOME}/.kit/settings.yml new=${XDG_CONFIG_HOME}/kit/config.yml
  printf 'theme: dark\ntoolchain: brew\n' > "${old}"
  run kit-bash 'use toolchain && toolchain:recorded system'
  [[ "$output" == *"${new}: the system toolchain moved here from ${old}"* ]]
  [ "$(yq '.toolchain // "none", .theme' "${old}")" = $'none\ndark' ]
  printf 'toolchain: mise\n' > "${old}"
  run choose ambient system
  [[ "$output" == *"${old} still names a system toolchain, which kit no longer reads; remove it"* ]]
  [ "$(yq .all.system_toolchain "${new}")" = ambient ]
}
```

- [ ] **Step 2: Run** `tests/config.bats tests/toolchains.bats`. Expected: fail, `config` is no module.
- [ ] **Step 3: Write `config.sh`, mark the schema's eighteen properties, and move `settings:stages` and the system
  choice onto it.** `docs/configuration.md` holds the three keys, the order above, each key's kind, and this list.
- [ ] **Step 4: Run** them with `tests/{flow,kit,tools}.bats` and `bin/ci`'s ajv line. Expected: pass.
- [ ] **Step 5: Commit**, subject `#135 kit: a person's preferences are in ~/.config/kit/config.yml, by reach`.

## Part 3: The seams' contracts

Every provider answers every call its seam requires. Three calls are optional: the toolchain's `remedy`, which mise,
nix, docker, and devcontainer lack, and `run`, which only docker and devcontainer have, and the policy's `warn`,
which patterns lacks. No test calls the forge's `require`, `pr-set-body`, `pr-last-for-branch`, or `pr-here`.

### Task 4: One contract for the five seams, and a suite that holds every provider to it

**Files:**

- Create: `kit/modules/seams.json`, `docs/seams.md`, `tests/seams.bats`
- Modify: `kit/modules/bash/forge.sh:51`, `kit/modules/bash/tracker.sh:48`, `docs/trackers.md:90-105`,
  `docs/architecture.md:149-187`, `tests/forge-conformance.bats`

**Interfaces:**

- Produces: `modules/seams.json`, by seam, `forge`, `tracker`, `toolchain`, `dependencies`, `policy`, each `{module,
  calls}`, a call `{call, arguments, prints, json, none, optional, changes}`. `call`, `arguments`, and `prints` are
  the header's; `none` is `nothing` for a call marked "or nothing"; `json` is what a command prints: a string for
  one text, and `null` for no match or for a call that changes something; `true` or `false` for a call that only
  succeeds or fails, as `usable`; an array of strings for one item a line, as `path`, or space-separated, as
  `assignees`; objects for tab-separated fields, `noun` `{sigil, name}`, `pr-state` `{state, commit}`,
  `pr-closing-issues` `[{number, state, repository}]`, `inputs` `[{kind, name}]`, `pins` `[{name, pinned,
  installed}]`, `outdated` `[{name, installed, latest}]`; and JSON as it is, one array for a document a line.
  `docs/seams.md` holds decision 7's protocol and a table a seam: `Call`, `Prints`, `A command prints`, `No answer`.

- [ ] **Step 1: Write the failing tests.** In `tests/seams.bats`: "each seam's header lists the contract's calls,
  with their arguments" compares each seam's calls, as `<call> <arguments>` lines, with its header's, read between
  `# Every provider implements the calls below` and `# see:` by this step's sed; "docs/seams.md has a row for every
  call" greps ``| `<call> <arguments>` `` for each; "every provider answers every call its seam requires" asserts
  that the optional calls are `remedy`, `run`, and `warn`, and that each `modules/bash/<seam>/*.sh` defines
  `<seam>/<provider>:<call>` for every other. In `tests/forge-conformance.bats`: "every provider answers require,
  pr-last-for-branch, pr-here, and pr-set-body the same way", HEAD on `7-fix-the-thing`: the first and `pr-set-body
  3 'a new body'` exit 0, the others print `3`. The sed: `sed -E -n 's/^#   ([a-z][a-z-]*( [^ ]+)*)(  .*)?$/\1/p'`.
- [ ] **Step 2: Run** `tests/seams.bats tests/forge-conformance.bats`. Expected: fail, no `modules/seams.json`.
- [ ] **Step 3: Write `seams.json` from the five headers, and `docs/seams.md` from it,** which `docs/trackers.md`
  and `docs/architecture.md` point at; `forge.sh:51` and `tracker.sh:48` put two spaces before their description.
- [ ] **Step 4: Run** them with `tests/modules.bats`, then `kit lint`. Expected: pass.
- [ ] **Step 5: Commit**, subject `#135 kit: each seam's calls are one contract, which holds every provider`.

## Part 4: A verdict a CI run can sign

§ settles 4 names the flag: "`kit ci --json` prints its verdict as an unsigned in-toto statement"; § Trust, the
form: "an in-toto statement in a signed envelope", whose "subject is an artifact's digest, or a commit by its
hash". A job signs the file `kit ci --json > kit-verdict.json` wrote; no delivered workflow changes.

### Task 5: kit ci --json prints its verdict as an unsigned in-toto statement

**Files:**

- Create: `kit/modules/bash/verdict.sh`
- Modify: `kit/bin/kit-ci`, `kit/modules/bash/policy.sh`, `docs/architecture.md` § The checks, `tests/ci.bats`

**Interfaces:**

- Produces:
  - `kit ci [--json | --jq]`. With `--json`, each check's output goes to stderr, stdout holds the statement, one
    line, and it exits 0 or 1 as without it. With no commit at HEAD, it runs no check, says `kit ci --json speaks
    of a commit, and HEAD has none`, and exits 1.
  - `policy:digest` prints `sha256:` and `file:digest` of `policy:rules`' JSON and then each `.rego` under
    `policy_rego`, in `LC_ALL=C` order of path.
  - `verdict:statement <check>=<passed|failed>…` prints the statement. The subject's name is `workspaces:member-of`
    at the root, else the root's path; `kit` is `release:version`; `run` is the name, `:run/`, and `GITHUB_RUN_ID`,
    else `CI_PIPELINE_ID`, else null; `at` is the second, in UTC.

```json
{"_type": "https://in-toto.io/Statement/v1",
 "subject": [{"name": "github.com/empowerite/example", "digest": {"gitCommit": "2dad35529c403673effe27426a583a9f6bd0c199"}}],
 "predicateType": "https://github.com/empowerite/kit/verdict/v1",
 "predicate": {"verdict": "failed", "kit": "0.1.0",
   "checks": [{"name": "gate", "verdict": "passed"}, {"name": "check", "verdict": "passed"},
              {"name": "linked", "verdict": "failed"}, {"name": "policy", "verdict": "passed"}],
   "policy": {"engine": "patterns", "digest": "sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"},
   "run": "github.com/empowerite/example:run/42778922819", "at": "2026-11-10T17:04:05Z"}}
```

- [ ] **Step 1: Write the failing tests** in `tests/ci.bats`, in its `ci-world`, where `kit linked` alone refuses;
  a second asserts, before any commit, status 1, no stdout, and the line above; then, on a desk, `"run": null`.

```bash
@test "kit ci --json prints its verdict as an unsigned in-toto statement about HEAD, alone on stdout" {
  printf 'ci:\n  echo gate ran\n' > justfile && git add --all && git commit --quiet --message '#1 a gate'
  run --separate-stderr env -u GITHUB_EVENT_PATH GITHUB_RUN_ID=42 "${kit_checkout}/bin/kit" ci --json
  [[ "$status" -eq 1 && "$stderr" == *"gate ran"* && "$stderr" == *"1 of 4 checks refused"* ]]
  [[ "$(jq -r '._type + " " + .predicateType' <<<"$output")" == *in-toto.io/Statement/v1\ *empowerite/kit/verdict/v1 ]]
  [ "$(jq -c .subject <<<"$output")" \
    = "[{\"name\":\"github.com/empowerite/example\",\"digest\":{\"gitCommit\":\"$(git rev-parse HEAD)\"}}]" ]
  [ "$(jq -c '[.predicate.checks[] | [.name, .verdict]]' <<<"$output")" \
    = '[["gate","passed"],["check","passed"],["linked","failed"],["policy","passed"]]' ]
  [ "$(jq -r '.predicate | .verdict, .run' <<<"$output")" = $'failed\ngithub.com/empowerite/example:run/42' ]
  [[ "$(jq -r '.predicate | .policy.digest + " " + .at' <<<"$output")" =~ ^sha256:[0-9a-f]{64}\ [0-9T:-]{19}Z$ ]]
}
```

- [ ] **Step 2: Run** `tests/ci.bats`. Expected: the two fail with 2, `kit ci` takes no `--json`.
- [ ] **Step 3: Write `verdict.sh` and `policy:digest`, and give `kit-ci` its usage line**, `# kit ci [--json |
  --jq]`, with a line for each flag, as `kit-toolchains` has; the docs say that on a desk the statement is a finding.
- [ ] **Step 4: Run** it with `tests/{help,syntax,serve,completion}.bats`, `kit lint`, and `kit syntax --check`.
- [ ] **Step 5: Commit**, subject `#135 kit: kit ci --json prints its verdict as an unsigned in-toto statement`.
