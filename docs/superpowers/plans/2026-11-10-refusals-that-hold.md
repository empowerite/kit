# Refusals That Hold Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every kit verb that promises not to destroy, or not to write, keeps the promise: `kit rebase`, `kit tidy`,
`kit serve`'s tools, the kit's regions, `kit packages remove`, `kit work`, `kit commit`, and `kit submit`.

**Architecture:** Each fix is a refusal, or a narrower write, decided before the verb changes anything, in the module
that owns the verb, and proven by a bats test that replays the review's scenario and asserts the state is as it was.
`git:push` becomes the one way a branch reaches its remote, `region:has` the one judge of a region's markers, and
`kit serve` gets one contract, enforced where a call's arguments become the verb's words.

**Tech Stack:** bash, jq, git, bats, and the fake `gh` in `tests/fakes/fake-tool`.

**Spec:** the review's findings Workflow 1, 2, 6, 7, 9, 10, 11, 13, and 14; Serve 1, 2, 3, 4, 5, 7, and 9;
Documents 1; and Delivery 1, 2, 10, 11, and 13. Each task names the scratch scripts its reviewer reproduced the
finding with, as the reviewer named them; the scripts are not kept, and each finding's scenario holds the same
steps. The plan Fail Closed owns the reads and checks that fail and pass anyway: Workflow 4 and 5, and
Delivery 5 and 7.

**Review:** [the findings](../reviews/2026-11-10-code-review-findings.md) of the review of 2026-11-10, cited
here by area and number: Serve 8 is item 8 under its Serve heading.

## Global Constraints

- kit commit only, never kit submit or push; every subject starts with `#141`, the issue for this plan.
- `export PATH=$PWD/kit/bin:$PATH` before `kit commit`.
- Peers share the worktree: stage named paths only, guard that the index holds only them, never leave anything
  staged. A commit step is `git add` of the task's files, `git diff --cached --name-only` showing exactly
  those, then `kit commit --message "<subject>" --message "Co-Authored-By: Claude <noreply@anthropic.com>"`.
- Everything under `kit/` is written for a consumer, and a path in it is written from `kit/`:
  `modules/bash/git.sh`.
- Lines are 120 cells at most.
- A verb's header comment is its help page and its usage line is its grammar (`kit syntax --check` holds them): a
  new flag is added to the header's usage line and described there, and `docs/` says the same. This plan adds no
  flag; a header it changes says what the verb now refuses.
- A refusal changes nothing: it is decided before the verb's first write, exits 1, and names what to do.
- "Run `x.bats`" is `KIT_TOOLCHAIN=mise kit/bin/kit tools run bats tests/x.bats`. In a test's code, a `#` line is
  setup, a run, or an assertion to write out with the file's helpers and the review's script, and assertions
  that share a line go one to a line.

## Review Focus

1. No test races two sessions through `kit rebase`: a push that lands between its fetch and its push must be
   refused by the lease, with nothing overwritten.
2. A squash-merged branch whose change took commits this clone never fetched: `kit tidy` keeps it, the safe
   side, with a count that is wrong.
3. Linux, and a git or jq older than this desk's 2.56 and 1.8: `--force-with-lease=<ref>:` expecting nothing,
   `git stash apply --index` over untracked files, `readlink -f` on a missing path, the 128 KB cap on one word.
4. GitLab: `pr-head-for-branch` is proven against a fixture's `sha` alone, and no workflow tool ran against it.
5. An unstaged `require_signing: false`: the setting is read from the disk's `kit.yml`, the policy from the tree.
6. A file with CRLF line endings: `region:has` sees no marker line, so `kit update` appends a second region.

---

## Decisions for the owner

1. `kit rebase` when the branch's remote has commits this checkout lacks: take them, or refuse. Default: take
   them. `kit submit` names `kit rebase` as the remedy, and the deny lists leave an agent no other verb.
2. `check` and `lint` over MCP run commands the repository's files name: annotate both `readOnlyHint: false`, or
   keep the hint and have `check` skip generated regions, which still leaves `lint`. Default: annotate both.
3. `flow`'s `workspace` over MCP reads other clones' uncommitted paths: do not offer it, or offer and document
   it. Default: do not offer it; kittui, which needs `--workspace`, does not go through MCP.
4. `require_signing: true`: `kit commit` and `kit work`'s start commit sign, and refuse when git cannot; or they
   stay, and the guard's "every commit here is signed" is reworded. Default: sign or refuse.

## Task 1: kit rebase takes origin's commits, and pushes the branch to its own name

