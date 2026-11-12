# Fail Closed Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A read, a sub-command, or a check that fails is never taken for "nothing found", "absent", or "passed":
the verb says what it could not read and exits 1, and CI judges a pull request by its base branch's policy.

**Architecture:** Six mechanisms let a failure read as empty, and each gets one fix at every site: a helper that
reads a command's lines and keeps its status; `|| return` where a function leaned on errexit; capture, then match,
where `grep --quiet` ended a pipeline; one contract for every forge call, held by `tests/forge-conformance.bats`; a
check that refuses what it cannot classify; and one exit contract for the JSON verbs, which kittui's `Runner`
follows. Scans keep the idioms out: tests in `tests/modules.bats`, not a rule's `checks:`, as they name kit's seams.

**Tech Stack:** bash 4.4 or later, jq, git, bats; Go for `projects/tui`.

**Spec:** issue #69, "Fail closed", its Design sketch (`gh issue view 69`), for Tasks 1 to 7; issue #67, "CI judges a
pull request by its base branch's policy", its Design sketch, for Task 8. The sites are the review's.
Every line number was opened at 2dad3552, in `kit/modules/bash/` unless a directory is named.

**Review:** [the findings](../reviews/2026-11-10-code-review-findings.md) of the review of 2026-11-10, cited
here by area and number: Serve 8 is item 8 under its Serve heading.

## Decisions for the owner

