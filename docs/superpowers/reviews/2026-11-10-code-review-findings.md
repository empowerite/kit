# Code review, 2026-11-10: the findings

This page lists every finding of [the review](2026-11-10-code-review.md) of the tree at `2dad3552`, by area, each
area numbered from 1, so another page cites "Workflow 1". An item gives its severity and its kind, the place, the
defect, a scenario that shows it, the direction of a fix, and whether a reviewer confirmed it. Under each area,
further headings say what the reviewer found of the issues already filed, what holds, what no test covers, and
what the reviewer did not reach. The last section, Gaps, sets the specs, the plans, and the issues against the tree.

## Workflow

Reviewed the workflow verbs, from an issue to a merged pull request (`kit work`, `kit commit`, `kit submit`,
`kit rebase`, `kit tidy`, `kit assign`, `kit linked`, `kit policy`, `kit gate`, and the modules behind them under
`kit/modules/bash/`) at 2dad3552; every finding marked confirmed without a qualifier was reproduced in a throwaway
bare origin and clone with the tests' fake `gh`, on macOS, bash 5.3, and git 2.56.

1. Critical, bug. `kit/modules/bash/git.sh:317-325`. `kit rebase` force-pushes over commits that origin has and
   this branch lacks. `git:rebase` fetches the primary, then pushes with a bare `--force-with-lease`; with the
   primary being origin (the default), the fetch has just moved `origin/<branch>` to origin's tip, so the lease
   always holds, and `kit submit` names `kit rebase` as the remedy for exactly this state
   (`kit/modules/bash/commit.sh:534`). Scenario: session A pushes `#7 their work` to `7-fix-the-thing`; session B,
   without it, commits `#7 my work`; `kit submit` answers `origin refused the push … it has commits this branch
   lacks; run: kit rebase`, and `kit rebase` answers `✅ rebased onto origin/main and pushed`, exit 0. Origin's
   branch is then `my work` on `one`, A's commit is on no ref, and `git branch --contains <their sha>` on origin
   prints nothing. Fix: refuse (or integrate) when `@{upstream}` is not an ancestor of HEAD after the fetch, and
   push with `--force-with-lease=<branch>:<sha read before the fetch>`. Confirmed.
2. High, bug. `kit/modules/bash/git.sh:135,170,268`. `kit tidy` deletes commits that were never pushed. A worktree
   counts as finished by its change's state alone; the only local check is `git status --porcelain`, and the
   branch is then deleted with `git branch -D` once its upstream is gone. Scenario: a commit is made with
   `kit commit` and never submitted, the change merges, and the forge deletes the branch; `kit tidy` prints
   `removing … (MERGED)` and `deleting 7-fix-the-thing`, exit 0. The commit is on no ref, the branch and worktree
   reflogs went with them, and `git for-each-ref` lists only `main` and `origin/main`. The same holds for a
   closed, unmerged change whose remote branch was deleted, and for any local branch of the human's whose upstream
   is gone. Fix: keep the worktree and branch unless the local tip is the change's head commit (the forge knows
   it) or is reachable from the default branch; say how many commits origin never had. Confirmed.
3. High, bug. `kit/modules/bash/commit.sh:269-273`. The #61 guard misses a path that two unseen commits
   changed. `commit:behind` compares the staged blob with the blob before the last commit that touched the
   path, so an index that is two commits behind on a path matches neither and passes; #62 is not implemented
   either, so nothing brings the shared index up to a commit made from a private index. Scenario: session A
   commits `a` twice from `GIT_INDEX_FILE`, and the shared index shows `MM a`; session B stages `b.txt` and
   runs `kit commit --message '#7 B: add b.txt'` with no paths. The commit succeeds, exit 0, and its stat is
   `a | 2 --`, `b.txt | 1 +`: both of A's changes are undone, silently. The test in `tests/commit.bats` ("an
   index left behind …") covers one commit only. Fix: refuse when the staged blob equals the path's blob at
   any commit on HEAD since the index's base and the disk differs (walk `git log -- path`), or implement #62
   so the shared index never falls behind. Confirmed.
4. High, bug. `kit/modules/bash/issue.sh:432-446`, reached through `kit/modules/bash/commit.sh:144`. A failed read
   of the pull request body replaces the body with `Closes #N`. `issue:link-in-body` does
   `body=$(forge:pr-body …)` unchecked; called from `commit:attach` under `||`, errexit is off, so a failed read
   is an empty body, no match, and `forge:pr-set-body` writes a blank line and `Closes #12` over the report.
   Scenario: `kit commit --message '#12 also fixes the other thing'` on `7-fix-the-thing`, while
   `gh pr view 3 --json body` fails once (rate limit, network) and `gh pr edit` works; the fake's call log shows
   `pr view 3 --json body …` (exit 1) followed by `pr edit 3 --body` with `Closes #12` alone. This is an instance
   of #69 with a destructive write behind it. Fix: `body=$(forge:pr-body "${pr}") || return 1` before any edit;
   separately, an empty body gets a leading blank line. Confirmed.
5. High, bug. `kit/modules/bash/commit.sh:73-74`. The default-branch refusal fails open when the default branch
   cannot be read. `default_branch=$(git:default-branch)` runs inside `branch=$(commit:branch) || return 1`, so
   its failure is ignored; `default_branch` is empty and no branch equals it. Scenario: `kit.yml` says
   `primary: upstream` and the clone has only `origin` (every fresh clone of such a repository). In a worktree
   with `main` checked out and the branch pattern off, `kit commit` prints `❌ primary 'upstream' in kit.yml is not
   a remote` twice, then `✅ committed … #1 straight to main`; `kit submit` prints the same error, then
   `✅ pushed main: 1 commit(s)`, and origin's `main` has moved. With the default `policy.branch` the commit is
   stopped by accident (the pattern refuses `main`), but `kit submit` with nothing staged judges nothing and
   pushes all the same. Fix: `default_branch=$(git:default-branch) || return 1` in `commit:branch`; the same in
   `facts:document` (`kit/modules/bash/facts.sh:70`). Confirmed.
