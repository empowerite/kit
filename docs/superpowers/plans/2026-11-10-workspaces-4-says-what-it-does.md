# Workspaces 4: what is built says what it does Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every workspace input kit accepts does what its page says or is refused by name, a member that fails is seen
and counted, and the docs and the spec say what is built and what is not.

**Architecture:** Each fix lands where its rule lives. `modules/bash/workspaces.sh` checks the file inside
`workspaces:resolve` and `workspaces:current`, gains `workspaces:across`, the one function `kit flow` and `kit
workspace` ask for their workspace, and takes around every edit the lock `forge:snapshot` uses, a symbolic link that
names its process. `modules/bash/across.sh` passes a refusal's status through. kittui drops three menu lines that open
nothing, and reads `checkouts`. No file is created.

**Tech Stack:** bash, jq, yq, git, bats; Go and Bubble Tea v2 in Task 5.

**Spec:** `docs/superpowers/specs/2026-11-08-workspaces-design.md`, § The workspace file, § Which workspace kit starts
on, and § The order of the work. Findings: Serve 8, 10 to 13, 19 to 23, 26, 27; Command line 22; Gaps.

**Review:** [the findings](../reviews/2026-11-10-code-review-findings.md) of the review of 2026-11-10, cited
here by area and number: Serve 8 is item 8 under its Serve heading.

## Decisions for the owner

1. `match:` and `project:` in a group are read as nothing today, and no word says so. (a) Refuse the file, naming the
   group and the key as not built yet, exit 1. (b) Build `match:`: a forge call that lists an owner's repositories
   (each call in `modules/bash/forge.sh:36-80` reads one), a cache of resolved members, and forge reads for a member
   with no checkout, which is spec step 4 whole. Default: (a).
2. A member's case (Serve 19). (a) One spelling: the host in lower case, and the path too on `github.com`,
   `gitlab.com`, and hosts under them, so `github.com/NixOS/nixpkgs` prints as `github.com/nixos/nixpkgs`. (b) Keep
   each spelling and compare without case, at eight comparisons in `workspaces.sh`. Default: (a), one key.
3. A clone whose `kit.yml` does not parse (Serve 21) has no readable primary. (a) File it under origin, as kit
   reads that clone itself, and say so only where its member is drawn. (b) Count it as no checkout and warn on every
   walk of the roots, as the fail-closed plan beside this one has it. Default: (a); the two plans must agree.

## Global Constraints

- kit commit only, never kit submit or push; every subject starts with `#89`, as `#89 kit: …`.
- `export PATH=$PWD/kit/bin:$PATH` before `kit commit`.
- Peers share the worktree: stage named paths only, guard that the index holds only them, never leave anything staged.
- Everything under `kit/` is written for a consumer, and a path in it is written from `kit/`: `modules/bash/x.sh`.
- Lines are 120 cells at most.
- A verb's header comment is its help page and its usage line is its grammar (`kit syntax --check` holds them): a new
  flag is added to the header's usage line and described there, and `docs/workspaces.md` says the same.
- A change to `kit-flow`'s header regenerates `projects/tui/internal/syntax/testdata/flow.json` by `kit syntax flow`.
- Inside a repository, or with no group used, every form prints what it prints today: `tests/flow.bats` is unchanged.
- `projects/tui` stays at or above 98% of statements covered (98.0% today): each new Go statement has a test.
- Bats: `KIT_TOOLCHAIN=mise kit/bin/kit tools run bats tests/<file>.bats`. Go: `cd projects/tui && go test ./...`.

## Review Focus

1. Inside a repository, a `workspaces.yml` that does not parse changes nothing for `kit flow`: exit 0. Task 2.
2. The last group used is deleted: `--current` says none, so `kit flow` draws above desks again. Task 3.
3. A killed process's lock is taken over; a live one's is waited for, then refused with 1, the file as it was. Task 3.
4. `--rename` onto a name in use, onto `none`, or onto `Bad/Name` is refused with 2, the file as it was. Task 3.
5. A dropdown with no group and no favorite draws one dim line, never an empty box. Task 5.

---

## Task 1: The workspace file is checked wherever it is read

**Files:** modify `kit/modules/bash/workspaces.sh:128-184,255,316`,
`kit/modules/bash/across.sh:29,38,103,109,153,172,255,395`, `kit/bin/kit-workspace:212`; test as Step 1 names.

**Interfaces:**