1. A forge that cannot answer, at the desk (issue #69's question): `kit policy` fails wherever it runs, or on a CI
   runner only. Default: everywhere. The rule has no exception, and `kit commit` never reads the forge.
2. A JSON verb when part of the picture failed: the document and exit 1, or exit 0 and an `errors` array. Default:
   the document and exit 1, as `kit flow incoming` does and both headers say. Plan 3a chose 0 for `--workspace`.
3. A list longer than a page: read every page and fail past 10000 items, or read one page and say "first N of M"
   with exit 0. Default: every page; the other is a part that exits 0. GitLab pays four reads a merge request.
4. The default branch with no `<primary>/HEAD` (Workflow 25, Flow 29): keep `main`, or ask the
   remote and fail when it cannot say. Default: keep `main`: a remote added by hand has none, as in most tests.
5. What approves a weakening (issue #67's questions): a code owner's review, a label, a signed trailer, or none.
   Default: none. The base's rule judges the pull request that loosens it, so no change to a pattern is judged.

## Global Constraints

- kit commit only, never kit submit or push, after `export PATH=$PWD/kit/bin:$PATH`. A subject starts with `#69`,
  Task 8's with `#67`, and a second `--message` holds `Co-Authored-By: Claude <noreply@anthropic.com>`.
- Peers share the worktree: stage named paths only, guard that the index holds only them, never leave anything staged.
- Everything under `kit/` is written for a consumer, and a path in it is written from `kit/`: `modules/bash/1st.sh`.
- Lines are 120 cells at most.
- A verb's header comment is its help page and its usage line is its grammar (`kit syntax --check` holds them): a
  changed exit is written into the header's `Exits` line, and `docs/` says the same.
- The exit codes stay: 0 passed, 1 refused or could not run, 2 bad arguments. A failure is one ❌ line on stderr,
  the remedy and then what could not be read: `run: gh auth status (the pull requests could not be read)`.
- `wait "$!"` after a process substitution is bash 4.4's, the kit's floor; `projects/tui` stays at or above 98%.
- A bats file runs as `KIT_TOOLCHAIN=mise kit/bin/kit tools run bats tests/<file>.bats`; `run !` and `run
  --separate-stderr` need `bats_require_minimum_version 1.5.0` at its top. Step 4 ends with `kit lint`.
- Not here, but in the plan "Refusals that hold": `kit rebase`, what `kit tidy` deletes, `kit serve`, and regions.

## Review Focus

1. A read nested in another command's words, `"$(forge:pr-body …)"`: no scan sees it. Task 2 names five found.
2. A forge that answers with what is not JSON and exits 0, as a proxy's page: the call returns 1 (Task 4's test).
3. A repository with no forge, origin a local path: `change` is null and `kit policy` passes (Task 5's test).
4. A `kit.yml` yq cannot read: every verb stops with yq's line and exit 1, never as "no kit here" (Task 2's test).
5. A path staged, then removed before the hook runs: `kit lint` and `kit generate --check` exit 2, naming it.

---

## Task 1: `list:read`, and every `< <(`

**Files:** modify `1st.sh` (after `list:without`) and each file the scan names; test `tests/use.bats`,
`tests/modules.bats`, `tests/packages.bats`, `tests/tools.bats`.

**Interfaces:** `list:read <array> <command> [<argument>…]` empties the named array, runs the command, and fills
the array with its lines, a trailing empty line dropped. It returns the command's status, the array left empty when
that is not 0; stdin reaches the command. `list:read0` does the same for items that end in NUL: `readarray -d ''`
from a process substitution, then `wait "$!"`. A loop over fields becomes `list:read rows <command>… || return 1`,
`for row in "${rows[@]}"`, and `IFS=$'\t' read -r a b <<<"${row}"`; a verb exits, with `$?` where the producer
returns 2. `read -r … < <(producer) || return` stays: `read` fails on no output.

| Site | What fails | Taken for today | Must do |
| --- | --- | --- | --- |
| `bin/kit-update:52`, `bin/kit-install:44`, `package.sh:108,209,291` | `resolve:active` at a cycle (Delivery 5) | the packages before the cycle: one dropped, exit 0 | exit 1 after the cycle's line, `kit.yml` untouched; `resolve:active` prints nothing until the sort ends |
| `toolchain/brew.sh:34`, `bin/kit-tools:293` | `brew bundle list`, `brew list` (Command line 8) | nothing missing: "✅ brew installed"; `update` exits 1 unsaid | return 1 after `run: brew bundle list --file Brewfile (the Brewfile's tools could not be read)`; `update` says `<name>: the tools could not be updated; see above` |
| `bin/kit-lint:57,59,69`, `commit.sh:404,478,555` | `git diff --cached`, `git ls-files`, `git status` | nothing staged, "nothing to lint" | exit 1 after git's line |
| `bin/kit-lint:92`, `context.sh:336,395`, `template.sh:181`, `issue.sh:253` | `context:rules` (Task 6), `forge:pr-closing-issues` | no rule governs the path; the change closes nothing | return 1 |
| `1st.sh:116` | `bash_path` in a `kit.yml` yq cannot read | the kit's modules alone | read through `use:path-setting <root>`; `use:load-path` returns 2 |
| `serve.sh:458` | jq on a NUL in an argument (Serve 5) | the verb runs with fewer words | the call fails, nothing run |
| the other 130 or so: `settings:packages`, `settings:excludes` (22), the `yaml:` readers (16), `resolve:*` (17), `workspaces:*` (14), `git`, `git:*` (14), `place:*`, `render:layers`, `manifest:paths` (11), `forge:noun`, `forge:remotes` (7), `tracker:records` (5), `template:*`, `toolchain`, `dependencies:usable`, `sources:*` | the producer: a YAML file yq cannot read, git, the forge | an empty list | the same conversion |

- [ ] **Step 1: Write the failing tests**: the scan, in `tests/modules.bats`, and the three in the table.

```bash
@test "no verb or module reads a command's lines through < <(…), which loses its status, unless it cannot fail" {
  local pure='jq|awk|sed|printf|cat|find|compgen|for|while|"\$@"|list:without|use:find|use:modules|help:[a-z-]+'
  pure+='|region:markers|flow:(label|stage-args|fetch-diff)|context:(governed-by|embedded|read-kinds)'
  pure+='|commit:placed|welcome:[a-z-]+|serve:(watched|changed|resources)|new:seam-calls|generate:twice'
  pure+='|tracker:body-refs'
  local kept="< <\\((${pure})[ )]|(^|[ ;])read -r [^<]*< <\\(.*\\) \\|\\| (return|exit|continue)|^[^:]+:[0-9]+: *#"
  run bash -c 'grep --recursive --line-number "< <(" "$1/bin" "$1/modules/bash" | grep --invert-match -E "$2"' \
    scan "${kit_root}" "${kept}"
  [ -z "$output" ] || { printf '%s\n' "$output"; return 1; }
}
```

| Test | Given | Asserts |
| --- | --- | --- |
| `tests/use.bats`: "list:read fills an array with a command's lines and returns its status, the array empty when it fails" | `run kit-bash 'bad() { printf "a\n"; return 3; }; nul() { printf "x y\0z\0"; return "$1"; }; list:read rows printf "a\n\nb c\n"; echo "$? ${#rows[@]} [${rows[2]}]"; list:read rows bad; echo "$? ${#rows[@]}"; list:read0 items nul 0; echo "$? ${#items[@]} [${items[0]}]"; list:read0 items nul 4; echo "$? ${#items[@]}"'` | `[ "$output" = $'0 3 [b c]\n3 0\n0 2 [x y]\n4 0' ]` |
| `tests/packages.bats`: "a cycle among the packages refuses packages add with 1, and kit.yml stays as it was" | packages `a` and `b` of the repository's own, written as `tests/context.bats:94` writes one, each depending on the other; `kit install`; `cp kit.yml before.yml`; `run "${kit_checkout}/bin/kit" packages add a` | `[ "$status" -eq 1 ]`; `[[ "$output" == *"form a cycle among"* && "$output" != *"✅"* ]]`; `cmp kit.yml before.yml` |
| `tests/tools.bats`: "brew filling in fails when the Brewfile's tools cannot be read", beside "brew filling in lists…" | `route brew 'bundle list --file Brewfile --formula --cask' '!1'`; `run "${kit_root}/bin/kit" tools install` | `[ "$status" -eq 1 ]`; `[[ "$output" == *"could not be read"* && "$output" != *"✅ brew installed"* ]]` |

- [ ] **Step 2: Run them to see them fail**: `list:read: command not found`; the scan prints 151 lines.
- [ ] **Step 3: Write the two helpers, then convert every line the scan prints**, a module at a time.
- [ ] **Step 4: Run the whole suite**, `bin/ci`: all pass, the scan among them.
- [ ] **Step 5: Commit**, subject `#69 kit: list:read keeps a producer's status, and every < <( reads through it`.

## Task 2: A function that leaned on errexit

**Files:** modify the sites below; test `tests/modules.bats`, and `commit.bats`, `lint.bats`, and `work.bats` there.

**Interfaces:** a function called under `if`, `||`, or `&&` runs with errexit off, so `x=$(read)` that fails leaves
`x` empty and goes on. Every assignment from a forge or tracker read, or from `git:default-branch`, carries `||
return`. `issue:assign` returns 1, after its warning, when an issue could not be read or assigned.

| Site | What fails | Taken for today | Must do |
| --- | --- | --- | --- |
| `issue.sh:432` (Workflow 4) | `forge:pr-body` | an empty body: `pr-set-body` writes `Closes #N` over the report | `\|\| return 1` before any edit |
| `commit.sh:73`, `facts.sh:70`, `issue.sh:492`, `git.sh:304`, `releases.sh:57`, `flow.sh:427` (Workflow 5) | `git:default-branch`, as when `primary` names no remote | no default branch: the refusal to commit on it never fires | `\|\| return 1` |
| `issue.sh:62,66`, `bin/kit-assign:35` (Workflow 17) | the assignees' read, the assignment | unassigned; `kit assign` exits 0 | `issue:assign` warns, goes on, and returns 1; `issue.sh:215,497` warn and go on |
| `issue.sh:229,295,417`, `commit.sh:139`, `git.sh:119,142` | `forge:pr-url`, `forge:branch-for-issue`, the branch's change | no branch, no change: `kit work` makes a second | `\|\| return 1`; `kit commit` warns that the change could not be read; `kit tidy` keeps the worktree, and says so |
| `settings.sh:202`, `materialize.sh:314`, `bin/kit-hydrate:37` (found while verifying) | `settings:value version` on a `kit.yml` yq cannot read | no kit declared: nothing hydrated, `kit hydrate` exits 0, `kit lint` says ✅, `kit check` says `run: kit install` | `settings:version` returns 1 after yq's line; `materialize:hydrate`, `settings:require`, and `kit hydrate` stop with 1 |
| `package.sh:109,162`, `uninstall.sh:22` (Delivery 12) | `render:all` | rendered: `kit.yml` declares a package with no file placed | `\|\| return 1` |
| `bin/kit-tools:148-170,272-285` (Command line 2) | a locked provider, unusable before the toolchain installs | no layer: `install` skips npm and uv, exit 0 | install the toolchain layers, read `dependencies:toolchain-dirs` and `dependencies:usable` again, and exit 1 naming a provider whose lock is here and whose tool is still not on PATH |
| nested: `bin/kit-linked:68`, `issue.sh:236,264`, `forge/github.sh:303`, `forge/gitlab.sh:301` | the read inside `$(…)` | empty | read into a variable first, guarded |

- [ ] **Step 1: Write the failing tests.** The scan, "no module assigns a forge or tracker read unguarded": once
  a line that ends in a backslash joins the next, no line of `kit/modules/bash` without `||` or `if` matches
  `=\$\((forge:(pr-|issue-|branch-for-issue|repo|me)|tracker:(of|at) |git:default-branch|forge/gitlab:api )`.

| Test | Given | Asserts |
| --- | --- | --- |
| `tests/commit.bats`: "a body that cannot be read is never replaced: the commit stands, and kit says to add the line by hand" | `forge-world`; `sed -i.bak '/^pr view 3 /d' "${FAKE_DIR}/gh.routes"`; `printf 'x\n' > x && git add x`; `run kit commit --message '#9 x: a file' x` | `[ "$status" -eq 0 ]`; `[[ "$output" == *"could not be added to change 3's body"* ]]`; `run ! grep --quiet '^pr edit' "${FAKE_DIR}/gh.calls"` |
| `tests/commit.bats`: "with a primary that names no remote, kit commit refuses, since the default branch could not be read" | `printf 'primary: upstream\n' >> kit.yml`; `x` staged; the same `kit commit` | `[ "$status" -eq 1 ]`; `[[ "$output" == *"is not a remote"* && "$output" != *"✅ committed"* ]]` |
| `tests/lint.bats`: "a kit.yml yq cannot read stops kit lint and kit hydrate, never as a repository with no kit" | `printf 'packages: [kit\n' > kit.yml`; `run "${kit_checkout}/bin/kit" lint good.sh`; then `hydrate` | `[ "$status" -eq 1 ]` and `[[ "$output" != *"✅"* ]]`, for both |
| `tests/work.bats`: "kit assign exits 1 when an issue could not be assigned" | its forge world, `issue edit 7 --add-assignee *` routed `'!1'`; `run kit assign 7` | `[ "$status" -eq 1 ]`; `[[ "$output" == *"could not be assigned"* ]]` |

- [ ] **Step 2: Run them**: the scan names 17 lines; `pr edit 3 --body` is among the calls.
- [ ] **Step 3: Guard each site**; `forge/gitlab.sh:165-167` take `|| return 1` here, and Task 4 rewrites them.
- [ ] **Step 4: Run** the four files, and `tests/linked.bats`, `tools.bats`, `packages.bats`, `tidy.bats`.
- [ ] **Step 5: Commit**, subject `#69 kit: a read or a render that fails stops the function that asked`.

## Task 3: Two idioms that hide a status: `| grep --quiet`, and `|| true`

**Files:** modify `welcome.sh:248`, `kit/bin/kit-flow:330`, `workspaces.sh:235,359`, `1st.sh:437-438`, and the 92
lines of `kit/bin` and `kit/modules/bash` with `|| true` or `|| :`; test `tests/modules.bats`, `tests/welcome.bats`.

`grep --quiet` exits at its first match, the producer dies of SIGPIPE, and pipefail makes the match a failure
(Command line 5: 1492 runs of 1500). Each site captures the producer's text, with `|| return 1` where it can fail, then
matches it with `<<<`. A `|| true` stays, its reason in a comment, where the failure means "none", as `grep`'s does,
or costs only speed; it becomes `|| return 1` where the output is then read as data (sketch 2).

- [ ] **Step 1: Write the failing tests.** Two scans. "No pipeline ends in grep --quiet": with a line that ends
  in `|` or a backslash joined to the next, no line of `kit/bin` or `kit/modules/bash` matches
  `\| *grep [^|]*(--quiet|-q)( |$)`. "Every || true says why it swallows": a line that holds `|| true` or `|| :`
  ends in a comment, or the line before its statement is one. In `tests/welcome.bats`, "a provider is found
  whichever the list names first": with `package.json` and `pyproject.toml`, each with its lock, and `fake-script
  npm 'exit 0'` and `fake-script uv 'exit 0'`, `check dependencies-npm` never holds `is not on PATH`, in 20 runs.
- [ ] **Step 2: Run them**: the first scan names the five sites, and the check fails on most runs.
- [ ] **Step 3: Rewrite the five sites, and give each `|| true` its reason or its guard.**
- [ ] **Step 4: Run the whole suite**, `bin/ci`.
- [ ] **Step 5: Commit**, subject `#69 kit: no pipeline ends in grep --quiet, and every || true says why`.

## Task 4: The forge seam's contract, and the test that holds every provider to it

**Files:** modify `forge.sh` (header, `forge_list_limit=10000`), `forge/github.sh`, `forge/gitlab.sh`,
`kit/modules/jq/forge.jq`; test `tests/forge-conformance.bats`.

**Interfaces:** the contract, in `forge.sh`'s header. A call returns 0 and prints its answer; one marked "or
nothing" returns 0 and prints nothing only when the forge answered that there is none. Otherwise it returns 1,
prints nothing on stdout, and says one ❌ line that ends `(<what> could not be read)`. A list is whole or the call
fails: it reads every page, and fails at `forge_list_limit` items, saying `kit reads at most 10000`.
`forge/gitlab:ask <name> <what> <path> [<glab api option>…]` is `forge/github:ask` for `glab api`, saying `run: glab
auth status`; both fail an answer that is not the JSON asked for. `forge/gitlab:pages <name> <what> <path>` joins
the arrays `glab api --paginate` prints, a flag to confirm first: glab was not installed where this plan was written.

| Site | What fails | Taken for today | Must do |
| --- | --- | --- | --- |
| `github.sh:17-19,219,235,248`, `gitlab.sh:26-30,210-213,227-230,243-247` | the read, behind `2>/dev/null \|\| true` | no branch, no change | through `ask`: return 1 |
| `github.sh:224-232` | `gh pr view`, which exits 1 for "none" and for a failure alike | none | the event's number, else `gh pr list --head <branch> --state open` through `ask`; a detached HEAD has none |
| `github.sh:31,58,192,252`, `gitlab.sh:45-47,76-83,187-193,249-251` | the read; GitLab's `issue-state` prints `OPEN` for an error body | a status with no line, or an answer | through `ask` |
| `gitlab.sh:162-183` (Flow 2) | a merge request's discussions or closes, or jq | the merge request dropped, exit 0, the part kept as the snapshot | each read through `ask`, and jq's status returned |
| `gitlab.sh:165` (Flow 3) | the jobs of a fork's pipeline, asked under the target project | the merge request dropped | read under `.head_pipeline.project_id`, the target project where it names none; jobs that cannot be read are one check, `pipeline`, of `.head_pipeline.status` |
| `gitlab.sh:131,153`, `github.sh:112,143` (Flow 4) | a list past its first page | the whole list | `pages`; GitHub's pull requests loop on `pageInfo`, as `forge/github:rest-of-closes` does; `gh issue list --limit "${forge_list_limit}"` |
| `github.sh:152,155` | more than 100 threads or check contexts | all of them | ask `totalCount`; where it is more than the nodes, return 1 naming the pull request |
| `github.sh:308-309`, `gitlab.sh:156-158,173-174` | a run's jobs, the labels' colors, a fork's owner | none | stay: each says why in its comment |

- [ ] **Step 1: Write the failing tests** in `tests/forge-conformance.bats`:

```bash
@test "a forge that cannot answer fails every read with one line, and never answers nothing" {
  for forge in github gitlab
  do
    world "${forge}"
    printf 'repo view*\trepo.json\n' | tee "${FAKE_DIR}/gh.routes" > "${FAKE_DIR}/glab.routes"
    for call in 'branch-for-issue 7' 'pr-for-branch 7-fix-the-thing' 'pr-last-for-branch 7-fix-the-thing' \
      'pr-state-for-branch 7-fix-the-thing' pr-here 'pr-title 3' 'pr-body 3' 'pr-author 3' 'pr-closing-issues 3' \
      'issue-state 7' 'issue-assignees 7' 'issue-title 7' open-prs open-issues releases
    do
      read -r -a words <<<"${call}" && run --separate-stderr call "${words[@]}"
      [[ "$status" -eq 1 && -z "$output" && "$stderr" == *"could not be read)"* ]] \
        || { echo "${forge} ${call}: exit ${status}, printed '${output}', said '${stderr}'"; return 1; }
    done
  done
}
```

| Test | Given | Asserts |
| --- | --- | --- |
| "one merge request's read that fails fails the open changes, and no part is kept as the snapshot" | `world gitlab`; `mrs-list.json` of `[{"iid": 3}, {"iid": 4}]`; `merge_requests/4/discussions*` routed `'!1'` ahead of the world's routes; `run call open-prs` | `[ "$status" -eq 1 ]`; `[ ! -f "$(snapshot gitlab open-prs)" ]` |
| "a list in pages is read whole" | `world gitlab`, its routes for lists ending in `*`, since the call gains `--paginate`; `issues.json` holding two arrays, `[…][…]`, one issue in each; `run call open-issues` | `[ "$(jq --slurp length <<<"$output")" -eq 2 ]` |
| "a fork's pipeline that cannot be read is one check, the pipeline's own status" | `world gitlab`; `mr.json` with `head_pipeline` of `{"id": 11, "project_id": 99, "status": "failed"}`; `projects/99/pipelines/11/jobs*` routed `'!1'`; `run call open-prs` | `[ "$(jq --compact-output '[.checks.total, .checks.failing]' <<<"$output")" = '[1,"pipeline"]' ]` |
| "an answer that is not JSON fails the read" | `world github`; `issues.json` holding `<html>`; `run call open-issues` | `[ "$status" -eq 1 ]` |

- [ ] **Step 2: Run them**: the first fails at `github branch-for-issue 7: exit 0`.
- [ ] **Step 3: Write the contract into the header, `forge/gitlab:ask` and `pages`, and bring every row over.**
- [ ] **Step 4: Run** `tests/forge-conformance.bats`, `forge.bats`, `flow.bats`, `changes.bats`, `backlog.bats`.
- [ ] **Step 5: Commit**, subject `#69 kit: a forge call answers, answers none, or fails; a list is whole`.

## Task 5: What the verbs do with a forge that cannot answer

**Files:** modify `facts.sh:84-92`, `kit/bin/kit-policy`, `kit/bin/kit-linked:37,68`, `marks.sh:32`,
`docs/policy.md:46`; test `tests/policy.bats`, `tests/linked.bats`, `tests/flow.bats`.

**Interfaces:** `facts:change` prints the change for `facts:document`, and returns 0: `null` where `forge:provider`
names no forge or the forge answered that there is no open change; the change's object; or `{"error": "<line>"}`,
the line the provider said, less its ❌, when a read failed. `kit policy` refuses a document whose `change` has
`error` before the engine runs, as it refuses `commits` of null, so both engines and `--facts=<file>` refuse it:
`the change could not be read, so its title is unchecked: <error>; then run: kit policy`, exit 1. `kit linked`
exits 1 with the provider's line when `forge:pr-here` or the body's read fails. `marks:stages` lets the provider's
line through, prints the forge's marks as null, and returns 1 (Flow 23). `kit ci` counts each refusal.

- [ ] **Step 1: Write the failing tests**; the file's `change` routes `pr list --head *` to `[{"number": 3}]`.

| Test | Given | Asserts |
| --- | --- | --- |
| `tests/policy.bats`: "a forge that cannot answer is an error in the facts, and kit policy refuses it" | `branch 7-fix-the-thing "#7 cli: fix the thing"`; `change "Fix the thing"`; `: > "${FAKE_DIR}/gh.routes"`; `run "${kit_root}/bin/kit" policy` | `[ "$status" -eq 1 ]`; `[[ "$output" == *"the change could not be read, so its title is unchecked"* ]]`; `[ "$("${kit_root}/bin/kit" policy --input \| jq --raw-output '.change \| keys[]')" = error ]` |
| `tests/policy.bats`: "with no forge, and with a forge that says there is no change, the change is null" | the same branch, no `change`; then `forge: github` in `kit.yml` and `pr list --head *` routed to `[]` | `kit policy` exits 0 both times; `.change` of `kit policy --input` is `null` both times |
| `tests/linked.bats`: "linked fails with the forge's line when the forge cannot answer" | the file's failing gh shim; `run "${kit_checkout}/bin/kit" linked` | `[ "$status" -eq 1 ]`; `[[ "$output" == *"could not be read"* && "$output" != *"kit linked <number>"* ]]`; the test at line 14 gets a fake gh that answers `[]` |
| `tests/flow.bats:618`, "kit flow --marks … and none a stage it cannot read" | as it is | exit 1 where the forge cannot be read, `.desk` still a string |

- [ ] **Step 2: Run them**: `.change` is null, and `kit policy` exits 0.
- [ ] **Step 3: Write `facts:change`, the refusal, and the guards**; `docs/policy.md`'s `change` row says them.
- [ ] **Step 4: Run** the three files, and `tests/facts.bats`, `ci.bats`, `house-policy.bats`.
- [ ] **Step 5: Commit**, subject `#69 kit: the facts carry the forge's error, and kit policy refuses it`.

## Task 6: A check that classified or read nothing says so

**Files:** modify `context.sh`, `kit/bin/kit-lint`, `generate.sh`, and `materialize.sh`; test `tests/lint.bats`,
`tests/context.bats`, and `tests/generate.bats`. The workspace file's checks (Serve 8, 21, and 27) are
the plan Workspaces 4's, in its Tasks 1 and 4.

| Site | What fails | Taken for today | Must do |
| --- | --- | --- | --- |
| `context.sh:265-282` (Delivery 7) | awk dies on a kind's `content:` that is no regular expression | every path kindless: `kit lint` and `kit check` pass | `context:classify` returns 1 after `<file>: the kind <kind>'s content pattern '<pattern>' is not a regular expression`; `context:kinds` and `context:rules` return it |
| `context.sh:256` (Delivery 30) | another worktree's compile deletes the matcher in use | no kinds | prune matchers older than a day, never the others |
| `context.sh:80-113` | a repository that declares a kit version has no kind | nothing governed | return 1: `no kind is defined under .kit/kinds; run: kit hydrate` |
| `bin/kit-lint:87,139` (Documents 7) | a named path that is no file; no rule governs any path | skipped; "✅ every governed path passes" | exit 2: `'<path>' is not a file or a directory here`; and `ok "no rule governs the <n> path(s); nothing was linted"`, exit 0 |
| `generate.sh:378-381` (Delivery 25) | a named path that is no file | "no kit:generate directive to expand" | `generate:run` returns 2 with `kit lint`'s line |
| `materialize.sh:274-297` (Delivery 8) | a file added under `.kit/packages/` since the last full run | hydrated: the pulled rule is not in force | return 1 when `find .kit/packages -newer .kit/.gitignore -print -quit` prints a path |

- [ ] **Step 1: Write the failing tests.**

| Test | Given | Asserts |
| --- | --- | --- |
| `tests/lint.bats`: "a path that is not there is refused by name" | `run "${kit_checkout}/bin/kit" lint nosuch.sh` | `[ "$status" -eq 2 ]`; `[[ "$output" == *"'nosuch.sh' is not a file or a directory here"* ]]` |
| `tests/lint.bats`: "a kind whose pattern is no regular expression fails the lint, and names the kind" | `"${kit_checkout}/bin/kit" new kind odd --content='^#!.*(odd'`; `"${kit_checkout}/bin/kit" packages add local`; `run "${kit_checkout}/bin/kit" lint bad.sh` | `[ "$status" -eq 1 ]`; `[[ "$output" == *"the kind odd's content pattern"* && "$output" != *"✅ every governed"* ]]` |
| `tests/generate.bats`: "a named path that is not there is refused" | `run "${kit_checkout}/bin/kit" generate --check typo/AGENTS.md` | `[ "$status" -eq 2 ]` |
| `tests/context.bats:196`, "two lookups that compile changed kinds at once…" | as it is | the matcher of other kinds, `kinds-0.awk`, is still there afterwards |

- [ ] **Step 2: Run them**: `kit lint nosuch.sh` exits 0 with ✅.
- [ ] **Step 3: Write each row.** `kit-lint`'s and `kit-generate`'s `Exits` lines add "or a path not there".
- [ ] **Step 4: Run** the four files, and `tests/materialize.bats`, `across.bats`, `kinds.bats`.
- [ ] **Step 5: Commit**, subject `#69 kit: a check refuses a path that is not there and kinds it cannot compile`.

## Task 7: One exit contract for the verbs that print JSON, and kittui's Runner

**Files:** modify `flow.sh`, `across.sh`, `kit/bin/kit-flow`, `kit/bin/kit-workspace`, and `kit/kit.go` and
`flow/flow.go` under `projects/tui/internal/`; test `tests/flow.bats`, `tests/across.bats`, and the Go tests there.

**Interfaces:** the contract, for every form of `kit flow` and `kit workspace` that prints JSON. Exit 0: the
document is whole, and there is one. Exit 1 with a document: it is what could be read, and it names in itself what
could not, a remote's `error`, a member under `errors`, a stage of null. Exit 1 with nothing on stdout: nothing
could be read, and stderr's last ❌ line says why. Exit 2: the arguments. So `flow:above` returns 1 when a desk's
`kit flow` did (Flow 14); `across:merge` returns 1, after its document, when a member's status is not 0, and
`across:stage`, `across:picture`, and kit workspace's `document` return it, the verb printing the document first;
`across:quota-merge` says each failed member's line and returns 1 (Serve 12). In Go, `kit.ExitError` gains
`Stdout []byte`, which `Runner.output` fills. `partial(out []byte, err error) ([]byte, error)` gives the `Stdout`
and nil for an `*ExitError` of code 1 whose `Stdout`, trimmed, is valid JSON that opens with `{` or `[`, and its
arguments otherwise; `Status`, `MarksOf`, and `Quota` return through it (Flow 6, Apps 2). `shape`
and `across` in `flow.go` refuse nothing, `null`, and `{"rows": null}`: `printed nothing` (Apps 22).

- [ ] **Step 1: Write the failing tests.** In `kit_test.go`, indented as gofmt has it:

```go
func TestAStageThatPrintsItsDocumentAndExitsOneIsItsDocument(t *testing.T) {
    ctx := context.Background()
    r := script(t, `echo '{"fetched": true, "remotes": [{"name": "fork", "error": "fatal: no route"}]}'; exit 1`)
    if out, err := r.Status(ctx, "incoming", nil); err != nil || !strings.Contains(string(out), "fatal: no route") {
        t.Fatalf("Status = %q, %v", out, err)
    }
    if marks, err := script(t, `echo '{"desk": "1", "changes": null}'; exit 1`).Marks(ctx); marks["desk"] != "1" {
        t.Errorf("Marks = %v, %v", marks, err)
    }
}
```

| Test | Given | Asserts |
| --- | --- | --- |
| `kit_test.go`: `TestARefusalWhatIsNotJSONAndAnotherExitStayTheError` | `script(t, body).Status(ctx, "incoming", nil)` for each body of `echo '❌ no gh' >&2; exit 1`, `echo 'not json'; exit 1`, `echo '{}'; exit 2` | `err != nil` for each |
| `edges_test.go`: `TestNothingOrNullOnStdoutIsARefusalNotAnEmptyStage` | `shape("backlog", []byte("null"))`, `shape("incoming", nil)`, `across("backlog", []byte("{\"rows\": null}"))` | each returns an error that holds `printed nothing` |
| `tests/flow.bats`: "above desks, kit flow exits 1 when a desk's picture did, and still prints every desk" | two desks under one directory, one with a second remote whose path is gone; from the directory, `run --separate-stderr kit flow --json` | `[ "$status" -eq 1 ]`; `[ "$(jq length <<<"$output")" -eq 2 ]` |
| `tests/across.bats:116`, "a member whose checkout is gone…" | as it is | `[ "$status" -eq 1 ]`, its document as before; the test at line 127, no member cloned, stays 0 |

- [ ] **Step 2: Run them**: `Status` returns the `ExitError`; `kit flow --json` above desks exits 0.
- [ ] **Step 3: Write the bash and the Go.** Both headers' `Exits` lines, `docs/modules.md`'s exits, and
  `docs/workspaces.md:77-79` say "1 when … cannot be read: with --json, what was read is still printed".
- [ ] **Step 4: Run** the bats files, `(cd projects/tui && go vet ./... && go test ./...)`, and the gate: 98.0%.
- [ ] **Step 5: Commit**, subject `#69 kit, kittui: a JSON verb that read a part prints the part and exits 1`.

## Task 8: CI judges a pull request by its base branch's policy (#67)

**Files:** modify `policy.sh` (its header too), `kit/bin/kit-policy`, `kit/bin/kit-linked:53`, `docs/policy.md`; test
`tests/policy.bats`, `tests/linked.bats`, `tests/forge-conformance.bats`.

**Interfaces:** `policy:base` prints `origin/<default branch>`, the ref `facts:document` counts commits from. It
returns 1 when the ref is not here: `the base, origin/<default>, is not here, so its policy cannot be read; fetch it
with full history` (sketch 4). `policy:from-base <dir>` calls `policy:from "$(policy:base)" <dir>`, which sets
`settings_file` and `policy_rego`. Then it makes `<dir>/kit.yml` the disk's `kit.yml` with `policy`,
`policy_engine`, and `linked_exempt` as the base's has them, absent where it has none, so `forge`, `primary`, and
`facts` stay the checkout's. It adds to `<dir>/rego` each `.rego` a package in the base's `packages:` delivers to
`.kit/policy/` and the disk lacks: the repository's own from the base's `.kit/packages/`, the release's from
`${kit_root}/packages/`. On a branch past the default one, `kit policy` calls `policy:from-base` before it gathers
the facts, and `kit linked` reads `linked_exempt` from the base's `kit.yml`; on the default branch both read the disk.

- [ ] **Step 1: Write the failing tests.**

| Test | Given | Asserts |
| --- | --- | --- |
| `tests/policy.bats`: "a branch is judged by the policy on its base: a pattern it loosens, and a login it exempts, change nothing" | `kit.yml` of `policy: {commit_subject: "^#[0-9]+ "}`, committed and pushed to main; `branch 7-fix-the-thing "no reference"`; on disk, `kit.yml` of `policy: {commit_subject: ""}` and `linked_exempt: [ada]`; `change "Fix the thing" ada`; `run "${kit_root}/bin/kit" policy` | `[ "$status" -eq 1 ]`; `[[ "$output" == *"does not match policy.commit_subject"* && "$output" != *"exempt"* ]]` |
| `tests/policy.bats`: "Rego a branch removes still judges it" | `kit_bash_modules=policy`; a package of the repository's own that delivers `x.rego`, committed on main, removed on the branch; `run kit-bash "policy:from-base '${dir}'"`, `dir` a scratch directory | `[ -f "${dir}/rego/x.rego" ]` |
| `tests/policy.bats`: "a base that is not here fails" | `git update-ref -d refs/remotes/origin/main` on a branch; `run "${kit_root}/bin/kit" policy` | `[ "$status" -eq 1 ]`; `[[ "$output" == *"its policy cannot be read"* ]]` |
| `tests/forge-conformance.bats:389,402`, the tests of `kit linked` | each commits its `kit.yml` and points `refs/remotes/origin/main` at the commit | as they are |

- [ ] **Step 2: Run them**: the first exits 0, "exempt from the policy by kit.yml".
- [ ] **Step 3: Write `policy:base`, `policy:from-base`, and the verbs' reads**; `docs/policy.md` says so.
- [ ] **Step 4: Run** the three files, and `tests/house-policy.bats`, `commit.bats`, `ci.bats`.
- [ ] **Step 5: Commit**, subject `#67 kit: kit policy and kit linked judge a branch by the policy on its base`.