Workflow 1 and 11: `workflow/t2-rebase-clobber.sh`, `t15-pushdefault.sh`. **Files:** modify
`kit/modules/bash/git.sh:282-327`, `kit/bin/kit-rebase:6-16`, `docs/features.md:87-88`; test `tests/rebase.bats`.
**Interfaces:** `git:pushes-to <branch> [<path>]` prints `branch.<branch>.remote` and returns 0 when
`branch.<branch>.merge` is `refs/heads/<branch>`; otherwise it prints `origin` and returns 1. `git:push <path>
<branch> [<option>…]` pushes `refs/heads/<branch>` to the same name on the remote `git:pushes-to` prints, quietly,
with the options, and with `--set-upstream` when that returned 1. `git:rebase [<path>]`, after its four refusals,
notes HEAD, fetches the primary and the branch's remote, and reads `theirs`, the tip of
`refs/remotes/<remote>/<branch>`, or nothing. When `theirs` is not an ancestor of HEAD it says `taking <n> commit(s)
<remote>/<branch> has and this checkout lacks` and rebases onto it; then onto `<primary>/<default>`; then `git:push
<path> <branch> --force-with-lease=refs/heads/<branch>:<theirs>`. A failed rebase is aborted and HEAD reset:
`<branch> conflicts with <its target>; the rebase was aborted and the worktree is as it was`. A failed push returns
1: `<branch> is rebased here and was not pushed; see above, then run: kit rebase`.

- [ ] **Step 1: Write the failing tests** in `tests/rebase.bats`, with its `worktree` and `commit-in`:

```bash
@test "rebase takes the commits origin's branch has and this checkout lacks, and pushes both" {
  # 1-x pushed from ../wt; a clone ../other commits theirs.txt on it, pushed; ../wt commits mine.txt; kit rebase ../wt
  [ "$status" -eq 0 ]; [[ "$output" == *"taking 1 commit(s) origin/1-x has and this checkout lacks"* ]]
  git -C ../origin.git merge-base --is-ancestor "$(git -C ../other rev-parse HEAD)" 1-x
  # a second test, both committing file a: status 1, "1-x conflicts with origin/1-x", ../wt and origin's 1-x unmoved
}
@test "rebase pushes the branch to its own name on origin, never where push.default points" {
  # git worktree add ../wt -b 1-x origin/main; push.default upstream in it; one commit; origin's main is ${main}
  [ "$status" -eq 0 ]; [ "$(git -C ../origin.git rev-parse main)" = "${main}" ]
  [ "$(git -C ../wt rev-parse --abbrev-ref '@{upstream}')" = origin/1-x ]
}
```

- [ ] **Step 2: Run** `rebase.bats`: the three fail, origin's commit on no ref, and origin's `main` moved.
- [ ] **Step 3: Write `git:pushes-to` and `git:push`, and rewrite `git:rebase`.** Its comment, `kit-rebase`'s
  header, and `docs/features.md` item 4 say it takes the remote's commits first and forces only over that tip.
- [ ] **Step 4: Run** `rebase.bats`: all 8 pass.
- [ ] **Step 5: Commit**, subject `#141 kit: kit rebase takes origin's commits first, and pushes by name`.

## Task 2: kit tidy keeps commits no remote ever had

Workflow 2 and 14: `workflow/t1-tidy-unpushed.sh`, `t9-tidy-locked.sh`. **Files:** modify
`kit/modules/bash/git.sh:103-280`, `kit/modules/bash/forge.sh:68-70,283-285`, `kit/modules/bash/forge/github.sh`,
`kit/modules/bash/forge/gitlab.sh`, `kit/packages/kit/files/repo/.kit/templates/bash/forge-provider`,
`kit/bin/kit-tidy:6-20`; test `tests/tidy.bats`, `tests/forge-conformance.bats:61,122,245`.
**Interfaces:** `forge:pr-head-for-branch <branch>` prints the head commit of the latest pull request on the branch,
in any state, or nothing: GitHub's `headRefOid` from `gh pr list --head <branch> --state all`, GitLab's `sha` from
`merge_requests?state=all&source_branch=<branch>`. Each provider defines it between `pr-for-branch` and `pr-here`,
as the template orders them, failing soft as `pr-state-for-branch` does. `git:unpushed <branch>` prints `git
rev-list --count refs/heads/<branch> --not --remotes`; when that is not 0 it asks the forge for the head, and when
this clone has that commit it counts again with the head after `--not` too. `git:tidy-worktree` keeps a MERGED or
CLOSED worktree whose count is not 0: `keeping <path> (<branch>, change <state>, <n> commit(s) no remote ever had)`.
It removes the worktree before it settles; when git fails it says `keeping <path> (<branch>, <state>, git would not
remove it)` and returns 1. `git:tidy`'s branch sweep says `keeping <branch> (<n> commit(s) no remote ever had)`, and
warns `<branch> could not be deleted; see above` when `git branch -D` fails; it counts neither.

