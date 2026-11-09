# Workspaces 3a: kit flow across a workspace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `kit flow --workspace <workspace>` draws every stage across the workspace's members with checkouts, and
`kit flow repos` gives the 📚 repos tab's rows, so kittui's half of step 3 (plan 3b) has its data.

**Architecture:** A new module, `across`, resolves a workspace's members and their checkouts once, runs
`kit flow … --json` in every checkout at once, as `flow:above` runs each desk, and merges what each printed into one
document per stage, each row or object tagged with its member. `kit flow` gains `--workspace` on every form and a
`repos` stage; without `--workspace` nothing it prints changes.

**Tech Stack:** bash, jq, git, bats.

**Spec:** `docs/superpowers/specs/2026-11-08-workspaces-design.md`, step 3 of § The order of the work, and
§ Flow over a workspace, § Where Flow's data comes from.

This plan is step 3's kit half. Plan 3b, written once this lands, is kittui's: the 📚 repos tab, `member` as a field of
every stage, the counts as links, and switching Flow's reads to `--workspace`.

## Global Constraints

- kit commit only, never kit submit or push; every subject starts with `#89`, the issue for this plan.
- `export PATH=$PWD/kit/bin:$PATH` before `kit commit`.
- Peers share the worktree: stage named paths only, guard that the index holds only them, never leave anything staged.
- Everything under `kit/` is written for a consumer, and a path in it is written from `kit/`: `modules/bash/across.sh`.
- Lines are 120 cells at most.
- Without `--workspace`, every form of `kit flow` prints exactly what it prints today; the existing `tests/flow.bats`
  passes unchanged.
- A member is host and path, `github.com/empowerite/kit`, as `kit workspaces` names it.
- Step 3 reads the forge per member, as `kit flow` does in each checkout today. Batching by forge and owner, ☁️
  members' forge reads, and reading a tracker several members share once, so its issue is one row, are step 4's.
- A test's desk has origin's URL on GitHub, so `workspaces:member-of` names its member, and is never fetched: a test
  that draws the picture or incoming passes `--no-fetch`.

## Review Focus

1. A member whose checkout was moved or deleted since kit last walked the roots: its line is in `errors`, every other
   member draws, and the stage exits 0.
2. A member with no primary remote, or one that names no forge: its forge stages refuse for it alone, as `kit flow`
   in it refuses; its desk and worktrees draw.
3. A group in which no member has a checkout: every stage prints `{"rows": [], "errors": []}` and exits 0; `repos`
   lists each member, not cloned; the text picture says once that none is cloned.
4. A checkout path with spaces, and two members whose checkouts share a parent: each runs in its own directory, and
   the rows keep each one's member.
5. `--workspace` with a name that is neither a group nor a repository, or with `none`: exits 2 with `kit
   workspaces`' line, before any member runs.

---

## File structure

- Create `kit/modules/bash/across.sh`: members and checkouts, running `kit flow` in each, and merging the outputs.
- Create `tests/across.bats`: the module's and `kit flow --workspace`'s tests.
- Modify `kit/bin/kit-flow`: the grammar's `--workspace` and `repos`, the paragraphs that describe them, and the
  dispatch before `flow:render`.
- Modify `docs/workspaces.md`: a section on `kit flow --workspace`.

## Task 1: Members, their checkouts, and the merge

**Files:**

- Create: `kit/modules/bash/across.sh`
- Create: `tests/across.bats`

**Interfaces:**

- Consumes: `workspaces:resolve <workspace>` (JSON `{name, kind, members}`, exit 2 for neither),
  `workspaces:checkouts` (`member\tpath` lines), from `modules/bash/workspaces.sh`.
- Produces:
  - `across:members <workspace>` prints `member\tcheckout` per member, in the workspace's order, checkout empty for a
    member not cloned; the roots are walked once. Exit 2, with workspaces' line, for an unknown workspace or `none`.
  - `across:merge <dir> <kind>` reads `<dir>/<n>.member`, `<n>.out`, `<n>.err`, `<n>.status` for n = 0, 1, … and
    prints `{"rows": [...], "errors": [...]}` on one line. `<kind>` is `array` or `object`: for `array`, each row of a
    member's array gains `"member": <member>`; for `object`, the member's object gains it. A member whose status is
    not 0, or whose output does not parse, adds `{"member": <member>, "error": <its stderr's last line, less a
    leading ❌ and a space>}` to `errors` and nothing to `rows`. Members keep their order.

