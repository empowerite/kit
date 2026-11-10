# Contracts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** kit's public surface changes only as a reviewed diff: every verb's syntax, the MCP tools and resources,
the JSON kittui reads, each verb's exit statuses, and what the docs say of them are each held by a test in the gate.

**Architecture:** A `snapshot` helper in `tests/helper.bash` compares what kit prints with a committed file, and
rewrites it under `KIT_SNAPSHOTS=write`. Two more hold a stage's JSON to kittui's fixtures and to `fields.json`.
`tests/claims.bats` reads every markdown page with an awk extractor and a jq judge that calls `syntax::match`, whose
answer `kit syntax <verb> --match -- <word>…` prints: `status` 0, 1 with `missing`, or 2 with `refusal`.

**Tech Stack:** bash, jq, awk, git, bats; in `projects/tui`, one struct field added and one renamed.

**Spec:** issue #74, "Contracts: snapshot every verb's syntax and the MCP tools, and check the docs' claims against
them", § Design sketch. Read it with `gh issue view 74`.

**Review:** [the findings](../reviews/2026-11-10-code-review-findings.md) of the review of 2026-11-10, cited
here by area and number: Serve 8 is item 8 under its Serve heading.

## Decisions for the owner

1. The docs check is a test, `tests/claims.bats`, not a rule's check that `kit lint` runs (the issue's question):
   `kit lint` is not one of `kit ci`'s four, and most of what the test holds is this repository's alone.
2. The syntax snapshots stay in `projects/tui/internal/syntax/testdata/`, owned by `tests/syntax.bats`: `go test`
   reuses a cached result without rechecking a file outside the module. The others go in `tests/snapshots/`.
3. A record's keys that are no column are declared in a `carries` list on the stage in `fields.json`, 34 keys over
   13 stages, not in the test. It costs one field on `table.Schema`, whose reader refuses unknown keys.
4. `pr.source` in an underway row, which kittui reads to key a GitLab merge request as `!<number>` and kit never
   prints, becomes `pr.ref`: kit prints the reference the changes row carries as `ref`, and kittui reads that.
5. Two names: `KIT_SNAPSHOTS=write`, and the comment `<!-- kit:claims skip -->`, `{/* kit:claims skip */}` in MDX.

## Global Constraints

- kit commit only, never kit submit or push; every subject starts with `#74`, the issue for this plan.
- `export PATH=$PWD/kit/bin:$PATH` before `kit commit`.
- Peers share the worktree: stage named paths only, guard that the index holds only them, never leave anything staged.
- Everything under `kit/` is written for a consumer, and a path in it is written from `kit/`: `modules/bash/x.sh`.
- Lines are 120 cells at most.
- A verb's header comment is its help page and its usage line is its grammar (`kit syntax --check` holds them):
  a new flag is added to the header's usage line and described there, and `docs/` says the same.
- Run one file with `KIT_TOOLCHAIN=mise kit/bin/kit tools run bats tests/<file>.bats`, written `bats <file>` below.
- A snapshot is written only by a test run with `KIT_SNAPSHOTS=write`, and its diff is reviewed. A fixture follows
  kit, except for decision 4. A new file has the banner its kind's others have. `projects/tui` stays at or above
  98% coverage. Each commit names its task's paths, with the session's attribution as a second `--message`.

## Review Focus

1. A code span the wrap split over two lines, five today, is judged whole, on the line it opens on. Task 4.
2. A fence never closed is a finding, and one closes only on its own marker, as `…/styles/just.md:32` needs. Task 4.
3. A command with quotes, a pipe, a redirect, or a trailing comment is judged up to the operator. Task 4.
4. A key that only a later row of a stage holds: `shape` reads every row. Task 3's first test.
5. `KIT_SNAPSHOTS=write` left set in a gate run: `bin/ci` runs bats with it empty. Task 1's first test.

---

## Task 1: A snapshot helper, and every verb's syntax

