# Workspaces, step 1: `kit workspaces` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A `kit workspaces` verb that reads `~/.config/kit/workspaces.yml`, resolves a group's fixed members to their
local checkouts, applies the startup rule, and edits the file, all with `--json` for kittui to read later.

**Architecture:** One sourced module, `kit/modules/bash/workspaces.sh`, holds every rule; the verb
`kit/bin/kit-workspaces` parses arguments and prints. Checkouts are found by walking the configured roots with the
desk walker `kit flow` already has, moved to `git.sh` so both share it. A checkout's member is its primary remote's
host and path. Only kit reads and writes the file; writes go through `yq --inplace`, which keeps comments.

**Tech Stack:** bash 5, git, jq, mikefarah yq 4 (through `yaml.sh`), bats.

**Spec:** `docs/superpowers/specs/2026-11-08-workspaces-design.md`, steps "The workspace file", "Members and
checkouts", "Which workspace kit starts on", and "`kit workspaces`". This plan is the spec's step 1 of 7: fixed members
only. Queries, Projects, topics, labels, `--workspace` on `kit flow`, and kittui are later plans.

## Global Constraints

- The file: `${XDG_CONFIG_HOME:-$HOME/.config}/kit/workspaces.yml`. Kit's own state:
  `${XDG_CACHE_HOME:-$HOME/.cache}/kit/workspaces/`.
- Top-level keys of the file read in this step: `roots`, `favorites`, `workspaces` (a map of name to `{repos,
  exclude}`). Unknown keys (`match`, `project`) are ignored in this step, not refused.
- A member is `host/path`, the path whole (`gitlab.com/group/sub/repo` keeps every segment), with no `.git`, no scheme,
  no user.
- A checkout belongs only to the member its primary remote points at: `primary` in its `kit.yml`, else `origin`.
- The walk under each root goes `4` levels down, stops at a desk (a directory holding `.git`), and skips hidden
  directories and symbolic links.
- Startup rule: `--workspace` given wins; inside a repository, that repository; outside one, the last group used; else
  none. `--workspace none` asks for none.
- A group name matches `^[a-z0-9][a-z0-9._-]*$`; it never holds `/`, which is how a name and a member are told apart.
- Exits: 0 done; 1 the file cannot be read or holds a bad entry; 2 a usage error or an unknown workspace.
- Every file under `kit/` is written for a consumer and names paths from `kit/` (`modules/bash/workspaces.sh`). The
  verb's header follows the other verbs': banner with `(kit verb: workflow)`, usage lines, options, JSON shapes, one
  `Exits` line, `see:`.
- Commits go through `kit commit` with `export PATH=$PWD/kit/bin:$PATH` first; each subject starts with the workspaces
  issue number, written `#W` below, and ends with the `Co-Authored-By` trailer as a second `--message`.

## Review Focus

1. A hand-written `repos` entry given as a URL (`git@github.com:empowerite/edge.git`) or with `.git`: it should be read
   as the member `github.com/empowerite/edge`, not refused. A local path there is refused, naming the entry. Pinned in
   Task 3.
2. A desk under a root with no remotes, or whose primary remote is a local path: not a member of anything, skipped
   without a warning. Pinned in Task 2.
3. One member cloned twice under the roots: the first under the earliest root, then path order, is `checkout`; every one
   is in `checkouts`. Pinned in Task 2.
4. Editing a file that has comments, and editing when `~/.config/kit/` does not exist yet: comments stay where they
   were, and the directory and file are made. Pinned in Task 5.
5. Started in a linked worktree of a member, or in a repository with no remote: the first is that member; the second is
   kind `repository` with `name` the checkout's path and `member` null. Pinned in Task 4.

---

### Task 1: Move the desk walker to `git.sh`

**Files:**

- Modify: `kit/modules/bash/git.sh` (add `git:desks`)
- Modify: `kit/modules/bash/flow.sh:1305-1323` (remove `flow:desks`), `:1334` (call `git:desks`)
- Test: `tests/workspaces.bats` (new)