- [ ] **Step 1: Write the failing tests** in `tests/across.bats`, using `workspaces.bats`' `clone-at` and
  `workspaces-file` helpers (copy them; bats files do not share helpers but `helper.bash`'s):

```bash
@test "across:members gives each member and its checkout, in the group's order, empty for one not cloned" {
  clone-at "${BATS_TEST_TMPDIR}/code/b" https://github.com/o/b.git
  clone-at "${BATS_TEST_TMPDIR}/code/a dir/a" https://github.com/o/a.git
  workspaces-file <<EOF
roots: [${BATS_TEST_TMPDIR}/code]
workspaces:
  w: {repos: [github.com/o/a, github.com/o/c, github.com/o/b]}
EOF
  run kit-bash "across:members w"
  [ "$status" -eq 0 ]
  [ "$output" = "$(printf '%s\t%s\n' github.com/o/a "${BATS_TEST_TMPDIR}/code/a dir/a" github.com/o/c '' \
    github.com/o/b "${BATS_TEST_TMPDIR}/code/b")" ]
}

@test "across:members refuses a workspace that is neither a group nor a repository, and none, with 2" {
  run kit-bash "across:members nosuch"
  [ "$status" -eq 2 ]
  run kit-bash "across:members none"
  [ "$status" -eq 2 ]
}

@test "across:merge tags each row with its member, in order, and turns a failed member into its line" {
  d=${BATS_TEST_TMPDIR}/m && mkdir -p "$d"
  printf 'github.com/o/a' > "$d/0.member"; printf '[{"n":1},{"n":2}]' > "$d/0.out"; printf 0 > "$d/0.status"
  : > "$d/0.err"
  printf 'github.com/o/b' > "$d/1.member"; : > "$d/1.out"; printf 1 > "$d/1.status"
  printf 'a warning\n❌ gh cannot read o/b\n' > "$d/1.err"
  printf 'github.com/o/c' > "$d/2.member"; printf '[{"n":3}]' > "$d/2.out"; printf 0 > "$d/2.status"; : > "$d/2.err"
  run kit-bash "across:merge '$d' array"
  [ "$status" -eq 0 ]
  [ "$output" = '{"rows":[{"n":1,"member":"github.com/o/a"},{"n":2,"member":"github.com/o/a"},{"n":3,"member":"github.com/o/c"}],"errors":[{"member":"github.com/o/b","error":"gh cannot read o/b"}]}' ]
}

@test "across:merge tags an object stage's object with its member" {
  d=${BATS_TEST_TMPDIR}/m && mkdir -p "$d"
  printf 'github.com/o/a' > "$d/0.member"; printf '{"branch":"main"}' > "$d/0.out"; printf 0 > "$d/0.status"
  : > "$d/0.err"
  run kit-bash "across:merge '$d' object"
  [ "$output" = '{"rows":[{"branch":"main","member":"github.com/o/a"}],"errors":[]}' ]
}
```

  Each test sets `kit_bash_modules=across` in `setup`, with `hide-system-toolchains`, as `workspaces.bats` does.

- [ ] **Step 2: Run them to see them fail**

  Run: `KIT_TOOLCHAIN=mise kit/bin/kit tools run bats tests/across.bats`
  Expected: 4 fail: `across:members: command not found`, and the same for `across:merge`.

- [ ] **Step 3: Write `across:members` and `across:merge`** in `kit/modules/bash/across.sh`, with the module's
  header as the other modules have one, and `use workspaces` at its top.

- [ ] **Step 4: Run them to see them pass**

  Run: `KIT_TOOLCHAIN=mise kit/bin/kit tools run bats tests/across.bats`
  Expected: 4 pass.

- [ ] **Step 5: Commit**

```bash
git add kit/modules/bash/across.sh tests/across.bats
kit commit --message "#89 kit: across gives a workspace's members and checkouts, and merges what kit flow printed in each, every row tagged with its member" --message "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

## Task 2: One stage across a workspace

**Files:**

- Modify: `kit/modules/bash/across.sh`, `kit/bin/kit-flow`
- Test: `tests/across.bats`

**Interfaces:**

- Consumes: Task 1's `across:members`, `across:merge`.
- Produces:
  - `across:stage <workspace> <stage> <argument>…` runs `kit flow <stage> --json <argument>…` in every member's
    checkout at once, as `flow:above` runs each desk, and prints `across:merge`'s document: `array` for backlog,
    underway, changes, builds, and releases; `object` for desk, worktrees, and incoming. It exits 0 whatever a member
    did; exit 2 only from `across:members`.
  - `kit flow <stage> --workspace <workspace> --json [<argument>…]` prints it. Its grammar lines gain
    `[--workspace <workspace>]` on the picture's line and on every stage's. `--cached`, `--fresh`, `--no-fetch`, and
    the owner filters pass through to each member.
  - `kit flow changes --workspace <workspace> <number>…` exits 2, its line `a number names a pull
    request in one repository; run kit flow changes <number> in it`.

- [ ] **Step 1: Write the failing tests** in `tests/across.bats`. `desk-at <path>` makes a real desk at `<path>`,
  a bare origin beside it with one pushed commit, as `flow.bats`' `setup` does, and sets origin's URL to a GitHub URL
  with `git config remote.origin.pushurl` left to the bare one, so `workspaces:member-of` names the member and the
  fetch stays local; the test reads `desk`, which needs no forge.

```bash
@test "kit flow desk --workspace prints each cloned member's desk, tagged, and a moved checkout's line" {
  desk-at "${BATS_TEST_TMPDIR}/code/a" github.com/o/a
  desk-at "${BATS_TEST_TMPDIR}/code/b b" github.com/o/b
  workspaces-file <<EOF
roots: [${BATS_TEST_TMPDIR}/code]
workspaces:
  w: {repos: [github.com/o/a, github.com/o/b, github.com/o/c]}
EOF
  run kit flow desk --workspace w --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.member, .branch]]' <<<"$output")" = '[["github.com/o/a","main"],["github.com/o/b","main"]]' ]
  [ "$(jq -c .errors <<<"$output")" = '[]' ]
}