6. Medium, bug. `kit/modules/bash/issue.sh:504-514`. A failure after the sweep leaves the human's desk work
   in the stash, unsaid. The desk is stashed before the worktree exists, and the stash is applied only after
   the worktree, the hydrate, the start commit, and the push; any of those failing ends `kit work` (several
   by errexit, with git's words only) with the desk emptied. Scenario: the desk has a modified `a` and an
   untracked `notes.txt`, and `worktrees/7-fix-the-thing` exists as a plain directory; `kit work 7` prints
   `Sweeping the uncommitted work…`, then `❌ … exists but is not a worktree …`, exit 2 (the header's "usage
   error", for a refusal). The desk is clean, `notes.txt` is gone, and the only trace is `stash@{0}: On main:
   kit work #7 → 7-fix-the-thing`; a second run finds a clean desk and never applies it. A push that fails
   for the network does the same. Fix: on any failure after `issue:sweep-push`, put the stash back on the
   desk, or at least name it and the command. Confirmed.
7. Medium, bug. `kit/modules/bash/issue.sh:387-392`. The sweep takes whatever is on top of the stash
   when `git stash push` saved nothing. `issue:sweep-push` trusts `stash@{0}` after a push that may
   have saved nothing (`No local changes to save` exits 0), and `git status --porcelain` is non-empty
   for things a stash does not take, such as a submodule with work in it. Scenario: the desk shows
   `m lib` (a dirty submodule) and holds the human's old `stash@{0}: the human: an old idea`;
   `kit work 7` prints `✅ Swept the work into …/worktrees/7-fix-the-thing`. The stash list is then
   empty and the new worktree shows `A  idea.txt`: an unrelated stash was applied to the issue's
   branch and dropped. Fix: compare `git rev-parse -q --verify refs/stash` before and after the push;
   no new entry means nothing to sweep. Confirmed.
8. Medium, bug. `kit/modules/bash/issue.sh:297,303`. `kit work 7` adopts any branch whose last path part starts
   with `7-`. `git ls-remote --heads origin "7-*"` matches by trailing path component, and `*` crosses `/`.
   Scenario: origin has `release/7-x` (or `dependabot/npm/7-zip-min-2.0`) and no `7-…` branch; `kit work 7`
   prints `Creating worktree …/worktrees/release/7-x` and `Adding the start commit`, and origin's `release/7-x`
   then ends in `#7 start: Fix the thing`. Fix: ask for `refs/heads/${key}-*` and keep only names that start
   with `${key}-` (no slash before it). Confirmed.
9. Medium, bug. `kit/modules/bash/commit.sh:459,462` and `kit/modules/bash/issue.sh:379`. With
   `commit.cleanup = strip`, git drops the `#N …` subject that kit judged. kit never passes
   `--cleanup`, so a user's `commit.cleanup=strip` (or `scissors`) treats the subject line `#7 …` as a
   comment. Scenario: `kit commit --message 'fix the thing' --message 'WIP do not merge'` prints
   `✅ committed affd664: #7 fix the thing`, but `git log` shows the subject `WIP do not merge`: a
   subject the policy never judged, which `kit policy` then refuses in CI. With one message the result
   is `Aborting commit due to empty commit message`, exit 1, with no kit words; the start commit of
   `kit work` aborts the same way. Fix: `git commit --cleanup=whitespace` (or `verbatim`) in both
   places, and report the subject read back from HEAD. Confirmed.
10. Medium, bug. `kit/modules/bash/issue.sh:324-328,510-514`. `kit work` commits on the desk when the desk has the
    issue's branch. `issue:worktree` reuses whichever checkout has the branch, the root checkout included.
    Scenario: the human's desk is on `7-fix-the-thing` with a modified `a`; `kit work 7` prints
    `Reusing worktree <desk>`, `Adding the start commit`, `Swept the work into <desk>`, and
    `Worktree ready: <desk>`. The desk's work was stashed and re-applied, the desk's branch has a new commit that
    was pushed, and the agent is sent to a checkout where `kit commit` refuses; this contradicts `.kit/1st.md`
    ("An agent never … changes the desk in place"). Fix: when the branch is checked out in the main root, refuse
    and say so before touching anything. Confirmed.
11. Medium, bug. `kit/modules/bash/commit.sh:506` and `kit/modules/bash/git.sh:325`. A bare `git push` goes where
    git's configuration says, which can be the default branch. After the first push, `kit submit` runs
    `git push --quiet` with no remote and no refspec. Scenario: `push.default=upstream` and a branch made from
    `origin/main` (git sets `origin/main` as its upstream); `kit submit --message '#7 work'` prints `✅ pushed
    7-fix-the-thing: 1 commit(s)`, and origin's `main` then ends in `#7 work`. `push.default=matching` pushes
    other local branches as well, and the bare `--force-with-lease` push of `kit rebase` follows the same
    configuration. Fix: always `git push <remote> "refs/heads/${branch}:refs/heads/${branch}"`. Confirmed.
12. Medium, inconsistency. `kit/modules/bash/commit.sh:241-259`. The pre-commit checks read the disk, and
    the commit takes the index. `commit:checks` passes `${PWD}/${path}` to `kit lint` and
    `kit generate --check`; without paths, git commits the staged blob, which may differ. Scenario: stage a
    version a linter refuses, then fix the file on disk without staging; `kit commit` lints the fixed file
    and commits the refused one, and `commit:same-tree` does not help because the index did not change. A
    run shows the committed blob differing from the checked file; the lint half is from the code alone. Fix:
    check a checkout of the tree being committed (`git checkout-index` into scratch), or refuse a staged
    path whose disk copy differs from the index. Confirmed from the code.
13. Medium, inconsistency. `kit/modules/bash/commit.sh:434-437`. `require_signing: true` does not make
    `kit commit` require a signature. Only `kit work` (a key is configured) and `kit release` (`--sign`) read the
    setting, while `guard:require signing-key` says "every commit here is signed". Scenario: with
    `require_signing: true` and `commit.gpgsign=false`, `kit commit` warns and commits, and `git log --format=%G?`
    prints `N`; the start commit (`kit/modules/bash/issue.sh:379`) is unsigned too. `docs/packages.md:270`
    documents only the two effects, so the name promises more than the kit does. Fix: with
    `require_signing: true`, pass `--gpg-sign` (and refuse when it fails); keep the warning otherwise. Confirmed.
14. Low, bug. `kit/modules/bash/git.sh:168-172,268`. `kit tidy` reports a removal that failed, settles its
    issues, then dies. The status of `git worktree remove` is ignored; the worktree is counted and its branch
    marked free. Scenario: a locked worktree (or one holding a submodule) whose change merged gives
    `removing …`, git's `cannot remove a locked working tree`, then `deleting 7-fix-the-thing`, `error: cannot
    delete branch … used by worktree`, exit 1. The sweep stops there, so later gone branches (`8-next`) stay,
    and there is no summary line. Fix: return 1 from `git:tidy-worktree` when the removal fails, and do not
    let one `git branch -D` end the sweep. Confirmed.
15. Low, bug. `kit/bin/kit-rebase:31`. `kit rebase <path>` reads a relative path from the repository root,
    not from where it was typed. `kit:enter` changes to the root, and the raw argument goes to `git -C`.
    Scenario: from `desk/docs`, `kit rebase ../worktrees/7-fix-the-thing` prints
    `❌ ../worktrees/7-fix-the-thing has no branch checked out`; from a subdirectory, a path can also land on
    a sibling repository and rebase that. `kit commit` and `kit policy --facts` use `context:relative`. Fix:
    resolve the operand against `kit_invoked_in`. Confirmed.
16. Low, bug. `kit/modules/bash/git.sh:318-326`. `kit rebase` leaves a rewritten, unpushed branch with git's
    words only, and calls every failure a conflict. Scenario: on a branch with no upstream the rebase
    happens, then the push fails with git's advice to run `git push --set-upstream` (a command the deny list
    forbids), exit 128 (the header promises 0, 1, and 2; see #70); a rebase that fails for a missing
    `origin/<default>` is reported as `conflicts with …`. Fix: check the push and say what state the branch
    is in; tell a conflict from another failure. Confirmed.
17. Low, inconsistency. `kit/modules/bash/issue.sh:66`. `kit assign` exits 0 when the assignment failed. The
    header says "1 when one could not be"; `issue:assign` only warns, while `issue:assign-to` returns 1. Scenario:
    a failed assignment prints `⚠️ #7 could not be assigned to ada; assign it by hand.`, exit 0. Fix: count the
    failure and return it; `kit work` can keep treating it as a warning. Confirmed.
18. Low, missing. `kit/modules/bash/issue.sh:496-498`. `kit work` starts a closed issue without a word. No
    state is read, and the result is a branch and a draft change that `kit linked` then refuses (a closed
    issue does not count). Scenario: with issue 7 CLOSED, `kit work 7` proceeds. Fix: refuse a closed issue,
    or warn and name the rule of `kit linked`. Confirmed.
19. Low, bug. `kit/modules/bash/commit.sh:404`. A staged type change is invisible to the checks:
    `--diff-filter=ACMRD` leaves out `T`. Scenario: a file is replaced by a symbolic link and staged (`T  a`).
    Alone, `kit commit` says `nothing is staged`; beside other staged paths it is committed without the guard,
    the linters, or `commit:behind` seeing it (the second half follows from `git commit` taking the whole
    index). Fix: add `T` to the filter. Confirmed.
20. Low, inconsistency. `kit/modules/bash/policy/patterns.sh:12-13,27-29` and `kit/modules/bash/policy.sh:65`. The
    two engines disagree on the same facts. Scenario: a title of 100 two-byte characters is allowed by OPA and by
    patterns under a UTF-8 locale, and refused by patterns under `LC_ALL=C`, where `{1,120}` counts bytes; kit
    sets no locale, so a desk and a container can differ. A facts document whose `policy` lacks a key makes
    patterns match against the literal `null` (`branch '7-x' does not match policy.branch: null`), while the Rego
    rule is undefined and allows. Both ran with opa 1.21.4. Fix: read each pattern with `// ""`; match under a
    fixed UTF-8 locale, or say the limit is bytes. Confirmed.
21. Low, missing. `kit/modules/bash/issue.sh:488`. The primary forge issue gets no `Closes` line, so a
    reused branch is not linked. The primary is left to "the branch the forge creates". Scenario: when the
    branch was found on origin instead (made by hand, or adopted as in finding 8), nothing links the
    change to the issue, and `kit linked` fails in CI although `.kit/workflow.md` says `kit work` makes
    the link. Fix: always add `Closes #N` for the primary; `issue:link-in-body` is idempotent. Plausible;
    a real forge is needed to show the missing link.
22. Low, bug. `kit/modules/bash/issue.sh:247-273`. Settling a closed change unassigns the author though
    another open change closes the same issue; `issue:settle` looks at the closed change alone.
    Scenario: closing a superseded change (CI's Settle job, or `kit tidy`) comments "this is backlog
    again" and takes the author off an issue their newer change is working on. Fix: skip an issue that
    an open change of the same author closes. Plausible.
23. Low, bug. `kit/modules/bash/issue.sh:311-312,421`. `kit work` can make a branch or a title the default policy
    refuses. Scenario: a non-forge issue whose title has no ASCII letter or digit gives the branch `ab4521-`,
    which `policy.branch` refuses, so every `kit commit` in the new worktree is refused
    (`branch=[ab4521-] … REFUSED`). The change's title is the issue's title, and one over 120 characters fails
    `change_title` in CI. Fix: fall back to the bare key or `-work` for an empty slug; shorten the title to the
    policy's limit. Confirmed for the branch; plausible for the title.
24. Low, inconsistency. `kit/modules/bash/facts.sh:75-78`. `kit policy` counts commits from `origin/<default>`
    while the default branch's name comes from the primary. Scenario: in a fork (`primary: upstream`), the commits
    judged are those past the fork's own, possibly stale, `main`. This is the same family as #68, which lists only
    the pushes; `kit/modules/bash/issue.sh:376,409` do the same. Fix: none given. Plausible.
25. Low, bug. `kit/modules/bash/git.sh:36-38`. The default branch is the literal `main` when `<primary>/HEAD`
    is not set. Scenario: the default-branch refusals of `kit commit` and `kit rebase` then do not apply to a
    `master` or `trunk`, and `kit policy` reports `the fork from origin's default branch is not here`. git
    2.48 and later set the ref on fetch (it could not be triggered on git 2.56); an older git, or a
    `git remote add` without a fetch of HEAD, can leave it unset. Fix: fall back to
    `git ls-remote --symref <primary> HEAD`, and fail rather than guess. Plausible.
26. Low, bug. `kit/modules/bash/generate.sh:69,318`, reached through `kit commit`. A path that starts with `-`
    is read by grep as an option. Scenario: `kit commit` on `-dash.txt` prints
    `grep: unknown --directories option`, and `kit generate --check` passes, so a stale region in such a file is
    not caught. Paths with spaces, quotes, `*`, and non-ASCII names otherwise commit correctly, from a
    subdirectory too. Fix: none given; the module is outside this review's scope. Confirmed.
27. Low, bug. `kit/bin/kit-gate:34`. The gate is detected with a bare `just --summary`, outside the
    toolchain that will run it. Scenario: on a desk where `just` is only in the project toolchain,
    `kit gate` fails with `the root justfile could not be read: … command not found` instead of running
    `just ci` through `toolchain:run`. Fix: none given. Plausible.

### Filed issues (Workflow)

- #62 (the shared index brought up to a private-index commit): not implemented; see finding 3.
- #61: implemented by content (`commit:behind`, `commit:moved`, `commit:same-tree`), with the hole in finding 3.
- #59: delivered (`commit:named` uses `--with-tree=HEAD`; covered by tests).
- #67: unchanged. `kit linked` and `kit policy` read `linked_exempt`, `policy`, and `gate` from the checkout.
- #68: unchanged, and wider than filed: `kit/modules/bash/facts.sh:75-78` and the bare pushes in finding 11.
- #69: unchanged (`kit/modules/bash/facts.sh:84-88`), and finding 4 is a destructive case of it; `issue:assign`
  (`kit/modules/bash/issue.sh:62`) and the closing-issues read of `issue:settle`
  (`kit/modules/bash/issue.sh:250-253`) are others.
- #70: unchanged; exit 128 (finding 16) and exit 2 for a refusal (finding 6) are further instances.
- #64 and #76: not implemented (no `policy/conventional` package; `kit tidy` has no `--wait`).

### Untested (Workflow)

- `kit rebase` when origin's branch has commits the local one lacks, and on a branch with no upstream.
- `kit tidy` with unpushed commits, with a removal that fails, and with a closed, unmerged change.
- `kit commit` from a shared index two commits behind, with `commit.cleanup=strip`, and with a type change.
- The attach of `kit commit` when the body cannot be read.
- `kit work` with a failure after the sweep, with the desk on the issue's branch, with a
  closed issue, and with a `<dir>/<key>-…` branch.
- `kit linked` has one test of its own (no pull request found); the closed-issue, other-repository, and
  body-reference paths are untested outside forge-conformance.

### Not reviewed (Workflow)

- `kit/bin/kit-ci`, `kit/bin/kit-root`, `kit/modules/bash/super.sh`, and `kit/modules/bash/release.sh` were read
  only, and nothing was run against them. `kit release` was run for `--lower` alone (it works); signing, a failed
  push, and the fail-open read of `releases:on-origin` were not.
- The GitLab and Azure DevOps providers behind these verbs; `kit settle` and `kit close` end to
  end; `kit work --title` and `--pr` beyond reading.
- Two sessions racing inside one `kit commit` (the window between `commit:moved` and
  `git commit`) was reasoned about, not reproduced.
- Linux: everything ran on macOS (bash 5.3, git 2.56); no GNU-versus-BSD difference was found by reading.
- The house Rego in `.kit/packages/house`, and `tests/*.bats` beyond their test names and setups.

## Delivery

Reviewed the delivery and content core of kit's bash (the `1st`, `materialize`, `place`, `manifest`, `package`,
`packages`, `sources`, `resolve`, `settings`, `yaml`, `generate`, `region`, `template`, `render`, `new`, `file`,
`context`, `lint`, `facts`, `install`, and `uninstall` modules under `kit/modules/bash/`, the `template`, `merge`, and
`version` modules under `kit/modules/jq/`, and the verbs `kit`, `kit hydrate`, `kit install`, `kit update`,
`kit uninstall`, `kit packages`, `kit generate`, `kit region`, `kit template`, `kit new`, `kit kinds`, `kit context`,
`kit lint`, `kit check`, `kit facts`, `kit gather`, `kit root`, and `kit version`); every file in scope was read in
full, and 27 of the 30 findings were reproduced in throwaway repositories with HOME, `KIT_HOME`, and the caches
isolated as `tests/helper.bash` isolates them, on macOS with bash 5.3.20, jq 1.8.2 and 1.7.1-apple, and yq 4.54.1.

1. High, bug. `kit/modules/bash/region.sh:207-238`. `region:remove` never checks that the region closes, so a
   missing `kit:end` deletes the rest of the file. `region:strip` sets `skip = 1` at the directive and clears
   it only at the end line; `region:remove` calls it without `region:has`, whose status 2 exists exactly
   "because a write would drop everything after the directive". Scenario: `.gitignore` holds `mine-before`, the
   directive, two kit lines, `mine-after-1`, and `mine-after-2`, and the `# kit:end .gitignore` line was lost
   (a hand edit, a merge). After `kit uninstall` (or `kit packages remove <pkg>`) the file is `mine-before` and
   a blank line, both `mine-after` lines are gone, and the verb prints `✅ kit is uninstalled`, exit 0.
   `uninstall:path` has no edit check for a region kind either. Fix: `region:remove` calls `region:has` first
   and returns 1 on status 2, as `region:write` does. Confirmed.
2. High, bug. `kit/modules/bash/region.sh:161-164`. `region:write` looks for the region under the symlink
   target's absolute name, so a linked file gets one more region on every write. `path=$(release:target "$1")`
   is the absolute real path, and `region:has "${path}"` derives its markers from that path (`region:name`
   returns it whole), so it looks for `# kit:generate /abs/…/shared/gitignore`, never finds it, and
   `region:filled` appends; the comment of `release:target` names a linked file as the supported case.
   Scenario: `.gitignore -> shared/gitignore` (or `.claude/CLAUDE.md -> ../AGENTS.md`), then `kit update`. Each
   run prints `.gitignore: the region regenerated` and appends a full copy of the region, so after two runs the
   file holds three, and `kit check` fails with `the region '.gitignore' is opened twice`. Fix: test with the
   destination's markers against the target's content; `region:has` takes the marker name separately from the
   file it reads (or is passed `begin` and `end`). Confirmed.
3. High, bug. `kit/modules/bash/materialize.sh:182-187,335-336`. A foreign file under `.kit/` is left alone for
   exactly one command, then overwritten. `materialize:all` protects "a path the manifest never recorded that
   holds a different file", but `materialize:hydrate` then calls `manifest:write repo`, which records every
   delivered path whether or not it was placed; the path then has a record, `materialize:hydrated` fails (a
   writable regular file), and the next verb's full run replaces it. Scenario: a repository with its own tracked
   `.kit/rules/bash.md` (or any file at a path a package delivers) and a declared `kit.yml`. `kit hydrate` prints
   `⚠️ .kit/rules/bash.md: a different file is there already; move it aside…`, the content is `MY OWN NOTES`, and
   `.kit/manifest` already holds a line for it. The next command, `kit kinds bash`, prints `⚠️ .kit/rules/bash.md
   was changed in place, or came from another release; replaced`, and the file is the kit's. `kit update` has the
   same order problem: its comment at `kit/bin/kit-update:199` says the path "is still unrecorded, and stays as it
   is", and line 201 records it. Fix: `manifest:text` leaves out a path that `materialize:all` refused (return the
   refused set, or record only paths whose destination `materialize:ok`). Confirmed.
4. High, bug. `kit/modules/bash/settings.sh:168-176`. The line-oriented rewrite takes only two-space-indented
   lines as a block, so a valid `kit.yml` with an unindented list is corrupted. `skip && /^  /` is the only
   continuation the awk knows, while yq reads an unindented `- kit` under `packages:` correctly, so the file
   works until the first write. Scenario: `kit.yml` is the four lines `version: unreleased`, `packages:`,
   `- kit`, and `answers: …` (zero-indent sequences are what many YAML formatters emit).
   `kit packages add style/json` makes it `packages:`, then `- kit` and `- style/json` indented by two spaces,
   then the unindented `- kit`, then `answers:`. yq then refuses it (`did not find expected key`), and every verb
   says `run: kit install (kit.yml declares no kit version…)`. `uninstall:declaration` uses the same awk and
   leaves the stray `- kit` behind. Fix: a block is the key line plus every following line that is indented or
   starts with a dash and a space; or delete the three keys with output in the style of `yq -i 'del(.version,
   .packages, .excludes)'` and re-insert them. Confirmed.
5. High, bug. `kit/modules/bash/resolve.sh:192-214`, with `kit/bin/kit-update:52`,
   `kit/modules/bash/package.sh:108,209`, and `kit/bin/kit-install:44`. A dependency cycle prints an error, and
   the callers carry on with the partial set. `resolve:active` prints packages as it places them and returns 1
   at the cycle; every caller reads it through `readarray … < <(…)`, where the status is lost. Scenario: the
   active package `c` gains a dependency that closes a cycle (`c → a → b → a`). `kit update` prints `❌ the
   packages' depends and includes lines form a cycle among: a b c kit style/json`, then `── kit update …
   packages: kit style/json ──`, `c.txt: retired … removed`, and `✅ kit unreleased is installed`, exit 0, and
   `kit.yml` does not list `c` afterwards. With `a ↔ b`, `kit packages add a` prints the cycle error twice,
   then `✅ packages: kit style/json`, exit 0, with `a` not added. Fix: `names=$(resolve:active …) || exit 1`
   before `readarray`, as `package:named` already does for `resolve:expand`; and print nothing from
   `resolve:active` until the sort completes. Confirmed.
6. High, bug. `kit/modules/jq/template.jq:102`. `indices` returns byte offsets in jq 1.7.x, the version the
   kit accepts, so template matching fails on any non-ASCII text. `match` mixes `$text | indices($p.literal)`
   (bytes in jq 1.7.1 and earlier, code points from 1.8) with `$text[$from:$at]` (code points);
   `kit/modules/bash/welcome.sh:292` and `kit/modules/bash/syntax.sh:329` accept "jq 1.7 or later", and macOS
   ships 1.7.1 as `/usr/bin/jq`. Scenario: a desk with the system jq, `kit install agents/generic`, then
   `kit check`. With jq 1.8.2 the answer is `✅ … every delivered file matches`; with `/usr/bin/jq` first on
   PATH the same repository gives `❌ AGENTS.md does not have the shape of the agents/generic package's host
   template; it leaves AGENTS.md at line 4` (the region's own text has `⚠️`). `kit install agents/generic`
   into a repository whose `AGENTS.md` opens `# Guía del agente` is refused with `does not open as the … host
   template has it`, and installs with 1.8.2; the shape check of `kit lint` goes through the same `match`.
   Fix: find the literal without `indices` (`$text[$from:] | index(…)` has the same defect; use `test` or
   `match` with an escaped literal, or scan on code points in the manner of `ltrimstr`), or require jq 1.8 in
   the three places that state the minimum. Confirmed.
7. High, bug. `kit/modules/bash/context.sh:265-277,336-337`. One kind with an invalid `content:` regex
   makes every path kindless, and `kit lint` passes. The compiled classifier is one awk program; a bad
   dynamic regex is a fatal awk error, awk prints nothing, and `context:rules` reads "no kinds" as "no
   rules". This is a concrete instance of #69 that its list does not have (the kinds file parses; the
   classifier dies). Scenario: any active package, a local one or a source's, defines a kind with
   `content: ["^#!.*(odd"]`. Before, `kit lint bad.sh` prints `❌ bad.sh does not have the shape of the
   bash template`, exit 1; after `kit new kind odd --content='^#!.*(odd'` and `kit packages add local`, it
   prints awk's `syntax error in regular expression`, then `✅ every governed path passes its
   rule's linter`, exit 0. `kit lint --all-files` does the same, `kit context bad.sh` prints nothing with
   exit 0, and `kit check` passes. Fix: `context:classify` checks awk's status (`PIPESTATUS`) and returns
   1; `context:rules` and `kit-lint` propagate it; `context:load` (or `kit new kind`) compiles each pattern
   once and names the kind that fails. Confirmed.
8. Medium, bug. `kit/modules/bash/materialize.sh:274-297`. The fast path cannot see a file added to an active
   package, so a pulled rule is silently not in force. `materialize:hydrated` checks only the paths
   `.kit/.gitignore` already names and the mtime of `kit.yml`; a file added under
   `.kit/packages/<active>/files/repo/.kit/` (tracked, so it arrives by `git pull`) changes neither.
   Scenario: with a local package active, add `.kit/packages/local/files/repo/.kit/rules/proto.md` governing
   the kind `proto`. `kit context a.proto` prints nothing, `.kit/rules/proto.md` does not exist, and
   `kit lint a.proto` prints `✅ every governed path passes`. After `kit hydrate`
   (`→ .kit/rules/proto.md: placed`) the rule is named. A fresh clone or CI gets the rule; the desk that
   pulled does not. Fix: the fast path also compares the newest mtime under `.kit/packages/` (one
   `find -newer .kit/.gitignore -print -quit`) with `.kit/.gitignore`. Confirmed.
9. Medium, bug. `kit/modules/bash/place.sh:240-247`. `place:retire` reads its candidates from `.kit/manifest`
   alone, so in a fresh clone or worktree a retired file is never removed. The manifest is ignored by git: in a
   clone or a new worktree it is absent, and after any hydrating verb it has been rewritten from the running
   release (`kit/modules/bash/materialize.sh:336`), so the manifest minus the delivered paths is empty.
   `recorded-digest` in `kit/bin/kit-update` falls back to the base release for digests; retire has no such
   fallback for `repo`. Scenario: two release copies, with `OLD.txt` delivered by 0.1.0 only. The same checkout
   with its manifest prints `OLD.txt: retired in 0.2.0, removed`. In a `git clone` of it,
   `KIT_BASE=old new/kit update` prints `✅ kit 0.2.0 is installed`, `OLD.txt` stays, and `kit check` prints
   `✅ every delivered file matches`. The original checkout gives the same result when any verb ran under 0.2.0
   before `kit update` (`touch kit.yml; kit kinds`). Fix: the candidates are the union of `manifest:paths` and the
   base release's delivered paths, for `repo` as well, each judged by digest against the base's file. Confirmed.
10. Medium, bug. `kit/modules/bash/region.sh:72-82,183-191`. `region:has` accepts any file that has both marker
    lines somewhere, so `region:filled` drops the repository's lines when the markers are out of order or doubled.
    Scenario: a file of the lines `top`, `# kit:end .gitignore`, `middle`, `# kit:generate .gitignore`, `old`,
    `bottom-1`, and `bottom-2`; `kit update` prints `the region regenerated`, `bottom-1` and `bottom-2` are gone,
    and the region has no end line. A file with a second directive after a complete region loses everything after
    the second directive the same way. Both shapes are what a bad merge leaves. Fix: `region:has` requires exactly
    one directive, followed by exactly one end line (one awk pass returning 2 otherwise). Confirmed.
11. Medium, bug. `kit/modules/bash/region.sh:76-77`. The marker is passed to grep as a pattern with no `--`
    before it, and for `.sql`, `.lua`, and `.hs` files the marker itself starts with `--` and a space.
    Scenario: a local package with `regions/repo/schema.sql`; `kit update` prints `grep: unrecognized
    option '-- kit:generate schema.sql'` and appends a second region, and `kit check` reports both
    `schema.sql has no kit region; run: kit update` and `the region 'schema.sql' is opened twice`. The
    result is the same with the BSD grep and the grep on PATH. Fix: `grep … -e "${begin}" -- "${path}"` (or
    the awk test that `region:read` already uses). Confirmed.
12. Medium, bug. `kit/modules/bash/package.sh:109,162,239-243`. `package:select` calls remove and add under `||`,
    which turns errexit off inside them, and neither checks `render:all`. Scenario: a package `tp` whose template
    needs an unset answer. `kit packages add tp` prints the answer error, exit 1, with `kit.yml` unchanged.
    `kit packages select kit tp` prints the same error, then `── kit packages add tp ──` and
    `✅ packages: kit tp`, exit 0. `kit.yml` then declares `tp`, none of its files is placed, and every later verb
    (`kit check`, anything that hydrates) fails on the same error until `kit.yml` is edited by hand. Fix:
    `render:all … || return 1` in `package:add`, `package:remove`, and `uninstall:run`. Confirmed.
13. Medium, bug. `kit/modules/bash/place.sh:216-218`, through `kit/modules/bash/package.sh:184`.
    `kit packages remove` deletes a copied file the repository edited, and `kit uninstall` leaves the same file
    alone. `uninstall:path` compares before it calls `place:remove`; `package:remove` does not, though the header
    of `kit/modules/bash/uninstall.sh` says the two behave alike ("A copy the repository edited is the one path
    left alone"). Scenario: `.github/workflows/kit-ci.yml` with a hand-added line. `kit uninstall --dry-run`
    prints `⚠️ … edited since the kit placed it; it is left as it is`, while `kit packages remove github/ci`
    prints `→ removing .github/workflows/kit-ci.yml`, and the file is gone. Fix: move the `cmp` guard from
    `uninstall:path` into `place:remove` (or into a shared `place:take-out`). Confirmed.
14. Medium, missing. `kit/bin/kit-update:50`, `kit/modules/bash/package.sh:15-22`, and
    `kit/modules/bash/render.sh:99`. A release that drops or renames a package leaves the repository with no
    verb that works. Scenario: a new release without `style/json`. `kit update` prints `'style/json' is not a
    package in reach`, exit 2 (the header documents 0 and 1 only, which adds to #70);
    `kit packages remove style/json` prints `'style/json' is not one of the choices`, exit 2;
    `kit uninstall --dry-run` gives the same refusal. Only a hand edit of `kit.yml` gets out. Fix: `kit update`
    drops a recorded package the release does not have, says so, and lets retire take its paths;
    `package:remove` accepts a name that is recorded though not in reach. Confirmed.
15. Medium, bug. `kit/modules/bash/new.sh:231-233`. `kit new kind` writes `--path` and `--content` in YAML double
    quotes, where a regex backslash is an escape. Scenario: `kit new kind proto --content='^// proto\.v[0-9]+'`
    writes `- "^// proto\.v[0-9]+"`; after `kit packages add local`, the verbs `kit kinds`, `kit lint`, and
    `kit context` all fail with `found unknown escape character` and `the kinds in it cannot be read`. `\b`, `\t`,
    and `\n` parse and silently become control characters. `kit/modules/bash/yaml.sh:8-9` says a pattern is
    written single-quoted for this reason. Fix: write the pattern in single quotes with each single quote doubled;
    the double-quote refusal at `kit/modules/bash/new.sh:188` can then go. Confirmed.
16. Medium, bug. `kit/modules/bash/sources.sh:89,119-125`. `--strip-components 1` removes `packages/` from a
    tarball laid out as the header describes, and the empty source is cached without a word. The header and
    `docs/packages.md:304` say "Its packages are under packages/ in the tarball"; only a tarball with one wrapping
    directory works (the forge's tag tarballs, and the test fixture's `top/`). Scenario: a tarball of
    `packages/guides/…` with its digest pinned; `kit packages add guides` fetches it, then prints
    `'guides' is not one of the choices`, exit 2. The cache holds `<digest>/guides/package.yml`, `sources:roots`
    skips it silently, and it is never fetched again. Fix: after unpacking, refuse a source with neither
    `packages/` nor `kit/packages/` (`the source team holds no packages/`) and do not cache it; or strip only when
    the tarball has a single top-level directory. Confirmed.
17. Low, bug. `kit/bin/kit-template:35`. Under pipefail, grep's status 1 on a file with no ASCII hole ends the
    verb silently. Scenario: `kit template fix a.tpl b.tpl`, where `a.tpl` has no hole, prints nothing and exits 1
    with `b.tpl` untouched; in the other order `b.tpl` is rewritten, then the verb exits 1 with no line for
    `a.tpl`. Fix: `count=$(grep --count …)` with `|| true`, or `{ grep … || true; } | wc -l`. Confirmed.
18. Low, bug. `kit/modules/bash/context.sh:163-166`. `context:glob-ere` loops forever on a `[` that never closes,
    and keeps `[!x]` as written. `expr` prints 0, so `i += -1` is followed by `i += 1`. `[!b]` means "not b" to
    `context:glob-matches` and "! or b" to the awk classifier, and an extglob such as `*.@(sh|bash)` matches in
    bash and is escaped to a literal in the ERE. Scenario: `context:glob-ere 'notes[1.md'` runs until a 5-second
    alarm kills it; `context:glob-matches adc 'a[!b]c'` is true while `[[ adc =~ ^a[!b]c$ ]]` is false. A kind
    with `paths: ["notes[1.md"]` hangs every verb that classifies. Fix: treat an unclosed `[` as a literal
    (`\[`); translate a leading `!` to `^`. Confirmed.
19. Low, bug. `kit/modules/jq/template.jq:66-69` against `kit/modules/jq/template.jq:24-25`. `bind_parts` (used by
    `check`) drops the newline of a bound hole that stands alone on a line, while `bind` (used by `unapply` and
    `leaves`) keeps it. Scenario: for the template `# head\n«name»\nrest\n`, `render({name:"foo.sh"})` makes
    `# head\nfoo.sh\nrest\n`, `parts | bind_parts | match` gives `{"ok":false,"line":2}`, and `unapply` gives `{}`
    (a match). The same holds for a dotted settings hole, so `kit new file` makes a file that `kit lint` refuses
    and the host test of `kit check` accepts. No shipped template has such a line. Fix: in `bind_parts`, a bound
    hole of kind `lines` becomes `$bound[$h] + "\n"`. Confirmed; latent.
20. Low, bug. `kit/modules/bash/lint.sh:28`. `lint:pinned` reads `uses:` by regex, so valid YAML spellings
    pass unpinned. Scenario: of five unpinned steps, `- uses: actions/checkout@v4` and
    `uses: "actions/cache@v4"` are reported; `- {uses: actions/checkout@v4}`,
    `- "uses": actions/setup-node@main`, and `uses:` with the value on the next line are not, and the exit
    status reflects only the two. Fix: read `.jobs[].steps[].uses` (and the `jobs.*.uses` of a reusable
    workflow) through `yaml:yq`, keeping the regex only for the line number and comment. Confirmed.
21. Low, bug. `kit/modules/bash/package.sh:179-187`. Removing a package that overrides another's path leaves the
    overrider's file in place: `place:beyond` lists only paths no remaining package delivers. Scenario: `p2`
    overrides the `a.txt` of `p1`; `kit packages remove p2` prints `✅ packages: kit p1`, `a.txt` says `from p2`,
    and `kit check` prints `❌ a.txt differs from the pinned release; run: kit update`. Fix: after removal,
    re-place every path whose winning source changed between `active` and `remaining`. Confirmed.
22. Low, bug. `kit/modules/bash/render.sh:77`. A rendered template is written with `printf >`, so it loses the
    source's mode; `docs/packages.md:111` says a template is "placed like a `files/` file", and `file:copy` keeps
    the mode. Scenario: `files/repo/f.sh` and `templates/repo/t.sh`, both 755 in the package, arrive as
    `-rwxr-xr-x f.sh` and `-rw-r--r-- t.sh`. Fix: the equivalent of `chmod --reference` after writing
    (`[[ -x "${file}" ]] && chmod +x "${out}/${rel}"`). Confirmed.
23. Low, bug. `kit/modules/bash/yaml.sh:141-170`. A cached result is keyed by the read's name and the path and
    validated by the file's text only; the expression, the options, and the release are not part of it. Scenario:
    `yaml:yq c.yml "value a" ".a"` prints `1`, and `yaml:yq c.yml "value a" ".b"` prints `1` again. After a kit
    upgrade that changes a reader's expression (or yq's behavior), an unchanged `kit.yml` or rule replays the old
    release's answer from `~/.cache/kit/yaml/` until its text changes or seven days pass; the header's "never read
    stale" holds for file edits only. Fix: add a digest of the expression, the options, and `release:version` (or
    `kit_root`) to the entry's name or header. Confirmed; latent.
24. Low, bug. `kit/modules/bash/uninstall.sh:152-166`. A `kit.yml` that held only the declaration, with no
    modeline, is left as a one-byte file: `grep --invert-match` over an empty here-string sees one empty line and
    "finds something else". Scenario: `kit.yml` holds `version` and `packages` only; `kit uninstall` prints
    `kit.yml: version and packages dropped`, and the file is 1 byte (`\n`), though the header promises it is
    removed "when nothing else is in it". Fix: test `[[ -z "$(grep --invert-match '^#
    yaml-language-server:' <<<"${kept}" | awk NF)" ]]`. Confirmed.
25. Low, bug. `kit/modules/bash/generate.sh:91-95`. A named path that does not exist is dropped by discovery,
    so a typo passes. Scenario: `kit generate nosuchfile.md` and `kit generate --check typo/AGENTS.md` print
    `✅ generate: no kit:generate directive to expand`, exit 0; the "no such file" answer of `generate:expand`
    (status 2) is unreachable for a named path. Fix: `generate:discover` passes a named path that is not a
    file through (or `generate:run` refuses it with 2); the pre-commit hook's staged list uses
    `--diff-filter` to leave deletions out. Confirmed.
26. Low, bug. `kit/bin/kit:28-31`. The dispatcher sources `kit/modules/bash/log.sh` and calls `log:now` before
    anything checks the bash version. Scenario: with stock macOS bash 3.2 first on PATH, `kit version` prints
    `log.sh: line 80: printf: '(': invalid format character` and an arithmetic syntax error, then
    `kit needs bash 4.4 or later`, then the two `log.sh` errors again. Fix: the `BASH_VERSINFO` test from
    `kit/modules/bash/1st.sh:21-25` at the top of `kit/bin/kit`. Confirmed.
27. Low, bug. `kit/modules/bash/region.sh:230-238`. `region:strip` rewrites blank lines outside the region:
    `held`, set by leading blank lines, survives until after the first text line. Scenario: a file opens with two
    blank lines and has its markers intact; after `kit uninstall` the two leading blank lines are gone, a third
    blank line appears between `mine-before` and `mine-2`, and one trailing blank line is dropped. Fix: clear
    `held` when nothing is printed yet; drop only the single blank line directly before the directive. Confirmed.
28. Low, inconsistency. `kit/bin/kit-hydrate:11` and `kit/modules/bash/materialize.sh:324-332`. The code does not
    keep two statements about hydration. Scenario:
    - The header of `kit/bin/kit-hydrate` says it "writes … `.kit/manifest` when it is missing";
      `materialize:hydrate` rewrites it on every full run (line 336; its own comment says "It rewrites
      .kit/manifest"). This rewrite is what feeds findings 3 and 9.
    - The header of `materialize:hydrate` says "It warns when this release is not the declared one"; the
      warning is inside the full run only, so on the fast path a 0.2.0 kit runs every verb over the copies of
      0.1.0 with no word (`kit kinds bash` under the new release printed nothing about versions).

    Fix: correct the header, and warn on the fast path too (one `settings:version` comparison,
    already cached). Confirmed from the code and a run.
29. Low, inconsistency. Several files, each named below. Headers and messages disagree with the code. Scenario:
    - `kit/modules/bash/file.sh:4` says "Every file the kit writes in a repository … goes through here". That
      is not so for `kit/bin/kit-new:194` (`template:render … > "${path}"`, which leaves an empty file that
      `kit new` then refuses to overwrite if rendering fails),
      `kit/modules/bash/new.sh:220-239,279,332-336,367-371,405,482-489,511-514`, `kit/bin/kit-template:38`
      (`sed -i`), `kit/modules/bash/1st.sh:60`, and `kit/modules/bash/place.sh:92`.
    - `kit/modules/bash/new.sh:190` (`--${key%s}`): the refusal for `--kinds=a"b` names `--kind=`, and for
      `--embeds` names `--embed=`; neither flag exists.
    - `kit/modules/bash/new.sh:413-418`: when the owner file has no frontmatter the awk inserts nothing, and
      line 418 says "listed the … variant under templates:" all the same; a new variant is inserted first, so
      it is tried before the existing ones, which the header ("in the order they are tried") does not say.
    - `kit/modules/bash/region.sh` (exact-line markers) against `kit/modules/bash/generate.sh:54-55` (any
      punctuation around the marker): `<!--kit:generate AGENTS.md-->` or an indented directive is a region to
      `kit generate` and "no region" to `region:has`, so `kit update` appends a second one.
    - The path rule of `AGENTS.md`: every module in scope ends its header with `# see: docs/packages.md` (or
      `docs/generate.md`, `docs/context.md`, `docs/templates.md`), a path that is not under `kit/` and not in
      the tarball; `kit/modules/bash/new.sh:432,479,486` name `kit/modules/…` and `tests/jq/` from the
      repository root (deliberately, for the kit's own source, but against the rule as written).

    Fix: none given. Confirmed from the code.
30. Low, bug. `kit/modules/bash/context.sh:244-260`. The compiled classifier is shared by every worktree, and each
    compile deletes all others. Two worktrees whose kinds differ (the kit's own workflow: one edits a kind in its
    worktree) each run `find … ! -name kinds-<key>.awk -exec rm`; the key is the kinds' text alone, so a release
    that changes `context:compile` reuses the old program. Scenario: a process that was handed the other path
    reaches `awk -f` after the file is gone, and the failure reads as "no kinds" (the fail-open of finding 7); the
    window is between the return of `context:matcher` and awk's open. Fix: do not delete on compile (prune by age,
    as `kit/modules/bash/yaml.sh` does), and add `release:version` to the key. Plausible; read only.

### Filed issues (Delivery)

- #66: unchanged in the code. `materialize:all` (`kit/modules/bash/materialize.sh:161-175`) removes a
  manifest path with `rm -f` through a symlinked directory under `.kit/`; `manifest:paths` filters `..` and
  `/` only. In the same class, `sources:fetch` passes the `url` of `kit.yml` to curl with no `--`, so
  `url: -K<file>` is read as an option, before the digest check.
- #69: unchanged at `kit/modules/bash/facts.sh:84-92` (`2>/dev/null` and `|| true` around the forge
  reads). Findings 7, 8, and 30 are further instances.
- #70: `kit update` exiting 2 (finding 14) belongs with it.
- #65 and #71: not examined.

### Not reviewed (Delivery)

- Home-base delivery (`~/.kit/manifest`, the `grep -F "  ${path}"` substring match of `manifest:write home`, two
  repositories sharing the manifest, `kit uninstall` in one removing what another uses): no shipped package
  delivers to `home`, so nothing was run, and the code path is untested in practice.
- Linux and GNU runs: everything was run on macOS only.
  `cmp --silent` and `readlink -f` on macOS 12.x were not checked.
- `kit facts`, `kit gather`, `kit root`, `kit version`, and the tree and text reads of
  `kit/modules/bash/facts.sh`: read, not exercised.
- `guard:arguments`, `guard:jq`, and the hand-off to the TUI in `kit/modules/bash/1st.sh` (they lean on
  `kit/modules/bash/syntax.sh`, outside this scope).
- Concurrency of two kit commands in one checkout (`file:replace` on
  `.kit/manifest` and `kit.yml`), beyond reading.
- The three-way merge of `kit update` and `kit/modules/jq/merge.jq` under conflicting types (a scalar
  where the release has an array): read only; the failure is reported as "not JSON jq can read", which
  is the wrong message but leaves the file alone.
- A bats run of the suites in scope; the kit was run directly instead.

## Flow

Reviewed `kit flow` and the forge and tracker seams (`kit/modules/bash/flow.sh`, the providers under
`kit/modules/bash/forge/` and `kit/modules/bash/tracker/`, the `forge`, `backlog`, and `underway` modules
under `kit/modules/jq/`, and kittui's decode structs); findings were checked with the tests' fake tool
standing in for `gh` and `glab`, by feeding fixture JSON straight into the jq modules (a "jq run" below), and
by reading, and no bats file and no shellcheck was run.

1. High, bug. `kit/modules/jq/forge.jq:61`. `checks` keeps one run per check name, so a failed check is hidden by
   a same-named check that started later. Scenario: the workflows `CI` and `Docs` both have a job `test` (or one
   workflow runs on `push` and `pull_request`); `CI/test` fails at 10:00:00 and `Docs/test` passes at 10:00:02,
   and the pull request is `ready`, `✅ 1 / 1`. A jq run gives
   `{"state":"ready","checks":{"state":"SUCCESS","passed":1,"total":1,"failing":""}}`, and a live read-only
   GraphQL read of cli/cli shows that `CheckRun.name` is the bare job name and that several runs of one name sit
   on one head commit. Fix: group by check suite or workflow plus name (ask `checkSuite { workflowRun { workflow {
   name } event } }`), and treat only a later attempt of the same suite as a rerun. Confirmed.
2. High, bug. `kit/modules/bash/forge/gitlab.sh:165-183`. The per-merge-request reads (jobs, discussions,
   `closes_issues`) have no `|| return 1`; errexit is off inside `$(…)`, so a failed read becomes an empty
   `--argjson`, jq fails, the merge request is dropped, the loop goes on, and the call exits 0. The partial
   list is then kept as the 60-second snapshot. Scenario: two merge requests are open, and the discussions
   read of !3 fails once (a fake glab that fails `merge_requests/3/discussions*`). `kit flow changes --json`
   lists only !4 (`[4]`), `kit flow underway --json` prints `[]`, and `kit flow backlog --json` lists #7 and
   #9, so issue #7, which !3 closes, is drawn in backlog as unclaimed. All exit 0, with only a stray
   `jq: invalid JSON text passed to --argjson` on the first reader's stderr. Fix: `|| return 1` on each
   read, and check jq's status in the loop. Confirmed.
3. High, bug. `kit/modules/bash/forge/gitlab.sh:165`. A fork merge request's head pipeline lives in the
   fork's project, but its jobs are asked under the target project (`projects/<target>/pipelines/<id>/jobs`),
   which answers 404. By the mechanics of finding 2 the merge request vanishes; when it is the newest one
   (the last in the loop), every forge stage fails with `jq: error: Cannot index string with
   string ("name")`. Scenario: with a fake glab that puts the 404 body on stdout and exits 1,
   `kit flow changes --json` exits 1 with that jq error, and backlog likewise.
   `tests/forge-conformance.bats:282` masks it: the fake answers pipeline 11 under the target project. Fix:
   read jobs under `.head_pipeline.project_id`; a pipeline that cannot be read gives no checks, not no merge
   request. Plausible; GitLab's 404 is not verified live, and kit's behaviour on it is confirmed.
4. High, bug. `kit/modules/bash/forge/gitlab.sh:131,153` and `kit/modules/bash/forge/github.sh:143,112`. The reads
   take hard first pages, with no pagination and nothing said. Scenario: on GitLab,
   `issues?…per_page=100…sort=asc` is the 100 oldest open issues, so in a project with more, new issues never
   reach the backlog and `--today` is always empty; merge requests stop at 100. On GitHub,
   `pullRequests(first: 100, CREATED_AT ASC)` with no `pageInfo` drops the newest pull requests, and backlog then
   lists the issues they close as unclaimed; `gh issue list --limit 1000` keeps the 1000 newest, so the two
   providers drop opposite ends of an "oldest first" list. Also truncated are `reviewThreads(first: 100)`,
   `contexts(first: 100)` (a failing check past the 100th is not seen), and GitLab's jobs, discussions, labels,
   and releases at `per_page=100`. No `--paginate` or `hasNextPage` is anywhere but in `closingIssuesReferences`.
   Fix: page until done, or say "first N of M" in the stage. Confirmed by reading.
5. Medium, bug. `kit/modules/jq/backlog.jq:12`. An issue counts as claimed when any open change closes an issue
   with the same number, whatever its repository. Scenario: PR #3 says `Fixes other/repo#9`; the local open issue
   #9, "Local nine", disappears from backlog and is not in underway either, which shows "Nine in ANOTHER repo"
   instead: backlog is `[{"number":7}]`, and the underway row has `organization: other, repository: repo`. This
   breaks "every open issue is in backlog or underway". Fix: match on organization and repository plus number
   (the documents already carry them). Confirmed.
6. Medium, bug. `projects/tui/internal/kit/kit.go:562-573` against `kit/modules/bash/flow.sh:443-449`.
   `kit flow incoming --json` prints its whole document on stdout and exits 1 when any remote's fetch fails, with
   nothing on stderr, and `Runner.output` throws stdout away on a non-zero exit. Scenario: with a dead second
   remote, the command exits 1, stdout holds both remotes with `error` on one, and stderr is 0 bytes. In kittui
   that one dead secondary remote blanks the whole incoming stage as "kit flow incoming exited 1 with nothing on
   stderr", and `remote.Error` (`projects/tui/internal/flow/rows.go:89,696`) is unreachable for one repository.
   Fix: exit 0 with the errors in the document for `--json`, or have `Status` keep stdout on exit 1 for incoming.
   Confirmed for the bash run; the Go side by reading.
7. Medium, bug. `kit/modules/bash/flow.sh:280-285`. The `NR == FNR` idiom of `flow:fetch-diff` misfires when the
   "before" snapshot is empty: every line of "after" is read as an old ref and printed as gone. Scenario: in a
   repository with no tags, the first `kit flow incoming` after `git remote add` (or after `git init` and
   `remote add origin`) reports "2 branches pruned" in place of "2 new branches":
   `{"branches":{"new":[],"updated":[],"gone":["main","x"]}}` from a fresh remote. Fix: `FILENAME == ARGV[1]`, or
   pass the snapshots as awk variables. Confirmed.
8. Medium, bug. `kit/modules/jq/forge.jq:201`. GitLab's `unresolved` counts unresolved notes, where GitHub's
   counts threads. Scenario: one open thread with three notes gives `{"unresolved":3}`, drawn as `💬 3 unresolved`.
   Fix: count discussions with `any(.notes[]; .resolvable and (.resolved | not))`. Confirmed.
9. Medium, bug. `kit/modules/jq/forge.jq:182-183`. A failed GitLab job with `allow_failure: true` is a failed
   check, so a merge request whose pipeline "passed with warnings" is `failing:gate`. Scenario: a jq run gives
   `{"state":"FAILURE","failing":"lint"}`. Also, `requested_changes` in `detailed_merge_status` is not
   `blocked:approval` (line 199), where GitHub's `CHANGES_REQUESTED` is. Fix: a job that is
   `failed and .allow_failure` counts as passed; add `requested_changes`. Confirmed.
10. Medium, bug. `kit/modules/bash/forge/github.sh:96,165,205,261,363,402`. `gh api` (user, graphql, `rate_limit`)
    goes to gh's default host, github.com unless `GH_HOST` is set or a single host is logged in, while
    `gh repo view` and `gh pr list` follow the checkout's remote. Scenario: gh is logged in to github.com and a
    GitHub Enterprise host, and the repository is on the enterprise host (`kit/modules/bash/forge.sh:17` supports
    that detection). The GraphQL read asks github.com for the enterprise owner and name, every forge stage fails
    with "run: gh auth status", and `forge:me` gives the github.com login, so `--mine` matches the wrong person.
    `gh api --help` says `--hostname … (default "github.com")`, and kit passes no `--hostname` and sets no
    `GH_HOST`. Fix: pass `--hostname "$(forge:host <primary>)"`. Plausible.
11. Medium, bug. `kit/modules/bash/forge/github.sh:17`. `branch-for-issue` reads `gh pr list` with its default
    `--limit 30` (`gh pr list --help`: `--limit int (default 30)`), so an issue whose pull request is not among
    the 30 newest open ones has "no branch". Scenario: `kit work` (`kit/modules/bash/issue.sh:295`) then makes the
    duplicate `<N>-<slug>-1` that the comment above it says it prevents; GitLab reads 100. Fix:
    `--search "head:<N>-"` or a high limit. Confirmed.
12. Medium, bug. `kit/modules/bash/forge/github.sh:301` and `kit/modules/bash/forge/gitlab.sh:299`. builds filters
    the 50 newest runs of any kind; it does not ask for 20 unowned ones. Scenario: in a repository where pull
    request runs fill the last 50, the stage is short or says "no build since the last 20 runs" although main has
    builds. Runs for `release`, `delete` (this repository's own `.github/workflows/retire-pre-releases.yml`),
    `workflow_run`, and `merge_group` are in neither changes nor builds. Fix: ask the forge by branch and event
    (`gh run list --branch <default> --event push`, `pipelines?ref=…&source=…`). Confirmed by reading.
13. Medium, inconsistency. `kit/modules/bash/forge/github.sh:146` against
    `kit/modules/bash/forge/github.sh:184-189`. A change's `author` comes from GraphQL, which
    names a bot `dependabot`; `forge:pr-author` uses REST on purpose to get `dependabot[bot]`.
    Scenario: `kit flow changes --owner='dependabot[bot]'` matches nothing, and a login copied
    from the picture does not match what `pr-author` gives. A live read of cli/cli shows
    GraphQL's `{"__typename":"Bot","login":"copilot-swe-agent"}` and REST's `.user.login` as
    `Copilot` for the same pull request; `dependabot` likewise. Fix: one spelling in both (append
    `[bot]` when `__typename == "Bot"`). Confirmed.
14. Medium, bug. `kit/modules/bash/flow.sh:1334-1351`. Above desks the exit status is always 0, in text and JSON,
    even when a desk's picture exited 1; `kit/bin/kit-flow:229` promises 1. Scenario: at the desk the exit is 1 (a
    dead remote); one level up it is 0 both ways. Fix: collect each desk's status. Confirmed.
15. Medium, bug. `kit/modules/jq/underway.jq:16,22` and `kit/modules/bash/changes.sh:41`. "here", "local", and
    the worktree's own change are matched on the branch name alone. Scenario: a fork pull request from
    `cy:main` is drawn `💻 here` with the desk's path as `worktree`
    (`"here":true,"worktree":"…/repo","pr":{"fork":"cy"}`); in a worktree, `changes` shows a fork's pull
    request that shares the branch name. Fix: `here` and `local` only when `.fork == null`. Confirmed.
16. Medium, bug. `kit/modules/jq/forge.jq:132,202`. `closes` takes every linked issue, open or closed (the
    GraphQL query does not ask `state`, and GitLab's `.state` is ignored). Scenario: a closed issue linked to an
    open pull request is a row in underway, "the open issues closed by an open pull request". Fix: ask and
    filter on state. Plausible; the missing filter is confirmed, and that the forges list closed issues is
    inferred from `pr-closing-issues` printing a state.
17. Medium, bug. `kit/modules/bash/forge/gitlab.sh:159-184` with `kit/modules/bash/forge.sh:88`. Four sequential
    glab calls run per merge request (plus one per fork). Scenario: past roughly 50 merge requests the read
    outlasts `forge_snapshot_wait` (120 s) and kittui's 2-minute stage deadline
    (`projects/tui/internal/flow/flow.go:116`): waiters give up and each starts the same full read, or all three
    stages are killed and nothing is ever kept. Fix: none given. Plausible.
18. Low, bug. `kit/modules/jq/forge.jq:23`. `epoch` accepts only a time that ends in `Z`. Scenario: a
    self-managed GitLab with a non-UTC time zone answers `+02:00`, and one such time fails the whole
    stage: `jq: error: date "2026-10-01T11:00:00+02:00" does not match format`. Fix: normalise the offset
    first. Plausible; the jq failure is confirmed.
19. Low, bug. `kit/modules/bash/flow.sh:194-204,817` and `kit/modules/bash/underway.sh:167`. Widths are
    `${#text}`: code points in a UTF-8 locale, bytes in `LC_ALL=C`. Scenario: a CJK title clipped to 40
    "characters" takes 80 columns on a 60-column terminal; under `COLUMNS=61 LC_ALL=C` the clip cuts inside a
    character, a backlog row gives `cut: Illegal byte sequence`, and iconv rejects the line. Fix: measure
    display width, and never slice bytes. Confirmed.
20. Low, inconsistency. `kit/bin/kit-flow:17-19,39` against the grammar. The page says a stage stays in text
    "with --text", and `flow:render` (`kit/modules/bash/flow.sh:1284`) handles it, but the grammar refuses it.
    Scenario: `kit flow backlog --text` prints `❌ --text cannot be combined with backlog`, exit 2; at a terminal a
    single stage can only be printed as text by piping. Fix: none given. Confirmed.
21. Low, bug. `kit/modules/bash/forge/gitlab.sh:314`. The project path is wrong for three spellings of the
    remote. Scenario: `gitlab.com:group/proj.git` (no user; `forge:url-host` accepts it) gives
    `gitlab.com:group/proj`; a trailing slash gives `group/proj/`; a GitLab under a sub-path gives
    `gitlab/group/proj` (the sed was run on each). Fix: none given. Confirmed.
22. Low, bug. `kit/modules/bash/forge/gitlab.sh:211,228,244`. `source_branch=$1` is not
    URL-encoded. Scenario: a branch with `+`, `&`, or `#` finds no merge request, which reads as
    "no merge request". Fix: none given. Plausible.
23. Low, bug. `kit/modules/bash/marks.sh:32`. Marks that cannot be read (rate limit, offline) become `{}`: every
    forge stage's mark is null, exit 0, so the stage is never stale and nothing says why. The changes mark is also
    blind to status contexts that are not Actions runs and to runs past the 10 newest. Scenario:
    `kit flow --marks` with gh failing prints nulls, exit 0. Fix: none given. Confirmed.
24. Low, bug. `kit/modules/bash/forge.sh:320-324`. `forge:forget` removes the snapshots, but a read already in
    flight writes its pre-change answer afterwards, stamped with the second it began. Scenario: that answer is
    served for up to a minute after the change. Fix: a generation stamp the writer checks. Plausible.
25. Low, inconsistency. `tests/flow.bats:745` against `kit/modules/jq/forge.jq:65`. The unit test pins a shape the
    forge never gives. Scenario: the test feeds `failing: 2` and expects `❌ 2 failing 3 / 5`; the real field is
    the first failed check's name (in Go, `Failing string`), so the cell reads
    `❌ build (ubuntu, 20) failing 3 / 5`. Fix: none given. Confirmed.
26. Low, inconsistency. `kit/modules/bash/forge/gitlab.sh`, at every error. Every GitLab failure names one remedy,
    while GitHub has `forge/github:limited`. Scenario: a 429, a 5xx, and a lost network alike print "run: glab
    auth status". The assign, close, comment, and unassign calls of `kit/modules/bash/tracker/ado.sh` fail
    silently (`2>/dev/null`, no remedy), against `kit/modules/bash/tracker.sh:34`. Fix: none given. Confirmed.
27. Low, inconsistency. `projects/tui/internal/flow/fields.json` and `projects/tui/internal/flow/rows.go`.
    The catalog, the Go structs, and the bash disagree on a few fields. Scenario: the `event` enum of builds
    lists `pull_request`, which the stage never emits; the Go structs read `url` on issue, underway, pull,
    and release, and `pr.source`, which the bash never writes (always empty); `kit/bin/kit-flow:96` omits
    `labels` from the underway row. No other type or nullability mismatch was found (the `reset: null` of
    quota decodes to 0). Fix: none given. Confirmed.
28. Low, missing. `tests/forge-conformance.bats`. The suite never calls `require`, `pr-set-body`,
    `pr-last-for-branch`, or `pr-here`, and has no test for a failed sub-read, a list past one page, a fork merge
    request whose pipeline is in the fork, cross-repository closes, same-named checks, or a first fetch. Scenario:
    the defects of findings 1, 2, 3, 4, 5, and 7 pass the suite unseen. Fix: none given. Confirmed.
29. Low, bug. `kit/modules/bash/git.sh:33-41`. With no `<primary>/HEAD` (git before 2.48, some CI clones) the
    default branch is silently `main`. Scenario: on a `master` repository ahead and behind are 0, and builds drops
    every push. Fix: none given. Plausible; git 2.5x sets the ref on fetch, and it could not be reproduced.

### Filed issues (Flow)

- #60: unchanged; `settings_stages` is `… changes builds incoming releases`.
- #68 and #50: not fixed, and uneven. The GitLab provider reads the primary's URL; the GitHub provider never tells
  gh which remote is primary (no `GH_REPO`, no `-R`), so gh's own choice of base repository answers, while the
  snapshot directory is keyed by the primary's URL.
- #69: present in `branch-for-issue`, `pr-for-branch`, `pr-last-for-branch`, `pr-state-for-branch`, and `pr-here`
  (`2>/dev/null || true`, both providers), GitHub's run jobs, GitLab's label colours, and marks; GitLab's
  `issue-state` prints `OPEN` for an error body. Findings 2 and 3 are worse cases of it.
- #77: unimplemented; `source` is only github or gitlab (`kit/modules/jq/forge.jq:208`, "for now").
- #26: the `took` of builds is `updatedAt - startedAt` of the latest attempt, and `age` is from the
  first; it looks handled. For checks, see finding 1.
- #63: not examined.

### Not reviewed (Flow)

- `kit/modules/bash/across.sh` (`--workspace`: `{rows, errors}`, marks, and quota across members).
- kittui's Go beyond the decode structs, `Status` and `output`, and the `--fresh` path.
- The legend of `kit help flow` against `kit/modules/jq/glyphs.json` line by line (state and status glyphs do
  match `projects/tui/internal/flow/fields.json`).
- Real GitLab behaviour (the 404 of fork pipelines, time-zone offsets, glab's
  output on HTTP errors), seen only through the fake.
- The lock under real contention (pid reuse, SIGKILL) and the timing of `forge:snapshot`;
  `kit/modules/bash/tracker/ado.sh` against a real `az`.
- No bats file was run, and shellcheck was not run.

## Serve

Reviewed `kit serve`, the workspace verbs (`kit workspaces`, `kit workspace`, and `--workspace`), and the
logs (`serve.sh`, `workspaces.sh`, `workspace.sh`, `across.sh`, and `log.sh` under `kit/modules/bash/`,
and `kit/modules/jq/serve.jq`); findings were reproduced in a scratch HOME with a fake `gh` first on PATH,
on macOS with bash 5.3, jq 1.8.2, and yq 4.54.1.

1. Critical, bug. `kit/modules/jq/serve.jq:127-136,105-108`. An operand's value is never checked against its
   `enum` or against the words the tool refuses, so the listing-only `packages` tool runs `add`, `remove`, and
   `select`. Scenario: `tools/call {"name":"packages","arguments":{"package":["add","style/yaml"]}}` makes the
   server run `kit packages add style/yaml`; `kit.yml` gains `- style/yaml`, `.kit/styles/yaml.md` is placed, and
   the answer is `isError:false`, on a tool annotated `readOnlyHint:true`. The result text is `── kit packages add
   style/yaml ── … ✅ packages: kit style/yaml`, `git status` shows `M kit.yml`, and the MCP log has
   `outcome=ok kit packages add style/yaml`. `tests/serve.bats:136` only tries `{"subcommand":"add"}` and
   `{"add":true}`. Fix: enforce `enum` in `serve::argv`, refuse an operand equal to a refused word, and re-check
   the built argv with `kit syntax <verb> --match` against the offered lines only. Confirmed.
2. High, bug. `kit/modules/bash/serve.sh:61,408`. `check` is annotated `readOnlyHint:true` but runs the command
   of every `kit:generate` directive written in the repository's files. Scenario: an agent writes `notes.md`
   holding a directive named `x` whose command is `date > /elsewhere/PWNED; echo hi`, then calls the `check`
   tool; the command runs and writes outside the repository, and the answer is `✅ generate: …`,
   `isError:false`. A harness that
   auto-approves read-only tools gives shell execution to an agent that only has file edits. `lint` likewise
   runs each rule's `lint:` command (not reproduced). Fix: under `kit_serve_hydrated`, have `check` skip
   generated regions or only compare them without running them; or annotate `check` and `lint` as not
   read-only. Confirmed for `check`; plausible for `lint`.
3. High, bug. `kit/modules/bash/serve.sh:433,535,316`. A `params` that is not an object kills the server: a jq
   failure in a plain assignment under errexit. Scenario:
   `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":[1]}` (or `resources/read` with
   `"params":"kit://1st.md"`) gets no answer; the server exits 5 with `jq: error … Cannot index array with
   string "name"` on stderr, and every later request is lost. Fix: refuse a non-object `params` with -32602 in
   `serve:handle`; make each handler's jq failure a -32603 answer. Confirmed.
4. High, bug. `kit/modules/bash/serve.sh:293,309,445-447,223`. Results, arguments, and the pending line travel on
   jq's command line or in the environment, so a large message kills the server. Scenario: a tool output of 1.2 MB
   (`syntax` with `verb` set to `flow` 50 times) or a 1.5 MB request line gives `serve.sh: line 293: …/jq:
   Argument list too long`, no answer, and exit 126; the same happens at line 447. On Linux one argument is capped
   at 128 KB; `kit syntax` alone already makes a 116 KB result, and `flow --json` on a real repository will pass
   it. Fix: pass JSON through files or stdin (`--slurpfile`, `--rawfile`, `input`), never `--argjson "$big"`.
   Confirmed on macOS; the Linux threshold is plausible.
5. Medium, bug. `kit/modules/bash/serve.sh:458`. A NUL in an argument makes `jq --raw-output0` fail inside a
   process substitution; the verb then runs with fewer words than the call gave, and reports success. Scenario:
   `help {"verb":["version","a\u0000b"]}` runs `kit help version` with `outcome=ok`, and stderr has
   `Cannot dump a string containing NUL with --raw-output0`. The same shape on `close`, `assign`, or `unassign`
   acts on a different list than the one requested. Fix: refuse a string holding U+0000 with -32602 in
   `serve::argv`; check jq's status before running. Confirmed.
6. Medium, missing. `kit/modules/bash/serve.sh:463-467`. A tool's verb has no timeout, and
   `notifications/cancelled` is ignored. Scenario: one stalled verb (a `gh` call, a linter) blocks
   every later request, `ping` included; the call is a synchronous subshell, and `serve:handle`
   drops every notification. Fix: run the verb under a deadline, answer `isError` when it passes,
   and honor cancellation. Confirmed from the code.
7. Medium, inconsistency. `kit/bin/kit-serve:16-19`. The header, which is `kit help serve`, says the
   tools are the eleven verbs that "do not change anything" and that "a call does not place .kit/".
   Scenario: the server offers 17 tools, six of which write to the forge, and `work` makes a branch, a
   worktree, and a pull request; `tests/serve.bats:190` is titled "every tool says it reads only". Fix:
   bring the header in step with `docs/serve.md`. Confirmed.
8. Medium, bug. `kit/modules/bash/workspaces.sh:133`, `kit/bin/kit-flow:286`, and `kit/bin/kit-workspace:212`.
   `--workspace` never runs `workspaces:check`, so a file that `kit workspaces` refuses yields an empty,
   error-free answer. Scenario: with `exclude: [nonsense]` in a group, `kit flow desk --workspace grp --json`
   prints `workspaces.sh: line 133: excluded[…]: bad array subscript`, then `{"rows":[],"errors":[]}`, exit 0, and
   `kit workspace repos` does the same. The spec's anti-goal is a total that looks complete when it is not. Fix:
   call `workspaces:check` in `workspaces:resolve`; skip an empty key in `workspaces:fixed`. Confirmed.
9. Medium, inconsistency. `kit/modules/bash/serve.sh`, in the `workspace` property of flow and the paths of
   `lint`, `context`, and `kinds`. `docs/serve.md` says the server works "in the repository it starts in",
   but tools reach outside it. Scenario: `flow {"workspace":"github.com/acme/a","stage":"desk","json":true}`
   returns another clone's uncommitted paths, and `lint {"path":["/abs/outside.sh"]}` returns shellcheck
   findings that quote that file's lines (`export SECRET_TOKEN=hunter2 …`). No resource URI escapes: the
   match is exact against the list. Fix: refuse paths that resolve outside the root; drop or gate
   `workspace` in the flow tool, or document it. Confirmed.
10. Medium, bug. `kit/modules/bash/workspaces.sh:405-415`. Every edit is an unlocked read-modify-write
    through `yq --inplace`, so concurrent writers lose updates while each reports success. Scenario: 12
    parallel runs of `kit workspaces --add small github.com/acme/mN` print 12 "added" lines and leave 8
    members in the file. A single write is atomic (a rename), and a symlinked file is kept. Fix: take a lock
    (a `mkdir` lock or `flock`) around `workspaces:edit`. Confirmed.
11. Medium, inconsistency. `kit/bin/kit-flow:250-288` and `kit/bin/kit-workspace`. The startup rule in
    `docs/workspaces.md:40-46` and the spec applies only to `kit workspaces --current`. Scenario: outside a
    repository after `kit workspaces --use small`, `--current` says `📚 small (the last group used)`, but
    `kit flow --json` prints the above-desks array. `kit flow --workspace none` and
    `kit workspace --workspace none` fail with `'none' is no workspace`, exit 2, and `kit workspaces --new none …`
    succeeds, making a group that `--current --workspace none` can never name. Fix: have both verbs resolve
    through `workspaces:current`; reserve the name `none`. Confirmed.
12. Medium, bug. `kit/modules/bash/across.sh:430-438`. `across:quota-merge` drops every member whose read failed,
    without a word. Scenario: `kit flow --quota --workspace small` with the forge unreadable prints `[]`, exit 0;
    in one repository the same failure is `❌ … the rate limits could not be read`, exit 1. Fix: print each failed
    member's line on stderr, and exit 1 when none could be read. Confirmed.
13. Low, bug. `kit/modules/bash/workspace.sh:125-126`. The branch name is spliced into a sed replacement.
    Scenario: for the branch `feat/a&b` the commits rows say `"branches":["feat/ab"]`; for the branch
    `pipe|name` sed fails with `bad flag in substitute command`, and the branch is missing from every row. Fix:
    append the branch with awk or a bash loop. Confirmed.
14. Low, bug. `kit/modules/bash/log.sh:70`. A long line is appended in several writes, so concurrent calls
    interleave. Scenario: 30 parallel calls with a 40 KB argument leave 30 newlines but 13 malformed lines, one
    call's text spliced into another's at 1024-byte multiples; a long `kit commit` message or `--title` qualifies.
    Fix: truncate the logged command line below one write's size. Confirmed.
15. Low, bug. `kit/bin/kit:33-34` and `kit/modules/bash/serve.sh:459`. Argument lines are
    logged verbatim, credentials included, and kept 14 days. Scenario: `kit tools run --
    sh -c true --token=ghp_SECRET123` and `kit workspaces --show https://user:ghp_URLTOKEN@github.com/acme/a.git`
    are both in `.kit/logs/kit-*.log` in full. Fix: redact URL userinfo and words
    shaped like `--token=`; do not log what follows `kit tools run`. Confirmed.
16. Low, bug. `kit/modules/bash/serve.sh:251-288`. Invalid requests are not refused with -32600. Scenario:
    `"method":123` gets -32601 with the message `Method not found:` and no name; `{"jsonrpc":"2.0","id":3}`
    gets silence; `"id":null`, `{"a":1}`, `[1]`, and `true` are echoed in a result; a missing `jsonrpc`, or
    `"1.0"`, is accepted; a second object on the same line is dropped unanswered; the id `1e3` comes back
    as `1E+3`; and a batch array is refused although 2025-03-26, which the server offers, requires
    receiving batches. Correct are -32700, an unknown method, unanswered notifications, string and number
    ids, and exit 0 at EOF. Fix: none given. Confirmed.
17. Low, missing. `kit/modules/bash/serve.sh:277-286,541`. The server has no `resources/templates/list`, though
    resources are declared and the documented URIs are templates, and it names a missing resource with another
    code than the revisions through 2025-11-25 do. Scenario: `resources/templates/list` answers -32601, and a
    missing resource is -32602, where those revisions name -32002. Fix: none given. Confirmed for the behaviour;
    plausible for the code of the 2026-07-28 revision.
18. Low, bug. `kit/modules/bash/serve.sh:538`. `awk -v uri=` expands backslash
    escapes. Scenario: `kit://rules/ba\163h` reads the bash rule and is echoed as its
    uri. Fix: pass the uri through `ENVIRON`. Confirmed.
19. Low, bug. `kit/modules/bash/workspaces.sh:195`. Members match by exact case. Scenario:
    `github.com/Acme/A` shows ☁️ beside a clone of `github.com/acme/a`. Fix: lowercase the
    host, and the path on GitHub and GitLab. Confirmed.
20. Low, missing. `kit/bin/kit-workspaces:180-191`. No form deletes or renames a group. Scenario:
    `--unfavorite` of a favorite whose group is gone fails with exit 2, so the favorite stays until the file
    is edited by hand. Fix: none given. Confirmed.
21. Low, bug. `kit/modules/bash/workspaces.sh:51,81`. One clone under the roots with an unparsable `kit.yml` is
    a problem for every workspace command (the #69 pattern). Scenario: its yq error is printed on every
    workspace command, even for a group that does not hold it, and the clone is then filed under `origin`
    whatever its `primary`. Fix: none given. Confirmed.
22. Low, bug. `kit/modules/bash/git.sh:44-46`, through `kit/modules/bash/workspace.sh:26`. In a bare repository
    `git:main-root` is the bare directory's parent. Scenario: `kit workspace` prints `repos lists a
    workspace's repositories…` and exits 1 (the header says 2), and a stage prints `fatal: not a git repository`
    three times, or reads an enclosing repository. Fix: none given. Confirmed.
23. Low, inconsistency. `kit/bin/kit-workspace:243-248,140`. Over a workspace, a member that cannot be read
    is a ❌ line and exit 0 in text too; the header says 1. Scenario: `kit workspace` over a group with one
    unreadable member exits 0, where `kit flow --workspace` as text exits 1. Fix: none given. Plausible;
    from the code, with no failing member produced.
24. Low, bug. `kit/modules/bash/serve.sh:171-241`. The reload misses three edges. Scenario: a deleted file is
    never seen (`find -type f -newer`); `serve:resume` withholds both `list_changed` notifications from any
    pending line that contains the text `"initialize"`; and each reload leaves the scratch directory of
    `file:temp` behind. Fix: none given. Confirmed from the code.
25. Low, missing. `kit/modules/bash/serve.sh:61-62`. The tool list is a hand-kept array; only the schemas
    come from syntax. Scenario: `workspaces` and `workspace` are not offered, though the `workspace`
    property of flow says "as kit workspaces names it"; no test snapshots the schemas, and
    `tests/contract.bats` covers mise only. Fix: none given. Confirmed.
26. Low, bug. `kit/modules/bash/workspaces.sh:71-85`. A root written twice with a different spelling
    lists each clone twice. Scenario: with both `…/roots` and `…/roots/` among the roots, `checkouts`
    holds each clone twice. Symlinked clones and bare repositories are never checkouts, which
    `docs/workspaces.md` does not say. Fix: none given. Confirmed.
27. Low, bug. `kit/modules/bash/workspaces.sh:89-95,146`. A `workspaces.yml` that is a scalar passes
    `workspaces:check`. Scenario: the list prints jq parse errors and an empty listing with exit 0,
    and `--new` exits 5. Fix: none given. Confirmed.

### Filed issues (Serve)

- #58: implemented. `kit/bin/kit-serve:43-51` hands over to the checkout's own `kit-serve`;
  `serve:reload` restarts between messages; `tests/serve.bats:292` and
  `tests/serve.bats:300` cover both. Its edges are in finding 24.
- #71: fixed. `log:days` reads `log_days` through yq
  (`kit/modules/bash/log.sh:39-47`), and `tests/log.bats:95` covers it.
- #69: present in workspaces (finding 21).
- #74: not implemented (finding 25).

### Holds (Serve)

- Resource URIs: `kit://rules/../../etc/passwd` is refused.
- Operands that start with `-` are refused; an option's dash value is read as its value
  (`--toolchain --outdated`, `--workspace --fresh`).
- `flow` gets `--no-fetch`; `--owner` values with spaces stay one word.
- Stdout carries only protocol lines.
- A log that cannot be written (a read-only directory, `logs`
  being a file, outside a repository) never fails the verb.
- Checkout paths with spaces, linked worktrees, a detached HEAD, a repository
  with no remote, and empty groups give valid JSON.

### Not reviewed (Serve)

- The text of the 2026-07-28 revision itself (result fields, -32022, `cacheScope`).
- Linux and GNU runs: the 128 KB argument cap, and `find -newer` on filesystems with coarse mtimes.
- SIGTERM and a closed stdout: the stop line and the scratch sweep.
- Reproducing a hung verb and a failing member in `kit workspace --workspace`.
- kittui's use of `kit workspaces --json`.
- The side effects of `tools --toolchain nix`; `syntax --choices` reaching the forge.
- The workflow tools against a real forge.
- `kit:move-settings` running in every verb.
- Unbounded parallelism in `across:start` on large groups.
- `tests/workspaces.bats`, `tests/workspace.bats`, and `tests/across.bats` were read by title only.

## Command line

Reviewed the grammar (`kit syntax`, the guard, `kit help`, and `kit completion`), the toolchains and dependencies
(`kit toolchains`, `kit tools`, and their providers), and `kit welcome`, the hand-off to kittui, and `kit tutorials`;
findings were reproduced in scratch repositories with a fake `mise`, `brew`, `nix`, `uv`, `npm`, `go`, and `kittui` on
an isolated PATH, and no real install or update ran.

1. High, bug. `kit/modules/jq/syntax.jq:74-81`, with `kit/bin/kit-tools:6`. `kit tools run <command>…` refuses any
   command line that holds one of the flags of `kit tools` itself. The usage line opens with the optional
   `[--toolchain <name>]`, so `opens()` never sees the literal `run`, the alternative that holds `--check` wins,
   and `reclaim` is never reached; only `run -- <command>` works. Scenario: `kit tools run prettier --check .`
   prints `❌ kit tools needs --outdated`, exit 2 (on a terminal it hands off to the TUI form instead of running);
   `kit tools run npm outdated --json`, and the same with `--jq` or `--outdated`, print `unknown argument 'run'`;
   `kit tools run echo -h` shows kit's own page; and `kit syntax tools --match -- run prettier --check .` gives
   status 1. Fix: let `opens` look past leading optional groups, as `firsts` does; add a test. Confirmed.
2. High, bug. `kit/bin/kit-tools:148-170`. The dependency layers are worked out before the toolchain installs, so
   the first `kit tools install` on a fresh machine skips npm and uv and exits 0 all the same. Scenario: mise
   pins node, `package.json` and its lock are present, and nothing is installed yet; only `mise install` runs,
   and a second run does `npm ci` (with a fake mise whose `bin-paths` answers only after `install`, the first run
   made 0 npm calls). Where a runner already has a system npm, `npm ci` runs with that node, since
   `toolchain_dirs` is the stale pre-install value. `tests/dependencies.bats:150` puts npm on PATH from the
   start. Fix: recompute `toolchain_dirs` and the usable providers after the toolchain layer installs, and fail
   when a locked provider is unusable after that. Confirmed.
3. High, bug. `kit/modules/bash/dependencies/uv.sh:33-40`. The whole `uv.lock`, as JSON, is passed to jq as
   one `--argjson` argument. Scenario: a 1.4 MB lock (400 packages, 12 wheels each) gives
   `jq: Argument list too long`, then `❌ uv: the pinned tools could not be read`, so `kit tools`,
   `--outdated`, and the check of `kit welcome` all fail. macOS has an `ARG_MAX` of 1 MiB; Linux caps a single
   argument at 128 KiB, which an ordinary lock exceeds. Fix: feed the lock on stdin or with `--slurpfile`.
   Confirmed on macOS; the Linux threshold is not reproduced.
4. High, bug. `kit/modules/bash/dependencies/uv.sh:36-40` and `kit/modules/bash/dependencies.sh:104-108`. Every
   package in the lock is expected in `.venv`, including ones a marker excludes on this platform. Scenario: any
   project that depends on click or pytest locks `colorama` (`sys_platform == 'win32'`); `kit tools` shows
   `❌ colorama`, `dependencies:installed uv` fails, and the `dependencies-uv` check of welcome never holds, with a
   fix (`kit tools install`) that cannot satisfy it, so `kit welcome --text` exits 1 for good. Fix: judge with
   `uv sync --frozen --check`, or compare only what uv resolves for this platform. Confirmed.
5. Medium, bug. `kit/modules/bash/welcome.sh:248`. `dependencies:usable | grep --quiet` under `pipefail`
   fails whenever grep matches before the last provider's line is written, so the check reports a tool as
   missing that is there. Scenario: `package.json` and `pyproject.toml`, each with its lock, and npm and uv
   both on PATH; the `dependencies-npm` check of welcome says `package-lock.json is here, but npm is not
   on PATH`, with no fix. `welcome:dependencies npm` gave that line 6 times of 6, and the pipeline alone
   failed 1492 times of 1500; with npm alone it is right, which is all `tests/welcome.bats:195` covers. The
   same shape is at `kit/bin/kit-flow:330` and `kit/modules/bash/workspaces.sh:235,359`, not tested here.
   Fix: capture the list, then match it with `<<<`. Confirmed.
6. Medium, bug. `kit/modules/bash/toolchain.sh:331-336`. `kit toolchains choose brew` is refused when brew
   is the only usable project toolchain. Scenario: on macOS with a Brewfile and nothing else, it prints
   `❌ brew fills in for the tools brew does not pin; choose brew`, exit 1, and after
   `kit toolchains choose ambient` there is no command back to brew. With two others the join is wrong too:
   `mise docker does not pin; choose mise docker` (`${usable[*]// / or }` substitutes per element).
   `docs/toolchains/using-brew.md` says brew alone is "the project toolchain, as any other is". Fix: refuse
   only when `usable` does not hold brew; join the names properly. Confirmed.
7. Medium, bug. `kit/modules/bash/toolchain.sh:325-357` and `kit/bin/kit-toolchains:44-45`. `choose`
   reports success while an active toolchain overrides the choice. Scenario: with `MISE_SHELL` set (as step
   1 of `docs/toolchains/using-mise.md` sets it up) and both `mise.toml` and `flake.nix` present,
   `kit toolchains choose nix` prints `✅ kit uses nix for the project`; the next `kit toolchains` pins
   mise, and `kit tools` uses mise. `docs/toolchains/using-nix.md:83` says the command switches. Fix: say
   that the active one wins, or refuse. Confirmed.
8. Medium, bug. `kit/modules/bash/toolchain/brew.sh:34-35,39-47,70-77`. brew filling in swallows a failure.
   Scenario: when `brew bundle list` fails (a bad Brewfile), `kit tools install` prints `✅ brew installed the
   tools Brewfile pins that mise does not`, exit 0, because `readarray < <(…)` hides the status;
   `kit tools update` then exits 1 with no line about brew at all. Fix: capture the pins with
   `$(…) || return 1`, and say what failed. Confirmed.
9. Medium, bug. `kit/modules/bash/1st.sh:242-250,289-292`, `kit/modules/bash/welcome.sh:623-627`,
   `kit/bin/kit-tools:93`, and `kit/bin/kit-toolchains:85`. Colour is decided by stderr but written to
   stdout. Scenario: at a terminal, `kit welcome --checks | cat`, `kit toolchains | cat`, and
   `kit tools | cat` put `^[[90m…^[[0m` in the piped data (seen on a pty with `cat -v`). Fix: stdout writers
   test `-t 1`, as `help:styled` does. Confirmed.
10. Medium, inconsistency. `kit/modules/bash/1st.sh:345,429`, `kit/modules/bash/syntax.sh:323-330`, and
    `kit/modules/bash/welcome.sh:286-299`. The jq check of welcome can never fail, and the refusal
    assumes mise. Scenario: with no jq, `kit welcome` and `kit welcome --checks` print `❌ run: mise
    install (jq is not on PATH)`, exit 2, in a repository with no mise; a jq 1.6 (`--raw-output0` is
    1.7) gives `…; run: mise install`. `docs/features.md:161` says "No tool manager is assumed". The
    same line meets a missing linter in `kit lint`. Fix: let welcome run before the guard's jq
    requirement, and name `kit tools install` or the system toolchain's command. Confirmed for a
    missing jq; the old-jq case is read from the code.
11. Medium, bug. `kit/modules/bash/welcome.sh:263-279`. A tool counts as "not installed" when
    `<tool> --version` exits non-zero. Scenario: a house rule with `lint: go vet` or `lint: gofmt -l` gives
    `❌ go is not installed`, with `kit tools install` as the fix, for good, while `kit tools` (which uses
    `command -v`) lists the same tool as found; `go --version` and `gofmt --version` exit 2. Fix: tell
    "absent" from "does not answer --version". Confirmed.
12. Medium, bug. `kit/modules/bash/tui.sh:39-44`. The build's cache key covers `*.go`, `go.mod`, and `go.sum`
    only. Scenario: `projects/tui/internal/flow/fields.json` is embedded with `//go:embed` (and hand-edited in
    five commits), and an edit to it alone leaves kit running the stale kittui. Nothing prunes the cache either:
    the reviewer's machine holds 200 `kittui-*` binaries, 1.5 GB, and a `_test.go` edit makes one more. Fix:
    digest every file under `projects/tui` that the build reads, and drop old builds. Confirmed.
13. Medium, bug. `kit/modules/bash/tui.sh:54-72`, `kit/bin/kit-help:58-77`, `kit/modules/bash/1st.sh:446-449`, and
    `kit/bin/kit-tutorials:57-58`. `--probe` answers yes whenever go is on PATH, and a failed build is fatal.
    Scenario: in a checkout whose TUI does not compile (a peer mid-edit), `kit help commit`, `kit tutorials`, and
    `kit commit` with nothing given all exit 1 on a terminal with the compiler's error (seen on a pty with a
    failing fake go); the last never says what was missing. The headers promise text "without a kittui". Fix: fall
    back to text, or to the refusal, when the build fails. Confirmed.
14. Medium, inconsistency. `kit/modules/bash/1st.sh:421-428` and `kit/bin/kit-help:8`. `kit <verb> --help` on a
    terminal opens kittui's palette (building kittui first in a checkout); the header says it "prints the same
    page as `kit help --text <verb>`". Scenario: on a pty the guard runs `kittui … help commit`;
    `tests/help.bats:117` runs without a terminal. Fix: exec `kit-help --text`. Confirmed.
15. Medium, bug. `kit/bin/kit-new:104-131`. The verb dispatches on `$1` although the guard takes options
    anywhere. Scenario: `kit new --package=x kind foo` passes the guard as `kind` (`--match` gives status 0
    with the literals `["kind"]`), then runs the `file` branch with the path `foo`;
    `kit new --rule=bash file x.sh` loses `--rule` to the `shift`. Fix: dispatch on `syntax_literals[0]`.
    Confirmed from the code; the verb itself was not run.
16. Medium, inconsistency. `kit/bin/kit-kinds:35-41`, `kit/bin/kit-help:35-55`, `kit/bin/kit-usage:22`,
    `kit/bin/kit-linked:30`, `kit/bin/kit-completion:32`, and `kit/bin/kit-packages:46`. A `--` that the guard
    accepts reaches the verb as a word. Scenario: `kit kinds -- README.md` prints a bogus `--` row;
    `kit usage -- version` and `kit help --text -- x` print `unknown verb '--'`; `kit linked -- 5` prints
    `'--' is not a number`; `kit completion -- zsh` prints nothing, exit 0. `tests/arguments.bats:128` covers
    `kit context` only. Fix: read `syntax_operands`, or drop `--`. Confirmed.
17. Medium, bug. `kit/modules/bash/toolchain/docker.sh:167-174`. Only `${PWD}` is mounted. Scenario: in
    a linked worktree, where `kit work` puts all work, `.git` is a file that points outside the mount,
    so any gate that calls git fails inside the container; the run also has no `--interactive` and
    passes no environment. `tests/tools.bats:470` uses a fake docker. Fix: mount the common git
    directory too. Plausible; read from the `docker run` line.
18. Low, inconsistency. `kit/bin/kit-new:22` and `kit/modules/bash/new.sh:573-594`. #41 does not hold for
    `kit new issue [--list] [--json]`. Scenario: `kit new issue --jq` prints `--jq cannot be combined with issue`,
    and `--json` without `--list` is silently ignored. The test (`tests/arguments.bats:153`) looks per verb, and
    `new` has a `--jq` of another meaning. Fix: none given. Confirmed.
19. Low, bug. `kit/modules/jq/syntax.jq:91-92`. The refusal names the wrong pair. Scenario:
    `kit flow --workspace w --me` prints `--workspace cannot be combined with --workspace`. Fix:
    name a key the other alternative lacks. Confirmed.
20. Low, inconsistency. `kit/modules/bash/syntax.sh:478-486`. `kit syntax <verb> --match` skips the
    closed choices the guard applies, though the header says it "judges the words as the guard judges".
    Scenario: `kit syntax tutorials --match -- --tag nosuch` gives status 0, and
    `kit tutorials --tag nosuch` exits 2. Fix: none given. Confirmed.
21. Low, inconsistency. `kit/modules/jq/completion.jq:22-46`. Completion ignores the alternative chosen. Scenario:
    `kit flow backlog -<TAB>` offers `--stages --me --marks --quota --text --no-fetch`, each refused with
    `backlog`; `kit new kind <TAB>` and `kit new provider <TAB>` offer files; `kit tools --json <TAB>` offers
    `install`; and `kit packages <TAB>` offers no package. Nothing is offered for values the headers enumerate:
    `toolchains choose <name>`, `tools --toolchain`, `welcome --fix <check>`, `new verb --group=`,
    `new provider <seam>`, and `--workspace`. Fix: none given. Confirmed.
22. Low, inconsistency. The usage lines of `kit/bin/kit-flow`, `kit/bin/kit-generate`, `kit/bin/kit-facts`,
    `kit/bin/kit-policy`, `kit/bin/kit-new`, and `kit/bin/kit-workspaces`. The same idea is flagged two
    ways. Scenario: `kit flow --owner alice` prints `--owner needs a value` while `kit flow --workspace w`
    works; `kit workspaces --new=g` works while `--add=g` prints `unknown flag`, because
    `(--add | --remove) <name>` makes `--add` a flag and `kit/bin/kit-workspaces:70` reads it as an option.
    Fix: one form, or a refusal that says `write --owner=alice`. Confirmed.
23. Low, bug. `kit/modules/bash/tutorials.sh:76-77` and `kit/bin/kit-tutorials:71-75`. Two exits of
    `kit tutorials` are not the documented ones. Scenario: with the catalogue missing, `kit tutorials <id>` exits
    2 (`<tutorial> … is not one of the choices`), not the documented 1; and the grammar offers
    `--json <tutorial>`, which the verb refuses with 2. Fix: none given. Confirmed.
24. Low, inconsistency. `kit/bin/kit:23` and `kit/bin/kit-tui:35-46`. Bare `kit` with no terminal
    does not fall back to text. Scenario: it exits 2 with `kit tui needs an interactive terminal;
    run: kit help tui`, where `kit help`, `kit flow`, and `kit welcome` fall back to text; `kit -V`
    is an unknown verb. Fix: none given. Confirmed.
25. Low, bug. `kit/modules/bash/toolchain.sh:380-384`. `kit toolchains choose ambient` fails with git's words
    outside a repository, and a recorded `ambient` shows nowhere. Scenario: outside a git repository the
    command prints git's own `fatal: not a git repository`, exit 1; with `ambient` recorded, no row of
    `kit toolchains` has 📌. Fix: none given. Confirmed.
26. Low, inconsistency. `kit/modules/bash/toolchain/mise.sh:82-85` and `kit/bin/kit-tools:294`. The seam and the ✅
    line say `mise upgrade` "rewrites the pins". Scenario: `mise upgrade` without `--bump` moves within the range
    and never rewrites `mise.toml`. Also, `toolchain/mise:file` names `mise.toml` when only `mise.local.toml` is
    present, and the header of `kit/bin/kit-new` leaves `plumbing` out of the values of `--group`, which
    `new:verb` accepts. Fix: none given. Confirmed from the code.
27. Low, inconsistency. `kit/modules/bash/dependencies/uv.sh:17-26,44`. `--outdated` asks
    `uv pip list --outdated` (the index's latest, transitive packages included) while `update` is
    `uv lock --upgrade`, within the bounds of `pyproject.toml`. Scenario: `kit tools --outdated --check` can stay
    red after `kit tools update`. Fix: none given. Plausible.
28. Low, missing. `kit/modules/bash/welcome.sh:539`. The git check takes any version, while kit calls
    `git rev-parse --path-format=absolute` (git 2.31) in `kit/modules/bash/toolchain.sh:371,382`,
    `kit/modules/bash/welcome.sh:522`, and in `context.sh`, `flow.sh`, and `git.sh` of the same directory.
    Scenario: an older git echoes the option back as a path. Fix: none given. Plausible.

### Filed issues (Command line)

- #74: not done. The snapshots that exist are current, and `tests/syntax.bats:235` enforces
  that, but 10 of 44 verbs have none: assign, close, facts, gather, settle, tutorials,
  unassign, uninstall, workspace, and workspaces.
- #41: holds except for `kit new issue` (finding 18).
- #70 and #65: not examined.

### Holds (Command line)

- `kit syntax --check` is clean, and all 44 verbs parse.
- Each verb has a group in `kit help` and is offered by
  `kit completion candidates`, and `kit completion zsh | zsh -n` passes.
- All 34 files in `projects/tui/internal/syntax/testdata/` equal the output of `kit syntax <verb>`.
- `kit/modules/tutorials.yml` matches the one top-level page under `projects/tutorials/src/content/`.

### Not reviewed (Command line)

- kittui's Go side: how its forms read `kit syntax`, `--choices`, and `--extend`.
- `kit flow`, `kit workspace`, and `kit workspaces` beyond their grammar.
- Real brew, nix, docker, devcontainer, npm, and uv output; only mise has a contract test (`tests/contract.bats`).
- Linux runs (the BSD and GNU differences were read, not exercised), and jq 1.7.0 and 1.7.1 with `--args --`.
- Whether the tutorial site URLs resolve; the zsh completion inside a live zsh.
- The wrapping of `help:reflow`; narrow terminals beyond the floor of 60 in `flow:width`.
- Outside this scope: `~/.cache/kit/yaml` holds 29,387 entries on the reviewer's machine, and nothing prunes it.

## Components

Reviewed kittui's components at 2dad3552 (the `grid`, `table`, `tabs`, `shell`, `style`, `pane`, `form`,
`palette`, `syntax`, and `testkit` packages under `projects/tui/internal/`); each confirmed finding was
reproduced by a throwaway Go test in a copy of the module unless it says "by reading", and in that copy the
existing suites pass with `-race` and `go vet ./...` is clean.

1. Critical, bug. `projects/tui/internal/form/fields.go:288-293,258-266` and
   `projects/tui/internal/form/command.go:28-30`. A one-of row's choice is kept by the row's ordinal in the walk,
   so once an earlier row's choice changes it lands on another row, unchecked. Scenario: in the `kit packages`
   form on `remove` (`kit tui packages remove`, or chosen by hand), press ↑↑↑, ⇧⇥, then ␣ to choose the
   first usage line; `Chosen[1]=2` then belongs to `[--active | --inactive]`, and `n.Alternatives[2][0]` panics
   with an index out of range. In range it is silent: in `kit tools`, choosing the second line with `update` and
   then the first line makes the command line `kit tools --jq`, which nobody chose. Fix: key a choice by the
   group's path in the grammar, not a running count; drop the choices of rows that left. Confirmed.
2. Critical, bug. `projects/tui/internal/table/column.go:299-301` and
   `projects/tui/internal/shell/shell.go:430-436`. The header's column menu is drawn from `HeaderTop()`, which
   is -1 once no row shows, and `shell.Over` indexes `lines[top+k]` with no lower bound. Scenario: click a
   column's name while a read is out (Refresh, Watch, the marks' minute); the read answers with no rows, or with a
   refusal, and the next `View` panics with `index out of range [-1]`. Fix: `ColumnMenu.Box` gives no box when the
   header draws no name, as the heading case does at line 292; `Over` skips lines above 0. Confirmed.
3. High, bug. `projects/tui/internal/table/outline.go:60-65,231-281,337-391` and
   `projects/tui/internal/grid/grid.go:455-458`. `Opens` measures every row's columns again for each node, so
   `ShowMore`, `ShowLess`, `Folding`, and `Changes` are O(n²); the `enabled` of flow
   (`projects/tui/internal/flow/menu.go:120-128`) asks `Folding` and `Changes` for each fold line each time
   the tab menu's box is drawn. Scenario: with the tab menu open, 500 rows take 0.9 s a frame and 2,000 rows
   take 13.6 s (4,000 rows take 54 s for the four fold lines, with rows cached as flow keeps them); one `<` on
   2,000 rows takes 2 s. Fix: lay the widths out once per pass (`grid.Opens` over all rows at once), and cache
   the menu's enabled lines per outline. Confirmed.
4. High, bug. `projects/tui/internal/shell/search.go:38-77` and `projects/tui/internal/form/model.go:129-149`,
   with no handler anywhere. Nothing takes `tea.PasteMsg`, and Bubble Tea v2.1.1 turns bracketed paste on unless
   `View.DisableBracketedPasteMode` is set, which `shell.View` does not do (`cursed_renderer.go:140` in Bubble
   Tea). Scenario: paste a title into a form's text field, or text into the search or a preset's name, and nothing
   is typed; `grep -rn PasteMsg` finds none outside `testkit`. Fix: `shell.Type` takes a paste as typed text;
   every screen routes `PasteMsg` to the control that types. Confirmed.
5. High, bug. `projects/tui/internal/table/heading.go:45-67,124-138`. `NamesIn` looks for the field's id in the
   heading over the groups, which draws its label (`Panel.Label`), or one value's name while a list is flat.
   Scenario: with workspace/repos grouped by `by`, labelled `member by`, or with assignees flattened and grouped
   (`assignee`), the name is no stop and takes no click, so its menu is out of reach there; a click on the level's
   ▾ does nothing (two groups) or folds the rows' level instead of the first (one group). Fix: have `group` give
   each name's span as it draws it, instead of searching the drawn text. Confirmed.
6. High, bug. `projects/tui/internal/grid/panel.go:574-593`. Groups sort by `natural`, which calls `Bug` and
   `bug`, or `v01` and `v1`, equal, but headings are cut by exact value, so the values interleave. Scenario:
   the labels `Bug, bug, Bug, bug` across a workspace's members give four headings, two pairs with the same
   key, each counting 1; one fold and one selection mark act on both. Fix: break the ties of `natural` by the
   exact text in the compare of `group`. Confirmed.
7. High, bug. `projects/tui/internal/grid/panel.go:531,828-832,871,890,962` and
   `projects/tui/internal/grid/grid.go:175-177`. "A row of one cell is an aside" is how grid tells a data row
   from a heading, and `hide` runs before `FoldAway`. Scenario: hide every column but one (Hide in the column
   menu allows it); rows stay under a closed heading, the header is gone, a row has no heading over it, and space
   on a heading selects nothing. A table of one field never sorts or groups. Fix: tell rows from asides by
   `Aside`, not by the count of cells. Confirmed.
8. Medium, bug. `projects/tui/internal/grid/panel.go:778-790` and
   `projects/tui/internal/table/heading.go:145-156`. A level's fold is kept by depth (`LevelKey(depth)`), and
   nothing clears it when the groups change; flow has no write of `LevelKey` on `setPanel`. Scenario: group by
   state then who, close the level of who, and ungroup state; the rows' level is hidden though nobody closed it.
   Close level 0 on one field and group by another, and nothing shows. Fix: key a level by its field's id, and
   the rows' level by a name of its own. Confirmed.
9. Medium, bug. `projects/tui/internal/tabs/tabs.go:590-603`. `hit` gives the last tab for any column past
   the canvas's end. Scenario: with two tabs on a 120-cell screen, a click at column 100 on any of the
   bar's lines selects the last tab; where it is selected already, `OnSelected` is true and its menu drops.
   Fix: return -1 where `at >= l.width`. Confirmed.
10. Medium, inconsistency. `projects/tui/internal/form/model.go:171-193` and
    `projects/tui/internal/palette/palette.go:207-231`. The form and the palette take clicks on the menu bar
    alone, against rule (a), plain clicks for every action. Scenario: a click on a toggle, an option, a list's
    row, the command line, the drawn `‹ verbs`, or a verb in the palette does nothing, and neither list takes the
    wheel; the only `MouseWheelMsg` is the one in flow. Fix: `Layout` gives each field's lines, as it gives the
    focused one's; a click focuses and acts as ␣ or ⏎. Confirmed by reading.
11. Medium, inconsistency. `projects/tui/internal/table/column.go:226-232,199-222` and
    `projects/tui/internal/table/columns.go:145-151`. Moving a column has no click and no menu line (rule a), and
    its keys are named on a menu's keys bar. Scenario: a column moves by ⇧←→ in the column menu and by ⇧↑↓
    in the fields dialog only, and flow names ⇧←→ on the keys bar of a menu
    (`projects/tui/internal/flow/layers.go:105-108`), which rule (c) keeps to esc, ↑↓, and ⏎. Fix: two menu lines,
    Move Left and Move Right, with their keys at the right; a click target in the dialog. Confirmed.
12. Medium, bug. `projects/tui/internal/form/fields.go:317-334`. `opens` looks at an alternative's
    first node only, so a literal after leading optional groups selects nothing and is dropped.
    Scenario: `kit tui tools run` (`kit tools [--toolchain <name>] (install | update |
    path | run <command>…)`) opens the form on the first usage line, with the command line `kit tools`.
    Fix: skip optional nodes before the first required one. Confirmed.
13. Medium, bug. `projects/tui/internal/form/model.go:355-360,800-813`. `refreshExtension` runs
    `kit syntax <verb> --extend` inside `New` and `Update`, on the update loop; `kit.Runner.Extend`
    (`projects/tui/internal/kit/kit.go:167`) is `r.run`. Scenario: on a verb that extends, each move
    to another field with new answers makes the screen stand still for a kit run. Fix: a command and
    a message, as `fetch` is. Confirmed by reading.
14. Medium, bug. `projects/tui/internal/form/model.go:605-660,662-689`. A flip of a list with states starts a
    read on each ␣, none superseding another, and `others` leaves the list's own values out of `asked`.
    Scenario: an older answer that lands last is not stale and replaces newer rows, and the note of a list's
    answer goes under whichever field has focus at that moment. Fix: `source.Load`, which the spec says the
    screens share, or a generation per key. Plausible; by reading.
15. Medium, missing. `projects/tui/internal/table/catalog.go:105-119`. `ReadCatalog` checks nothing that a table
    then indexes by. Scenario: with no preset named Default, `Views.Chosen` panics
    (`projects/tui/internal/table/views.go:45`); a field with no renderings makes `Automatic`
    (`projects/tui/internal/table/auto.go:47`) and `DrawnBy` (`projects/tui/internal/table/render.go:77`) panic;
    an unregistered rendering calls a nil func (line 91 of the same file); and a zero `Table` panics on `Picked`
    (`projects/tui/internal/table/view.go:375`). Fix: validate in `ReadCatalog`; add a `NewTable`. Confirmed.
16. Medium, bug. `projects/tui/internal/palette/palette.go:523-528`. A verb's page longer than the window says
    how many lines it hides, and no key or click shows them. Scenario: the page keys move the verbs, and the
    model keeps no offset for the page, so the hidden lines of a long page cannot be reached. Fix: an offset,
    paged while the pages show. Confirmed by reading.
17. Medium, bug. `projects/tui/internal/form/command.go:74-75,85-93`. The emitter writes a literal wherever it
    walks, and a value as it is. Scenario: `[file <path>]` with no path gives `kit x a file`; `[now]` is a text
    field whose value is never written (no shipped grammar has either shape). A value `--force` for `--title` or
    for an operand gives `--title --force --force`, with nothing to tell kit it is a value. Fix: write a literal
    inside `[ ]` only with an answer beside it; write `--name=value` or `--` for a value with a dash. Confirmed
    for the output; plausible for kit's reading of it, which was not run.
18. Medium, bug. `projects/tui/internal/shell/menubar.go:64-70` and
    `projects/tui/internal/shell/search.go:94-96`. The search field is cut at its end with `pane.Clip`, where the
    form uses `pane.Tail`, and the line is `2 + (width-1)` cells once `SearchAt` is 0. Scenario: with 50
    characters typed on a 40-cell screen, the caret and what is typed last are gone, and the line is 41 cells, so
    it wraps. Fix: `Tail`, and the two cells counted. Confirmed.
19. Low, inconsistency. `projects/tui/internal/grid/panel.go:37-38,1017` and
    `projects/tui/internal/tabs/tabs.go:188-196`. Circled glyphs are on the screen, against rule (d), no circled
    glyphs. Scenario: the radio is ⦾ ⦿ (U+29BE and U+29BF, circled bullets), the filter mark is ⊘ (U+2298, of the
    block of ⊜ and ⊖), and the tabs' numbers are ➀ ➊ (circled dingbat digits). Fix: the owner's call per glyph;
    ⊘ and ⦾ ⦿ are the likely misses. Confirmed by reading.
20. Low, inconsistency. `projects/tui/internal/shell/dropdown.go:58-66` against
    `projects/tui/internal/shell/placed.go:19-31`. The two overlays read a click on their own
    border differently. Scenario: a click on a menu's border closes it; a click on a dialog's
    border does nothing. Fix: one hit test. Confirmed.
21. Low, bug. `projects/tui/internal/grid/panel.go:201-231`. `SortKeys` keeps a gathered list's sort "for when it
    stands once per value again", and `sortedBy` wipes it. Scenario: the next turn of any column's sort wipes
    every field's kept sort. Fix: `sortedBy` leaves the lists it did not list. Confirmed.
22. Low, bug. `projects/tui/internal/table/columns.go:65,194-203`. A label wider than 16 cells, or 14 with ⊘, is
    not clipped. Scenario: in the fields dialog the sort and group cells of such a field stand right of where a
    click is read; no shipped label is that long. Fix: clip the label. Confirmed.
23. Low, inconsistency. `projects/tui/internal/form/model.go:877,900-905,969-976`. The form writes key hints
    under its fields, where the rule of shell is that a screen never writes its hints. Scenario: ctrl-n,
    which takes what is typed, is named there alone, as `ctrl-n`, and never on the status bar. Fix: the
    field's segment names them. Confirmed by reading.
24. Low, bug. `projects/tui/internal/palette/palette.go:218-226`. The click on the workspace menu is tried before
    the open recent dropdown. Scenario: with the recent dropdown open, a click on the workspace menu leaves both
    open. Fix: try the open dropdown first, as the keys go. Plausible; read, not run.
25. Low, bug. `projects/tui/internal/tabs/tabs.go:565-580,164`. `StaleAt` answers on a cell that
    draws no dot, and the keys bar leaves a key out. Scenario: `StaleAt` answers on the blank cell
    before ▹, where no dot is drawn (at widths 47 and 89); the keys bar says `1-9` and never names 0
    for the tenth tab. Fix: none given. Confirmed.
26. Low, bug. `projects/tui/internal/grid/grid.go:127-148` and `projects/tui/internal/grid/panel.go:966-980`.
    `natural` misorders negative and fractional numbers, and sorts and groups place a missing value differently.
    Scenario: `natural` puts -5 before -10 and 1.5 before 1.25, and calls two runs of over 20 digits equal; a sort
    puts no value first ascending, where groups put none last (#90). The shipped number fields are whole and not
    negative, so this waits for a field that is not. Fix: none given. Confirmed.
27. Low, bug. `projects/tui/internal/form/render.go:434-452` and
    `projects/tui/internal/form/command.go:31-35,113-125`. A narrow form overflows, and a wildcard line is
    mishandled. Scenario: under 44 cells a form's lines run past the edge (seen at 36 and 40), as `pane.Wrap` is
    never narrower than ten. A wildcard line without `--` is dropped without a word, `--bare` becomes `--bare=`,
    and `--title=x` there sets the named option. Fix: none given. Confirmed.
28. Low, inconsistency. `docs/superpowers/specs/2026-11-07-kittui-components-design.md` against the code. The spec
    and the code differ in five places. Scenario: `shell.Control` is not there; `Layers` serves flow alone, while
    the palette, the form, welcome, and tutorials each repeat the chain of open menus in keys, clicks, hints, and
    drawing (finding 24 is one result); `Notice.Fading` (`projects/tui/internal/shell/notice.go:37-43`) waits on
    `tea.Tick`, not `source.Clock`; the form reads kit by itself (finding 14); and the spec names four screens
    where `projects/tui/internal/shell/shell.go:34-45` lists ten. Fix: none given. Confirmed.
29. Low, missing. `projects/tui/internal/grid`, `projects/tui/internal/table`, and `projects/tui/internal/form`.
    Some code is dead, and the goldens pin the wrong thing. Scenario: no caller outside tests exists for
    `grid.Show`, `All`, `Mine`, `Others`, `Panel.Showing`, and `Panel.Shown`; for `table.ItemViews`, `FoldAt`,
    `DrawsGlyph`, `Registered`, and `ColumnMenu.LineOf`; or for `form.RenderStatic`. The form's 34 goldens are
    `RenderStatic`, plain and without focus, so none pins the live form; they also pin the first option's
    description beside a row with nothing chosen (`projects/tui/internal/form/testdata/packages.golden`:
    `--active   --inactive   only the active packages`). Fix: none given. Confirmed.
30. Low, inconsistency. `projects/tui/internal/table`, `projects/tui/internal/grid`, and
    `projects/tui/internal/shell`. Several things tie the table to Flow, against rule (h), a table that other
    screens can reuse. Scenario: `table.Folding` switches on the tab menu's words
    (`projects/tui/internal/table/outline.go:340-349`); the catalog and the presets file say `stages`
    (`projects/tui/internal/table/catalog.go:107`, `projects/tui/internal/table/store.go:18`);
    `Opening = "Default"`; `grid.Panel` holds `Owner`, `Me`, the search's `Query`, `Focused`, and `Recall`; grid
    imports shell for one glyph (`projects/tui/internal/grid/panel.go:34`); and
    `projects/tui/internal/shell/shell.go:34-88` lists the apps. Fix: none given. Confirmed.

### Holds (Components)

- The grid's `Draw`, `Header`, `HeaderAt`, `HeaderName`, `LinkAt`, and `Opens`: no panic and no line wider than
  the pane at widths 0 to 70 with CJK, ZWJ emoji, combining marks, and styled text, grouped or not.
- The tabs' `View`, `Update`, `StaleAt`, `OnSelected`, and `MenuAt`: no panic at widths 0 to 90; the
  menu's glyph is where `MenuAt` says at every scroll.
- The shell's key hints, kit menu, `Dialog`, and `MenuBar.Line`: no panic at widths 0 to 30.
- `grid.Draw` and `Panel.Filter` are linear: 14 ms and 7 ms at 5,000 rows.
- A disabled menu line does not fire by its shortcut or by a click (`Dropdown.choose`, `Shortcut`).
- `SelectedRows` gives only rows the filter lets through.

### Not reviewed (Components)

- The pill colors and `ink` of style; the code of testkit itself; the edges of `pane.Frame` beyond its tests.
- Wide runes in the form, the palette, and the menus' boxes; only grid and tabs were run with them.
- Presets: round trips of `Preset.Panel` and `PresetOf`, and Save, Rename, and Delete through a host.
- The popover's cursor after a reload shortens its list; `FilterValues` asked several times a frame.
- What kit does with a value that starts with a dash; ⇧ with an arrow, and
  each glyph's width, in Terminal.app and iTerm.
- `-race` on flow; how welcome and tutorials use shell.

## Apps

Reviewed kittui's apps, screens, and the bridge to kit at 2dad3552 (`projects/tui/cmd/kittui` and the `flow`,
`screens`, `app`, `kit`, `source`, `welcome`, `tutorials`, and `transcript` packages under `projects/tui/internal/`);
each confirmed finding was reproduced by a throwaway Go test in a copy of the tree unless it says "from the code", and
in the real tree `go vet ./...` is clean and `go test -cover` passes on the scoped packages (flow 98.5%, welcome
99.9%, screens 94.4%, kit 97.8%, tutorials 95.4%).

1. Critical, bug. `projects/tui/internal/flow/flow.go:1063-1098`. A reload decodes into the slice the stage
   already holds, so a `null` in the new answer keeps the value of the row that stood at that index before.
   `decodeRows` calls `json.Unmarshal(data, &s.issues)` (and `&s.under`, `&s.pulls`, `&s.runs`, `&s.rels`,
   `&s.repos`) without clearing the slice; only incoming, desk, worktrees, and the workspace rows are reset first.
   `encoding/json` reuses the elements of an existing slice, and a JSON `null` into a non-pointer Go field "has no
   effect"; kit prints `null` for exactly these: the `branch` of backlog (`kit/modules/jq/backlog.jq`), the
   `worktree`, `source`, `organization`, `repository`, and `pr.fork` of underway (`kit/modules/jq/underway.jq`),
   the `jobs[].step` of builds, and the `checks.failing` of changes. Scenario: backlog shows #5 (branch `5-five`)
   and #7 (no branch); #5 is closed, and Refresh, a watched reload, or the refresh after any action answers
   `[#7 branch:null]`; #7 then draws `🌿 5-five`, with the branch and the source that were those of #5. The same
   happens at every start: the kept copy (`--cached`) is decoded first, then the fresh answer inherits the kept
   row's fields, so underway can show a worktree and a fork that belong to another issue (#9 with the worktree
   `/w/1-one` and the fork `cy`). Wrong data is shown as fresh, with no stale dot. Fix: decode into a fresh value
   and assign (`var issues []issue; …; s.issues = issues`), for every stage. Confirmed.
2. High, bug. `projects/tui/internal/kit/kit.go:562-574` (`output`) and `projects/tui/internal/kit/kit.go:306-318`
   (`Status`). A stage that prints its document and exits 1 is shown as an empty refusal. kit's contract
   (`flow:incoming` and `flow:stage-json` in `kit/modules/bash/flow.sh`: "A stage that fails prints what it
   printed, as incoming does for every remote when one fails") is stdout plus exit 1 when one remote's fetch
   failed; `output` returns `nil` on any non-zero exit, and git's line was captured into the JSON, so stderr is
   empty. Scenario: two remotes, with the fork unreachable (offline, VPN). The tab of incoming shows `!` and
   `❌ kit flow incoming exited 1 with nothing on stderr`, the primary's commits and git's own line are dropped,
   and the red per-remote line of `memberIncoming` (`projects/tui/internal/flow/rows.go:696`) is unreachable over
   a repository. `fetched()` is false, so nothing reloads. Fix: `Status` keeps stdout on an `ExitError`, and the
   stage decodes it when it is a document. Confirmed.
3. High, bug. `projects/tui/cmd/kittui/main.go:53,75`, with
   `projects/tui/internal/screens/screens.go:126-132,309-321`. A form's command line is exec'd where kittui
   started, not where the screen that built it reads. The root moves its own runner to the workspace's checkout
   (`m.runner.Dir = msg.ws.Checkout`) and opens every later screen with it; `main` keeps the runner it made at
   start, and `execKit(r, …)` does `os.Chdir(r.Dir)`. The root has no accessor for its runner, and Help is never
   remade on a workspace pick, while its title names the new workspace. Scenario: run `kit flow` in repository A,
   pick repository B from the 📚 menu, then `=`, Help, and `commit` (or `submit`, `tidy`, `work 7`); the form's
   choices are read in B, and the verb runs in A. From a subdirectory the same split is there at start: screens
   read the checkout's top, and exec runs in the subdirectory. Fix: the form's result carries its runner (or the
   root exposes it), and `finish` execs with that. Confirmed.
4. High, bug. `projects/tui/internal/welcome/welcome.go:698-707`. PgUp, PgDn, Home, or End panics on a step whose
   search hides every check. `page` indexes `rows[min(max(at+n,0), len(rows)-1)]`; `down` guards `len(rows)==0`,
   `page` does not, and leaving the search sets `inPane = true` unconditionally (lines 492-494). The panic is also
   reachable while the checks are refused. Scenario: `kit welcome`, `2`, `/zzzz`, ⏎, then PgDn gives
   `index out of range [-1]`, and kittui dies (exit 1, with `root.Close()` skipped on that path,
   `projects/tui/cmd/kittui/main.go:61-64`); all four keys panic, in a package at 99.9% coverage. Fix: return when
   `len(rows)==0`; give the pane the keyboard from the search only when it has rows. Confirmed.
5. High, bug. `projects/tui/internal/welcome/welcome.go:351-361,385-396,611-617`. `n` or ⏎ runs the fix of a check
   the step does not show. `find` returns index 0 when the step has no rows, and `choosing()` then judges
   `m.checks[0]`, whatever step it is of. Scenario: on the toolchains step the search hides everything, then ⏎;
   the first check kit printed is a failed one with a fix, so `n` runs `kit welcome --fix <that id>`
   (`welcome --fix bash` in the run), with the terminal handed over, for a row that is nowhere on the screen. Fix:
   `choosing` and `showOnly` require `slices.Contains(m.rows(m.step), m.cursor)`. Confirmed.
6. High, bug. `projects/tui/internal/screens/screens.go:312`, `projects/tui/internal/flow/apps.go:76-100`, and
   `projects/tui/internal/flow/flow.go:191-205`. The words of `kit flow` reach the first app only. `screens.open`
   calls `flow.OpenApp(m.runner, to, nil)`, while `--no-fetch`, `--mine`, `--others`, and `--owner=` live on the
   first app's model. Scenario: `kit flow --no-fetch` opens Issues, which has no incoming, so the flag does
   nothing; `=` and Repositories then fetches every remote. After `kit flow --mine`, Changes from the menu shows
   everyone's. A workspace pick remakes the shown app without the words too, and without the maximized tab. The
   shipped test of these words asserts only the first app. Fix: the root keeps the words (or the parsed `NoFetch`
   and `Filters`) and gives them to every app it opens. Confirmed from the code.
7. Medium, bug. `projects/tui/internal/flow/act.go:152-169,184-187`. Add to Pull Request… can never be enabled.
   `mine()` reads the `changes` stage of this screen, and under #138 Issues has backlog, underway, and two
   planned tabs. Scenario: in Issues the line is never enabled for any login. Rule (b) says a menu line that can
   never apply is omitted; here a working feature is lost. Fix: read the caller's pull requests from kit (or
   from the Changes app through the root). Confirmed.
8. Medium, bug. `projects/tui/internal/flow/origin.go:17-45,88-97` and
   `projects/tui/internal/flow/rows.go:520-525,559,604,854,877`. Every issue and pull request link is guessed from
   `origin`. kit prints no `url` for an issue or a change (`issue` and `document` in `kit/modules/jq/forge.jq`),
   so `URL` is always empty and the link is `https://<origin host>/<last two path segments>/…`, in GitLab's shape
   only where the host's name contains "gitlab"; `ReadOrigin` names `origin`. Scenario: where the primary is not
   origin (`primary: upstream` in `kit.yml`), links open #N of the fork; a GitLab subgroup `group/sub/repo` gives
   `https://gitlab.com/sub/repo/-/issues/7`; a GitLab at `git.example.com` gives `/pull/3`; and the `organization`
   and `repository` of backlog (where the tracker holds the issue) are ignored. Fix: kit prints each row's page
   (as builds do, and as `kit workspace` does under `links`), and kittui stops building URLs. Confirmed.
9. Medium, bug. `projects/tui/internal/flow/rows.go:637-649`. The link of underway to a GitLab merge request never
   finds its row on Changes. `pullKey` falls back to `!N` when `u.PR.Source == "gitlab"`, but the `pr` of
   `kit/modules/jq/underway.jq` has no `source` and no `ref`, and Changes keys the row by `ref`, `!3`. Scenario:
   the link leads to `kittui:changes/#3`, and its row is keyed `!3`. Fix: `pr.ref` in that module, used as the
   key; this is the "References" decision of the entities spec. Confirmed.
10. Medium, bug. `projects/tui/internal/flow/flow.go:1338-1350`. A jump to a row that a closed group folds away,
    or that the search or a filter hides, lands nowhere and says nothing. `land` searches `m.rows(i)`, the rows
    after `FoldAway` and the panel, and a miss clears `landing` once the stage is loaded. Scenario: a link to a
    folded or a filtered row leaves the cursor unchanged, with no notice. Fix: search `grid.Every` and show the
    headings over the row (Show Selected does this); for a row the panel hides, say so in the notice, or clear the
    search as the `follow` of welcome does. Confirmed.
11. Medium, bug. `projects/tui/internal/flow/flow.go:973-996,1219-1227`. A reload keeps the cursor's index,
    not its row; `take` only clamps, and watched stages reload by themselves. Scenario: the cursor is on #7,
    a new issue sorts in above, and the cursor is on #5; space selects #5, and New… then reads
    `cursorMember()` of that row. welcome keeps its cursor by key; flow does not. Fix: remember the cursor's
    key before the decode and find it after. Confirmed.
12. Medium, bug. `projects/tui/internal/source/load.go:62-99`. A refresh does not end a stage's first load: the
    first read runs in `env.Ctx` with no cancel of its own (`l.stop == nil`), so `Again` cannot stop it.
    Scenario: Repositories opens, the `git fetch` of incoming is slow, and `r` starts a second
    `kit flow incoming` that fetches the same remotes beside the first (the first load ran to its end, 5 s,
    with a nil error). The README says "a read started again supersedes and ends the one before", and the
    shipped test supersedes a restart with a restart, never the first load. Fix: `NewLoad` and `Read` make the
    first read's context cancellable too. Confirmed.
13. Medium, bug. `projects/tui/internal/screens/screens.go:270-284,309-313,139-146` and
    `projects/tui/internal/flow/apps.go:88-93`. Opening an app runs kit on the update loop, and one verb's
    refusal closes the whole app. `show` and `answered` call `OpenApp`, which runs `kit flow --stages` and
    `kit workspace --stages` synchronously (a 30 s limit each). Scenario: the screen freezes on every first
    switch and on every cross-app link. Repositories asks `kit flow --stages` only for incoming; where that
    refuses (a `primary` that names no remote, a bad `flow` setting), `kit workspace` cannot open at all, though
    six of its seven tabs read git alone. Fix: open at once with the stages loading as a source; a verb that
    refuses costs its own tabs. Confirmed from the code.
14. Medium, inconsistency. `projects/tui/internal/flow`. Parts of the one
    Flow picture are left behind under #138. Scenario:
    - `projects/tui/internal/flow/flow.go:957-969`: `moved` (backlog, desk, worktrees, underway) reload after
      the fetch of incoming only when they share a screen with incoming. No app has both, so the reload never
      runs, and nothing fetches unless Repositories is opened.
    - `projects/tui/internal/flow/rows.go:613-627`: `here()` asks `m.find("desk")`; no app draws desk, so the
      desk's checkout draws `🌿 here`, never `💻 here` as `kit flow --text` and the underway legend say.
    - Dead in every app: the stages `desk` and `worktrees` (`known`, `deskRows`, `worktreeRows`,
      `checkoutRow`, `pullOf`, and their catalog entries, legends, and goldens), `mine()`,
      `Options.App == ""`, and the "kit flow" of `appLabel`.
    - Most of the tests of flow (`options()`, `eight`) drive that eight-stage picture, which `main`
      and `screens` never open: the 98.5% is largely of a configuration that does not ship, and the
      app paths (#138 and #139) have two test files.
    - `projects/tui/internal/flow/flow.go:973-987`: a refusal is cleared by `restart` alone; a later `again`
      that answers leaves `!` and the old line. This is reachable only through `reloads()`, so it is latent.

    Fix: none given. Confirmed.
15. Medium, bug. `projects/tui/internal/flow/flow.go:219-233` and `projects/tui/internal/source/poll.go:93-101`.
    One verb's marks failing discards the other's and can turn the poll off. The marks' read joins the errors of
    `kit flow --marks` and `kit workspace --marks`; `Poll.Take` drops the value on any error, and a first failure
    sets `off` until Refresh All. Scenario: in Repositories with the forge unreachable, `kit flow --marks` fails,
    so the six git tabs never go stale and Watch never reloads them, silently. Fix: return what answered; fail
    only when every verb did. Confirmed from the code.
16. Medium, bug. `projects/tui/internal/flow/flow.go:193-203` and `kit/bin/kit-flow:333`. Every word kittui does
    not know becomes a filter of every forge stage. Scenario: `kit flow --jq` on a terminal is not excluded from
    the kittui branch (the `--workspace` branch at lines 275-277 does exclude it), so kittui opens with `--jq` as
    a filter, and backlog and underway both refuse: `kit syntax flow --match -- backlog --json --jq` answers
    `--json cannot be combined with --jq`. `--fresh` passed this way rides on every load and on `--cached`, and a
    mistyped or second stage name (`kit flow underway changes`) is a filter too. Fix: `kit/bin/kit-flow` excludes
    `--jq`; `open()` keeps only the owner flags as filters. Confirmed from the code and `kit syntax`.
17. Medium, inconsistency. `projects/tui/internal/flow/fields.json` against `projects/tui/internal/flow/rows.go`
    and `projects/tui/internal/flow/workspace.go`. The `record` and `path` of the catalog are never read, against
    rule (f), a table driven by its catalog. `table.Catalog.Record` and `Field.Path` are decoded and asserted
    non-empty (`projects/tui/internal/flow/fields_test.go`), and nothing evaluates them. The stages of `kit flow`
    are built by hand: one Go struct, one slice on `stage`, one `xxxRows` builder, and one case in each of
    `decodeRows`, `records`, `empty`, `expected`, and `known` per stage, with renderers that type-assert Go
    payloads (`v.Of.(checkout)`, `*checks`). Only the stages of `kit workspace` are data-driven, and by field id,
    not by path. Scenario: these stand in the way of lifting the table out:
    - `incoming` has no catalog entry: no columns, presets, sort, group, or filter.
    - `flow.Model` is the app shell (menus, workspace, legend, quota, marks, actions) and the `table.Host` at
      once, for "the selected stage"; tutorials cannot host a table without it.
    - Actions are tied to the name `backlog` and parse row keys (`refOf` splits on the last space).
    - Tab glyphs and colors are Go maps by stage name
      (`projects/tui/internal/flow/glyph.go`), and planned tabs have neither.
    - Declared but typed otherwise: the `upstream` of workspace/branches and the `track` and `base` of
      workspace/worktrees are `"type": "number"` holding `[ahead, behind]`; the `worktree` of branches and the
      `worktrees` of commits are lists declared as sortable `text` with no `one`. Emitted but not declared:
      the `host`, `path`, `checkouts`, and `error` of repos, and the `ref` and `createdAt` of backlog. Paths
      that are not paths: `changes:.ref` and `.here, .local, .pr.author`.
    - No test holds the flow fixtures to what kit prints (the syntax fixtures have `tests/syntax.bats`):
      `projects/tui/internal/flow/testdata/backlog.json` has no `ref` or `createdAt`.

    Fix: none given. Confirmed.
18. Medium, inconsistency. `projects/tui/internal/flow/flow.go:1236-1276,750-763` and
    `projects/tui/internal/flow/legend.go:44-49`. Some actions have one way in, against
    rule (a), plain clicks for every action. Scenario:
    - A link to another tab or app (`kittui:`) is followed by a click on its glyph only; no key does it
      (`jump` is called from `click` and from `JumpMsg` alone). `o` opens pages (`://`) only.
    - Selecting one row is space or ⇧↑↓ only; a click moves the cursor and never selects. The menu has Select
      All and Clear Selection, and nothing for the row under the cursor.
    - The legend opens by `?` or F1 only; it has no menu line and no click target.
    - In tutorials, a click on a card opens the browser at once; there is no click that only moves the cursor.

    Fix: none given. Confirmed from the code.
19. Low, inconsistency. `projects/tui/internal/flow/menu.go:47-58,121-142` and
    `projects/tui/internal/flow/watch.go:55-66`. Lines that can never apply are drawn dim, against rule (b),
    which omits them. Scenario: on a planned tab the menu shows Refresh, Watch, Show More, Show Less, Select
    All, and Open Selected, all dim forever. On People, Trust, and Platform the app menu's Refresh All is live
    and does nothing, and Watch All and Stop Watching All are dim forever. Add to Pull Request… is another
    such line (finding 7). Fix: none given. Confirmed.
20. Low, inconsistency. `projects/tui/internal/flow/legend.go:137-141,227` and `Name()` in
    `projects/tui/internal/flow/host.go`. The legend has gaps, and internal names reach the screen. Scenario:
    - underway draws a pull request's state glyph (🔨 🔥 💬
      🏁); the states hang under the legend of changes only.
    - The legend's title, and `Name()`, which titles the table's dialogs, give the stage's internal id:
      `workspace/repos legend`, `planned/evidence legend`; the tab says `repos`.
    - Menu glyphs are Go constants outside `kit/modules/jq/glyphs.json` and reuse legend glyphs with another
      meaning: 🛑 for Stop Watching and for cancelled, 📝 for New… and for a draft release, 👥 for Assign to…
      and for the count of assignees (`projects/tui/internal/flow/act.go:27-34`,
      `projects/tui/internal/flow/watch.go:25-26`).
    - The worktrees column of repos draws ❗ for a detached worktree
      (`projects/tui/internal/flow/repos.go:138-142`), and the legend of
      repos has no ❗. This part is plausible, not run.

    Fix: none given. Confirmed except the last part.
21. Low, inconsistency. `projects/tui/internal/grid` and `projects/tui/internal/tabs`, pinned by
    `projects/tui/internal/flow/panel_test.go:575`. Circled glyphs are on the screens in scope, against rule (d),
    no circled glyphs. Scenario: `grid.Filtered = "⊘"` in the fields dialog (asserted in that test and in
    `projects/tui/internal/flow/testdata/columns.golden`) and the tab numbers `➀…➉` and `➊…➓`
    (`tabs.Marker`) on every flow, welcome, and tutorials screen. Both belong to the packages of the Components
    review; they are noted because the goldens of flow pin them. Fix: none given. Confirmed.
22. Low, bug. `projects/tui/internal/flow/flow.go:1055-1103,1156-1189`. `null` or nothing on stdout with
    exit 0 is an empty stage. Scenario: `null` for backlog or changes gives "no open issue…", count 0;
    nothing for incoming gives an empty stage; `null` for incoming gives one zero member; over a group,
    `{"rows":null}` is empty too. kit is unlikely to print these, but it is fail-open. Fix: `shape`
    refuses anything but the expected opener. Confirmed.
23. Low, bug. `projects/tui/internal/shell/browse.go:42-49`, called from
    `projects/tui/internal/flow/flow.go:519-526` and from `open` in
    `projects/tui/internal/tutorials/tutorials.go`. The opener's failure is never seen: `cmd.Start()` runs with no
    `Wait`. Scenario: `open ""` or a bad URL exits non-zero unseen, and the child is never reaped; a tutorial
    whose catalogue entry has no `url` says `Opened <title> in the browser`. Fix: wait in the command's goroutine;
    a tutorial without a link is not openable. Confirmed from the code.
24. Low, missing. `projects/tui/internal/tutorials/tutorials.go:98-110,300-330` and
    `projects/tui/internal/welcome/welcome.go:232-290`. Some keys and words do nothing. Scenario:
    tutorials has no Home, End, PgUp, PgDn, ←, or → (#98 covers welcome only); neither welcome nor
    tutorials takes the mouse wheel, which flow does. `kit tutorials <unknown>` and `--tag=<audience>`
    (the `=` form) are ignored without a word; an unknown tab name after `kit flow` becomes a filter
    instead (finding 16). Fix: none given. Confirmed from the code.
25. Low, bug. `projects/tui/internal/flow/flow.go:346-350,668-674`. A failed `kit flow --me` is swallowed and
    never asked again: `asked` stays true, and the error is dropped. Scenario: `@me` in a filter matches nothing,
    with no notice, for the session. Fix: none given. Confirmed from the code.
26. Low, bug. `projects/tui/cmd/kittui/main.go:61-64`. The error path skips `root.Close()` and the exit code.
    Scenario: after a panic, SIGTERM, or SIGINT from outside, kit children (each in its own process group,
    `projects/tui/internal/kit/kit.go:379-385`) are left running, an action among them; an outside SIGINT ends 1
    with an ❌ line, not 130. `screens.ask` and every `Runner.run` use `context.Background()`, so quitting never
    ends them either. Fix: none given. Confirmed from the code.
27. Low, bug. `projects/tui/internal/flow/act.go:376-397` and `projects/tui/internal/kit/kit.go:348-364`. An
    action can run twice at once, and has no limit. Scenario: nothing marks an action as running, so `a` twice
    starts two `kit assign`, and a menu action can start beside it. `Act` has the screen's context alone, so a kit
    that hangs leaves `→ kit work …` until quit. A refusal with no output shows a bare ❌ (`refusal("")`, with
    `ExitError.Error()` empty; `screens.refusalLine` then shows nothing at all). The selection is cleared on
    success even where it changed while kit ran. Fix: none given. Confirmed from the code.
28. Low, bug. `projects/tui/internal/screens/screens.go:121-132`. `--workspace <host/path>` leaves the first app
    and the later ones reading differently. Scenario: a repository given is not "moved", so the first app keeps
    `--workspace host/path` (rows keyed `member #N`), and apps opened later read without it (rows keyed `#N`); a
    cross-app link from the first never finds its row. Fix: none given. Plausible.

### Holds (Apps)

- `go test -race` over the scoped packages in the real tree passes with no race reported: `cmd/kittui`, flow (319
  s), screens, app, kit, source, welcome, and tutorials. `transcript` was not in that run.

### Not reviewed (Apps)

- `projects/tui/internal/app` and `projects/tui/internal/transcript` beyond a read; the palette and form paths
  (`Choices`, `Extend`, `Match`) and how they behave on a slow or failing kit.
- The migration of `presets.json` in `projects/tui/internal/flow/views.go` (corrupt or foreign-version files), and
  `column.go`, `heading.go`, and `labels.go` in the same package.
- Welcome over a group (the composition in `projects/tui/internal/welcome/members.go`, fixes run in each checkout
  in turn) and the markdown of `projects/tui/internal/welcome/page.go`.
- kit's side of the last picture (#88: `flow:kept` keys by checkout and arguments, cksum) beyond reading it.
- Real-terminal behaviour: the terminal's restore after a panic, ⌃C during `tea.ExecProcess`, and iTerm clicks.

## Infrastructure

Reviewed the packages under `kit/packages/` and the repository's infrastructure (`bin/release`,
`bin/ci`, `bin/tutorials`, the workflows under `.github/`, the bats suites under `tests/`, and the
tutorials site under `projects/tutorials/`) at 2dad3552; experiments ran in a scratch clone and a
scratch consumer repository, two parts were reviewed by sub-reviewers (their findings say so), and the
reviewer re-ran findings 1, 2, 5, 6, 7, 21, 22, 23, and 28.

1. Critical, bug. `bin/release:114`. `pack` calls `release:digest`, which does not exist (the function is
   `file:digest` as of 96eeb70a), so the published `.sha256` has no digest, and `pack` exits 0 all the same.
   Scenario: tagging `release/0.1.0` publishes a `kit-0.1.0.tar.gz.sha256` that holds `"  kit-0.1.0.tar.gz"` and
   no digest; `shasum --check` at line 93 of every consumer's `.github/workflows/kit-ci.yml` fails, and
   `kit update` (`kit/modules/bash/release.sh:118`) refuses the release. In the clone,
   `bin/release pack release/9.9.9-rc.1` printed `release:digest: command not found`, then `✅ packed`, and
   `shasum` on the result said "no properly formatted SHA checksum lines", exit 1. `tests/release.bats:59` only
   greps for the file name, so all 4 tests pass. The published file of rc.7 has a digest, so the break is newer
   than rc.7. Fix: call `file:digest`, fail on an empty digest, and assert 64 hex digits in the test. Confirmed.
2. High, bug. `tests/underway.bats:199`. The suite fails at HEAD, so the gate is red. Scenario: the legend of
   `kit help flow` reads "…this desk has it: under underway…" (`kit/bin/kit-flow:157`), and the test expects a
   line the legend does not have; `bats tests/underway.bats` in the clone prints `not ok 11`. The last 8 CI runs
   are also red, on a pushed commit 165 behind HEAD. Fix: update the assertion. Confirmed.
3. High, missing. `kit/modules/bash/tutorials.sh:31`. Nothing builds, publishes, or serves the tutorials site, so
   every URL that `kit tutorials` opens is dead. Scenario: `kit tutorials kit-tutorials` opens
   `https://empowerite.github.io/kit/tutorials/main/kit-tutorials/?kit=unreleased`, which answers 404; a release
   points at `kit.empowerite.dev`, which has no DNS. `has_pages` is false, there is no Pages workflow, and lines
   73–74, 88–89, 106–110, 118–121, and 157–159 of the tutorials spec are unbuilt (the deploy on a push to
   main, the per-release line, the pre-release copy and its retire, and the changelog check in `kit release`).
   Fix: build part 2 before a release ships the verb. Confirmed by a sub-reviewer.
4. High, inconsistency. `README.md:79`. `brew install empowerite/tap/kit` cannot work. Scenario:
   `gh api repos/empowerite/homebrew-tap/contents` answers "This repository is empty", and nothing
   in `bin/release` or `.github/workflows/release.yml` writes a formula. Fix: publish the formula
   from the release, or drop the line. Confirmed.
5. Medium, bug. `kit/modules/bash/git.sh:36`, with the template of the `github/ci` package. On a runner
   `refs/remotes/origin/HEAD` does not exist, so the default branch falls back to `main`, and a repository whose
   default is anything else fails `kit policy` on every pull request. Scenario: with the default branch `trunk`
   and a checkout as actions/checkout fetches it, `kit policy` prints "the fork from origin/main is not here…
   fetch it with full history (fetch-depth: 0…)", exit 1, though the history is full; after
   `git remote set-head origin trunk` the real refusal appears. Fix: read `GITHUB_BASE_REF` or the event's default
   branch on a runner, or set the head in the job. Confirmed.
6. Medium, bug. `.github/dependabot.yml:18`. Dependabot bumps the installed `.github/workflows/kit-ci.yml`, never
   the template, so its pull request always fails `kit check`; the same holds in every consumer that runs
   Dependabot on actions. Scenario: bumping the SHA of mise-action in the clone gives `kit-ci.yml differs from the
   pinned release; run: kit update`, exit 1. mise-action is already a major version behind (v5.2.7). Fix: ignore
   `kit-ci.yml` in Dependabot and bump the template by release, or have `kit check` accept a newer pin. Confirmed.
7. Medium, bug. `bin/release:106,136`. `pack` copies the working tree, not the tag, and `publish` never
   checks that the tag exists or that HEAD is it. Scenario: a desk publishes `release/0.1.0-rc.8` with an
   uncommitted edit and a stray `kit/bin/x.orig`, and both ship (the tarball from the clone holds
   `bin/STRAY.orig` and the edit); with the tag unpushed, `gh release create` makes it at the default
   branch's head (by reading: there is no `--verify-tag`). Fix: pack from `git archive <tag>`, and pass
   `--verify-tag`. Confirmed for `pack`; plausible for `gh`.
8. Medium, inconsistency. `bin/release:158`. Retiring deletes pre-releases that consumers pin. Scenario: the
   ruleset requires linear history (its rules include `required_linear_history`), so merging this branch leaves
   rc.6 and rc.7 unreachable, and the branch delete retires both; a consumer on rc.7 gets a 404 in the kit-on-path
   step, and `kit update` says "could not be fetched" (`kit/modules/bash/release.sh:106-110`). Fix: keep a
   pre-release until a later release exists, or keep its assets. Confirmed by reading.
9. Medium, missing. `.github/workflows/release.yml`. A release has no signature or attestation; the
   `.sha256` comes from the same release, so it proves transport only. Scenario: `verify` is the tagged
   commit's own `bin/release`, and only a branch ruleset exists (`gh api …/rulesets` lists
   `main-branch-protection` alone), so anyone who can push a tag can publish a "release" from a commit off
   main with an edited script. Fix: a tag ruleset for `release/*`, `actions/attest-build-provenance`, and a
   `verify` run from the script on main. Confirmed.