**Interfaces:**

- Produces: `git:desks <dir> <depth>`: prints every desk at or below `<dir>`, at most `<depth>` levels down, in path
  order, one absolute path per line. Same behavior as `flow:desks` today.

- [ ] **Step 1: Write the failing test** in a new `tests/workspaces.bats`, with the header block the other `.bats` files
      have, `load helper`, and a `setup()` that runs `hide-system-toolchains` and sets `kit_bash_modules=workspaces`
      (Task 2 creates the module; until then set it to `git`).

```bash
@test "git:desks stops at a desk, skips hidden directories and links, and goes four levels down" {
  root="${BATS_TEST_TMPDIR}/code"
  mkdir -p "${root}/a/.git" "${root}/a/inner/.git" "${root}/.hidden/h/.git"
  mkdir -p "${root}/1/2/3/4/.git" "${root}/1/2/3/4/5/.git"
  mkdir -p "${root}/x/y/z/w/v/.git"
  ln -s "${root}/a" "${root}/link"
  run kit-bash "git:desks '${root}' 4"
  [ "$status" -eq 0 ]
  [ "$output" = "${root}/1/2/3/4
${root}/a" ]
}
```

- [ ] **Step 2: Run it to make sure it fails**

Run: `kit tools run bats tests/workspaces.bats`
Expected: FAIL, `git:desks: command not found`.

- [ ] **Step 3: Move the function.** Cut `flow:desks` and its comment from `flow.sh`, paste them into `git.sh` as
      `git:desks` (recursive call renamed too), and change `flow:above`'s call. `flow.sh` already does `use git`.

- [ ] **Step 4: Run the new test and flow's**

Run: `kit tools run bats tests/workspaces.bats tests/flow.bats`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
export PATH=$PWD/kit/bin:$PATH
git add kit/modules/bash/git.sh kit/modules/bash/flow.sh tests/workspaces.bats
kit commit --message "#W git: the desk walker is git:desks, for kit flow above desks and the workspaces' roots alike" \
  --message "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>" \
  kit/modules/bash/git.sh kit/modules/bash/flow.sh tests/workspaces.bats
```

### Task 2: Members and checkouts

**Files:**

- Create: `kit/modules/bash/workspaces.sh`
- Test: `tests/workspaces.bats`

**Interfaces:**

- Consumes: `git:desks`; `forge:url-host <url>`; `yaml:value`, `yaml:list`.
- Produces:
  - `workspaces:file` → the file's path. `workspaces:cache` → the state directory's path (not created).
  - `workspaces:url-member <url>` → `host/path`, or nothing for a local path, a `file://` URL, or a path of fewer than
    two segments.
  - `workspaces:member-of <dir>` → the member of the checkout at `<dir>`, or nothing. Reads `primary` from
    `<dir>/kit.yml` only when that file exists, else uses `origin`; reads the URL with `git -C <dir> remote get-url
    --end-of-options <primary>`. Never `cd`s the caller.
  - `workspaces:roots` → each `roots` entry, a leading `~` expanded to `$HOME`, one per line, in file order.
  - `workspaces:checkouts` → `<member>\t<path>` for every desk under the roots that has a member, roots in order, paths
    in `git:desks` order within a root. A path reached from two roots is printed once.
  - `workspaces_walk_depth=4`.

- [ ] **Step 1: Write the failing tests.** Add a helper in the file, `clone-at <path> <url>`, which runs `git init
      --quiet` at `<path>` and `git -C <path> remote add origin <url>` (no network). Set `kit_bash_modules=workspaces`
      in `setup()`.