**Files:** modify `tests/helper.bash`, `tests/syntax.bats:235-243`, `bin/ci:14,62`, `docs/testing.md`, and
`projects/tui/README.md:232-233`. Create `tests/snapshot.bats`, `tests/snapshots/verbs.tsv`, and ten each of
`projects/tui/internal/syntax/testdata/<verb>.json` and `projects/tui/internal/form/testdata/<verb>.golden`.

**Interfaces:** produces `snapshot <path>`: it reads stdin, `<path>` absolute or from the repository's root, and
returns 0 when the file holds exactly that. Otherwise, a missing file too, it prints the diff and `stale: <path>;
rerun the test with KIT_SNAPSHOTS=write, and review the diff`, and returns 1. With `KIT_SNAPSHOTS=write` it writes the
file instead, making its directory, and returns 0.

- [ ] **Step 1: Write the failing tests**, in a new `tests/snapshot.bats` and in place of `tests/syntax.bats:235`:

```bash
@test "snapshot passes on the same bytes, fails on others, writes when told to, and the gate never tells it to" {
  printf 'a\nb\n' > "${BATS_TEST_TMPDIR}/kept"
  snapshot "${BATS_TEST_TMPDIR}/kept" <<<$'a\nb'
  run snapshot "${BATS_TEST_TMPDIR}/kept" <<<$'a\nc'
  [ "$status" -eq 1 ]
  [[ "$output" == *"stale: ${BATS_TEST_TMPDIR}/kept; rerun the test with KIT_SNAPSHOTS=write, and review the diff" ]]
  KIT_SNAPSHOTS=write run snapshot "${BATS_TEST_TMPDIR}/new/kept" <<<'a'
  [ "$status" -eq 0 ] && [ "$(cat "${BATS_TEST_TMPDIR}/new/kept")" = a ]
  grep --quiet '^KIT_SNAPSHOTS= BATS_TEST_TIMEOUT=300 bats ' "${kit_root}/../bin/ci"
}

@test "every verb's syntax is its snapshot under projects/tui/, every snapshot there is a verb's, and so is the list" {
  local verb fixture dir=projects/tui/internal/syntax/testdata
  for verb in "${kit_root}"/bin/kit-*
  do
    [[ -x "${verb}" ]] || continue
    "${kit_root}/bin/kit" syntax "${verb##*/kit-}" | snapshot "${dir}/${verb##*/kit-}.json"
  done
  for fixture in "${kit_root}/../${dir}"/*.json
  do
    [[ -x "${kit_root}/bin/kit-$(basename "${fixture}" .json)" ]] || { echo "no verb has ${fixture}"; return 1; }
  done
  "${kit_root}/bin/kit" syntax --list | snapshot tests/snapshots/verbs.tsv
}
```

- [ ] **Step 2: Run `bats snapshot` and `bats syntax`**: `snapshot: command not found`. Write `snapshot` in
  `tests/helper.bash`, above `in-order`, with a comment as the others have, and put `KIT_SNAPSHOTS=` and a space
  before `BATS_TEST_TIMEOUT=300` at `bin/ci:62`. Now the first passes, and the second says `stale: …/assign.json`.
- [ ] **Step 3: Write the snapshots**: run `bats syntax` with `KIT_SNAPSHOTS=write` set, then `(cd projects/tui && go
  test ./internal/form -update && go test ./...)`. All pass; the new files are `verbs.tsv`, and a `.json` and a
  `.golden` each for assign, close, facts, gather, settle, tutorials, unassign, uninstall, workspace, and workspaces.
- [ ] **Step 4: Write the docs.** `docs/testing.md` gains `## Contracts` after `## Running the tests`: how Step 3
  rewrites a snapshot, and a table of what is held, where, and by which test, which each later task adds a row to.
  `bin/ci:14` and `projects/tui/README.md:232-233` say what the suite now holds.
- [ ] **Step 5: Commit**, subject `#74 tests: every verb's syntax, and the list of verbs, has a snapshot`.

## Task 2: The MCP surface

**Files:** modify `tests/serve.bats`, `kit/bin/kit-serve:16-20`, and `docs/testing.md`. Create
`tests/snapshots/tools.json` and `tests/snapshots/resources.json`. It uses Task 1's `snapshot`.