10. Medium, bug. `kit/packages/github/ci/templates/repo/.github/workflows/kit-ci.yml:114`. `Settle` runs on
    `pull_request: closed`. Scenario: a fork's or Dependabot's pull request gets a read-only token and no
    secrets, so the comment and the unassign fail; a pull request with a merge conflict starts no workflow at
    all, so its issues stay assigned. Fix: `pull_request_target` with the base's checkout for Settle. Plausible;
    read from GitHub's documented behaviour, not run.
11. Medium, inconsistency. `kit/packages/agents/claude/files/repo/.claude/settings.json:4`. The deny list misses
    forms of the commands that `.kit/1st.md` names. Scenario: not matched are `git -C x commit`,
    `git -c k=v push`, `env X=1 git push`, `bash -c 'git push'`, `git switch --create`, `git checkout -B`,
    `git branch -d` and `git branch --delete --force`, and `xargs -n1 git push`. Matched are the bare commands, a
    command after `&&`, `command git push`, `X=1 git push`, and `--force-with-lease` (so the `--force` and `-f`
    entries are redundant). This overlaps #72. Fix: none given. Plausible; from Claude Code's permissions
    documentation, which also says argument patterns "are fragile", and not executed.
12. Medium, bug. `tests/helper.bash:59,68`. `helper:isolate` leaves `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`,
    `GIT_AUTHOR_*`, `GH_TOKEN`, `KIT_BASE`, `IN_NIX_SHELL`, `CODESPACES`, and `DOCKER_HOST` set. Scenario: the
    gate inside `nix develop`, the shell that `flake.nix` provides, has 5 failures in `tests/toolchains.bats`;
    with bats under an exported `GIT_DIR`, a test's commit lands in the caller's repository. Fix: unset what
    `git rev-parse --local-env-vars` lists, and the rest. Confirmed by a sub-reviewer.