```bash
@test "a remote's URL gives its member, every segment of its path kept" {
  run kit-bash 'for u in git@github.com:empowerite/edge.git https://github.com/empowerite/kit \
      ssh://git@gitlab.com/g/sub/repo.git /srv/repo.git file:///srv/r https://host/one
    do printf "[%s]\n" "$(workspaces:url-member "$u")"; done'
  [ "$output" = "[github.com/empowerite/edge]
[github.com/empowerite/kit]
[gitlab.com/g/sub/repo]
[]
[]
[]" ]
}

@test "a checkout's member is its primary remote's: kit.yml's primary, else origin" {
  clone-at "${BATS_TEST_TMPDIR}/fork" git@github.com:me/edge.git
  git -C "${BATS_TEST_TMPDIR}/fork" remote add upstream git@github.com:empowerite/edge.git
  run kit-bash "workspaces:member-of '${BATS_TEST_TMPDIR}/fork'"
  [ "$output" = "github.com/me/edge" ]
  printf 'primary: upstream\n' > "${BATS_TEST_TMPDIR}/fork/kit.yml"
  run kit-bash "workspaces:member-of '${BATS_TEST_TMPDIR}/fork'"
  [ "$output" = "github.com/empowerite/edge" ]
}

@test "the checkouts are every desk under the roots with a member, the earliest root first" {
  clone-at "${BATS_TEST_TMPDIR}/b/edge" git@github.com:empowerite/edge.git
  clone-at "${BATS_TEST_TMPDIR}/a/github.com/empowerite/kit" git@github.com:empowerite/kit.git
  clone-at "${BATS_TEST_TMPDIR}/a/kit-again" https://github.com/empowerite/kit.git
  git init --quiet "${BATS_TEST_TMPDIR}/a/no-remote"
  clone-at "${BATS_TEST_TMPDIR}/a/local" /srv/repo.git
  mkdir -p "${XDG_CONFIG_HOME}/kit"
  t=${BATS_TEST_TMPDIR}
  printf 'roots: [%s/b, %s/a, %s/a]\n' "$t" "$t" "$t" > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  run --separate-stderr kit-bash workspaces:checkouts
  [ "$status" -eq 0 ]
  [ -z "$stderr" ]
  [ "$output" = "$(printf '%s\t%s\n' \
    github.com/empowerite/edge "${BATS_TEST_TMPDIR}/b/edge" \
    github.com/empowerite/kit "${BATS_TEST_TMPDIR}/a/github.com/empowerite/kit" \
    github.com/empowerite/kit "${BATS_TEST_TMPDIR}/a/kit-again")" ]
}

@test "a root given with ~ is under HOME" {
  mkdir -p "${XDG_CONFIG_HOME}/kit"
  printf 'roots: ["~/Code"]\n' > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  run kit-bash workspaces:roots
  [ "$output" = "${HOME}/Code" ]
}
```

- [ ] **Step 2: Run them to make sure they fail**

Run: `kit tools run bats tests/workspaces.bats`
Expected: FAIL, the module does not exist.

- [ ] **Step 3: Write `kit/modules/bash/workspaces.sh`** with the functions in Interfaces. Header block as the other
      modules': what a workspace is, the file, the member rule, `see: docs/workspaces.md`. `use git yaml`.
      `workspaces:url-member` takes the host from `forge:url-host` and the path by stripping
      `scheme://[user@]host[:port]/` or `[user@]host:`, then a trailing `/` and `.git`.

- [ ] **Step 4: Run the tests**

Run: `kit tools run bats tests/workspaces.bats`
Expected: all pass.

- [ ] **Step 5: Commit** `kit/modules/bash/workspaces.sh` and `tests/workspaces.bats`, subject `#W workspaces: a
      checkout's member is its primary remote's host and path, found under the roots four levels down`.

### Task 3: The verb, listing and showing

**Files:**

- Create: `kit/bin/kit-workspaces` (executable)
- Modify: `kit/modules/bash/workspaces.sh`
- Test: `tests/workspaces.bats`

**Interfaces:**