- [ ] **Step 1: Write the failing tests** in `tests/tidy.bats`; `change-on` takes a third argument, the head, its
  fixture `[{"state": "$2", "headRefOid": "${3:-}"}]`. In `tests/forge-conformance.bats`, `pr-all.json` (`:61`)
  gains `"headRefOid": "1f2e3d4"`, `mr-states.json` (`:122`) `"sha": "1f2e3d4"`, and after `:245` `run call
  pr-head-for-branch 7-fix-the-thing` prints it.

```bash
@test "kit tidy keeps a worktree, and a gone branch, whose commits no remote ever had, though the change merged" {
  # world; worktree-on 7-fix-the-thing; a commit in it, never pushed; gone; change-on … MERGED "$(git rev-parse main)"
  # branch 8-idea tracking origin/8-idea, moved to a commit made with git commit-tree; gone 8-idea; kit tidy
  [ "$status" -eq 0 ]; [[ "$output" == *"(7-fix-the-thing, change MERGED, 1 commit(s) no remote ever had)"* ]]
  [[ "$output" == *"keeping 8-idea (1 commit(s) no remote ever had)"* ]]; [ -d worktrees/7-fix-the-thing ]
  # a second test pins the other side: the commit pushed before gone, and given as change-on's head, is removed
}
@test "kit tidy keeps a worktree git will not remove, says so, and goes on to the next" {
  # world; 7-fix-the-thing and 8-next, each worktree-on, gone, change-on MERGED; git worktree lock the first; kit tidy
  [ "$status" -eq 0 ]; [[ "$output" == *"(7-fix-the-thing, MERGED, git would not remove it)"* ]]
  [[ "$output" == *"removed 1 worktree(s) and 1 branch(es)."* ]]; [ ! -e worktrees/8-next ]
}
```

- [ ] **Step 2: Run** `tidy.bats` and `forge-conformance.bats`: the first test fails, both refs deleted; the last
  exits 1; conformance does not find the call. The squash test passes before and after.
- [ ] **Step 3: Write the seam call in `forge.sh`, both providers, and the template; then `git:unpushed` and the
  changes to tidy.** `kit-tidy`'s "It never touches" list gains the branch and the worktree it now keeps.
- [ ] **Step 4: Run** both files: all pass; `kit lint` over the two providers passes, the template's shape held.
- [ ] **Step 5: Commit**, subject `#141 kit: kit tidy keeps a branch whose commits no remote ever had`.

## Task 3: kit serve's contract, stated and enforced