13. Medium, bug. `tests/across.bats:352`. The test runs the real `gh` (24 calls) and passes whatever it answers;
    on the runner `GH_TOKEN` is set, so it reaches github.com. Scenario: with a `gh` that exits 1, the test passes
    too. Fix: use a fake `gh` there. Confirmed by a sub-reviewer.
14. Medium, bug. `tests/toolchains.bats:103`. `[ a ] && [ b ]` in the middle of a test cannot fail on its left
    side; the same is at `tests/dependencies.bats:64`, `tests/releases.bats:120`, `tests/serve.bats:160`, and
    `tests/uninstall.bats:26,47,66`. A `run pty` followed by only a negative match
    (`tests/across.bats:361–373`, `tests/flow.bats:516–520`, `tests/tui.bats:83`, `tests/workspace.bats:196`)
    passes on a crash. Scenario: the scratch test `[ 1 -eq 2 ] && [ 1 -eq 1 ]; true` is ok. Fix: one assertion
    per line, and assert the status. Confirmed by a sub-reviewer.
15. Medium, bug. `kit/modules/bash/forge/github.sh:165,205,363`. The owner and the name go to
    `gh api graphql` with `--field`, which types digits as Int; the fake answers any arguments. Scenario: a
    repository named `2048` gives `Variable $name of type String! was provided invalid value` (with the real
    gh 2.103.2). Fix: `--raw-field`. Confirmed by a sub-reviewer.