- `workspaces:check` also refuses, with `❌ <file>: …` and 1: a file that is not a map, `the file is a map of roots,
  favorites, and workspaces`; a group's `match` or `project`, `<group>'s <key> is not built yet; this kit reads a
  group's repos and exclude, so list its members under repos`; any other key, `'<key>' in <group> is no key of a
  group; a group has repos and exclude`. Passing, it sets `workspaces_checked=1`, so a subshell does not check again.
- `workspaces:resolve` runs it first; `workspaces:current`, before it reads the file: with a workspace other than
  `none` given, and outside a repository; `across:flow`, before `--stages` reads the names. Each `|| return 2` at the
  lines above, and `kit-workspace:212`'s `|| exit 2`, passes `$?` on: 1 for a file refused, 2 for a workspace unknown,
  and no document. `workspaces:fixed` skips an `exclude` that names no member.

- [ ] **Step 1: Write the failing tests**, the first in `tests/workspaces.bats`, the second in `tests/across.bats`:

```bash
@test "a file that is no map, and a group's key kit does not read, are refused by name" {
  for each in "workspaces: {q: {match: [{owner: github.com/o}]}}|q's match is not built yet" \
    "workspaces: {p: {project: github.com/orgs/o/projects/3}}|p's project is not built yet" \
    "workspaces: {w: {repo: [github.com/o/a]}}|'repo' in w is no key of a group" \
    "just a string|the file is a map of roots, favorites, and workspaces"
  do
    workspaces-file <<<"${each%%|*}"
    run --separate-stderr "${kit_root}/bin/kit" workspaces --json
    [[ "$status" -eq 1 && -z "$output" && "$stderr" == *"workspaces.yml: ${each#*|}"* && "$stderr" != *jq:* ]]
  done
}

@test "a file kit workspaces refuses is refused by kit flow and kit workspace, with 1 and no document" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/o/a], exclude: [nonsense]}}'
  for words in "flow desk" "workspace repos" "workspace branches"
  do
    run --separate-stderr "${kit_root}/bin/kit" ${words} --workspace w --json
    [[ "$status" -eq 1 && -z "$output" && "$stderr" == *"'nonsense' in w's exclude names no repository"* ]]
  done
}
```

- [ ] **Step 2: Run both files.** The first fails on exit 0; the second on `bad array subscript` and exit 0.
- [ ] **Step 3: Write the three refusals, the flag, and the calls.**
- [ ] **Step 4: Run `workspaces.bats`, `across.bats`, `workspace.bats`, and `flow.bats`.** All pass.
- [ ] **Step 5: Commit**, subject `#89 kit: the workspace file is checked wherever it is read`.

## Task 2: One startup rule for every verb

**Files:** modify `kit/modules/bash/workspaces.sh`, `kit/bin/kit-flow:17-25,46-47,240-288`,
`kit/bin/kit-workspace:14-17,149-178`, `kit/bin/kit-workspaces:119-127`, and the `flow.json` fixture.

**Interfaces:**

- `workspaces:across [<workspace>]` prints the workspace a verb draws across: one given, unless it is `none`; with
  none given, `workspaces:current`'s name when its `why` is `last`; else nothing. It returns that function's status.
- `kit-flow` and `kit-workspace` run `workspace=$(workspaces:across "${workspace}") || exit $?` after their
  `--workspace` loops; `kit-flow` names `workspaces` in its `use`, and skips the call for `--me`, which reads one
  forge. Every later line reads `workspace` as today, so `--workspace none` draws what the verb draws with no group:
  the repository here, or outside one `kit flow`'s above-desks picture and `kit workspace`'s refusal with 2.
- `workspaces:name-refusal <name>` prints the line at `kit-workspaces:125`, or for `none`, `'none' is not a
  workspace's name: --workspace none names no workspace`, or nothing; `workspaces:check` refuses by it with 1, `--new`
  with 2, and Task 3's `--rename` with 2.
- The pages: `kit-flow`'s above-desks bullet gains "with no group used last, or with --workspace none", its
  `--workspace` line "none names no workspace"; `kit-workspace:14` adds "and outside one, the last group used".

- [ ] **Step 1: Write the failing test** in `tests/across.bats`:

```bash
@test "outside a repository the last group used is drawn; --workspace none draws none; inside one, nothing changes" {
  local-desks
  "${kit_root}/bin/kit" workspaces --use w
  run --separate-stderr "${kit_root}/bin/kit" flow --json --no-fetch
  [ "$(jq -c '[.desk.rows[].member]' <<<"$output")" = '["github.com/o/a","github.com/o/b"]' ]
  run --separate-stderr "${kit_root}/bin/kit" workspace repos --json
  [ "$(jq -c '[.rows[].by]' <<<"$output")" = '["fixed","fixed","fixed"]' ]
  run --separate-stderr "${kit_root}/bin/kit" flow --json --no-fetch --workspace none
  [ "$(jq -r type <<<"$output")" = array ]
  run "${kit_root}/bin/kit" workspace repos --json --workspace none
  [[ "$status" -eq 2 && "$output" == *"kit workspace reads a repository"* ]]
  run "${kit_root}/bin/kit" workspaces --new none github.com/o/a
  [[ "$status" -eq 2 && "$output" == *"--workspace none names no workspace"* ]]
  cd code/a && workspaces-file <<<'workspaces: [unclosed'
  run --separate-stderr "${kit_root}/bin/kit" flow desk --json
  [[ "$status" -eq 0 && -z "$stderr" && "$(jq -r .branch <<<"$output")" == main ]]
}
```

- [ ] **Step 2: Run the file.** It fails on its first assertion: after `--use`, the picture is the above-desks array.
- [ ] **Step 3: Write the two functions, the two verbs' calls, and the pages;** regenerate `flow.json`.
- [ ] **Step 4: Run the four files Task 1 ran, `syntax.bats`, and `kit syntax --check`.** All pass.
- [ ] **Step 5: Commit**, subject `#89 kit: kit flow and kit workspace start where kit workspaces --current says`.

## Task 3: The registry survives two writers, and has the verbs to manage it

**Files:** modify `kit/modules/bash/workspaces.sh:14,27-67,100-114,405-450`, `kit/bin/kit-workspaces:4-52,63-192`.

**Interfaces:**

- `workspaces:lock` takes the file's lock for this command, or returns 0 when this command holds it. The lock is
  `workspaces.yml.lock` in `workspaces:cache`'s directory, kit's own state: a symbolic link that names `$$`, made by
  `ln -s` and removed at exit by `forge:unlock`'s rule, as `forge:snapshot`'s is
  (`modules/bash/forge.sh:342-358,393`), since macOS has no `flock`, and a `mkdir` lock names no holder. The module's
  `use` gains `file`, for `file:at-exit`.
- A lock whose target is no number, or names a process `kill -0` cannot find, is removed and taken. Otherwise it waits
  0.1 second a turn, `workspaces_lock_wait=300` turns, then says `<file> is being edited by process <holder>; run the
  command again` and returns 1. `workspaces:edit` takes the lock first, and `kit-workspaces` takes it at the top of
  each action that edits, before the action reads the names, so its check and its write are one turn.
- The usage lines: each form that names its target is an option with a value on a line of its own, with no `( | )`
  group, so `--add=w` is `--add w`, as `--new=w` is `--new w` (Command line 22): `--add <name> <repository>…`, `--remove
  <name> <repository>…`, `--rename <name> <new-name>`, `--delete <name>`, `--favorite <workspace>`, `--unfavorite
  <workspace>`. The page's JSON paragraph ends "The forms that edit print one line, and take no --json."
- `workspaces:rename <name> <new>` sets the key through yq's `key`, `(.workspaces[strenv(kit_ws_name)] | key) =
  strenv(kit_ws_new)`, which keeps the group's place and comments; it and `workspaces:delete <name>` touch `favorites`
  only inside `with(select(has("favorites")); …)`, so the file gains no key. `--rename` refuses with 2 a group
  unknown, a new name `workspaces:name-refusal` refuses, and one in use, `📚 <new> is a workspace already; choose
  another name`; it keeps the group the last used, and says `📚 <name> is now 📚 <new>`. `--delete` refuses a group
  unknown with 2, asks nothing, and says `📚 <name> is deleted; its checkouts are as they were`.
- `--unfavorite` takes out a favorite written as given before it asks `workspaces:kind` (Serve 20).
- `workspaces:spelled <member>` prints Decision 2's spelling; `workspaces:url-member` and `workspaces:normalize` print
  through it. `workspaces:roots` prints each root once, without its trailing slashes (Serve 26).

- [ ] **Step 1: Write the failing tests** in `tests/workspaces.bats`:

```bash
@test "twelve --add at once all land; a dead holder's lock is taken over, and a live one's refuses with 1" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/a/m0]}}'
  for n in 1 2 3 4 5 6 7 8 9 10 11 12; do "${kit_root}/bin/kit" workspaces --add w "github.com/a/m${n}" & done; wait
  lock="${XDG_CACHE_HOME}/kit/workspaces/workspaces.yml.lock"
  true & gone=$!; wait "${gone}"; ln -s "${gone}" "${lock}"   # ln fails when an --add left its lock behind
  kit-bash 'workspaces:add w github.com/a/m13'
  sleep 30 3>&- & holder=$!; ln -s "${holder}" "${lock}"
  run kit-bash 'workspaces_lock_wait=2; workspaces:add w github.com/a/m14'
  kill "${holder}"
  [[ "$status" -eq 1 && "$output" == *"workspaces.yml is being edited by process ${holder}"* ]]
  [ "$(kit-bash 'workspaces:fixed w' | grep -c .)" -eq 14 ]
}