Serve 1, 2, 7, 9 and Documents 1: `serve/t9.sh`, `t14.sh`, `t10.sh`. The contract, which each document
states in its own words: a tool that says `readOnlyHint: true` runs no command the repository's files name, takes
no word that acts, reads no path outside the repository, changes no file git sees, and never fetches. **Files:**
modify `kit/modules/jq/serve.jq:111-151`, `kit/modules/bash/serve.sh:17-33,60-101,427-457`,
`kit/bin/kit-serve:16-19`, `docs/serve.md`, `kit/packages/kit/files/repo/.kit/1st.md:81-83`, `README.md:202`,
`docs/packages/agents/claude.md:33-36`; test `tests/serve.bats`.
**Interfaces:** `serve::argv($args; $opts)` gives two more faults, checked where `starts with -` is, in this order:
an operand's value that is a literal of the verb's own syntax, offered or not, `<key>'s value <v> is a word kit
<verb> acts on`; and a value off its property's `enum` or `items.enum`, a glob too, `<key>'s value <v> is not one
of: <the choices, comma-separated>`. In `serve.sh`, `serve_acts` gains `check` and `lint`, each `{"readOnlyHint":
false, "destructiveHint": false, "idempotentHint": true}`; `serve_hidden` gains `[flow]='--workspace'`;
`serve_paths=(context kinds lint)` names the tools whose operands are paths. `serve:inside <path>` succeeds when the
path, read from the root with every symbolic link followed, is the root or under it; one that does not exist is
judged by its nearest directory that does, and is outside when a `..` is left after that. `serve:tools-call` adds
the listing-only sentence to the `acts on` refusal too, and, for a tool in `serve_paths`, refuses each word that
does not start with `-` and is not `serve:inside`: `-32602`, `Invalid params: <word> is outside the repository kit
serve runs in`.

- [ ] **Step 1: Write the failing tests** in `tests/serve.bats`, with its `serve`, `call`, `modern`, and `answer`.
  Bring the existing lists of the tools that do not read only (`:111-116`, `:215-216`) to the eight names, the
  open-world assertion at `:115` to the six workflow tools by name, and `:190`'s title to what it now proves.

```bash
@test "a listing-only tool refuses an action given as an operand's value, and a value off its enum, unrun" {
  serve "$(call 1 packages '{"package":["add","style/yaml"]}')" "$(call 2 packages '{"package":["nosuch"]}')"
  answer 1 | jq --exit-status '.error | .code == -32602 and (.message | contains("add is a word kit packages acts on"))
    and (.message | contains("kit serve offers kit packages as a listing only"))'
  answer 2 | jq --exit-status '.error.message | contains("value nosuch is not one of: ")'
  # kit.yml's cksum is as before the calls, and .kit/styles/yaml.md is not there
}
@test "check and lint say they run the repository's commands; a tool that reads only leaves the repository" {
  # notes.md holds a region whose command is `touch ran; echo hi`; before=$(git status --porcelain; cksum kit.yml)
  # tools/list as 1, then a call of each of context (notes.md), kinds, packages, help, and version
  answer 1 | jq --exit-status '[.result.tools[] | select(.annotations.readOnlyHint | not) | .name]
    == ["check", "lint", "assign", "close", "gather", "settle", "unassign", "work"]'
  [ ! -e ran ]; [ "$(git status --porcelain; cksum kit.yml)" = "${before}" ]
}
@test "a path outside the repository is refused however it is spelled, and flow offers no workspace" {
  # ${BATS_TEST_TMPDIR}/outside.sh, and link.sh, a link to it; calls 1 to 5: lint with its absolute path, context
  # with ../outside.sh, kinds with path link.sh, context with nosuch/../../outside.sh, kinds with kind = the path
  answer "${id}" | jq --exit-status '.error | .code == -32602 and (.message | contains("is outside the repository"))'
  # call 6, flow with workspace x and stage desk: "unknown argument workspace"; call 7, context kit.yml: a result
}
```

- [ ] **Step 2: Run** `serve.bats`: the three fail. Answer 1 is a result and `kit.yml` gained `style/yaml`; the
  list holds six names; answers 1 to 5 are results.
- [ ] **Step 3: Write the code, then the documents.** Each names three groups: nine tools that read only
  (`context`, `flow`, `help`, `kinds`, `packages`, `syntax`, `toolchains`, `tools`, `version`); `check` and
  `lint`, which run the repository's commands; six workflow tools that act. `kit-serve:16-19` and
  `serve.sh:17-33` drop "the verbs that read and report, and do not change anything" and the list of eleven.
  `docs/serve.md` moves `check` and `lint` out of the first table, says the contract where "Every other tool
  carries `readOnlyHint`" stands, and adds to "A call that does not fit" a word the verb acts on, a value off
  its enum, a path outside the repository, and `workspace`. `1st.md:81-83`, `README.md:202`, and
  `claude.md:33-36` name the groups in place of "the read-only verbs", and `claude.md` adds that a yes there
  offers the tools that close issues too.
- [ ] **Step 4: Run** `serve.bats` and `jq.bats`: all pass; `kit syntax --check` passes.
- [ ] **Step 5: Commit**, subject `#141 kit: kit serve holds each operand to its enum and to the repository`.

## Task 4: A region's markers are judged once, before any write

Delivery 1, 2, 10, and 11: `delivery/t1`, `t3`, `t18`, `t8`. **Files:** modify
`kit/modules/bash/region.sh:70-82,159-178,204-226`, `kit/modules/bash/place.sh:212-215`,
`kit/modules/bash/package.sh:179-187`, `kit/modules/bash/uninstall.sh:16-38,58-83`, `kit/bin/kit-uninstall:23`;
test `tests/kit.bats`, `tests/uninstall.bats`.
**Interfaces:** `region:has <dest> [<file>]` takes the markers from `<dest>` and reads `<file>`, `<dest>` when not
given, in one awk pass, with no grep. It returns 0 for exactly one directive line with exactly one end line after
it; 1 for neither line, or no file; 2 otherwise, with an error: the present "opens with … and never closes" for one
directive and no end line, else `<file>: the kit's region is marked wrongly: '<begin>' and '<end>' must each be
there once, in that order`. `region:write` and `region:remove` call `region:has "$1" "${path}"`, `path` being
`release:target`'s; on 2 both return 1 and write nothing, and on 1 `region:remove` returns 0 and leaves the file.
`place:remove` says "the kit's region taken out" after `region:remove` succeeds, and returns its status.
`package:remove` and `uninstall:run` count the paths `place:remove` could not take out; with any, they stop before
the manifests and the declaration and return 1: `<n> path(s) could not be taken out, each named above; settle each,
then run: kit packages remove <named>`, or `… then run: kit uninstall`.