16. Medium, bug. `projects/tutorials/scripts/verify-dist.mjs:21`. The script passes on a broken dist. Scenario:
    with `assets/` deleted, a page emptied, `index.html` removed, or a dead link added, it exits 0 each time. Fix:
    walk the routes, and resolve every in-base href and src. Confirmed by a sub-reviewer.
17. Medium, bug. `projects/tutorials/src/version.ts:27`. A pre-release does not meet its own version. Scenario: a
    reader on `0.1.0-rc.7` is told "Needs kit 0.1.0; you have 0.1.0-rc.7" on that pre-release's own site. Fix:
    treat a pre-release of the same version as met. Confirmed by a sub-reviewer.
18. Medium, bug. `projects/tutorials/src/reader.tsx:28` and `projects/tutorials/src/routes.tsx:125`. An MDX link
    resolves against the base, not the page; nothing checks targets; and there is no not-found route. Scenario:
    `[x](dollar-signs/)` in `projects/tutorials/src/content/kit-tutorials/index.mdx` gives a dead href, and the
    build and the verify step exit 0. Fix: none given. Confirmed by a sub-reviewer.
19. Medium, bug. `projects/tutorials/vite.config.ts:43`. A `TUTORIALS_BASE` without a trailing slash is
    accepted. Scenario: the build exits 0, and its pre-rendered pages are empty. Fix: normalise the base; throw
    when render returns "". Confirmed by a sub-reviewer.