- Consumes: Task 2's functions.
- Produces:
  - `workspaces:names` → group names, file order. `workspaces:favorites` → favorites, file order. `workspaces:fixed
    <name>` and `workspaces:excluded <name>` → entries normalized to members.
  - `workspaces:check` → returns 1, after one `error` line naming the file and the entry, for: a name that breaks the
    name rule; a `repos` or `exclude` entry that normalizes to no member; a file yq cannot parse. Returns 0 for a
    missing file.
  - `workspaces:normalize <entry>` → a member from `host/path`, or from a URL via `workspaces:url-member`; from an
    existing directory via `workspaces:member-of`; nothing otherwise.
  - `workspaces:resolve <workspace>` → one JSON line, returns 2 for an unknown workspace (a name not in the file, or a
    string with no `/` that is no group):
    `{name, kind: "group"|"repository", members: [{repository, host, path, forge, remotes, checkout, checkouts, by}]}`.
    `forge` is `forge:of` run in the checkout for its primary, or for no checkout `github` for `github.com`, `gitlab`
    for `gitlab.com`, else null. `remotes` is `[{name, url, forge}]`, the primary first, `[]` without a checkout.
    `checkout` is a path or null; `checkouts` every path, in Task 2's order. `by` is `"fixed"` in this step. A group's
    members are its fixed list without its exclusions, in file order. A member string resolves as kind `repository` with
    that one member, `by: "given"`.
  - `workspaces:list-json` → `{file, roots, favorites, workspaces: [{name, members, favorite}]}`, `members` the count
    after exclusions.
- The verb's usage lines, exactly (Tasks 4 and 5 add the rest of the options' bodies; all forms go in the header now so
  `kit syntax` and the help tests see the final shape):

```text
# kit workspaces [--text | --json | --jq]
#
# kit workspaces --show <workspace> [--text | --json | --jq]
#
# kit workspaces --current [--workspace <workspace>] [--text | --json | --jq]
#
# kit workspaces --use <workspace>
#
# kit workspaces --new <name> [<repository>…]
#
# kit workspaces (--add | --remove) <name> <repository>…
#
# kit workspaces (--favorite | --unfavorite) <workspace>
```

  `<workspace>` is a group's name or a member; `<repository>` is a member, a git URL, or a checkout's directory. Text
  form: the list prints `📚 <name> <count>` per group, a `★` before favorites, and the file's path when there are none;
  `--show` prints one line per member, `<repository>  <checkout or ☁️>`.

- [ ] **Step 1: Write the failing tests.** A helper `workspaces-file` writes its stdin to
      `${XDG_CONFIG_HOME}/kit/workspaces.yml`. Run the verb as `"${kit_root}/bin/kit" workspaces …` from inside
      `fresh-repo`, as other verb tests do.

A second helper, `empowerite-file`, makes the clone and writes the file the first two tests share:

```bash
empowerite-file() {
  clone-at "${BATS_TEST_TMPDIR}/code/edge" git@github.com:empowerite/edge.git
  workspaces-file <<YML
roots: [${BATS_TEST_TMPDIR}/code]
favorites: [empowerite]
workspaces:
  empowerite:
    repos: [github.com/empowerite/kit, github.com/empowerite/edge, github.com/empowerite/attic]
    exclude: [github.com/empowerite/attic]
  solo:
    repos: [github.com/me/solo]
YML
}
```