- [ ] **Step 1: Write the failing tests**, two in `tests/kit.bats`, where `update` is `env PATH="$(shims mise
  curl)" "${kit_checkout}/bin/kit" update`, and the third in `tests/uninstall.bats`:

```bash
@test "a region in a linked file is found under the link's name, so an update never adds a second" {
  # .gitignore is a link to shared/gitignore, which holds build/; kit install; update twice
  [ "$status" -eq 0 ]; [ -L .gitignore ]
  [ "$(grep --count --line-regexp --fixed-strings '# kit:generate .gitignore' shared/gitignore)" -eq 1 ]
}
@test "markers out of order, or doubled, refuse the write and leave the file; a marker that starts with -- is text" {
  # kit install; .gitignore becomes: top, the end line, middle, the directive, old, bottom-1, bottom-2; update
  [ "$status" -eq 1 ]; [ "$(cat .gitignore)" = "${before}" ]
  [[ "$output" == *"'# kit:generate .gitignore' and '# kit:end .gitignore' must each be there once, in that order"* ]]
  # the same for a whole region, then mine, a second directive, and tail: status 1, and the tail line is there
  # then a sound .gitignore and a local package extra, made as kit.bats:336-337 makes it, whose
  # regions/repo/schema.sql holds one line; kit packages add extra; update
  [ "$status" -eq 0 ]; [[ "$output" != *"grep:"* ]]
  [ "$(grep --count --line-regexp --fixed-strings -e '-- kit:generate schema.sql' schema.sql)" -eq 1 ]
}
@test "uninstall refuses a region that never closes, keeps every line after it, and keeps the declaration" {
  # .gitignore holds mine-before; kit install; mine-after-1 and mine-after-2 appended; the end line deleted; uninstall
  [ "$status" -eq 1 ]; [[ "$output" == *"1 path(s) could not be taken out"* ]]
  [[ "$output" == *".gitignore: the kit's region opens with '# kit:generate .gitignore' and never closes"* ]]
  [ "$(cat .gitignore)" = "${before}" ]; grep --quiet '^version: ' kit.yml
  # kit packages remove kit: status 1, and .gitignore is still ${before}
}
```

- [ ] **Step 2: Run** `kit.bats` and `uninstall.bats`: the three fail. `shared/gitignore` has three directives;
  the update exits 0 without `bottom-1`; the uninstall exits 0 and `.gitignore` is `mine-before` alone.
- [ ] **Step 3: Rewrite `region:has`, and change `region:write`, `region:remove`, `place:remove`,
  `package:remove`, `uninstall:run`, and `uninstall:path`.** `kit-uninstall:23` adds the new cause of exit 1.
- [ ] **Step 4: Run** `kit.bats`, `uninstall.bats`, and `packages.bats`: all pass, the test at `kit.bats:330` too.
- [ ] **Step 5: Commit**, subject `#141 kit: a region is one directive with one end line after it, or no write`.

## Task 5: kit packages remove leaves a copy the repository edited

Delivery 13: `delivery/t17`. The review's fix, a guard inside `place:remove`, would stop `place:retire`,
which calls it with `/dev/null` as the source (`place.sh:273`), so the guard is a predicate its callers share.
**Files:** modify `kit/modules/bash/place.sh` (after `place:remove`), `kit/modules/bash/package.sh:179-187`,
`kit/modules/bash/uninstall.sh:58-68`, `kit/bin/kit-packages:14-15`; test `tests/packages.bats`.
**Interfaces:** consumes Task 4's loop in `package:remove`. `place:edited <base> <path> <source>` succeeds when the
path is placed as a copy, is not `materialize:owned`, and its destination differs from `<source>`. `uninstall:path`
uses it in place of its own three tests. `package:remove` warns `<dest>: edited since the kit placed it; it is
left as it is` and goes on to the next path, as `kit uninstall` does.

- [ ] **Step 1: Write the failing test** in `tests/packages.bats`:

```bash
@test "remove leaves a copied file the repository edited, names it, and takes out the rest" {
  # kit install github/ci agents/cursor; a line appended to .github/workflows/kit-ci.yml; kit packages remove both
  [ "$status" -eq 0 ]; grep --quiet --line-regexp '# edited' .github/workflows/kit-ci.yml
  [[ "$output" == *".github/workflows/kit-ci.yml: edited since the kit placed it; it is left as it is"* ]]
  [ ! -e .cursor/rules/kit-bash.mdc ]; [[ "$(declared packages)" != *"github/ci"* ]]
}
```

- [ ] **Step 2: Run** `packages.bats`: it fails, `kit-ci.yml` gone.
- [ ] **Step 3: Write `place:edited` and use it in both callers.** `kit-packages`' `remove` paragraph gains "a
  copied file the repository edited is named and left".
- [ ] **Step 4: Run** `packages.bats`, `uninstall.bats`, and `kit.bats`: all pass.
- [ ] **Step 5: Commit**, subject `#141 kit: kit packages remove leaves a copied file the repository edited`.

## Task 6: kit work never works on the desk, and never loses or borrows a stash

Workflow 6, 7, and 10: `workflow/t4-sweep-orphan.sh`, `t5b-sweep-submodule.sh`, `t18-desk-branch.sh`.
**Files:** modify `kit/modules/bash/issue.sh:37,320-346,383-411,491-515`, `kit/bin/kit-work:10-17`; test
`tests/work.bats`.
**Interfaces:** `issue:start` refuses, right after `issue:branch` and before `issue:assign`, which moves below it,
when the main root has `<branch>` checked out: `refusing to start <primary>: its branch <branch> is checked out on
the desk, <main-root>, which is the human's; switch the desk to another branch, then run kit work again`.
`issue:sweep-push` prints the new stash's hash, or nothing when `refs/stash` is the same before and after the push;
with nothing, `issue:start` says `Nothing to sweep: what is uncommitted on <branch> is not work a stash takes`.
`issue_swept` holds the hash from the push until `issue:start` calls `issue:sweep-apply`, which behaves as it does.
`issue:sweep-restore <root>`, which `issue:start` registers with `file:at-exit` in its own shell once it holds a
hash, does nothing when `issue_swept` is empty; otherwise it applies the stash on the desk with `--index`, drops it,
and says `The swept work is back on <branch>.`, or, when that fails, `the desk's uncommitted work is in the stash:
git -C '<root>' stash apply <hash>`.
`issue:worktree` returns 1, a refusal, for a path that is not a worktree.

- [ ] **Step 1: Write the failing tests** in `tests/work.bats`. Add `desk-world`, the five lines the sweep test
  opens with (`:226-230`): the install, `forge: github`, `notes.txt`, the init commit, and `world`.

```bash
@test "kit work refuses when the desk has the issue's branch checked out, before it assigns, sweeps, or commits" {
  # desk-world; git fetch origin; git switch 7-fix-the-thing; notes.txt changed; kit work 7
  [ "$status" -eq 1 ]; [ "$(git status --porcelain)" = " M notes.txt" ]; [ -z "$(git stash list)" ]
  [[ "$output" == *"refusing to start #7: its branch 7-fix-the-thing is checked out on the desk"* ]]
  run ! grep --quiet 'add-assignee' "${FAKE_DIR}/gh.calls"   # and HEAD and origin's branch are where they were
}
@test "kit work that stops after the sweep puts the desk's work back, and leaves no stash" {
  # desk-world; notes.txt changed to mine, scratch.txt untracked; mkdir -p worktrees/7-fix-the-thing; kit work 7
  [ "$status" -eq 1 ]; [[ "$output" == *"The swept work is back on main."* ]]
  [ "$(cat notes.txt)" = mine ]; [ "$(cat scratch.txt)" = scratch ]; [ -z "$(git stash list)" ]
}
@test "a desk whose only uncommitted work is inside a submodule sweeps nothing, and the human's own stash stays" {
  # ${BATS_TEST_TMPDIR}/lib, one file l, added as the submodule lib with git -c protocol.file.allow=always;
  # desk-world; git stash push of a staged idea.txt, message 'an old idea'; printf 'wip\n' >> lib/l; kit work 7
  [ "$status" -eq 0 ]; [[ "$output" == *"Nothing to sweep"* ]]
  [ "$(git stash list --format=%gs)" = "On main: an old idea" ]; [ ! -e worktrees/7-fix-the-thing/idea.txt ]
}
```

- [ ] **Step 2: Run** `work.bats`: the three fail. The first exits 0 with a start commit on the desk's branch; the
  second exits 2 with the desk empty and one stash; the third leaves no stash, and `idea.txt` in the worktree.