- [ ] **Step 1: Write the failing tests** in `tests/serve.bats`:

```bash
@test "tools/list and resources/list are their snapshots: each tool's name, hints, and input, and each resource" {
  serve "$(modern 1 tools/list)" "$(modern 2 resources/list)"
  answer 1 | jq --sort-keys .result.tools | snapshot tests/snapshots/tools.json
  answer 2 | jq --sort-keys .result.resources | snapshot tests/snapshots/resources.json
}

@test "every verb is a tool or is named here as not one, and kit-serve's header names the tools" {
  # not offered: each writes, needs a terminal, or is a gate's or a script's own; workspace and workspaces are open
  local tools header withheld='ci commit completion facts gate generate hydrate install linked new policy rebase
    region release root serve submit template tidy tui tutorials uninstall update usage welcome workspace workspaces'
  serve "$(modern 1 tools/list)"
  tools=$(answer 1 | jq --raw-output '[.result.tools[].name] | join(" ")')
  [ "$(tr -s ' \n' '\n' <<<"${tools} ${withheld}" | LC_ALL=C sort)" \
    = "$("${kit_checkout}/bin/kit" syntax --list | cut -f 1 | LC_ALL=C sort)" ]
  header=$(sed -n 's/^#   - \([a-z, ]*\)$/\1/p' "${kit_checkout}/bin/kit-serve" | tr -d , | paste -s -d ' ' -)
  [ "${header}" = "${tools}" ]
}
```

- [ ] **Step 2: Run `bats serve`**: `stale: tests/snapshots/tools.json`, and no bullet of names alone in the header.
- [ ] **Step 3: Rewrite `kit/bin/kit-serve:16-20`.** "Its tools are verbs. These read and report, and change
  nothing:", then the bullet `#   - check, context, flow, help, kinds, lint, packages, syntax, toolchains, tools,
  version` on one line; "These act on issues and pull requests, and their annotations say so; work also makes a
  branch, a worktree, and a pull request:", then `#   - assign, close, gather, settle, unassign, work`. The other
  sentences stay. Retitle the test at `tests/serve.bats:190`: "a reading tool changes nothing: …".
- [ ] **Step 4: Write the two snapshots** with `KIT_SNAPSHOTS=write`: 17 tools, each with `annotations` and
  `inputSchema`, and 35 resources, the same on a second run. `bats serve`, `bats help`, and `kit syntax --check` pass.
- [ ] **Step 5: Commit**, subject `#74 kit: kit serve's tools and resources have snapshots; its header names them`.

## Task 3: The JSON kittui reads

**Files:** modify `tests/helper.bash`, `tests/snapshot.bats`, `tests/flow.bats`, `tests/workspace.bats`,
`docs/testing.md`, `kit/modules/jq/underway.jq:20`, `projects/tui/internal/table/catalog.go:15-20`, and, under
`projects/tui/internal/flow/`, `rows.go:56,597,637-647`, `fields.json`, and `testdata/`'s `backlog.json`,
`underway.json`, `changes.json`, and `incoming-no-fetch.json`.

**Interfaces:** produces `table.Schema.Carries []string`, tagged `json:"carries"`; an underway row's `pr.ref`; and:

- `shape` reads JSON on stdin and prints each key path once, sorted, a list's items under `[]`:
  `jq -r '[paths | map(if type == "number" then "[]" else "." + . end) | join("")] | unique[]'`.
- `same-keys <fixture>` reads kit's JSON on stdin, `<fixture>` a path as `snapshot` takes one. Where the two shapes
  differ it prints `only kit prints: <paths>` and `only <fixture> has: <paths>`, each when it has any, and returns 1.
- `fields-hold <stage> [<catalog>]` reads a stage's JSON on stdin; the catalog is kittui's `fields.json` unless
  given. Over the records at the stage's `record`, it prints the three lines the test shows, each when it has any,
  and returns 1 when it printed. A field's `path` is split at each comma; a part after a `<stage>:`, and `.member`,
  which only a workspace's rows have, are passed over. A path's first key, or one in `carries`, is declared.