@test "--rename and --delete keep the comment, the favorites, and the last group; a member is matched once" {
  clone-at "${BATS_TEST_TMPDIR}/code/one" https://GitHub.com/A/One.git
  printf 'roots: ["%s", "%s/"]\nfavorites: [w, gone]\nworkspaces:\n  # the core\n  w: {repos: [github.com/a/one]}\n' \
    "${BATS_TEST_TMPDIR}/code" "${BATS_TEST_TMPDIR}/code" | workspaces-file
  for words in "--use w" "--new z github.com/a/two" "--rename=w core" "--unfavorite gone" \
    "--add=core GitHub.com/A/One github.com/a/three"
  do
    "${kit_root}/bin/kit" workspaces ${words}
  done
  for words in "--rename core z" "--rename core none" "--rename core Bad/Name" "--rename no x" "--delete no"
  do
    run "${kit_root}/bin/kit" workspaces ${words}
    [ "$status" -eq 2 ]
  done
  grep --quiet '# the core' "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$(jq -c '[.name, .why, .favorites[].name, (.members[].checkouts | length)]' <<<"$output")" = \
    '["core","last","core",1,0]' ]
  "${kit_root}/bin/kit" workspaces --delete core
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$(jq -c '[.kind, .favorites]' <<<"$output")" = '["none",[]]' ]
}
```

- [ ] **Step 2: Run the file.** The first counts fewer than 14; the second stops at `unknown flag '--rename=w'`.
- [ ] **Step 3: Write the lock, the usage lines and the page, the two actions, and the functions above.**
- [ ] **Step 4: Run `workspaces.bats` five times, then `across.bats` and `kit syntax --check`.** All pass.
- [ ] **Step 5: Commit**, subject `#89 kit: edits of the workspace file take a lock; a group is renamed or deleted`.

## Task 4: A member that fails is visible and counts

**Files:** modify `kit/modules/bash/workspaces.sh:49-54,189-212`, `kit/modules/bash/across.sh:299-319,420-438`,
`kit/modules/bash/workspace.sh:40-62,116-127`, `kit/bin/kit-workspace:140-141,191-204,267-294`.

**Interfaces:**

- `across:quota-merge <dir>` says each member whose status is not 0, `❌ <member>: <across:line of its .err>`, on
  stderr, and returns 1 when members ran and none could be read (Serve 12). Its test at `tests/across.bats:283`
  gives each of its three a `.member` and the failed one an `.err`, and asserts that line.
- `kit workspace` as text, a stage or the picture, draws all it draws today, then exits 1 when a document held an
  entry under `errors`, or a repos row an `error`; JSON carries them and exits 0, as kittui reads it (Serve
  23). The page's Exits line reads "1 when a repository, a member, or the workspace file cannot be read".
- `workspace:remote-facts` reads `remotes=$(forge:remotes) || return $?` before its loop: the process substitution at
  `:60` drops the failure today, so `kit workspace branches` exits 0 where `remotes` exits 1. `workspace:commits`
  appends each branch to its hashes in a bash read loop, with no sed (Serve 13).
- `kit-workspace:192,200` ask `git rev-parse --show-toplevel`, as `workspaces:current` does, so a bare repository is
  outside a repository, and is refused at `:202` with 2 (Serve 22).
- `workspaces:member-of` drops yq's line. `workspaces:unread <checkout>` prints `<checkout>/kit.yml does not parse, so
  its primary is read as origin: <yq's line>`, or nothing; `workspaces:member-json` warns with it for the member's
  first checkout, and `across:repos-of` puts it in a row's `error` that the desk's read left empty (Decision 3).

- [ ] **Step 1: Write the failing tests**, the first in `tests/across.bats`, the second in `tests/workspace.bats`:

```bash
@test "a member whose limits cannot be read, or whose kit.yml does not parse, says so where it is drawn" {
  two-desks
  PATH="$(shims gh)" run --separate-stderr "${kit_root}/bin/kit" flow --quota --workspace w
  [[ "$status" -eq 1 && "$output" == '[]' && "$stderr" == *"❌ github.com/o/a: "*"❌ github.com/o/b: "* ]]
  printf 'primary: [\n' > "${BATS_TEST_TMPDIR}/code/a/kit.yml"
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show github.com/o/c --json
  [ -z "$stderr" ]
  run --separate-stderr "${kit_root}/bin/kit" workspace repos --workspace w --json
  [[ "$(jq -r '.rows[0].error' <<<"$output")" == *"kit.yml does not parse, so its primary is read as origin"* ]]
}

@test "a member that cannot be read is its line, and the text exits 1; & and | in a branch; a bare repository" {
  code=${BATS_TEST_TMPDIR}/code
  history-at "${code}/a" github.com/o/a && history-at "${code}/b" github.com/o/b
  git -C "${code}/a" branch 'feat/a&b' 7-fix && git -C "${code}/a" branch 'pipe|name' 7-fix
  printf 'forges:\n  "-x": github\n' > "${code}/b/kit.yml"
  printf 'roots: [%s]\nworkspaces: {w: {repos: [github.com/o/a, github.com/o/b]}}\n' "${code}" | workspaces-file
  for stage in remotes branches commits
  do
    run "${kit_root}/bin/kit" workspace "${stage}" --workspace w --text
    [[ "$status" -eq 1 && "$output" == *"❌ github.com/o/b: '-x' in forges in kit.yml is not a remote name"* ]]
  done
  run --separate-stderr "${kit_root}/bin/kit" workspace commits --workspace w --json
  [[ "$status" -eq 0 && "$(jq -c '.rows[0].branches | sort' <<<"$output")" == '["7-fix","feat/a&b","pipe|name"]' ]]
  git init --quiet --bare "${BATS_TEST_TMPDIR}/bare.git" && cd "${BATS_TEST_TMPDIR}/bare.git"
  run "${kit_root}/bin/kit" workspace worktrees --json
  [[ "$status" -eq 2 && "$output" == *"kit workspace reads a repository"* && "$output" != *fatal:* ]]
}
```

- [ ] **Step 2: Run both files.** The first fails on `[]` with exit 0; the second where `remotes --text` exits 0.
- [ ] **Step 3: Write the six changes.**
- [ ] **Step 4: Run `across.bats`, `workspace.bats`, `workspaces.bats`, `flow.bats`, and `exits.bats`.** All pass.
- [ ] **Step 5: Commit**, subject `#89 kit: over a workspace a member that cannot be read is said and counted`.

## Task 5: kittui's workspace menu and repos tab

**Files:** modify `projects/tui/internal/shell/books.go:1-7,38-44,126-161`, `internal/flow/repos.go:19-53,96-116`;
test `internal/shell/books_test.go:72,103`, `internal/{welcome,flow}/books_test.go`, `internal/flow/repos_test.go`.

**Interfaces:**

- `BooksMenu.lines()` gives the refusal, the groups, and the favorites, a rule only between two that are there, and no
  line that opens what is not built: by the owner's rule a line that can never act is omitted, and the three at
  `books.go:158-160` have no target until spec step 7. The constants `browse`, `create`, and `manage` go. With no line
  left it gives one that never acts: `Item{Glyph: Books, Name: "no group yet · kit workspaces --new <name>
  <repository>…"}`. The three tests that look for `Browse…` or `Manage workspaces…` look for `w 2` alone.
- `repo` gains `Checkouts []string` (`json:"checkouts"`), which kit prints and kittui drops today. The `checkout`
  value's `Raw` is every clone, the one kit uses first, or `Checkout` alone where kit gave no list. `pathCells` draws
  each as a value in the open row; the closed row's summary is the first's, then a space and `+N` for N others.

- [ ] **Step 1: Write the failing tests**, with the files' own `names`, `inGroups`, `answer`, and `checkoutCell`:

```go
// shell/books_test.go, TestBooksDrawsNoLineThatOpensNothing:
//   names(BooksMenu{Workspace: inGroups}) gives "w 2|v 1||w 2|o/b · not cloned", joined by |, three of them acting;
//   names(BooksMenu{Workspace: kit.Workspace{Kind: "none"}}) gives one line that starts "no group yet", not acting
// flow/repos_test.go, TestAMemberClonedTwiceNamesItsOtherCloneOpenAndCountsItClosed, with HOME=/home/u: the row
//   {"repository": "github.com/o/a", "remotes": [], "by": "fixed", "checkout": "/home/u/Code/o/a",
//    "checkouts": ["/home/u/Code/o/a", "/home/u/old/a"]}, answered, draws its checkout's cell c, as checkoutCell does:
slices.Equal(c.Values, []string{style.Off("~/Code/o/a"), style.Off("~/old/a")}) && c.Summary == style.Off("…/o/a +1")
```

- [ ] **Step 2: Run `go test ./internal/shell/ ./internal/flow/`.** Both fail: `Browse…` is a line; no `+1`.
- [ ] **Step 3: Write the two changes,** and the four tests' new expectations; `:103` clicks `o/b · not cloned`.
- [ ] **Step 4: Run `go test ./...` and the gate's coverage lines, `bin/ci:63-70`.** All pass, at 98.0% or more.
- [ ] **Step 5: Commit**, subject `#89 kittui: the workspace menu draws no line that opens nothing`.

## Task 6: The docs and the spec say what is true

**Files:** modify `docs/workspaces.md:24-29,36-50,52-67,81-82,122-126`,
`docs/superpowers/specs/2026-11-08-workspaces-design.md:3,125-129,159-167,196-199,271-285`.

- [ ] **Step 1: Run the checks that fail today.** `grep -c 'Each form prints JSON' docs/workspaces.md` prints 1, and
  `grep -c 'bare repository\|--rename\|kit/workspaces/last' docs/workspaces.md` prints 0.
- [ ] **Step 2: Write `docs/workspaces.md`.** § The file: Task 1's refusals, and `none`. § Members and checkouts:
  Decision 2's spelling, Decision 3's line, a root written twice, and that a clone behind a symbolic link, and a bare
  repository, are no checkouts. § Which workspace kit starts on: `kit flow`, `kit workspace`, and `kit workspaces
  --current` follow the rule, other verbs run in their folder's repository, and the last group is kept in
  `kit/workspaces/last` under `XDG_CACHE_HOME`, else `~/.cache`. § Commands: `--rename`, `--delete`, and "The list,
  `--show`, and `--current` print JSON with `--json`; the forms that edit print one line". § Still to come: exactly
  the spec's steps 4 to 7, in its words.
- [ ] **Step 3: Write the spec.** Line 3: "steps 1 to 3 built, 4 to 7 designed". After line 129: `match` and `project`
  are refused by name until built. Lines 159-167: `none` is no group's name, and `kit workspace` takes `--workspace`
  too. Lines 196-199: the three lines join the dropdown with step 7. After line 282: steps 1 to 3 are built.
- [ ] **Step 4: Run `kit lint` over both files, and `kit generate --check`.** Both pass; the first grep prints 0.
- [ ] **Step 5: Commit**, subject `#89 docs: workspaces say what is built, and that steps 4 to 7 are to come`.

## Not built here: spec steps 4 to 7, what each starts from, and its size

- Uncloned members' forge reads (step 4, L): `across:start` skips a member with no checkout (`across.sh:175`); a stage
  reader takes its repository from the checkout's remote, and `workspaces:member-json` knows a forge by host.
- Batching by forge and owner (step 4, L): GitHub's provider reads by GraphQL, one repository a query
  (`modules/bash/forge/github.sh:165,205,363`); GitLab and Azure DevOps stay one `kit flow` per member.
- The caches and queries (step 4, M to L): `workspaces:cache` holds `last` alone; `forge:snapshot` and
  `flow:stage-json` are the idioms; Decision 1's (b) lists what `match:` takes, and `by` gains `query`.
- An issue shown once (step 4, M): rows are keyed by member (`internal/flow/rows.go:498`); each issue has `source`.
- Clone on action (step 4, M): `internal/flow/act.go:244` refuses through `m.nowhere`; kit needs a verb that clones.
- Labels and topics (step 5, S and M): `git config --get-all kit.workspace` in each checkout; a topic needs step 4.
- Project narrowing (step 6, M to L): the forge seam has no Project call; backlog and underway narrow to its items.
- The Workspaces screen (step 7, L): `kit workspaces --json` and this plan's edit verbs are its data; `kit-workspaces`
  has no hand-off to kittui; Task 5's three lines return. Apps 28 is in no task here.