```bash
@test "--json lists the groups, their member counts, and the favorites" {
  empowerite-file
  fresh-repo >/dev/null
  run --separate-stderr "${kit_root}/bin/kit" workspaces --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output '[.workspaces[] | [.name, .members, .favorite]]' <<<"$output")" \
    = '[["empowerite",2,true],["solo",1,false]]' ]
}

@test "--show resolves a group's members to their checkouts, and a missing one to none" {
  empowerite-file
  fresh-repo >/dev/null
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show empowerite --json
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output .kind <<<"$output")" = group ]
  [ "$(jq --compact-output '.members[0] | [.repository, .checkout, .forge, .by]' <<<"$output")" \
    = '["github.com/empowerite/kit",null,"github","fixed"]' ]
  [ "$(jq --compact-output '.members[1] | [.repository, .checkout, .forge, .by]' <<<"$output")" \
    = '["github.com/empowerite/edge","'"${BATS_TEST_TMPDIR}"'/code/edge","github","fixed"]' ]
  [ "$(jq --compact-output '.members[1].remotes' <<<"$output")" \
    = '[{"name":"origin","url":"git@github.com:empowerite/edge.git","forge":"github"}]' ]
}

@test "a repos entry given as a URL is read as its member; a path is refused, naming it" {
  workspaces-file <<YML
workspaces:
  w:
    repos: [git@github.com:empowerite/edge.git]
YML
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show w --json
  [ "$(jq --raw-output '.members[0].repository' <<<"$output")" = github.com/empowerite/edge ]
  workspaces-file <<YML
workspaces:
  w:
    repos: [/srv/nowhere]
YML
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show w --json
  [ "$status" -eq 1 ]
  [[ "$stderr" == *"workspaces.yml"*"/srv/nowhere"* ]]
}

@test "an unknown workspace is a usage error; a member is a workspace of one" {
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show nonesuch
  [ "$status" -eq 2 ]
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show github.com/me/x --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output '[.kind, .members[0].by, .members[0].checkout]' <<<"$output")" \
    = '["repository","given",null]' ]
}

@test "with no file, the list is empty and says where the file goes" {
  run --separate-stderr "${kit_root}/bin/kit" workspaces --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output '[.workspaces, .favorites, .file]' <<<"$output")" \
    = '[[],[],"'"${XDG_CONFIG_HOME}"'/kit/workspaces.yml"]' ]
}
```

- [ ] **Step 2: Run them to make sure they fail**