20. Low, bug. `bin/release:158`. Under `pipefail`, `git branch … | grep --quiet` reads SIGPIPE as "no branch", and
    deletes a pre-release that branches reach. Scenario: with 2500 remote branches (126 KB) the delete arm ran 40
    of 40 times; with 300, never. Fix: capture the output first. Confirmed at that scale.
21. Low, bug. `kit/packages/kit/files/repo/.kit/templates/bash/*-provider`. The provider and workflow templates
    make files the linters refuse. Scenario: in a consumer repository, `kit new file modules/bash/forge/x.sh`
    makes a file that starts with a blank line and drops `--purpose`; with `style/bash`, `kit lint` refuses it at
    once. `kit new provider` names `bash/` as the remedy, skips it while the directory is missing, and adds
    `//forge/x.sh is outside the repository`. `kit new file .github/workflows/x.yml` accepts empty `on:`,
    `permissions:`, and `jobs:` holes, and actionlint refuses the result. Fix: none given. Confirmed.
22. Low, bug. `kit/packages/kit/files/repo/.kit/kinds/*.yml`. The kinds misclassify common files. Scenario:
    `Justfile` and `.justfile` are not `just`; `#!/usr/bin/env -S bash` is not `bash`; `#!/bin/sh` is `bash`,
    so shellcheck passes bashisms in a POSIX script; `.github/workflows/sub/y.yml` is `workflows` (`*`
    crosses `/`); `action.yml` is not, so the `pinned` check never sees the `uses:` of a composite action. In
    this repository, 64 `.bats` files and `.envrc` are of no kind and unlinted (`kit kinds` over probes and
    all 774 tracked files). Fix: none given. Confirmed.