@test "a member whose checkout is gone is its line in errors, and the others still draw" {
  # as above; then: rm -rf "${BATS_TEST_TMPDIR}/code/b b" after the roots were walked once, by running
  # across:members first in the same kit-bash, then across:stage w desk
  ...
  [ "$(jq -c '[.rows[].member]' <<<"$output")" = '["github.com/o/a"]' ]
  [ "$(jq -r '.errors[0].member' <<<"$output")" = github.com/o/b ]
}

@test "a group with no member cloned prints no rows and no errors, and exits 0" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/o/a]}}'
  run kit flow desk --workspace w --json
  [ "$status" -eq 0 ]
  [ "$output" = '{"rows":[],"errors":[]}' ]
}

@test "kit flow --workspace refuses a workspace it does not know, and a pull request's number, with 2" {
  run kit flow desk --workspace nosuch --json
  [ "$status" -eq 2 ]
  run kit flow changes --workspace w --json 7
  [ "$status" -eq 2 ]
  [[ "$output" == *"run kit flow changes <number> in it"* ]]
}

@test "without --workspace, kit flow desk --json prints the desk's object as before" {
  desk-at "${BATS_TEST_TMPDIR}/code/a" github.com/o/a
  cd "${BATS_TEST_TMPDIR}/code/a"
  run kit flow desk --json
  [ "$(jq -r 'type' <<<"$output")" = object ]
}
```

  The second test's elided lines follow the first's setup; write them out.

- [ ] **Step 2: Run them to see them fail**

  Run: `KIT_TOOLCHAIN=mise kit/bin/kit tools run bats tests/across.bats`
  Expected: the five new tests fail: kit flow refuses `--workspace` as an unknown option, with 2.

- [ ] **Step 3: Write `across:stage`, and in `kit-flow`** add the grammar and, after `forge:primary`'s line, a branch
  taken when `syntax_options[--workspace]` is set: a stage with `--json` goes to `across:stage`; a stage without
  `--json`, and the picture, are Task 4's, and until then exit 2 with "--workspace draws with --json for now". The
  refusal of a number comes first. `use across` beside `use flow tui`.

- [ ] **Step 4: Run every flow test**

  Run: `KIT_TOOLCHAIN=mise kit/bin/kit tools run bats tests/across.bats tests/flow.bats`
  Expected: all pass; `flow.bats` unchanged.

- [ ] **Step 5: Commit**

  `kit/modules/bash/across.sh tests/across.bats kit/bin/kit-flow`, subject `#89 kit: kit flow <stage> --workspace
  --json prints the stage across the workspace's members with checkouts, each row tagged with its member, and a member
  that fails as its line`.

## Task 3: The repos stage

**Files:**

- Modify: `kit/modules/bash/across.sh`, `kit/bin/kit-flow`
- Test: `tests/across.bats`

**Interfaces:**

- Consumes: `across:members`; `workspaces:member-json <member> <by> <checkouts>` (`{repository, host, path, forge,
  remotes, checkout, checkouts, by}`); the desk stage's JSON (`branch`, `ahead`, `behind`, `detached`).
- Produces:
  - `across:repos <workspace>` prints `{"rows": [...], "errors": []}`, a row per member in order:
    `workspaces:member-json`'s object, `by` the member's (`fixed` for a group, `given` for a repository), plus `branch`,
    `ahead`, `behind`, from `kit flow desk --json` run in the checkout, each null for a member not cloned; and
    `error`, the desk's line where it failed, else null. Every member is a row, cloned or not, so `errors` stays empty
    here; a row carries its own.
  - `kit flow repos --workspace <workspace> --json` prints it. `kit flow repos --json` without `--workspace`, in a
    repository, prints the same for that repository alone, `by` `here`; outside one it exits 2, "repos lists a
    workspace's repositories: run it in one, or give --workspace".
  - `kit flow repos [--workspace <workspace>]` without `--json` draws a line per member: the workspace's glyph, the
    member, its forge, its branch with ↑ahead ↓behind where either is not 0, and its checkout's path, or "not cloned",
    dim.
  - `kit flow --stages --workspace <workspace>` prints `repos`, then the stages `settings:stages` gives where kit runs.
    Without `--workspace`, `--stages` is as it is; plan 3b decides whether `repos` joins it there.

- [ ] **Step 1: Write the failing tests** in `tests/across.bats`, with `desk-at` from Task 2:

```bash
@test "kit flow repos --workspace gives a row per member, cloned or not, with its desk's branch" {
  desk-at "${BATS_TEST_TMPDIR}/code/a" github.com/o/a
  git -C "${BATS_TEST_TMPDIR}/code/a" commit --quiet --allow-empty --message ahead
  workspaces-file <<EOF
roots: [${BATS_TEST_TMPDIR}/code]
workspaces:
  w: {repos: [github.com/o/a, github.com/o/c]}
EOF
  run kit flow repos --workspace w --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.repository, .by, .branch, .ahead, .checkout != null]]' <<<"$output")" = \
    '[["github.com/o/a","fixed","main",1,true],["github.com/o/c","fixed",null,null,false]]' ]
}