- [ ] **Step 1: Write the helpers' failing test** in `tests/snapshot.bats`, then the helpers in `tests/helper.bash`:

```bash
@test "shape gives every key path of every row, and fields-hold names what a catalog and a stage do not share" {
  run shape <<<'{"a": [{"b": 1}, {"c": null}], "d": []}'
  [ "$output" = $'.a\n.a[]\n.a[].b\n.a[].c\n.d' ]
  printf '%s\n' '{"stages": [{"stage": "s", "record": ".rows[]", "carries": ["key", "gone"], "fields": [' \
    '{"path": ".a, .b.c"}, {"path": ".member"}, {"path": "t:.x"}, {"path": ".no"}]}]}' > "${BATS_TEST_TMPDIR}/c.json"
  run fields-hold s "${BATS_TEST_TMPDIR}/c.json" <<<'{"rows": [{"a": 1, "b": {"c": 2}, "key": "k", "extra": 0}]}'
  [ "$status" -eq 1 ] && [ "${lines[0]}" = "s: declared and never printed: .no" ]
  [ "${lines[1]}" = "s: printed and not declared: extra" ] && [ "${lines[2]}" = "s: carried and never printed: gone" ]
}
```

- [ ] **Step 2: Write `contract-world`** in `tests/flow.bats`, after `empty-forge`: `forge-world`, then a value under
  every key a stage prints, as it ran in scratch. Issue 11 in `issues.json` gains a label, a `name` and a `color`.
  Change 3 in `open-prs.json` is edited with jq, as `tests/underway.bats:115-117` edits one: `labels: {nodes: [that
  label]}`, and the same on its closing issue; `isCrossRepository: true`; `headRepositoryOwner: {login: "cy"}`; and a
  commit of `{committedDate: <two hours ago>, author: {user: {login: "dee"}}, statusCheckRollup: {contexts: {nodes:
  [<a CheckRun named ci, COMPLETED, FAILURE, with its two times>]}}}`. `route 'run view * --json jobs' jobs.json`
  answers as `tests/builds.bats:48-52` does, a step failed. The desk, and a worktree made from `origin/9-other`, each
  have a commit not pushed and a file not committed; a branch `old` has lost its upstream. Last, with no fetch after,
  a second clone pushes a commit as `main main:feature main:7-fix-the-thing HEAD:refs/tags/v2` and deletes `9-other`.
- [ ] **Step 3: Write the stages' failing tests**: the one below in `tests/flow.bats`, and one of its name in
  `tests/workspace.bats`, where, after `history-at "${BATS_TEST_TMPDIR}/code/kit" github.com/empowerite/kit` and `cd`
  there, each of `repos remotes branches worktrees commits tags` goes from `kit workspace <stage> --json` to
  `same-keys …/workspace-<stage>.json` and `fields-hold workspace/<stage>`.

```bash
@test "each stage prints the keys of the fixture kittui's tests decode, and the fields its catalog declares" {
  contract-world
  local stage fixtures=projects/tui/internal/flow/testdata
  "${kit_root}/bin/kit" flow incoming --json --no-fetch | same-keys "${fixtures}/incoming-no-fetch.json"
  for stage in backlog underway desk worktrees changes builds releases incoming
  do
    "${kit_root}/bin/kit" flow "${stage}" --json > "${BATS_TEST_TMPDIR}/${stage}.json"
    same-keys "${fixtures}/${stage}.json" < "${BATS_TEST_TMPDIR}/${stage}.json"
    [[ "${stage}" == incoming ]] || fields-hold "${stage}" < "${BATS_TEST_TMPDIR}/${stage}.json"
  done
}
```

- [ ] **Step 4: Run `bats flow` and `bats workspace`.** Expected: `same-keys` fails for backlog, underway, changes,
  and incoming without a fetch, on the paths Step 5 names; `fields-hold` then fails for all thirteen stages.