23. Low, bug. `kit/modules/bash/context.sh:402`. The hook misreads three inputs. Scenario: it tells Claude that a
    file outside the repository (`~/.claude/plans/x.md`, `/tmp/x.sh`) is governed by this repository's rules, and
    logs it; the matcher is an unanchored regex, so it fires on NotebookEdit, whose `notebook_path` it does not
    read; and a missing `kit` exits 127, which Claude Code treats as non-blocking. Paths with spaces work. Fix:
    none given. Confirmed; the 127 case by documentation.
24. Low, inconsistency. `kit/packages/agents/cursor` and `kit/packages/agents/copilot`. The seven per-rule files
    match the rules, but they are hand copies of the kinds' globs and name the rule only: no style, no embedded
    kind (a justfile brings the bash rule under Claude's hook, not here), and nothing for YAML. Scenario: a kind
    that a repository changes leaves them stale. Fix: none given. Confirmed by reading.
25. Low, inconsistency. `kit/**`. Delivered files break the path rule of `AGENTS.md`. Scenario: nine `see:` lines
    write `kit/bin/kit-flow` and the like (`releases.sh:10`, `marks.sh:18`, `underway.sh:26`, `changes.sh:12`,
    `backlog.sh:18`, `builds.sh:15`, `serve.sh:53`, `commit.sh:26`, and `flow.sh:62` under `kit/modules/bash/`);
    `kit/modules/tutorials.yml:5–9` names `projects/tutorials` and runs `bin/tutorials` (a consumer that vendors
    the release and has a `projects/tutorials` would run its own); the delivered
    `kit/packages/policy/opa/files/repo/.kit/policy/kit.rego:4` and
    `kit/packages/kit/files/repo/.kit/kit.schema.json:6,245` point at `docs/*.md`, which a consumer does not have;
    and `kit/bin/kit-workspaces:21` names `github.com/empowerite/edge`. Fix: none given. Confirmed.
26. Low, inconsistency. `AGENTS.md:15` and `README.md:271`. Three statements about the layout and
    the release do not match the tree. Scenario: both lines say "`libexec/kittui`, built from
    `tui/`", and it is four `libexec/kittui-<os>-<arch>` files built from `projects/tui/`; the
    `bin/` row of the README omits `bin/tutorials`; and "A tag is never reused" does not hold for a
    retired pre-release. Fix: none given. Confirmed.
27. Low, inconsistency. `kit/packages/workflow/files/repo/.kit/workflow.md:31`. The rule says never to commit a
    plan as a file. Scenario: this repository, which takes the `workflow` package, commits
    `docs/superpowers/specs/…` and `docs/superpowers/plans/…`. Fix: none given. Confirmed.
28. Low, missing. `.github/dependabot.yml` and `.github/workflows/`. The infrastructure has six gaps. Scenario:
    Dependabot does not watch `projects/tui/go.mod` (it ships in the release) or the site's npm;
    `.github/workflows/kit-ci.yml` has no `concurrency` and no `merge_group`; the release workflows have no
    timeout; the gate never cross-compiles the four release platforms; a pre-release publishes without the gate;
    and zsh completion is skipped on the runner (`tests/completion.bats:53`). Fix: none given. Confirmed.
29. Low, bug. `bin/ci:28,33,57`. ajv exits 0 on a glob that matches nothing. Scenario:
    `.kit/packages/house/package.yml` and the frontmatter of the house style, `house.md`, are never
    validated (both are valid). Fix: none given. Confirmed.
30. Low, bug. `bin/tutorials:47`. The catalogue builder takes pages it should refuse. Scenario: a page with no
    front matter is skipped with exit 0; `top: "true"` counts; and an id is any directory name (`My Page/` gives a
    URL with a space). Fix: none given. Confirmed by a sub-reviewer.
31. Low, inconsistency. `docs/testing.md:8`. The gate's list omits `npm ci` (network) and the Go steps, and
    several tests pass for the wrong reason. Scenario: `tests/jq/forge.test:101` passes only under UTC;
    `tests/template.bats:291` passes when both sides are empty; and `run ! grep` passes on a missing file
    (`tests/release.bats:86`, `tests/tools.bats:205`). Fix: none given. Confirmed by a sub-reviewer.
32. Low, bug. `projects/tutorials`. The site has five small defects. Scenario:
    `projects/tutorials/src/main.tsx:19` hydrates an unrendered root in dev;
    `projects/tutorials/src/content/kit-tutorials/dollar-signs.mdx` is a test fixture shipped as a page that
    `projects/tutorials/scripts/verify-dist.mjs` hard-codes; `projects/tutorials/src/components/Since.tsx:13` does
    not say how to upgrade (line 83 of the spec); audience ids are checked on top pages only; and build errors do
    not name the file (lines 54–56 of the plan). Fix: none given. Confirmed by a sub-reviewer.

### Holds (Infrastructure)

- `kit check` and `kit generate --check` pass in the clone, so the installed copies match their sources.
- A fresh consumer install of every package passes `kit lint --all-files` and `kit check`.
- The three pinned action SHAs match their tags.
- `mise.lock` covers all four platforms with checksums.
- `kit.schema.json` and `settings:validate` know the same 18 keys.

### Untested (Infrastructure)

- The verb `kit template` (`template fix`) has no bats test; `gather` and `root` have the thinnest.
- `bin/release retire`.
- The refusals of `kit/modules/bash/toolchain/dnf.sh` and
  `kit/modules/bash/toolchain/pacman.sh`; glab and az have fakes and no contract test.
- Nothing enforces a test per bash module; `tests/jq.bats` enforces a test file per jq module, not per function.

### Not reviewed (Infrastructure)

- The matching of Gemini's `tools.exclude`; whether `flake.nix` evaluates (nix was not run); any Linux run.
- `kit lint --all-files`, lychee, and the full suite at HEAD; about 40 of 62 bats files were not run.
- The handling of the tarball by mise and Homebrew; kittui's tutorials screen.
- The rule and style texts against each other in depth; Rego against patterns beyond reading
  (they agree on the three rules and the wording).

## Documents

Reviewed the documents against the code (`README.md`, `AGENTS.md`, the pages under `docs/`, the delivered `.kit/`
pages, the package descriptions, and the verb headers that `kit help` prints); commands marked as run were run in a
throwaway repository with HOME and the caches moved into scratch, and each finding's kind says whether a document
is wrong, lacks something, or contradicts another.

1. High, wrong. `kit/packages/kit/files/repo/.kit/1st.md:81-83`, `README.md:202`,
   `docs/packages/agents/claude.md:33-34`, and `kit/bin/kit-serve:16-17` against
   `kit/modules/bash/serve.sh:61-62`. All four say `kit serve` offers "the read-only verbs", and the header of
   `kit/bin/kit-serve` (what `kit help serve` prints) lists 11 tools that "do not change anything". Scenario:
   `tools/list` against `kit serve` returns 17 tools; assign, close, gather, settle, unassign, and work come back
   with `readOnlyHint:false`, and close, settle, and unassign with `destructiveHint:true`. `docs/serve.md:34-48`
   is the only page that says so. Fix: the documents and the header; the notice of `agents/claude` ("say yes")
   should also say the tools can close issues. Confirmed.
2. High, missing. `docs/features.md:44-53`, `docs/architecture.md:217-218`, and `README.md:17-28` against
   `kit/bin/kit-ci:26`. `docs/features.md` says "The four checks are fixed. What each one runs is found at run
   time", then lists the linters, the kit's own checks (`pinned`), and the templates; those belong to
   `kit lint`, which is not one of the four (gate, check, linked, policy). "What verifies what" gives
   `kit lint` as the verifier without saying where it runs: only in `kit commit`, in the opt-in lefthook hook
   (`--no-verify` skips it), or when the repository's own gate calls it, as `bin/ci:25` does in this
   repository. Scenario: a consumer whose gate is `make ci` has no CI enforcement of the SHA-pin check or the
   template shapes; `kit ci` prints four sections, none of them lint. Fix: the documents; say so in both
   tables and in "What no check covers". Confirmed.
3. Medium, contradiction. `README.md:259`, `docs/features.md:108-109`, and `docs/architecture.md:145` against
   `kit/bin/kit-flow:324-339` and `docs/features.md:207`. The documents say "`kit flow <stage>` prints one
   stage" and that `--text` prints. Scenario: on a terminal a bare `kit flow <stage>` execs kittui, and
   `--text` is refused with a stage: `kit flow desk --text` prints `--text cannot be combined with desk`.
   `kit workspace <stage>` does take `--text`, and `kit/bin/kit-tui:40` also sends people to
   `kit flow --text`. Fix: give stage lines `--text`, or have the documents say `--json` or a pipe. Confirmed;
   the refusal ran, and the terminal branch is from the code.
4. Medium, wrong. `docs/tutorials.md:11,22`, `README.md:123`, and the purpose line in `kit help` against
   `kit/bin/kit-tutorials:8-9,67-87`. The documents say "A tutorial named opens in the browser." Scenario: on a
   terminal it opens kittui's tutorials screen on that card; the browser opens only without a terminal or with
   `--text`. Fix: the documents. Confirmed from the code.
5. Medium, wrong. `docs/tutorials.md:50-54` and `projects/tutorials/README.md:9-11` against
   `.github/workflows/`. The links that `kit tutorials` prints do not resolve. Scenario: `kit tutorials --text`
   prints `https://empowerite.github.io/kit/tutorials/main/kit-tutorials/?kit=unreleased`;
   curl gets 404, and `kit.empowerite.dev` does not connect. No workflow builds or
   deploys the site (only kit-ci, release, and retire-pre-releases exist). Fix: deploy
   it, or have the documents say it is not published. Confirmed on the day of the review.
6. Medium, wrong. `README.md:76-80` against `github.com/empowerite/homebrew-tap`.
   `brew install empowerite/tap/kit` cannot work: the tap repository exists and is empty.
   Scenario: `gh api repos/empowerite/homebrew-tap/contents` answers "This repository is empty".
   Fix: publish a formula, or drop the line. Confirmed.
7. Medium, wrong. `kit/bin/kit-lint:6-9` and `docs/modules.md:197-200` against the behavior of `kit lint`.
   `<path>…` is "the files to check", and exit 2 is for "a path kit does not know", but a path that does not exist
   reports success. Scenario: `kit lint nosuch.sh` prints `✅ every governed path passes its rule's linter`,
   exit 0. Fix: the code; refuse, or say nothing was linted. Confirmed.
8. Medium, wrong. `docs/architecture.md:213` against `kit/packages/github/ci/package.yml:11`. The row says "The
   gate ran with the pinned tools … locally and in CI". Scenario: the delivered default is
   `ci_toolchain: ambient`, where nothing is pinned or installed (`docs/toolchains/using-no-toolchain.md:22`).
   Fix: the row says "when a toolchain is named". Confirmed.
9. Medium, contradiction. `docs/features.md:306-310` against `docs/architecture.md:216,219-220`. "What no
   check covers" names one gap (rules followed after reading), while the table in `docs/architecture.md`
   has three more "no check" rows: an agent asked for the rules, the forbidden git commands, and the desk
   unchanged. Scenario: neither lists the lint gap (finding 2), a human's raw `git commit`, or that on
   the default branch `kit policy` judges no branch or commit (a run prints "on main, no branch or commit
   is judged"). Fix: `docs/features.md`. Confirmed.
10. Medium, contradiction. `docs/delivery.svg:81-84,45-46` against `docs/architecture.md:82-83` and
    `bin/release:106`. The diagram says `.kit/manifest` holds "version, packages, digests", `kit.yml` is "yours,
    never written", and `docs/` is "packed, placed nowhere". Scenario: the manifest holds digests only, kit writes
    three keys into `kit.yml`, and the tarball carries no `docs/`. Fix: the SVG. Confirmed.
11. Medium, wrong. `kit/bin/kit-linked:6,11-12` against `kit/bin/kit-linked:67-93` and
    `docs/trackers.md:52-54`. The header says a linked issue counts only when it "belongs to this repository".
    Scenario: the code also counts an open issue of any declared tracker named with `Closes AB#4521` in the
    body. Fix: the header. Confirmed from the code.
12. Medium, contradiction. `projects/tui/README.md:6-23,29-31` against
    `projects/tui/internal/flow/apps.go:24-37` and `docs/features.md:189-209`. The README has "six screens"
    with a flow picture and a workspace picture; kittui opens the seven apps (#138), which the README never
    names. Scenario: its example `kit tutorials screens` (line 30) names a tutorial that does not exist
    (`'screens' is not one of the choices`), and its Layout omits `internal/grid`, `internal/pane`, and
    `internal/screens`. Fix: the README. Confirmed.
13. Medium, wrong. `kit/packages/policy/opa/package.yml:5` (shown by `kit packages`) against
    `kit/packages/policy/opa/files/repo/.kit/policy/kit.rego`. The description says "covering names and files".
    Scenario: `kit.rego` has three deny rules (branch, commit subject, change title) and none over `tree`;
    `docs/packages/policy/opa.md:7` is right. Fix: the description. Confirmed.
14. Low, contradiction. `AGENTS.md:15` and `README.md:271` against `README.md:345` and `bin/release:48`. Both
    lines say "libexec/kittui, built from `tui/`". Scenario: the source is `projects/tui/`, and the files are
    `libexec/kittui-<os>-<arch>`. Fix: both lines. Confirmed.
15. Low, missing. `docs/modules.md:229-240` against `kit/modules/jq/`. The jq module table lists 10
    modules, and the directory has 14. Scenario: `glyphs.jq`, `serve.jq`, `version.jq`, and `workspace.jq`
    are not in the table. Fix: the document. Confirmed.
16. Low, contradiction. `kit/packages/kit/files/repo/.kit/1st.md:126-129` against `README.md:185-186` and the
    WORKFLOW group of `kit help`. Three lists of "the workflow commands" disagree. Scenario: `1st.md` has 11,
    including context, lint, gate, and hydrate, and tells agents to "run them … in no other way"; the README
    has 12, and `kit help` has 14 (it adds workspace and workspaces); assign, close, gather, settle, and
    release are not in `1st.md`. Fix: `1st.md`. Confirmed.
17. Low, wrong. `docs/generate.md:29-34,149` and `docs/packages/agents/generic.md:4-5` against
    `kit/packages/agents/generic/regions/repo/AGENTS.md`. The sample region and "the sentence" are
    not the region's text. Scenario: the region is a heading and two sentences, with the
    `kit hydrate` fallback (`README.md:152` has it right), as `kit packages add agents/claude` writes
    it into `AGENTS.md`. Fix: the documents. Confirmed.
18. Low, wrong. `kit/bin/kit-help:8` against `kit/modules/bash/1st.sh:426` and `kit/bin/kit-help:58`. The header
    says "`kit <verb> --help` prints the same page as `kit help --text <verb>`". Scenario: the guard execs
    `kit-help <verb>` without `--text`, so on a terminal with kittui it opens the palette. Fix: pass `--text`, or
    correct the header. Plausible; not run on a terminal.
19. Low, contradiction. `kit/bin/kit-new:87` against `kit/modules/bash/help.sh:19` and `docs/modules.md:175`. The
    header says `--group` "is workflow, files, checks, installation, interfaces, or help". Scenario: the code and
    `docs/modules.md` also take `plumbing`. Fix: the header. Confirmed.
20. Low, missing. `kit/bin/kit-install:13` and `kit/bin/kit-packages:6` against
    `docs/packages.md:286-310`. Both headers say packages come from "the release and .kit/packages/".
    Scenario: the sources under `sources:` in `kit.yml` are a third origin, which the Exits line of
    `kit/bin/kit-packages` itself mentions. Fix: the headers. Confirmed.
21. Low, missing. `README.md:275`, `docs/testing.md:8-9`, `bin/ci:2`, and `projects/tui/README.md:215` against
    `bin/ci:49-71`. The lists of what `bin/` holds and what the gate runs are short. Scenario: `bin/` is listed
    without `bin/tutorials`; the one-line lists of `docs/testing.md` and `bin/ci` omit the checks of the tutorials
    site (the ajv front matter check and `npm run check`); and the README of the TUI says "bin/ci runs the same
    three" and omits the 98% coverage floor that fails the gate. Fix: the documents. Confirmed.
22. Low, wrong. `docs/packages.md:95-96` and `bin/ci:9-10` against `bin/ci:27-37`. They say the
    gate "validates every package.yml" and "every rule and style". Scenario: the globs cover
    `kit/packages/` only; `.kit/packages/house/package.yml` and `styles/house.md` are not
    validated. Fix: the globs or the wording. Confirmed.
23. Low, missing. `README.md:289,291`, `docs/packages.md:336`, and
    `kit/packages/kit/files/repo/.kit/1st.md:151-153` against `kit/packages/kit/files/repo/.kit/`. The lists
    of what the kit package delivers are short. Scenario: they omit the five
    `.kit/templates/bash/<seam>-provider` templates, and the README also omits `.kit/welcome/`;
    `README.md:291` says `.kit/templates/bash/` links into the house package, while the provider templates
    there link into `kit`. `docs/packages/kit.md:27` is right. Fix: the documents. Confirmed.
24. Low, contradiction. `docs/architecture.md:14-15` against `docs/modules.md:176-177`. "`bin/` holds only
    commands a person runs" stands against seven plumbing verbs "that scripts, the hooks, and the TUI call rather
    than a person". Scenario: `.kit/README.md:49` and `.kit/welcome/7-make-it-yours.md:50` say `kit help` "lists
    every command", and it sums up plumbing in one line. Fix: the documents. Confirmed.
25. Low, wrong. `docs/context.svg:6-7` and `docs/loop.svg` against `docs/architecture.md:107-108`. The diagrams
    say "Hook: kit context before every write". Scenario: the hook fires on Write, Edit, and MultiEdit, and "a
    write through the shell does not reach it". Fix: the SVGs. Confirmed.
26. Low, contradiction. `docs/features.md:92-106` against `kit/bin/kit-flow:8` and `kit flow --stages`.
    The page says "Work moves left to right through eight stages:", and its bullets then put incoming
    third. Scenario: the order is backlog, underway, desk, worktrees, changes, builds, incoming,
    releases. Fix: reorder the bullets. Confirmed.
27. Low, missing. `AGENTS.md:24-26` and `bin/refresh:5-7` against `README.md:297`. The installed copies are
    listed without `.mcp.json`. Scenario: `.mcp.json` is an installed copy, as `README.md:297` says, and
    neither list names it. Fix: the documents. Confirmed.
28. Low, missing. `kit/packages/github/ci/templates/repo/.github/workflows/kit-ci.yml:7-8` against line
    87 of the same file. Step 2 of the header gives two sources of kit. Scenario: the step also uses
    any `kit` already on PATH, whatever its version, and downloads nothing
    (`docs/packages/github/ci.md:8` says so). Fix: the header. Confirmed.
29. Low, wrong. `docs/context.md:67-77` against `kit/packages/kit/files/repo/.kit/rules/bash.md` and
    `kit/packages/style/bash/files/repo/.kit/styles/bash.md`. The sample frontmatter pairs the rule's
    `lint: shellcheck` with the style's `script` and `module` templates. Scenario: no shipped file
    reads that way. Fix: show one real file. Confirmed.

### Undocumented

These are in no document, only in a verb header or the code.

- Verbs: `kit root` (0 mentions); `kit template` and `kit usage` once each.
- Flags: `kit assign --to`, `kit close --not-planned`, `kit unassign --all`, `kit work --pr`, the `--all` and
  `--message` of `kit commit` and `kit submit`, the `--all`, `--mine`, `--others`, `--owner`, `--fresh`, and
  `--cached` of `kit flow`, the `--content`, `--kinds`, and `--embeds` of `kit new kind`,
  `kit new package --description`, and `kit syntax --match`.
- Environment: `KIT_BREW_PREFIXES` (code only), `CLICOLOR_FORCE`, `COLUMNS`, `KIT` (the kit
  that kittui runs), `GH_CONFIG_DIR` and `GLAB_CONFIG_DIR`, `DOCKER_HOST`, `DOCKER_CONTEXT`,
  and `DOCKER_CONFIG`, and `TUTORIALS_PROJECT` (`bin/tutorials`). `KIT_GENERATE_FILE` and
  `KIT_HOME` are in the documents but in no verb header.
- State: the last workspace used, under `${XDG_CACHE_HOME}/kit/workspaces`
  (`docs/workspaces.md` says "the last group used", not where).
- `kit.yml` keys: none; all 18 schema keys are in `docs/packages.md`, and
  `kit/modules/bash/settings.sh` accepts the same set.
- `kit syntax --json` is not a flag; `kit syntax` prints JSON by default.

### Holds (Documents)

- The key table of `kit.yml`; the 19 packages and their dependencies; the empowerite bundle.
- Kinds, rules, styles, and the template table; both deny lists; the policy defaults and the 65536-byte cap.
- Log file names and `log_days`; where toolchain choices are kept, and the providers' commands.
- The delivered CI workflow against `docs/packages/github/ci.md`; the base fetch of `kit update`
  with its digest check; the gate detection order.
- `kit packages remove kit` writing `excludes:`; `kit check` failing on the missing
  lefthook `extends`; the exit codes of `kit hydrate`.
- The worker contract's ten steps; the seven steps of welcome; the app table in `docs/features.md`.

### Not reviewed (Documents)

- The plans and specs under `docs/superpowers/`.
- Anything that needs a real terminal: kittui's keys, menus, and legend, and the long
  interaction prose of `projects/tui/README.md`.
- Forge-backed behavior (`kit work`, `assign`, `close`, `gather`, `settle`, `tidy`, the `flow` stages with data,
  `release`): the headers were read, not run; gh was unauthenticated in scratch.
- `docs/toolchains/using-docker.md`, `docs/toolchains/using-devcontainer.md`, and
  `docs/toolchains/using-nix.md` beyond the commands found in the providers.
- The mise `version_prefix` install line and `minimum_release_age_excludes`; no `release/0.1.0` exists
  (the latest is `release/0.1.0-rc.7`, a pre-release), so the mise pin and the "latest release"
  download through `gh` cannot work until it does.
- External pages cited as "verified on 2026-10-30" (Cursor, Copilot, Gemini), the MCP spec
  revision, and the timings in `docs/testing.md`.
- Rule and style prose (`.kit/rules/*.md`, `.kit/styles/*.md`) beyond the frontmatter.

## Gaps

This section lists what is designed, planned, or filed and not built, or not fully built, as of 2026-11-10 at HEAD
`2dad3552` of the branch `1-the-first-release-v0-1-0`. Every spec and plan was read, and plan tasks and issues were
checked against the code, the tests' names, `git log`, and read-only `kit` runs; the bats suite and vitest were not
run, and kittui's Go suite was run once by a sub-reviewer (all packages pass, 98.0% total coverage). Three facts frame
the rest. First, the shipped surface of the CLI (45 verbs) is built and tested, and the gaps are in four places: the
release itself, the four decisions the entities spec says the first release must settle (none is met), twelve design
issues that PR #2 does not close (none has a commit), and kittui's apps (20 of 32 tabs are placeholders). Second, the
closing list of PR #2 leaves out exactly #62, #63, #64, #66, #67, #68, #69, #72, #73, #74, #76, and #77 (`gh pr view
2 --json closingIssuesReferences`); each has zero commits naming it and no code, two carry the `bug` label (#66
and #69), and one is the only CI hardening (#67). Third, the pushed branch is 165 commits behind the worktree
(`git rev-list --count origin/1-the-first-release-v0-1-0..HEAD`), and no CI run has passed on the pushed head
`6edaa6b6`: run 42778922819 fails five bats tests on Linux.

### Missing or partial

Each table row is one item that is missing or partial. Size
is S for under a day, M for days, and L for a week or more.

#### Release

Sources: issue #1 § Done when, `README.md` § Release and § Install, and PR #2 § Not checked.

| Source | Status | Missing | Evidence | Size |
| --- | --- | --- | --- | --- |
| #1 "v0.1.0 is tagged and the release exists" | Missing | Only pre-releases exist; the newest, rc.7 (2026-10-31), is 551 commits behind HEAD and predates kittui's apps, workspaces, and tutorials | `gh release list`; `git rev-list --count release/0.1.0-rc.7..HEAD` gives 551 | S once the rest holds |
| #1 "pass `kit check` in CI" | Partial | CI is red on the pushed head: tests 113 (`tests/help.bats`), 261 (incoming), 340 (materialize; `bin/tutorials catalogue` fails), 466 (modules), and 570 (underway). Four of the five test files are untouched after that head; not re-run locally | `gh run view 42778922819 --log-failed`; `git log 6edaa6b6..HEAD -- tests/…` | S–M |
| `README.md` § Install, Homebrew | Missing | `brew install empowerite/tap/kit` cannot work: the tap repository is empty, and nothing in `bin/release` or `.github/workflows/release.yml` writes a formula | `gh api repos/empowerite/homebrew-tap/contents/Formula` answers "This repository is empty"; a search of `bin/release` and `.github/workflows/release.yml` for `tap`, `formula`, or `brew` has 0 hits; `terraform/projects/homebrew-tap/README.md` in the infrastructure repository ("Formula/kit.rb and the workflow that bumps it are the tap's own") | M |
| PR #2 § Not checked | Partial | `kit update` across two real releases was never run (a synthetic release and `KIT_BASE` only); `kit work` on a fresh issue was never run end to end | The body of PR #2, § Not checked | S–M |
| #1 "a fresh repository can pin the release, install, work, and pass in CI" | Missing | No consumer but kit itself: the sibling clones edge, enchiridion, infrastructure, and jig have no `kit.yml` | `ls <clone>/kit.yml` fails in all four | M |
| PR #2 body | Partial | The body, which the workflow calls the report, describes the rc-era kit (`kit issue`, `kit status`, modules, `just`) and nothing of #56–#139 | The body of PR #2, § What changed | S |
| #20 | Partial | Sources are built; the manifest does not record each file's source and version, so an edited file from a source is not three-way merged | The closing comment of #20 ("Left out") | M |

#### Entities spec

Source: `docs/superpowers/specs/2026-11-09-entities-design.md`.

| Source | Status | Missing | Evidence | Size |
| --- | --- | --- | --- | --- |
| § What the first release settles, 1: references | Missing | No full reference (`host/path#7`, `…!3`, `…@sha`, `sha256:…`) in any JSON. `kit workspace --json` rows carry a stage-local `key` and links of another form, `row:<stage>/<key>`; `kit flow --json` rows carry no key at all, and kittui makes one in Go | `kit/modules/jq/workspace.jq:32`; `projects/tui/internal/flow/workspace.go:85-88`; `projects/tui/internal/flow/rows.go:498`; `kit workspace commits --json` | M |
| § settles, 2: owner and reach | Missing | No `~/.config/kit/config.yml`, no `all` or `selected` layout, and no setting-or-preference mark in the schema; `~/.kit/settings.yml` has not moved. The person files are `workspaces.yml`, `kittui.json`, `presets.json`, `search-history.json`, and `~/.kit/settings.yml` | `kit/modules/bash/toolchain.sh:386-388`; `grep -ri preference kit` has 0 hits; the schema keys in `kit/packages/kit/files/repo/.kit/kit.schema.json` | M |
| § settles, 3: command-provider contracts | Partial | Only the calls of the tracker seam are written down (`docs/trackers.md:90-105`), as text columns, not JSON in and out. Forge, toolchain, dependencies, and policy have their calls in bash headers only; no `provider: command` exists | `docs/architecture.md:149-187`; `grep -ri 'provider: command' kit docs` finds the spec only | M |
| § settles, 4: a verdict CI can sign | Missing | `kit ci` takes no arguments and has no `--json`; no in-toto statement anywhere | `kit syntax ci` gives `"usage":["kit ci"]`; a search of `kit`, `bin`, and `docs/*.md` for `in-toto` or `predicateType` has 0 hits | S–M |
| § Platform | Missing | `kit platform`; the `services:` and `bind:` keys | No `kit/bin/kit-platform`; the schema has neither key | L (after the release, order 4) |
| § Configuration | Missing | `kit config <key>` | No `kit/bin/kit-config` | M (order 4) |
| § Actors, identity seam | Missing | Person, Team, Role; the identity seam and its four calls; no verb lists people, teams, or agents | No `identity` in `kit/modules/bash`; the three tabs of People are `planned/` (`projects/tui/internal/flow/apps.go:31`) | L (order 2) |
| § Work | Missing | Mappings (`map:` under `trackers`), iteration, milestone, priority, estimate, board, and cadence | The tracker schema keys are `closed_state`, `new`, `organization`, `prefix`, `project`, `provider`, `removed_state`, `type`, and `url` | L (order 3) |
| § Delivery | Missing | Artifact as an entity, environment, deployment, target, and fleet; no cd seam | `projects/tui/internal/flow/apps.go:30`; no such module | L (order 5, 8) |
| § Trust | Missing | Evidence, finding, store, control, waiver, and approval; `kit verify` | No `kit/bin/kit-verify`; `projects/tui/internal/flow/apps.go:32-33` | L (order 1) |
| § Tools | Missing | The tool catalog, and `kit check` reporting an unapproved tool | No `catalog` in `kit/modules/bash/toolchain.sh` | M (order 6) |
| § Conversation | Missing, deferred by the spec | Everything; the spec says chat is not part of the first version | The spec, § The companion bot | L |

#### kittui apps

Sources: #138, #139, #123, and `docs/features.md:207-209`.

| Source | Status | Missing | Evidence | Size |
| --- | --- | --- | --- | --- |
| #138 | Partial | 20 of 32 tabs are placeholders that print "has no data source yet": iterations, milestones, reviews, checks, merged, artifacts, environments, deployments, and all 12 tabs of People, Trust, and Platform. Three whole apps show nothing | `projects/tui/internal/flow/apps.go:26-35`; `projects/tui/internal/flow/view.go:423` | See the rows below |
| #138 Platform: toolchains, tools | Partial | The data exists as `kit toolchains --json` and `kit tools --json`; the tabs are not wired, and there is no field catalog entry for either | `kit syntax toolchains`, `kit syntax tools`; `projects/tui/internal/flow/fields.json` (13 stages, neither) | M |
| #138 Trust: policy; Platform: settings | Missing | `kit policy` and `kit facts` print no row JSON; no verb lists settings | `kit syntax policy` gives `kit policy` with `--input` or `--facts=<file>` only | M each |
| #138 Changes: reviews, checks, merged | Missing | `kit flow changes` already reads checks and threads per change, but no stage lists them a row each, and none lists merged changes | The changes fields in `projects/tui/internal/flow/fields.json`: `checks`, `threads` | M |
| #138 Issues: milestones | Missing | The forge has milestones natively; no seam call or stage | No `milestone` in `kit/modules/bash/forge.sh` | M |
| #123 "holds for every app" | Partial | `kit flow <stage>` and `kit workspace <stage>` open their apps; People, Trust, and Platform have no verb or word that opens them, only the app menu | `projects/tui/internal/flow/apps.go:53-71` (`AppOf`); no `kit-people`, `kit-trust`, or `kit-platform` | S once verbs exist |
| `projects/tui/README.md` | Partial | Describes six screens with a flow picture and a workspace picture | `projects/tui/README.md:6-21`; not in `git show --stat ed17d55e` | S |

#### Design issues with no commit

Sources: the twelve design issues that PR #2 does not close; no commit names any of them.

| Source | Status | Missing | Evidence | Size |
| --- | --- | --- | --- | --- |
| #66 trust (bug) | Missing | No `kit trust`, no trust record, no hardened git calls; `kit tools install` runs `mise trust` unasked; `bash_path` and the self-serve handoff run a clone's code | `kit/modules/bash/toolchain/mise.sh:43`; a search of `kit` for `kit trust` or `hooksPath=/dev/null` has 0 hits | L |
| #69 fail closed (bug) | Missing | A forge read that fails leaves `change` null, so `change_title` goes unchecked; no audit of the `true` fallbacks after a failing command (27 modules hold them) | `kit/modules/bash/facts.sh:84-91` | M |
| #67 base-branch policy | Missing | CI judges by the pull request's own `kit.yml` and Rego | `kit/modules/bash/policy.sh:131`; no `weaken` logic | M |
| #72 enforcement tiers | Missing | No report of advisory, best-effort, local floor, and authoritative | A search of `kit/bin` and `kit/modules/bash` for `advisory`, `authoritative`, or `ruleset` has 0 hits | M |
| #73 leases | Missing | No `refs/kit/leases`, no verb | `grep -rni lease kit` finds only `--force-with-lease` | L |
| #74 contracts | Partial | Help pages are snapshotted (`tests/help.bats`); no snapshot of every `kit syntax <verb>` or of `tools/list`, and no docs-claims check. `tests/contract.bats` is something else (the real answers of mise) | `tests/contract.bats:1-8` | M |
| #76 `kit tidy --wait` | Missing | The flag | `kit syntax tidy` gives `kit tidy [--dry-run]` | S |
| #77 every tracker in the picture | Missing | Backlog and underway read `forge:open-issues` alone; no tracker `open-issues` call | `kit/modules/bash/backlog.sh:32`; `docs/trackers.md:90-105` | L |
| #62 shared index after a private-index commit | Missing | No sync of the shared index, no warning line | `grep -rn 'staged differently' kit tests` has 0 hits | S |
| #63 where a new branch belongs | Missing (undecided question) | No decision written | No commit; the issue has no answer | S to decide |
| #64 policy/conventional | Missing | No package, no `policy:` in `package.yml`, no `commit_reference` | `ls kit/packages/policy` lists `opa` only | M |
| #68 origin or primary (question) | Missing (undecided) | `kit work` and `kit submit` use `origin` | `kit/modules/bash/issue.sh:297-315,491,511`; `kit/bin/kit-submit:8` | S to decide, S–M to build |

#### Workspaces

Sources: `docs/superpowers/specs/2026-11-08-workspaces-design.md`, plans 1, 2, 3a, and 3b, and #89.

| Source | Status | Missing | Evidence | Size |
| --- | --- | --- | --- | --- |
| Spec § The workspace file | Missing | `match:` and `project:` are ignored without a word: a group with only `project:` has no members and no warning | `kit workspaces --show <group> --json` gives `"members":[]`; `kit/modules/bash/workspaces.sh:146-184` | S to refuse; L to build (step 4), M–L (step 6) |
| Spec § Labels on repositories | Missing | `topic:` and `label:` (`git config kit.workspace`) | No `kit.workspace` in `kit/modules/bash/workspaces.sh` or `kit/modules/bash/across.sh` | M (step 5) |
| Spec § Which workspace kit starts on | Partial | `kit flow` with no `--workspace` ignores the last group; `--workspace none` is refused | `kit/bin/kit-flow:283-288` | S |
| Spec § `kit workspaces`; § The 📚 control | Partial | No Workspaces screen; Browse…, New workspace…, and Manage workspaces… are drawn and can never act | `projects/tui/internal/shell/books.go:158-160` | L (step 7) |
| Spec § Flow: ☁️ members | Missing | A member with no checkout adds nothing to any tab, and only the repos tab and Welcome say so | `kit/modules/bash/across.sh:175` | L (step 4) |
| Spec § Flow: Project narrowing; an issue shown once | Missing | Neither; rows are keyed per member | `projects/tui/internal/flow/rows.go:498` | M each |
| Spec § Flow: clone on action | Missing | An action on an uncloned member refuses, where the spec asks and clones | `projects/tui/internal/flow/act.go:244`; `ca03c5b3` | M |
| Spec § Where Flow's data comes from; § State and cost | Partial | One `kit flow` per member per stage, no batching by forge and owner; only `last` is cached | `kit/modules/bash/across.sh:163-199`; `kit/modules/bash/workspaces.sh:297-310` | L (step 4) |
| Spec § Flow: repos tab | Partial | kit prints `checkouts`; kittui's row does not read it | `projects/tui/internal/flow/repos.go:19-38` | S |
| Plan 3a task 4 | Partial | `--me --workspace` prints "`--workspace` cannot be combined with `--workspace`"; `--quota` has a unit test of the merge, not the fake-`gh` test | `tests/across.bats:283-299` | S |
| `docs/workspaces.md` | Partial | States the start rule for kit as a whole; says every form prints JSON (`--unfavorite x --json` exits 2) | `docs/workspaces.md:43-47,66-67` | S |

#### Tutorials

Sources: `docs/superpowers/specs/2026-11-06-tutorials-design.md`,
`docs/superpowers/plans/2026-11-06-tutorials-index.md`, and #86.

| Source | Status | Missing | Evidence | Size |
| --- | --- | --- | --- | --- |
| Spec § Goals, permanent and temporary copies; #86 part 2 | Missing | The site is built in the gate and deployed nowhere, so every link `kit tutorials` prints is dead; no removal of a pre-release's copy | `.github/workflows/` holds `kit-ci.yml`, `release.yml`, and `retire-pre-releases.yml` only; `gh api repos/empowerite/kit/pages` answers 404 | L |
| Spec "`kit release` refuses a version with no entry" | Missing | The changelog gate, and any changelog | `grep -rni changelog bin kit/bin kit/modules/bash .github` has 0 hits | S |
| Spec "at least one tutorial, done well"; #86 part 3 | Partial | One page, about the tutorials themselves, and a build fixture; none for the `ui` or `maintainer` audience; What's new and Changelog are one sentence each | `projects/tutorials/src/content/kit-tutorials/` (`index.mdx`, `dollar-signs.mdx`); `kit/modules/tutorials.yml` (one entry) | M–L |
| Spec "says how to upgrade" | Partial | `<Since>` says which kit is needed, not how to get it | `projects/tutorials/src/components/Since.tsx:13` | S |
| Plan task 4, 98% coverage | Partial | The tutorials package is at 95.4% (`click` 69.0%, `key` 88.7%); the total is 98.0%, exactly the floor | The sub-reviewer's `go test -coverprofile` run | S |
| `docs/tutorials.md` | Partial | Says a named tutorial opens in the browser; on a terminal it opens kittui on its card | `docs/tutorials.md:11,21`; `kit/bin/kit-tutorials:8-10,83` | S |

#### kittui components

Sources: `docs/superpowers/specs/2026-11-07-kittui-components-design.md`, #85, #92, and #103.

| Source | Status | Missing | Evidence | Size |
| --- | --- | --- | --- | --- |
| #92 large tables | Partial | No windowing: every row is drawn, and the line cache is keyed by the cursor, so each ↓ redraws all rows (a probe: 7 ms at 300 rows, 71 ms at 3,000, 225 ms at 10,000). Tests stop at 300 rows | `projects/tui/internal/flow/view.go:419-453,460`; `projects/tui/internal/flow/cost_test.go:68,137` | M–L |
| Spec § Goals, one ordered list of controls | Partial | `shell.Layers` is used by flow alone; welcome, tutorials, palette, and form keep their own if-chains | `projects/tui/internal/flow/layers.go:17`; `projects/tui/internal/palette/palette.go:210-279` | M |
| Spec § Table; #85 | Partial | `incoming` is outside the catalog and drawn by hand; the stages of `kit flow` keep Go record builders; only flow imports `internal/table`, so reuse by a second screen is unproven | `projects/tui/internal/flow/fields.json`; `projects/tui/internal/flow/rows.go:551-962` | M–L |
| #85 tasks 6 and 8 | Missing | Double click and drag | No "double" or mouse-motion handling under `projects/tui` | S–M |
| Spec § Keys; #103 | Partial | Nothing shows which keys arrive in Terminal.app and iTerm; Home and End work and stay unnamed | `projects/tui/internal/flow/view.go:204-222` | S–M |
| Spec § Frame, alert; § Source, clock | Partial | Any key clears an alert, where the spec says it stays until esc; a notice's fade is a raw `tea.Tick`, not `source.Clock` | `projects/tui/internal/flow/flow.go:373-374`; `projects/tui/internal/shell/notice.go:42` | S |
| Spec § Terms | Partial | Comments keep names the spec retires ("keys bar" 121 hits, "app's menu" 69, "preset line" 82) | grep counts | S |
| Mouse wheel; `NO_COLOR` | Partial | The wheel works in flow only; kittui has no `NO_COLOR` code or test (the CLI honors it, `kit/modules/bash/1st.sh:18`) | `projects/tui/internal/flow/flow.go:376` | S |

### Each source document

- Entities spec: a design only. Of "what the first release settles", 0 of 4 are met (3 missing, 1 partial). Code
  and the entities of kit's content exist; every new domain is missing, as the spec intends. kittui's apps (#138)
  and links (#139) are built ahead of their data.
- Workspaces spec: steps 1–3 of its order of work are done; steps 4–7 are missing and have no issue of their own.
  Two of ten design subsections are done, seven partial, and one missing.
- Workspaces plan 1: 6 of 6 tasks done (`kit/modules/bash/workspaces.sh`,
  `kit/bin/kit-workspaces`, and 27 tests in `tests/workspaces.bats`).
- Workspaces plan 2: 7 of 7 done; its glyph is the one that #108 chose.
- Workspaces plan 3a: tasks 1 and 2 done; task 3 superseded by #134
  (`9f1520ff`, as `kit workspace repos`); task 4 partial.
- Workspaces plan 3b: tasks 1, 2, 3, and 5 done; task 4 superseded by #127 (`80465865`), #134, and #138.
- Tutorials spec: everything inside the repository is built (the site, the catalogue, the verb,
  the screen, and the front-matter check in `bin/ci:50-57`). Publishing and the changelog gate
  are missing; the content is one fixture tutorial.
- Tutorials plan: 5 of 5 tasks done; the coverage target of task 4 is missed for its
  package. The plan itself defers parts 2 and 3.
- Components spec: all 7 moves done as scoped; three goals partial (layers beyond flow, table reuse, and the
  clock). Its "four screens" text is superseded by `ed17d55e`, and its outline terms by `a0528ade`.
- Issue #1: the delivered tree, the workflow, and self-use are done; the
  tag, a green CI, and a fresh consumer are not.
- Issues #3–#61, #65, #70, and #71: done, by commits naming each (or under #1 for #15 and #50). The
  comment on #3 that it stays open "for the adapters" is stale:
  `kit/packages/agents/{cursor,copilot,gemini}` exist. #20 is partial.
- Issues #62–#64, #66–#69, #72–#74, #76, and #77: missing, all twelve; #74 is partial.
- Issues #78–#139: done, except #85 (partial), #86 (parts 2 and 3), #89 (steps 4–7), #92 (partial), #123
  (partial), #138 (20 placeholder tabs), and #135 (a design, by intent).
- In-repository markers: no TODO, FIXME, or XXX. Fourteen bats skips, all for a tool absent from PATH (opa six
  times). Two `t.Skip`, both benign. `docs/features.md` § What no check covers and `docs/testing.md` § Still to
  measure each hold one honest line and no work item.

### Planned nowhere

- A Homebrew formula and the step that bumps it. The README advertises the
  tap; the tap is empty, and no kit issue covers it.
- macOS in CI. Every job runs on `ubuntu-latest`
  (`.github/workflows/kit-ci.yml:39,116`, `.github/workflows/release.yml:32`), while
  the README supports macOS and its stock bash 3.2 is the documented trap.
- A smoke test of the packed release. `bin/release publish` builds kittui for four platforms; nothing installs the
  tarball and runs `kit version` or `kit tui` from it before publishing.
- Live conformance for GitLab and Azure DevOps. Both providers are tested against fakes alone;
  `docs/trackers.md:106-108` says the fixtures cannot prove a real `az` answers the same way.
- A changelog or release notes. The notes of a release are the mise pin only (`bin/release:132-137`); the only
  changelog designed is that of the tutorials site.
- A security policy. There is no `SECURITY.md`, while #66 documents five ways a clone runs code.
- A contributing guide and a CODEOWNERS file. Neither exists; the sketch of #67 leans on a code owner's review.
- Completion for bash and fish. `kit completion zsh` is the only shell (`kit syntax completion`).
- A version and a migration rule for the person files. `presets.json` has a version
  (`projects/tui/internal/table/store.go:17`); `kittui.json`, `search-history.json`, and `workspaces.yml`
  have none, and the entities spec will move them.
- An upgrade test from a real earlier release. PR #2 says no upgrade path from a pre-release is kept; after v0.1.0
  the first real `kit update` will be the first ever run across two published releases.
- Signing or attestation of kit's own tarball. A `.sha256` is published; the entities spec defers signing, and
  nothing says kit's own release is the first subject.
- A refreshed body of PR #2. It is the report the workflow requires, and it predates most of the branch.

Windows is not a gap: the README says to use WSL.