Run: `kit tools run bats tests/workspaces.bats`
Expected: FAIL, `kit: 'workspaces' is not a kit verb` (or kit's equivalent line).

- [ ] **Step 3: Implement** the module functions above and `kit/bin/kit-workspaces` on `kit-tutorials`' pattern: `source
      1st.sh`, `kit:enter`, `use workspaces`, `guard:arguments workspaces "$@"`, a `while` over the arguments, then one
      branch per form. Run `workspaces:check` before any read; exit 1 on its failure. Options other than the list and
      `--show` exit 2 with `error "kit workspaces <option> is not built yet"` until Tasks 4 and 5.

- [ ] **Step 4: Run the new tests and every verb-wide test**

Run: `kit tools run bats tests/workspaces.bats tests/syntax.bats tests/help.bats tests/exits.bats tests/completion.bats`
Expected: all pass. If `syntax.bats` refuses a usage line, run `kit syntax workspaces` to read why and reshape that
line, keeping the forms above.

- [ ] **Step 5: Commit** the verb, the module, and the tests, subject `#W kit workspaces: lists the groups and
      favorites, and shows a group's members with their checkouts`.

### Task 4: The startup rule

**Files:**

- Modify: `kit/modules/bash/workspaces.sh`, `kit/bin/kit-workspaces`
- Test: `tests/workspaces.bats`

**Interfaces:**

- Consumes: `kit_invoked_in` (set by `kit:enter`), `workspaces:member-of`, `workspaces:names`.
- Produces:
  - `workspaces:last` → the last group used, from `$(workspaces:cache)/last`, or nothing when the file is missing or
    names a group no longer in the file.
  - `workspaces:remember <name>` → writes `<name>` to that file, making the directory.
  - `workspaces:current [<workspace>]` → one JSON line, `{kind: "repository"|"group"|"none", name, member, checkout,
    why: "given"|"repository"|"last"|"none"}`. Given `none`: kind `none`, why `given`. Given anything else: as
    `workspaces:resolve` names it, why `given`, exit 2 when unknown. Else inside a repository (`git -C
    "${kit_invoked_in}" rev-parse --show-toplevel`): kind `repository`, `checkout` the top level, `member` its member or
    null, `name` the member or, with none, the checkout's path. Else the last group, why `last`. Else kind `none`, why
    `none`. `name`, `member`, `checkout` are null where they do not apply.
  - The verb: `--current` prints it; `--use <workspace>` remembers a group and exits 2 for a member (`error "kit
    workspaces --use remembers a group; a repository is the workspace inside it"`) or an unknown name.

- [ ] **Step 1: Write the failing tests**

```bash
@test "inside a repository, the workspace is that repository, even in a linked worktree" {
  fresh-repo >/dev/null
  git remote add origin git@github.com:empowerite/edge.git
  git commit --quiet --allow-empty -m one
  git worktree add --quiet "${BATS_TEST_TMPDIR}/wt" -b wt
  cd "${BATS_TEST_TMPDIR}/wt"
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$(jq --compact-output '[.kind, .name, .why]' <<<"$output")" \
    = '["repository","github.com/empowerite/edge","repository"]' ]
}

@test "a repository with no remote is named by its path, with no member" {
  fresh-repo >/dev/null
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$(jq --compact-output '[.kind, .name, .member]' <<<"$output")" = '["repository","'"${repo}"'",null]' ]
}

@test "outside a repository, the workspace is the last group used, or none" {
  workspaces-file <<YML
workspaces:
  empowerite:
    repos: [github.com/empowerite/kit]
YML
  mkdir -p "${BATS_TEST_TMPDIR}/outside" && cd "${BATS_TEST_TMPDIR}/outside"
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$(jq --compact-output '[.kind, .why]' <<<"$output")" = '["none","none"]' ]
  run "${kit_root}/bin/kit" workspaces --use empowerite
  [ "$status" -eq 0 ]
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$(jq --compact-output '[.kind, .name, .why]' <<<"$output")" = '["group","empowerite","last"]' ]
}

@test "--workspace wins, none included, and --use refuses a repository" {
  fresh-repo >/dev/null
  git remote add origin git@github.com:empowerite/edge.git
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --workspace none --json
  [ "$(jq --compact-output '[.kind, .why]' <<<"$output")" = '["none","given"]' ]
  run --separate-stderr "${kit_root}/bin/kit" workspaces --use github.com/empowerite/edge
  [ "$status" -eq 2 ]
}

@test "a last group no longer in the file is forgotten" {
  mkdir -p "${XDG_CACHE_HOME}/kit/workspaces" && printf 'gone\n' > "${XDG_CACHE_HOME}/kit/workspaces/last"
  mkdir -p "${BATS_TEST_TMPDIR}/outside" && cd "${BATS_TEST_TMPDIR}/outside"
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$(jq --raw-output .kind <<<"$output")" = none ]
}
```

- [ ] **Step 2: Run them to make sure they fail**

Run: `kit tools run bats tests/workspaces.bats`
Expected: the five new tests FAIL with exit 2, "not built yet".

- [ ] **Step 3: Implement** the three functions and the two options.

- [ ] **Step 4: Run the tests**

Run: `kit tools run bats tests/workspaces.bats`
Expected: all pass.

- [ ] **Step 5: Commit**, subject `#W kit workspaces: the startup rule, a repository inside one, else the last group
      used, and --workspace over both`.

### Task 5: Editing the file

**Files:**

- Modify: `kit/modules/bash/workspaces.sh`, `kit/bin/kit-workspaces`
- Test: `tests/workspaces.bats`

**Interfaces:**

- Consumes: `workspaces:normalize`, `workspaces:names`, `workspaces:check`, `kit_yq`.
- Produces: `workspaces:edit <expression>` → makes the file's directory, and the file with a two-line header comment (`#
  workspaces.yml - kit's workspaces; see: kit help workspaces`) when missing, then runs `yq --inplace <expression>` on
  it with values passed through `strenv(...)`, never interpolated into the expression. Then the verb's options:
  - `--new <name> [<repository>…]`: exit 2 for a name that breaks the rule or exists; adds `workspaces.<name>.repos`
    with the members, normalized, in order, without duplicates.
  - `--add <name> <repository>…` / `--remove <name> <repository>…`: exit 2 for an unknown group or a repository that
    normalizes to no member; `--add` appends the ones not there, `--remove` drops the ones there. Neither touches
    `exclude`.
  - `--favorite <workspace>` / `--unfavorite <workspace>`: exit 2 for an unknown workspace; add once, or drop.
  - Each prints one `step` line saying what changed, and nothing when nothing did.

- [ ] **Step 1: Write the failing tests**

```bash
@test "--new makes the file and its directory, and the group with its members" {
  fresh-repo >/dev/null
  git remote add origin git@github.com:empowerite/edge.git
  run "${kit_root}/bin/kit" workspaces --new empowerite . \
    https://github.com/empowerite/kit.git github.com/empowerite/kit
  [ "$status" -eq 0 ]
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show empowerite --json
  [ "$(jq --compact-output '[.members[].repository]' <<<"$output")" \
    = '["github.com/empowerite/edge","github.com/empowerite/kit"]' ]
  run "${kit_root}/bin/kit" workspaces --new empowerite
  [ "$status" -eq 2 ]
  run "${kit_root}/bin/kit" workspaces --new Bad/Name
  [ "$status" -eq 2 ]
}

@test "--add, --remove, --favorite, and --unfavorite keep the file's comments" {
  workspaces-file <<'YML'
# mine
roots: [~/Code] # where I clone
workspaces:
  w:
    # the core
    repos: [github.com/a/one]
YML
  fresh-repo >/dev/null
  "${kit_root}/bin/kit" workspaces --add w github.com/a/two github.com/a/one
  "${kit_root}/bin/kit" workspaces --remove w github.com/a/one
  "${kit_root}/bin/kit" workspaces --favorite w
  "${kit_root}/bin/kit" workspaces --favorite w
  run --separate-stderr "${kit_root}/bin/kit" workspaces --json
  [ "$(jq --compact-output '[.favorites, .workspaces[0].members]' <<<"$output")" = '[["w"],1]' ]
  grep --quiet '^# mine$' "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  grep --quiet '# where I clone' "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  grep --quiet '# the core' "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  "${kit_root}/bin/kit" workspaces --unfavorite w
  run --separate-stderr "${kit_root}/bin/kit" workspaces --json
  [ "$(jq --compact-output .favorites <<<"$output")" = '[]' ]
}

@test "editing a group that is not there is a usage error" {
  fresh-repo >/dev/null
  run "${kit_root}/bin/kit" workspaces --add nope github.com/a/one
  [ "$status" -eq 2 ]
  run "${kit_root}/bin/kit" workspaces --favorite nope
  [ "$status" -eq 2 ]
}
```

- [ ] **Step 2: Run them to make sure they fail**

Run: `kit tools run bats tests/workspaces.bats`
Expected: the three new tests FAIL with exit 2, "not built yet".

- [ ] **Step 3: Implement** `workspaces:edit` and the five options. Remove the "not built yet" branch.

- [ ] **Step 4: Run the tests**

Run: `kit tools run bats tests/workspaces.bats`
Expected: all pass.

- [ ] **Step 5: Commit**, subject `#W kit workspaces: --new, --add, --remove, --favorite, and --unfavorite edit the
      file, its comments kept`.

### Task 6: The page, and the gate

**Files:**

- Create: `docs/workspaces.md`
- Modify: `docs/features.md` (a line for workspaces, beside the others, linking the page)

**Interfaces:**

- Consumes: everything above.

- [ ] **Step 1: Write `docs/workspaces.md`** for someone using kit: what a workspace is; the file, with the spec's
      example cut down to what this step reads (`roots`, `favorites`, `workspaces.<name>.repos`, `exclude`); what a
      member is and how a checkout is found; the startup rule; the commands, one example each. Say that queries,
      Projects, `kit flow --workspace`, and kittui's 📚 come later, without dates.

- [ ] **Step 2: Run the gate**

Run: `kit gate`
Expected: exits 0. Lint, markdown, links, the bats suite, and the Go tests all pass.

- [ ] **Step 3: Commit** `docs/workspaces.md` and `docs/features.md`, subject `#W docs: workspaces, the file, members
      and checkouts, the startup rule, and kit workspaces`.