- [ ] **Step 5: Bring the two sides together.**
  - `underway.jq:20`: `pr` gains `ref: $pr.ref` after `number`. `rows.go:56`: `Source string`, tagged `source`,
    becomes `Ref string`, tagged `ref`; `pullKey(member, ref string, number int)` gives `m.keyOf(member, ref)` when
    no changes row matches and `ref` is given, else the key of `#<number>`; `rows.go:597` passes `u.PR.Ref`. In
    `underway.json`, each `pr`'s `"source": "github"`, which kit never printed, becomes `"ref": "#<its number>"`.
  - Kit prints, and a fixture gains: in `backlog.json`, each row's `"ref": "#<its number>"` and `"createdAt"`; in
    `changes.json`, each row's `"fork": null` and `"pushed": {"age": 3600, "by": "<its author>"}`, and each closed
    issue's `"labels"`, one not empty, `"source"`, `"organization"`, and `"repository"`, as `backlog.json` has
    them. Each remote in `incoming-no-fetch.json` loses `"default"`, which kit prints only after a fetch.
  - `catalog.go`: `Schema` gains `Carries` after `Record`. `fields.json`: each stage gains `"carries"` after
    `"record"`. `workspace/repos`: checkouts, error, host, path. `backlog`: createdAt, ref. `underway`: worktree.
    `desk` and `worktrees`: changes, commits, detached. `changes`: closes, createdAt, fork, pushed. `builds`:
    attempt, id, jobs, url. `releases`: at, name, tag. The other five `workspace/` stages: key, links.
- [ ] **Step 6: Run** `bats flow`, `bats workspace`, `bats underway`, `bats jq`, and the Go tests. An assertion that
  pins `pr`'s keys gains `ref`, and a golden that moves is rewritten with `go test ./internal/flow -update`.
  **Commit**, subject `#74 kit, kittui: each stage's JSON is held to kittui's fixtures and fields.json`.

## Task 4: The commands the docs name

**Files:** create `tests/claims.bats`, `tests/claims/spans.awk`, and `tests/claims/claims.jq`. Modify
`docs/testing.md` and the six lines Step 3 names.

**Interfaces:** produces, in `tests/claims.bats`, whose `setup` sets `root=$(cd "${kit_root}/.." && pwd)`:

- `pages` prints every tracked `*.md` and `*.mdx` but those under `docs/superpowers/`, `tests/`, and a `testdata/`.
- `spans <page>…` runs `spans.awk` from the root and prints `<page>\t<line>\t<where>\t<text>`. `where` is `inline`
  for a code span; `fence` for a line of a fenced block whose language is `sh`, `bash`, `shell`, `zsh`, or
  `console`, less a leading `$` and space; and `fault`, with the text `the fence is never closed`. A fence closes
  on a line of its own marker, at least as long, with nothing after it. A paragraph's lines are joined by a space
  before its spans are read, and a span's line is the one it opens on. After a line that holds `kit:claims skip`,
  nothing is printed up to the next blank line, or, where a fence opens next, to its close.
- `commands <page>…` judges each span and fence line that starts with `kit` and a space, and prints
  ``<page>:<line>: `<text>`: <why>`` for each it refuses and `<page>:<line>: <text>` for each fault; it returns 1
  when it printed. `setup_file` writes every verb's syntax once, `kit syntax > "${BATS_FILE_TMPDIR}/specs.json"`.
- `claims.jq`, a module of `def`s as the jq rule requires, imports `syntax` from `kit/modules/jq`. `words` gives a
  text's words: quotes hold a word together and are dropped, and the words stop before the first that is `|`, `||`,
  `&&`, `;`, `<`, `>`, `>>`, or `2>&1`, or starts with `#`. `judge($specs)` gives one span's `why`, or nothing, by
  these rules in order, over the words after `kit <verb>`:
  1. `<verb>` is a placeholder, `--help`, or `-h`: nothing. It is no verb: `kit <verb> is not a verb`.
  2. A `--name`, up to its `=`, is no flag or option of the verb, which has no wildcard: `kit <verb> takes no --name`.
  3. A word is a form, not a value, so the line is not matched: it holds `<…>`, `${`, `…`, or `...`, starts with
     `[` or `(`, ends with `]` or `)`, is `|`, or ends with `=`: nothing.
  4. `syntax::match` gives status 0: nothing. Status 2: its refusal, but for an inline span whose last word is an
     option that `needs a value`. Status 1: nothing inline; in a fence, `it stops before <missing>`.