- [ ] **Step 3: Change `issue:start`, `issue:sweep-push`, and `issue:worktree`, and write `issue:sweep-restore`**;
  add `file` to the module's `use`. `kit-work`'s header says it refuses a branch the desk has checked out, and
  that work it swept goes back to the desk when it stops before the worktree has it.
- [ ] **Step 4: Run** `work.bats`: all pass, the two sweep tests at `:225` and `:245` among them, unchanged.
- [ ] **Step 5: Commit**, subject `#141 kit: kit work refuses the desk's branch, and gives swept work back`.

## Task 7: kit commit and kit submit push by name, sign when required, and keep the subject

Workflow 9, 11, and 13: `workflow/t6-cleanup.sh`, `t15-pushdefault.sh`, `t24-misc.sh`. **Files:** modify
`kit/modules/bash/commit.sh:28,434-437,456-465,489-511`, `kit/modules/bash/issue.sh:374-381`,
`kit/bin/kit-commit:12`, `kit/bin/kit-submit:7-8`, `docs/packages.md:270`, `tests/helper.bash`; test
`tests/commit.bats`, `tests/work.bats`.
**Interfaces:** consumes Task 1's `git:pushes-to` and `git:push`, and Task 6's `desk-world`. `commit:submit` pushes
with `git:push . <branch>` in both places; when `git:pushes-to` returned 1 it is the first push, with the present
"which is its upstream from now on", and otherwise `ahead` counts from `refs/remotes/<remote>/<branch>`.
`commit:commit` and `issue:start-commit` pass `--cleanup=whitespace` to `git commit`, and `commit:commit` reports
the subject read back from HEAD. With `require_signing` true, both call `guard:require signing-key` and pass
`--gpg-sign`; `kit commit` then prints no "commits here are not signed", and when git fails says `the commit is
refused: require_signing is true and git could not sign it; see above`. `signer [<status>]`, new in
`tests/helper.bash`, writes a script, sets it as `gpg.program` here, and sets `user.signingkey`. The script drains
stdin, exits `<status>` when that is not 0, and otherwise runs `printf '\n[GNUPG:] SIG_CREATED \n' >&2`, the line
git looks for after a newline, and prints a `-----BEGIN PGP SIGNATURE-----` block on stdout.

- [ ] **Step 1: Write the failing tests**, two in `tests/commit.bats`, with its `kit`, one in `tests/work.bats`:

```bash
@test "kit submit pushes the branch to its own name on origin, whatever push.default and its upstream say" {
  # git branch --set-upstream-to=origin/main; push.default upstream; origin's main is ${main}; x staged; kit submit
  [ "$status" -eq 0 ]; [ "$(git rev-parse --abbrev-ref '@{upstream}')" = origin/7-fix-the-thing ]
  [ "$(git -C "${work}/origin.git" rev-parse 7-fix-the-thing)" = "$(git rev-parse HEAD)" ]
  # a second kit submit, of y, says "pushed 7-fix-the-thing: 1 commit(s)", and origin's main is still ${main}
}
@test "with require_signing, kit commit signs or refuses; commit.cleanup=strip cannot drop the subject it judged" {
  # require_signing: true appended to kit.yml; git config commit.cleanup strip; x staged
  # kit commit --message 'x: a file' with no key: status 1, and the output names git config user.signingkey
  # signer; kit commit --message 'x: a file' --message 'WIP'
  [ "$status" -eq 0 ]; [[ "$output" != *"commits here are not signed"* ]]
  [ "$(git log -1 --format=%s)" = "#7 x: a file" ]; git cat-file commit HEAD | grep --quiet '^gpgsig '
  # signer 2; y staged; kit commit --message 'y: a file'
  [ "$status" -eq 1 ]; [[ "$output" == *"require_signing is true and git could not sign it"* ]]
}
@test "kit work's start commit keeps its subject under commit.cleanup=strip, and is signed when signing is required" {
  # kit.yml holds require_signing: true; desk-world; git config --global commit.cleanup strip; signer; kit work 7
  [ "$status" -eq 0 ]; git -C worktrees/7-fix-the-thing cat-file commit HEAD | grep --quiet '^gpgsig '
  [ "$(git -C worktrees/7-fix-the-thing log -1 --format=%s)" = "#7 start: Fix the thing" ]
}
```

- [ ] **Step 2: Run** `commit.bats` and `work.bats`: the three fail. Origin's `main` moved; the first `kit commit`
  exits 0, unsigned; `kit work` stops at "Aborting commit due to empty commit message".