@test "kit flow repos in a repository lists it alone; outside one, without --workspace, it refuses with 2" {
  desk-at "${BATS_TEST_TMPDIR}/code/a" github.com/o/a
  cd "${BATS_TEST_TMPDIR}/code/a"
  run kit flow repos --json
  [ "$(jq -c '[.rows[] | [.repository, .by]]' <<<"$output")" = '[["github.com/o/a","here"]]' ]
  cd "${BATS_TEST_TMPDIR}"
  run kit flow repos --json
  [ "$status" -eq 2 ]
}

@test "kit flow repos draws a line per member, not cloned dim" {
  # the first test's workspace
  run kit flow repos --workspace w
  [[ "$output" == *"github.com/o/a"*"main ↑1"* ]]
  [[ "$output" == *"github.com/o/c"*"not cloned"* ]]
}

@test "kit flow --stages --workspace puts repos first" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/o/a]}}'
  run kit flow --stages --workspace w
  [ "${lines[0]}" = repos ]
  [ "${#lines[@]}" -eq 9 ]
}
```

- [ ] **Step 2: Run them to see them fail**

  Run: `KIT_TOOLCHAIN=mise kit/bin/kit tools run bats tests/across.bats`
  Expected: the four fail: `repos` is no stage.

- [ ] **Step 3: Write `across:repos` and its text drawing, and the `repos` grammar line**: `kit flow repos
  [--workspace <workspace>] [--json | --jq]`, with its paragraph under the stages': its JSON's fields, as above.
  `repos` is not among the stages the flow setting may name.

- [ ] **Step 4: Run every flow test**

  Run: `KIT_TOOLCHAIN=mise kit/bin/kit tools run bats tests/across.bats tests/flow.bats`
  Expected: all pass.

- [ ] **Step 5: Commit**, subject `#89 kit: kit flow repos lists a workspace's repositories, each with its forge,
  remotes, checkout, and desk's branch, cloned or not, and --stages --workspace puts it first`.