- [ ] **Step 1: Write the failing tests** in `tests/claims.bats`:

````bash
@test "a command is judged by its verb, its flags, and its syntax; a form, a beginning, and what is skipped pass" {
  cd "${BATS_TEST_TMPDIR}"
  cat > page.md <<'EOF'
`kit commit`, `kit packages add`, `kit context <path> --json`, and `kit close --duplicate-of` pass. A span the wrap
splits, `kit flow desk
--text`, is one, and `kit nosuch x`, `kit commit --bogus`, and `kit version now` are each refused.

<!-- kit:claims skip -->
`kit nosuch` is an example that is meant to fail.

```sh
kit commit --message "a b" x.txt | cat
kit facts > fixture.json   # a comment
$ kit commit
```

```sh
kit version
EOF
  run commands page.md
  [ "${lines[0]}" = 'page.md:2: `kit flow desk --text`: --text cannot be combined with desk; run: kit help flow' ]
  [ "${lines[1]}" = 'page.md:3: `kit nosuch x`: kit nosuch is not a verb' ]
  [ "${lines[2]}" = 'page.md:3: `kit commit --bogus`: kit commit takes no --bogus' ]
  [ "${lines[3]}" = 'page.md:3: `kit version now`: kit version takes no arguments; run: kit help version' ]
  [ "${lines[4]}" = 'page.md:11: `kit commit`: it stops before --message or --file' ]
  [ "${lines[5]}" = 'page.md:14: the fence is never closed' ] && [ "${#lines[@]}" -eq 6 ] && [ "$status" -eq 1 ]
}

@test "every command the pages name is one its verb takes" {
  # shellcheck disable=SC2046  # a page's path has no space
  commands $(pages)
}
````

- [ ] **Step 2: Run `bats claims`**: `commands: command not found`, twice. Write `spans.awk`, `claims.jq`, and the
  three functions; `commands` runs jq once over every span.
- [ ] **Step 3: Run it.** The first passes. The second prints six of the 845 commands a prototype in scratch found
  in 81 pages, 36 of them in fences. Write `kit x y` at `projects/tui/README.md:29` as `kit <verb> <argument>`, and
  `kit tools run` at `docs/logs.md:26`, `docs/testing.md:81`, `docs/toolchains.md:142` and `:150`, and
  `docs/toolchains/using-brew.md:54` as `kit tools run <command>…`, as its header does: the matcher refuses the bare
  words with status 2, where 1, missing `<command>`, is right. Both pass. The table's row shows decision 5's comment.
- [ ] **Step 4: Commit**, subject `#74 tests: every kit command a page names is one its verb's syntax takes`.

## Task 5: Settings, paths, lists, and counts

**Files:** modify `tests/claims.bats`, `docs/modules.md`, `docs/features.md`, `docs/architecture.md:205-224`,
`docs/testing.md`, and the lines Step 3 names. It uses Task 4's `pages`, `spans`, and `root`.

- [ ] **Step 1: Write the failing tests** in `tests/claims.bats`. Each prints what differs, then fails.
  - "every setting the schema has is one settings:validate knows, and the pages name no other". The keys are the
    `properties` of `kit/packages/kit/files/repo/.kit/kit.schema.json`. In a `fresh-repo`, with `<key>:` alone in
    `kit.yml`, `kit_bash_modules=settings kit-bash settings:validate` does not say `a key the kit does not know`;
    with `nosuch:` it does. The first cells of `docs/packages.md`'s rows from `version`'s to the next blank line are
    the keys. A code span of a word and a colon that "in", "of", or "under" and the span `kit.yml` follow is a key.
  - "a path a page names from the root, or from kit/, is there". An inline span that matches
    `^(bin|docs|kit|projects|tests|modules|packages)/[A-Za-z0-9_./-]*$` exists under the root, or under `kit/`.
  - "the lists a page gives of what the code holds are whole, and in its order". The first cells of four tables
    and lists are what the code holds: `docs/modules.md`'s rows of a `<name>.jq`, `kit/modules/jq/*.jq`; the
    bullets under "eight stages" in `docs/features.md`, `kit flow --stages`; its app table, each `{shell.<App>,` of
    `projects/tui/internal/flow/apps.go`; and `docs/serve.md`'s two tool tables, the names in `tools.json`.
  - "a count a page states is the count". On every page, a number, in digits or as a word up to twenty, before
    `stages` is the lines of `kit flow --stages`, and before `checks` the words after `for verb in` in `kit-ci`.
- [ ] **Step 2: Run `bats claims`.** Expected: settings passes, on 20 spans and 18 rows; paths names 7 of 120; lists
  names four modules and the bullets' order; counts names `docs/features.md:217`, "two checks".
- [ ] **Step 3: Fix each.** Write the whole path: `.git/kit/context.log`, once, at `docs/context.md:12`, and
  `${XDG_CONFIG_HOME:-~/.config}/kit/workspaces.yml` at `docs/features.md:225` and `docs/workspaces.md:8`. A file a
  reader would write is an example: put decision 5's comment on the line before the paragraphs that hold
  `docs/modules.md:120` and `docs/packages.md:284`, and write the example at `docs/templates.md:67`, an item of a
  list, as `scripts/deploy.sh`. Add rows to `docs/modules.md:229-240` for `glyphs.jq`, `serve.jq`, `version.jq`, and
  `workspace.jq`, each from its module's banner. Move the `incoming` bullet of `docs/features.md:92-106` between
  `builds` and `releases`, and write line 217 as "two of the checks".
- [ ] **Step 4: Write the docs.** `docs/architecture.md` § What verifies what gains three rows, each verified by
  "this repository's gate" and its tests: a verb's grammar and the MCP tools change only as a reviewed diff; the
  JSON kittui reads is what kit prints; a page names verbs, flags, settings, and paths that exist. `docs/features.md`
  § What no check covers gains a paragraph: no check reads what a page says a verb does, a value of a closed set,
  as `kit tutorials screens` at `projects/tui/README.md:30`, the words of a picture, or where a link leaves for.
- [ ] **Step 5: Run** `bats claims`, `kit lint docs`, and `kit generate --check`: all pass. **Commit**, subject
  `#74 docs, tests: the settings, the paths, the lists, and the counts the pages give are the code's`.

## Task 6: Exit statuses

**Files:** modify `tests/exits.bats`, `docs/testing.md`, and the `Exits` lines of `kit/bin/kit-check`, `kit-gate`,
`kit-hydrate`, `kit-update`, `kit-help`, and `kit-serve`.

Issue #70's three cases are done, by `d377c911`, but no test holds its rule, that a header names every status its
verb exits with. A status that reaches `exit $?` from a module stays with the tests that pin each case.

- [ ] **Step 1: Write the failing test** in `tests/exits.bats`, "an Exits line names 2, the guard's status for a
  usage error, and every status its verb's own file exits with". For each verb, the statuses are 2 and each number
  `grep --only-matching --extended-regexp '\bexit [0-9]+'` finds in its file. The header's paragraph from `# Exits`
  to the next `#` alone, on one line, matches `(Exits|,|;|and|or) <status>[ ,.]` for each; for the first that it
  does not, the test prints `kit <verb>: its Exits line does not name <status>` and fails. Run `bats exits`: it
  names 2 for `check`, then, as each is fixed, for `gate`, `hydrate`, and `update`, and 1 for `help` and `serve`.
- [ ] **Step 2: Say each in its header.** `check`, `gate`, `hydrate`, and `update` end their `Exits` sentence with
  ", and 2 on a usage error". `help` gains ", 1 when kittui cannot be built," before its 2, and `serve` ", 1
  without jq," before its 2. Each header keeps one line that starts with `Exits`.
- [ ] **Step 3: Run** `bats exits`, `bats help`, and `kit/bin/kit syntax --check`, then the gate, `kit gate`:
  all pass, and `✅ ci passed`. **Commit**, subject `#74 kit: every header names each status its verb exits with`.