- [ ] **Step 3: Write `signer`, then change `commit:submit`, `commit:commit`, and `issue:start-commit`**; add
  `settings` to `commit.sh`'s `use`. `kit-commit:12` says it signs when `require_signing` is true and refuses when
  git cannot, and warns otherwise; `kit-submit:7-8` that the branch goes to its own name on origin, whatever
  git's push configuration says; `docs/packages.md:270` that `kit commit` and the start commit sign or refuse.
- [ ] **Step 4: Run** `commit.bats`, `work.bats`, and `rebase.bats`: all pass, the test at `commit.bats:544` too.
- [ ] **Step 5: Commit**, subject `#141 kit: kit submit pushes by name, and kit commit signs when it is required`.

## Task 8: kit serve outlives a malformed or oversized message

Serve 3, 4, and 5: `serve/t2.sh`, `t12.sh`. **Files:** modify
`kit/modules/bash/serve.sh:178-241,248-301,430-478`, `kit/modules/jq/serve.jq:127-136`, `docs/serve.md`; test
`tests/serve.bats`.
**Interfaces:** consumes Task 3's `serve::argv` and `serve:tools-call`. `serve:handle` answers a request whose
`params` is there and is not an object with `-32602`, `Invalid params: params is an object`, before any handler
runs; such a notification gets nothing. Nothing a client sent, and nothing a verb printed, is a word of a command
line or a variable of the environment: `serve:reply` and `serve:error` keep their signatures and hand jq the result
and the message on stdin or with `--rawfile`; `serve:tools-call` gives `serve::argv` the call's arguments on stdin
and the spec with `--argjson`, and the result's command text with `--rawfile`; `kit_serve_pending` names a file that
holds the pending line, which `serve:resume` reads and removes. A value quoted in an error is cut to 200 characters.
`serve::argv` gives, before its other faults, `<key>'s value holds a NUL, which no command line carries`.
`serve:tools-call` writes the words to a file, checks jq's status, and answers `-32603`, `Internal error: the verb's
words could not be read`, when it failed. A verb the system will not run is a result with `isError`, status 126.

- [ ] **Step 1: Write the failing tests** in `tests/serve.bats`:

```bash
@test "params that is not an object, and a NUL in an argument, are refused, and the server goes on" {
  # an initialize of 2025-06-18 as id 0, then these, then a notification with params [1], then a ping as id 4
  serve … '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":[1]}' \
    '{"jsonrpc":"2.0","id":2,"method":"resources/read","params":"kit://1st.md"}' \
    "$(call 3 help '{"verb":["version","a\u0000b"]}')" …
  [ "$(wc -l < "${BATS_TEST_TMPDIR}/out")" -eq 5 ]; answer 4 | jq --exit-status '.result == {}'
  # answers 1 and 2: .error.code == -32602, and .error.message contains "params is an object"
  answer 3 | jq --exit-status '.error | .code == -32602 and (.message | contains("holds a NUL"))'
  run ! grep --quiet 'kit help version' .kit/logs/mcp-*.log
}
@test "a result over a megabyte, and a request line over a megabyte, are answered, and the server goes on" {
  serve "$(call 1 syntax "{\"verb\": $(jq --null-input --compact-output '[range(50) | "flow"]')}")" "$(call 2 version)"
  answer 1 | jq --exit-status '.result | .isError == false and (.content[0].text | length > 1000000)'
  # answer 2 is a result. Then, piped to kit serve as the serve helper pipes: request 3, a context call written
  # with printf, its path 1500000 a's, since call would put it on jq's command line; and call 4 version
  answer 3 | jq --exit-status '.error.code == -32602 or .result.isError'
  answer 4 | jq --exit-status '.result.isError == false'
}
```

- [ ] **Step 2: Run** `serve.bats`: both fail. The first has one line of output, the server gone with status 5;
  the second has no answer 1, and `jq: Argument list too long` in `err`.
- [ ] **Step 3: Change `serve:handle`, `serve:reply`, `serve:error`, `serve:tools-call`, `serve:reload`, and
  `serve:resume`, and add the NUL fault to `serve::argv`.** `docs/serve.md` adds the NUL to "A call that does not
  fit", and to "The protocol" that a request whose `params` is not an object is refused with `-32602`.
- [ ] **Step 4: Run** `serve.bats` and `jq.bats`: all pass, the reload tests at `:300` and `:325` among them. Then
  `kit lint` and `kit generate --check` pass over every file this plan changed.
- [ ] **Step 5: Commit**, subject `#141 kit: kit serve refuses params that are not an object, and a NUL`.