## Task 4: The picture, marks, and quota across a workspace

**Files:**

- Modify: `kit/modules/bash/across.sh`, `kit/bin/kit-flow`, `docs/workspaces.md`
- Test: `tests/across.bats`

**Interfaces:**

- Consumes: Tasks 1–3.
- Produces:
  - `kit flow --workspace <workspace> --json` prints `{stages, repos, <stage>…}`: `stages` as `--stages --workspace`
    prints them, `repos` as Task 3's, and each stage as Task 2's document. Each member runs `kit flow --json` once, all
    its stages at once, and the documents are cut from those; exit 0.
  - `kit flow --workspace <workspace>`, as text, draws `repos` as Task 3 does, then each cloned member's picture under
    its member, as `flow:above` draws each desk's under its path; with none cloned, one dim line says so. It exits 1
    when a member's picture did, as `flow:above` does not; the text says which.
  - On a terminal, `--workspace` keeps kit flow in text, as `--text` does, until plan 3b hands it to kittui.
  - `kit flow --marks --workspace <workspace>` prints one object, a mark per stage that changes when any member's
    does: the members' marks for the stage, joined in member order. `repos`' mark is the members' checkouts and
    branches, so a clone or a switch of branch marks it stale.
  - `kit flow --quota --workspace <workspace>` prints every forge the members touch, each once: the members' arrays
    joined and kept unique by forge.
  - `kit flow --me --workspace <workspace>` exits 2: "--me reads one forge; run it in a member".

- [ ] **Step 1: Write the failing tests** in `tests/across.bats`:

```bash
@test "kit flow --workspace --json is one object: the stages, repos, and each stage across the members" {
  # two desks, a and b, in w; stages limited by each desk's kit.yml to desk and worktrees
  run kit flow --workspace w --json
  [ "$status" -eq 0 ]
  [ "$(jq -r '.stages[0]' <<<"$output")" = repos ]
  [ "$(jq -c '[.desk.rows[].member]' <<<"$output")" = '["github.com/o/a","github.com/o/b"]' ]
  [ "$(jq -c '[.repos.rows[].repository]' <<<"$output")" = '["github.com/o/a","github.com/o/b"]' ]
}

@test "kit flow --workspace as text draws repos, then each member's picture under it" {
  run kit flow --workspace w --text
  [[ "$output" == *"github.com/o/a"*"github.com/o/b"* ]]
  [ "$(grep -c '^📚 github.com/o/' <<<"$output")" -ge 2 ]
}

@test "kit flow --marks --workspace changes a stage's mark when one member's changes" {
  run kit flow --marks --workspace w
  before=$(jq -r .desk <<<"$output")
  printf 'x\n' > "${BATS_TEST_TMPDIR}/code/b/x"
  run kit flow --marks --workspace w
  [ "$(jq -r .desk <<<"$output")" != "${before}" ]
}

@test "kit flow --me --workspace refuses with 2" {
  run kit flow --me --workspace w
  [ "$status" -eq 2 ]
}
```

  Each test's setup is two desks in `w`, with `stages desk, worktrees` committed in each desk's `kit.yml`, so no
  forge is read. `--quota` is tested with a fake gh in the members' PATH, as `flow.bats` tests it, asserting each
  forge once.

- [ ] **Step 2: Run them to see them fail**

  Run: `KIT_TOOLCHAIN=mise kit/bin/kit tools run bats tests/across.bats`
  Expected: the new tests fail with Task 2's "--workspace draws with --json for now", or exit 2 on `--marks`.

- [ ] **Step 3: Write the picture, marks, and quota**; replace Task 2's placeholder refusal. Add a section to
  `docs/workspaces.md`, "kit flow over a workspace", saying what `--workspace` and `repos` print, and that kittui's
  tab comes with step 3's second half.

- [ ] **Step 4: Run the gate's tests for kit flow and workspaces**

  Run: `KIT_TOOLCHAIN=mise kit/bin/kit tools run bats tests/across.bats tests/flow.bats tests/workspaces.bats`
  Expected: all pass. Then `kit lint` and `kit generate --check` pass.

- [ ] **Step 5: Commit**, subject `#89 kit: kit flow --workspace draws the picture across the members, as JSON and
  as text, and its marks and quota span them`.
