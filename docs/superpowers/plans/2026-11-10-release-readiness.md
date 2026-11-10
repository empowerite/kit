# Release Readiness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Nothing stands between this branch and a `release/0.1.0` a stranger can install: its digest verifies, the
gate is green, a release is packed from its tag and run before it is published, and the documents are true.

**Architecture:** Two parts. Part A is nine test-first tasks inside the repository: none pushes, tags, publishes, or
writes to GitHub, and it ends with `kit gate` passing. Part B is the owner's checklist: commands and their results.

**Tech Stack:** bash, jq, git, bats, Go, GitHub Actions, Dependabot, Homebrew.

**Spec:** issue #1 § Done when; pull request #2 § Not checked; `README.md` §§ Install, Maintaining, Release;
`../specs/2026-11-06-tutorials-design.md` line 88 (the changelog); and the review at `2dad3552`.

**Review:** [the findings](../reviews/2026-11-10-code-review-findings.md) of the review of 2026-11-10, cited
here by area and number: Serve 8 is item 8 under its Serve heading.

## Decisions for the owner

1. jq's minimum: keep 1.7 and make `modules/jq/template.jq:102` right on it, or require 1.8. Default: keep 1.7.
   GitHub's runners and macOS ship 1.7.1, and the delivered `ci_toolchain` is `ambient`, so 1.8 fails a consumer's CI.
2. macOS in CI: this repository's own `.github/workflows/macos.yml`, or a `ci_runner` matrix in the delivered
   template. Default: its own. A matrix renames the required check `CI` and bills every consumer macOS minutes.
3. The changelog gate: a `changelog` key in `kit.yml` that `kit release` reads, as the spec says, or `bin/release
   verify` alone. Default: the key. `verify` runs after the tag is pushed, and a tag ruleset then keeps that tag.
4. kittui gains `--version`, or the smoke test checks only its refusal off a terminal. Default: the flag.
5. Homebrew: the tap bumps itself on a schedule (infrastructure issue 458, no token in kit), a `release.yml` step
   with a token for the tap, or drop `README.md:76-80`. Default: the first, Part B step 6; not done by the tag, drop.
6. The first consumer: `edge` (6 files, a fresh repository) or `infrastructure` (rules of its own). Default: `edge`.

## Global Constraints

- kit commit only, never kit submit or push; every subject starts with `#1`, as `#1 <area>: …`.
- `export PATH=$PWD/kit/bin:$PATH` before `kit commit`.
- Peers share the worktree: stage named paths only, guard that the index holds only them, never leave anything staged.
- Everything under `kit/` is written for a consumer, and a path in it is written from `kit/`: `modules/bash/tui.sh`.
- Lines are 120 cells at most.
- A verb's header comment is its help page and its usage line is its grammar (`kit syntax --check` holds them):
  a new flag is added to the header's usage line and described there, and `docs/` says the same.
- A Commit step is `git add <the task's Files>`, then `kit commit --message "<the subject>"` with the session's
  `Co-Authored-By` trailer as a second `--message`. "Run `tests/x.bats`" is `KIT_TOOLCHAIN=mise kit/bin/kit tools
  run bats tests/x.bats`. In a test below, a trailing comment states the setup the assertion follows.
- `bin/refresh`, which the pre-commit hook runs, writes a change under `kit/packages/` into the installed copies.
  The minimums are bash 4.4, git 2.31, jq 1.7, and yq 4. `projects/tui` stays at or above 98% statement coverage.

## Review Focus

1. A consumer on the runner's jq 1.7.1 whose `AGENTS.md` holds text outside ASCII: `kit check` passes (Task 5).
2. `pack` with a tag that is not here, or a working tree that differs from the tag: refused, or ignored (Task 3).
3. `retire` when the forge's release list cannot be read: nothing is deleted, and it exits 1 (Task 3).
4. A release unpacked under a path with a space, as `~/Library/Application Support`: every smoke step runs (Task 4).
5. A file the repository edited, in a release that retires it: `kit update` keeps it and names it (Task 7).

---

## Part A: in the repository

Left out: Command line 12 and 13 reach a checkout alone; a release has a `version` file, so it never builds kittui.

### Task 1: The packed tarball's digest verifies

- **Files:** `bin/release:114-115`; test `tests/release.bats:54-76`.
- **Interfaces:** line 114 calls `release:digest`, which exists nowhere, so `.sha256` holds no digest and `pack`
  exits 0 (Infrastructure 1). `file:digest [<path>]` (`kit/modules/bash/file.sh:150`) comes with `use release`. `pack`
  holds the digest to `^[0-9a-f]{64}$`, and otherwise removes both files and exits 1.
- [ ] **Step 1: Write the failing tests.** Line 59's `grep` becomes `[[ "$(cat "kit-${version}.tar.gz.sha256")" =~
  ^[0-9a-f]{64}\ \ kit-9\.9\.9\.tar\.gz$ ]]` and `shasum --algorithm 256 --check --status
  "kit-${version}.tar.gz.sha256"`. A new test, "pack refuses, and leaves neither file, when no digest comes back",
  puts a `sha256sum` and a `shasum` that print nothing in `${fakes}`: `pack release/9.9.9 "${work}"` exits 1 with
  `kit-9.9.9.tar.gz has no sha256 digest; nothing is packed`, and `ls "${work}"` prints nothing.
- [ ] **Step 2: Run `tests/release.bats`.** Expected: both fail; the first prints `release:digest: command not found`.
- [ ] **Step 3: Write the digest into a variable with `file:digest`,** and the `.sha256` line only when it holds.
- [ ] **Step 4: Run it again.** Expected: 5 pass.
- [ ] **Step 5: Commit:** `#1 release: pack writes the tarball's sha256, and refuses when none comes back`

### Task 2: The gate is green at HEAD

- **Files:** `tests/underway.bats:199`.
- **Interfaces:** the whole suite at HEAD (736 tests, macOS) fails line 199 alone (Infrastructure 2). The five failures
  of run 42778922819, on the pushed head `6edaa6b6`, each run by name at `2dad3552` on macOS, pass at HEAD:
  - 113, `tests/exits.bats:14`, `tutorials: 0 Exits line(s) in its header`: since `4f6c3f83`.
  - 261, `tests/incoming.bats:111`, `--json` printed nothing after a failed fetch, on macOS too: since `7d32dbc0`.
  - 340, `tests/materialize.bats:73`, `bin/tutorials: No such file or directory`: since `38079cad`.
  - 466, `tests/template.bats:291`, forge's header lists `quota` and its template does not: since `72440e78`.
  - 570, `tests/underway.bats:81`, `labels` is among the keys: since `87bc1a4d`.
- [ ] **Step 1: Run `tests/underway.bats`.** Expected: `not ok 12 kit help flow holds the underway legend…`.
- [ ] **Step 2: Replace line 199** with the line `kit help flow` prints for `kit/bin/kit-flow:157`, which wraps:
  `[[ "$output" == *$'\n    🌱  a local branch: this desk has it: under underway, and no checkout does;\n'* ]]`
- [ ] **Step 3: Run the suite** as `bin/ci:62` does, after `kit/bin/kit hydrate`, which a fresh clone needs first:
  `kit/bin/kit tools run bats --jobs "$(getconf _NPROCESSORS_ONLN)" --timing tests/`. Expected: no `not ok`.
- [ ] **Step 4: Commit:** `#1 tests: kit help flow's underway legend reads as kit flow's header has it`

### Task 3: A release is its tag, and a pre-release outlives its branch

- **Files:** `bin/release` (header, `build-kittui`, `pack`, `publish`, `retire`), `tests/helper.bash`,
  `tests/release.bats`, `tests/tui.bats:106` (Infrastructure 7, 8, 20).
- **Interfaces:**
  - `tag-checkout <tag>` in `tests/helper.bash` commits all of a `checkout-copy`, edits included, and tags it. Every
    test that packs runs the copy's `bin/release`, not `${repo_bin}/release`.
  - `pack` reads `git archive --format=tar "refs/tags/<tag>" kit projects/tui` into a scratch directory, copies
    `kit/` from it, and builds kittui from it: `build-kittui <version> <directory> <tui>`. `publish` checks that
    `git rev-parse HEAD` is `<tag>^{commit}`, and passes `--verify-tag` to `gh release create`.
  - `retire` reads `gh release list --limit 1000 --json tagName,isPrerelease` once, captures `git branch --remotes
    --contains` before `grep` reads it, and deletes `release/X.Y.Z-s` only when no branch reaches it and
    `version::higher` (`modules/jq/version.jq`) is false for `X.Y.Z` over the releases: one at or above it exists.
- [ ] **Step 1: Write the failing tests** in `tests/release.bats`, and move its two packing tests onto `tag-checkout`.
  The fake `gh` of line 79 also prints `${FAKES}/releases.json` for `release list`, and exits 1 without that file.
  "publish refuses when HEAD is not the tag, and has gh verify the tag": one commit past the tag, `publish` exits 1
  with `HEAD is not release/9.9.9-rc.1; check it out, and publish from there`; on it, `gh.calls` holds `--verify-tag`.

```bash
@test "pack packs the tag's tree, not the working tree, and refuses a tag that is not here" {
  run bin/release pack release/9.9.9 "${work}"   # in a tag-checkout, kit/bin/STRAY.orig written after the tag
  [ -z "$(tar -tzf "${work}/kit-9.9.9.tar.gz" | grep STRAY)" ]
  run bin/release pack release/9.9.8 "${work}"
  [[ "$status" -eq 1 && "$output" == *"release/9.9.8 is not a tag here; tag the commit first"* ]]
}
@test "retire deletes a pre-release no branch reaches once a release at or above it exists, and no other" {
  run "${repo_bin}/release" retire topic   # origin's main holds 1.0.0 and 1.0.0-rc.2; no branch 1.0.0-rc.1, 1.1.0-rc.1
  [ "$(grep '^release delete' "${fakes}/gh.calls")" = "release delete release/1.0.0-rc.1 --cleanup-tag --yes" ]
  [[ "$output" == *"keeping release/1.1.0-rc.1: no release at or above 1.1.0 yet"* ]]
  rm "${fakes}/releases.json" && run "${repo_bin}/release" retire topic
  [[ "$status" -eq 1 && "$output" == *"the releases could not be read; nothing is retired"* ]]
}
```

- [ ] **Step 2: Run `tests/release.bats`.** Expected: the three fail: `STRAY.orig` is packed, and `publish` exits 0.
- [ ] **Step 3: Write `tag-checkout`, then `pack`, `publish`, and `retire`;** the header's paragraphs say the same.
- [ ] **Step 4: Run `tests/release.bats` and `tests/tui.bats`.** Expected: all pass.
- [ ] **Step 5: Commit:** `#1 release: pack packs the tag, publish refuses another HEAD, retire waits for a release`

### Task 4: The packed release runs before it is published

- **Files:** `projects/tui/cmd/kittui/main.go`, `main_test.go` beside it, `bin/release`, `tests/fakes/fake-go`,
  `tests/release.bats`, `.github/workflows/release.yml`.
- **Interfaces:**
  - `kittui --version` prints `kittui <version>` and exits 0, before the terminal check. `stdout io.Writer` joins
    the seams at `main.go:31-40`. `fake-go`'s script answers `--version` with the `-X main.version` it was given.
  - `bin/release smoke <tarball>` unpacks it under a scratch directory whose name holds a space and, with every
    directory that holds a `kit` off `PATH` and the unpacked `bin/` first, runs `kit version` (the file's version),
    `kit help`, `kit install` and `kit check` in a fresh repository, and `libexec/kittui-<platform> --version`.
  - `publish <tag> [<directory>]` packs unless `<directory>` holds the two files, runs `smoke`, then creates the
    release. `release.yml` runs `verify`, `pack "${GITHUB_REF_NAME}" dist`, and `publish "${GITHUB_REF_NAME}" dist`,
    with `timeout-minutes: 20` and `concurrency: release-${{ github.ref }}` (Infrastructure 28).
- [ ] **Step 1: Write the failing tests.** In `main_test.go`, `TestVersionPrintsTheVersionWithoutATerminal`: with the
  seam `stdout` a `bytes.Buffer` and `version` set to `9.9.9`, `run([]string{"--version"}, io.Discard)` returns 0
  and the buffer holds `kittui 9.9.9\n`. In `tests/release.bats`, "smoke runs the packed kit from a path with a
  space, and names the step that fails": for a tarball packed from a `tag-checkout` into `"${work}/a b"`, `smoke`
  prints `✅ kit-9.9.9.tar.gz runs: version, help, install, check, kittui`; for one whose `kit/bin/kit-check` is a
  script that exits 1, it exits 1 with `kit-9.9.10.tar.gz does not run: check`. And "with the real go, the four
  platforms build", skipped without go: `${fakes}/go` removed, `pack` and `smoke` exit 0, the gate's cross-compile.
- [ ] **Step 2: Run `(cd projects/tui && go test ./cmd/kittui/)` and `tests/release.bats`.** Expected: `undefined:
  stdout`; the bats tests exit 2 on the usage line.
- [ ] **Step 3: Write the flag, `smoke`, `publish`'s directory, the fake's answer, and the workflow's steps.**
- [ ] **Step 4: Run both, and `kit lint .github/workflows/release.yml`.** Expected: all pass.
- [ ] **Step 5: Commit:** `#1 release: a packed kit is run before it is published, and kittui --version names it`

### Task 5: The minimums are checked, and a remedy names no tool manager

- **Files:** `kit/bin/kit:18-21`, `kit/modules/jq/template.jq:102`, `kit/modules/bash/welcome.sh:539`,
  `kit/modules/bash/1st.sh:345`, `syntax.sh:329`, `yaml.sh:78,81`, and `tui.sh:91` beside it, `kit/bin/kit-check:67`,
  `README.md:94`, `kit/packages/kit/files/repo/.kit/welcome/2-before-toolchains.md:8`; the tests named below.
- **Interfaces:**
  - `kit/bin/kit` tests `BASH_VERSINFO` as `1st.sh:21-25` does, before it sources `log.sh` (Delivery 26).
  - `def offsets($literal)` in `template.jq`: where `$literal` starts in the input, in code points, overlapping
    starts included; line 102 calls it for `indices`, which counts bytes before jq 1.8. When `("éx" |
    indices("x")[0]) != 1`, it maps each byte offset to its code point by summing each code point's UTF-8 width:
    1, 2, 3, or 4 bytes for a value under 128, 2048, 65536, or above (Delivery 6).
  - `welcome:git` holds git to 2.31 as `welcome:jq` holds jq: `git 2.30.1 is too old; kit needs git 2.31 or later`
    (Command line 28). `tests/welcome.bats:135` fakes `git version 2.30.1` as it fakes jq 1.6.
  - `guard:missing <tool>` in `1st.sh` replaces the six `run: mise install` texts (Command line 10), which
    `arguments.bats:149`, `tui.bats:63,66`, and `yaml.bats:166,180` expect. For jq and yq: `<tool> is not on PATH;
    install it with this machine's tool manager, then run: kit welcome`. Any other ends `; run: kit tools install`.
- [ ] **Step 1: Write the failing tests:** the case below in `tests/jq/template.test`; in `tests/jq.bats`, "every jq
  module passes its tests on jq 1.7, where /usr/bin/jq is one", which skips unless `/usr/bin/jq --version` starts
  `jq-1.7` and runs `/usr/bin/jq -L "${kit_root}/modules/jq" --run-tests` over each `tests/jq/*.test`; and in
  `tests/kit.bats`, "under a bash before 4.4, kit says what it needs", which skips unless `/bin/bash` is before 4:
  `/bin/bash "${kit_root}/bin/kit" version` exits 1 and prints only `kit needs bash 4.4 or later; this is 3.2.…`.

```text
# Text outside ASCII before a literal: a hole's value is found by code points, on every jq the kit accepts.
import "template" as template; template::parts | template::match("# Guía ⚠️\nx\nend\n")
"# «title»\n«body»\nend\n"
{"ok":true,"values":{"title":"Guía ⚠️","body":"x\n"}}
```

- [ ] **Step 2: Run the files.** Expected: jq 1.7.1 gives `{"ok":false,"line":1}`; bash 3.2 prints `log.sh` errors.
- [ ] **Step 3: Write the four changes;** the README and the welcome page say `git 2.31 or later`.
- [ ] **Step 4: Run the files.** Expected: all pass; the new tests run on macOS, the jq one on GitHub's runners too.
- [ ] **Step 5: Commit:** `#1 kit: bash, git, and jq are held to their minimums, and a remedy names no tool manager`

### Task 6: CI covers what ships, and a test stands apart from its caller

- **Files:** create `.github/workflows/macos.yml`; modify
  `kit/packages/github/ci/templates/repo/.github/workflows/kit-ci.yml`, `docs/packages/github/ci.md`,
  `.github/dependabot.yml`, `.github/workflows/retire-pre-releases.yml`, `bin/ci:27-57`, `tests/helper.bash:68-76`,
  and the test lines Step 3 names; tests `tests/ci.bats`, `tests/world.bats`.
- **Interfaces:**
  - `macos.yml`, this repository's own: on `pull_request`, one job, `macos`, on `macos-latest`, 60 minutes at most,
    one run per pull request: `kit-ci.yml`'s pinned checkout and mise-action, `brew install bash`, then `bin/ci`.
  - The delivered `kit-ci.yml`: after checkout, `git remote set-head origin "${DEFAULT_BRANCH}"`, from `${{
    github.event.repository.default_branch }}` (Infrastructure 5). Job `CI` gains `concurrency`, group `kit-ci-${{
    github.event.pull_request.number }}`, cancelling the run in progress. `closed` moves to `pull_request_target`,
    on which `Settle` alone runs, from the base's checkout; each `if:` tests `github.event_name` (Infrastructure 10).
  - `dependabot.yml`: github-actions gains `exclude-paths: [".github/workflows/kit-ci.yml"]`, the template being
    its source (Infrastructure 6); `gomod` at `/projects/tui`, prefix `tui`, and `npm` at `/projects/tutorials`,
    prefix `tutorials`, both weekly and grouped. `retire-pre-releases.yml` gains `timeout-minutes: 10`.
  - `validate <schema> <glob>…` in `bin/ci` exits 1 with `nothing matches <globs>, so <schema> validated nothing`;
    the four ajv calls go through it, the house package's `package.yml`, rules, and styles too (Infrastructure 29).
  - `helper:isolate-in` also unsets `$(git rev-parse --local-env-vars)`, `GIT_AUTHOR_*` and `GIT_COMMITTER_*`,
    `GH_TOKEN`, `GITHUB_TOKEN`, `KIT_BASE`, `IN_NIX_SHELL`, `CODESPACES`, and `DOCKER_HOST` (Infrastructure 12).
- [ ] **Step 1: Write the failing tests:** the one below in `tests/ci.bats`; and in `tests/world.bats`, "a test's
  environment holds none of its caller's git, forge, or toolchain variables", which exports `GIT_DIR`, `GH_TOKEN`,
  `KIT_BASE`, `IN_NIX_SHELL`, and `DOCKER_HOST`, calls `helper:isolate-in`, and holds `[ -z "${!name+x}" ]` for each.

```bash
@test "the CI job sets origin's HEAD and cancels its earlier run, and Settle runs from the base on a close" {
  grep --quiet 'git remote set-head origin "${DEFAULT_BRANCH}"' "${workflow}"   # github/ci installed, as at line 46
  grep --quiet 'group: "kit-ci-${{ github.event.pull_request.number }}"' "${workflow}"
  [ "$(yq '.on.pull_request_target.types | join(",")' "${workflow}")" = closed ]
  [ "$(yq '[.updates[].package-ecosystem] | join(",")' "${kit_checkout}/../.github/dependabot.yml")" \
    = github-actions,gomod,npm ]
}
```

- [ ] **Step 2: Run both files.** Expected: the first fails at its first `grep`; the second on `GIT_DIR`.
- [ ] **Step 3: Write the workflows, `dependabot.yml`, the page, `validate`, and the unsets.** Then
  `across.bats:352` calls `fake-tools gh` first, so none leaves the machine (Infrastructure 13). Each `[ a ] && [ b ]`
  becomes a line per assertion (`toolchains.bats:103`, `dependencies.bats:64`, `releases.bats:120`, `serve.bats:160`,
  `uninstall.bats:26,47,66`), and each `run pty` followed only by a negative match gains `[ "$status" -eq 0 ]`
  (`across.bats:361-373`, `flow.bats:516-520`, `tui.bats:83`, `workspace.bats:196`; Infrastructure 14).
  `contract.bats:17`'s `read` ends in `|| true`, so a mise with nothing installed skips at line 19, not fails.
- [ ] **Step 4: Run the files touched, `kit lint` over the four workflows, then** `bash -c 'source <(sed -n
  "/^validate()/,/^}/p" bin/ci); validate x.json "nosuch/*.yml"'`. Expected: all pass; the last exits 1.
- [ ] **Step 5: Commit:** `#1 ci: macOS, origin's HEAD, Settle from the base, Dependabot over Go and npm, apart tests`

### Task 7: `kit update` holds across two packed releases

- **Files:** `kit/bin/kit-update:28,50`, `kit/modules/bash/place.sh:227-247`, `docs/packages.md`; test
  `tests/release.bats`, with Task 3's `tag-checkout` and `pack`, and `fake-curl` as `tests/fetch.bats:12-21` has it.
- **Interfaces:**
  - `kit update` drops a recorded package the release does not have, where `resolve:require` exits 2 today:
    `<package> is not in kit <current>; it leaves kit.yml, and what it delivered is retired` (Delivery 14).
  - `place:retire <base-root> <base> <previous> <current> <package>… [-- <dropped>…]`: the candidates are
    `manifest:paths` joined with what the base release delivers for the packages and the dropped ones, less what
    this release delivers. A path the manifest does not record is judged by the base's digest of it, as
    `recorded-digest` in `kit-update` judges (Delivery 9).
- [ ] **Step 1: Write the failing test.** `old` and `new` are the two tarballs' `bin/kit`, unpacked under `${work}`;
  `FAKE_RELEASES` is `${work}/releases`, exported, which `${fakes}/curl`, a link to `fake-curl`, serves.

```bash
@test "kit update between two packed releases removes a dropped file and a dropped package, in any clone" {
  checkout-copy && printf 'old\n' > kit/packages/kit/files/repo/OLD.txt && tag-checkout release/9.9.8
  git rm --quiet -r kit/packages/agents/cursor kit/packages/kit/files/repo/OLD.txt && tag-checkout release/9.9.9
  bin/release pack release/9.9.8 "${FAKE_RELEASES}" && bin/release pack release/9.9.9 "${work}"
  fresh-repo consumer >/dev/null && "${old}" install agents/cursor && git add --all && git commit --quiet --message in
  git clone --quiet . ../fresh && git clone --quiet . ../edited
  cd ../fresh && run "${new}" update
  [[ "$status" -eq 0 && "$output" == *"agents/cursor is not in kit 9.9.9; it leaves kit.yml"* ]]
  [[ ! -e OLD.txt && -z "$(ls .cursor/rules 2>/dev/null)" ]]
  grep -q 'kit-9.9.8.tar.gz.sha256$' "${FAKE_RELEASES}/curl.calls" && "${new}" check
  cd ../consumer && "${new}" kinds >/dev/null && run "${new}" update   # the manifest already describes 9.9.9
  [ ! -e OLD.txt ]
  cd ../edited && printf 'mine\n' >> OLD.txt && run "${new}" update
  [[ "$output" == *"OLD.txt: retired in 9.9.9, but edited since 9.9.8; it is left as it was"* ]]
}
```

- [ ] **Step 2: Run `tests/release.bats`.** Expected: the first update exits 2, `'agents/cursor' is not a package…`.
- [ ] **Step 3: Write the two changes;** `kit-update`'s header and `docs/packages.md` say a dropped package leaves.
- [ ] **Step 4: Run `tests/release.bats`, `tests/kit.bats`, and `tests/materialize.bats`.** Expected: all pass.
- [ ] **Step 5: Commit:** `#1 kit: kit update drops a package the release no longer has, and retires in a fresh clone`

### Task 8: A release needs its changelog entry

- **Files:** `kit/modules/bash/releases.sh:38-91`, `kit/bin/kit-release:9-13`, `kit/modules/bash/settings.sh:44`,
  `kit/packages/kit/files/repo/.kit/kit.schema.json`, `docs/packages.md:268`, `kit.yml`, `bin/release:131-137`;
  tests `tests/releases.bats`, `tests/release.bats`.
- **Interfaces:**
  - `changelog` in `kit.yml` names a directory of one entry per release. Here: `projects/tutorials/src/changelog`,
    which holds `0.1.0.mdx`, front matter `title:` and a body.
  - `releases:release` refuses a version with no suffix when the key is set and HEAD has no `<changelog>/<version>.*`.
  - `bin/release publish` writes the notes from the tag's entry, for a pre-release its version's without the
    suffix: the body after the front matter, each link whose target is not `https://` reduced to its text, then
    today's mise pin block. With no entry, the pin block alone.
- [ ] **Step 1: Write the failing tests.** In `tests/release.bats`' publish test, an entry `9.9.9.mdx` with the body
  `See [the page](kit-tutorials/).`, written before `tag-checkout`, and `grep -q '^See the page\.$'
  "${fakes}/gh.calls"`. In `tests/releases.bats`, in a desk as its test at line 26 makes one, "where kit.yml names a
  changelog, a release needs its entry there, and a pre-release does not": with `changelog: notes` pushed in
  `kit.yml`, `kit release 1.0.0` exits 1 with `1.0.0 has no changelog entry; write notes/1.0.0.md and commit it`,
  and `kit release 1.0.0-rc.1` exits 0; with `notes/1.0.0.mdx` committed and pushed, `kit release 1.0.0` exits 0.
- [ ] **Step 2: Run both files.** Expected: `kit release 1.0.0` exits 0 where 1 is expected; no `See the page.`.
- [ ] **Step 3: Write the key (schema, `settings:validate`, the key table), the refusal, and the notes.**
- [ ] **Step 4: Run both files and `kit syntax --check`.** Expected: all pass.
- [ ] **Step 5: Commit:** `#1 release: a release needs its changelog entry, and its notes are that entry`

### Task 9: The documents that describe the release are true

- **Files:** create `SECURITY.md`; modify `AGENTS.md:15`, `README.md:271,275,329-354`, `docs/testing.md:8-9`,
  `projects/tui/README.md:215`, `.github/workflows/release.yml:1-19`, `bin/ci:2,49`.
- **Interfaces:**
  - `AGENTS.md:15` and `README.md:271`: "`libexec/`, which holds `kittui-<os>-<arch>` for linux and darwin on amd64
    and arm64, built from `projects/tui/`". `README.md:275` names `bin/tutorials` (Documents 14, 21).
  - `README.md` § Release: the tag's tree is packed, and run, before it is published; a release needs its changelog
    entry; a pre-release goes once no branch reaches it and a release at or above it exists. `docs/testing.md:8-9`,
    `bin/ci:2`, and the tui README list what the gate runs, the site's checks and the 98% floor included.
  - `SECURITY.md`, from `kit new file SECURITY.md`, in three sections. Supported: the latest release. Reporting:
    GitHub's private vulnerability reporting here, never a public issue. What kit trusts: the repository it runs
    in, so run it only in a clone you would run `make` in. Issue #66 lists how a clone runs code today: `kit serve`
    runs the clone's `kit/bin/kit-serve` under `version: unreleased`; `bash_path` sources the clone's bash; a
    tracked `.kit/manifest` or `.kit/logs` under a symlink writes outside the clone; `kit tools install` runs `mise
    trust` unasked; a local rule's `lint:` and the `gate` run when a verb asks. A `.sha256` proves the download only.
- [ ] **Step 1: Add `SECURITY.md` to lychee's list** at `bin/ci:49`, and run that line. Expected: it fails, no file.
- [ ] **Step 2: Write the documents.**
- [ ] **Step 3: Run `kit lint SECURITY.md AGENTS.md README.md`, then `kit gate`.** Expected: `✅ ci passed`.
- [ ] **Step 4: Commit:** `#1 docs: the release as it is packed, run, and retired, and a security policy`

## Part B: the owner's checklist

- [ ] **1. Push.** `git push origin 1-the-first-release-v0-1-0`, then `gh pr checks 2 --watch`. Expected: `CI` and
  `macos` pass. No run has passed on Linux since `6edaa6b6`; a failure there is fixed under Part A's rules.
- [ ] **2. Rehearse.** In the worktree, `kit release 0.1.0-rc.8`, then `gh run watch`. Expected: the log holds
  `kit-0.1.0-rc.8.tar.gz runs: version, help, install, check, kittui`. In an empty directory, `gh release download
  release/0.1.0-rc.8 --repo empowerite/kit && shasum --algorithm 256 --check *.sha256`. Expected: `…: OK`.
- [ ] **3. Protect and attest** (Infrastructure 9). In `empowerite/infrastructure`, `terraform/projects/kit/main.tf`
  gains a `github_repository_ruleset` with `target = "tag"`, including `refs/tags/release/*` and excluding
  `refs/tags/release/*-*`, which `retire` deletes, with `creation`, `update`, and `deletion` kept to the admin role.
  Expected: `gh api repos/empowerite/kit/rulesets --jq '.[].target'` prints `branch` and `tag`. `release.yml` gains,
  between `pack` and `publish`, `actions/attest-build-provenance` pinned by SHA, with `subject-path: dist/*.tar.gz`,
  `id-token: write`, and `attestations: write`; after step 5, `gh attestation verify kit-0.1.0.tar.gz --repo
  empowerite/kit` exits 0. And `gh api --method PUT repos/empowerite/kit/private-vulnerability-reporting`, off today.
- [ ] **4. Rewrite pull request #2's body,** the report: `gh pr edit 2 --body-file <file>`. It says what kit is now
  (45 verbs, kittui's apps, workspaces, tutorials), what the gate ran, and the twelve issues it does not close, by
  number (Gaps § 1); § Not checked keeps `kit update` across two published releases. Then `gh pr ready 2`.
- [ ] **5. Tag,** once the owner has merged, at the desk on `main`: `kit release 0.1.0`. Expected: `✅ pushed
  release/0.1.0`; the `release` workflow passes; `gh release view release/0.1.0 --repo empowerite/kit --json
  assets,isPrerelease --jq '[.isPrerelease, .assets[].name]'` prints `false` and the two file names; step 2's
  `shasum` prints `kit-0.1.0.tar.gz: OK`. Deleting the merged branch then retires rc.6 to rc.8.
- [ ] **6. Fill the tap** (Decision 5; `empowerite/homebrew-tap` is empty), by pull request there. `Formula/kit.rb`:
  `url` is `releases/download/release/0.1.0/kit-0.1.0.tar.gz`; `sha256` is the first field of its `.sha256`;
  `depends_on` names bash, gh, jq, and yq; `install` is `libexec.install Dir["*"]` and `bin.install_symlink
  libexec/"bin/kit"`; `test` holds `kit version` to `version`. A scheduled workflow there, with the tap's own token,
  opens a pull request when `gh release view --repo empowerite/kit --json tagName` names a release newer than the
  formula's. Expected: `brew install empowerite/tap/kit && kit version` prints `0.1.0`.
- [ ] **7. Take the kit in `edge`** (Decision 6), as `README.md` §§ Install and Take the kit say: `gh issue create
  --title "Take kit 0.1.0" --body "Pin it, install it, pass CI."`; pin `"github:empowerite/kit" = { version =
  "0.1.0", version_prefix = "release/" }` under `[tools]` in `mise.toml`; `mise install && kit version`, expected
  `0.1.0`; `kit install github/ci workflow agents/claude`; add `gate: kit lint --all-files` to `kit.yml`; `kit work
  <N>`, which sweeps what install wrote into the worktree; there, `kit commit --message "#<N> kit: take kit 0.1.0"`
  and `kit submit`; `gh pr checks --watch`. Expected: `CI` passes on the runner's own jq and yq, kit downloaded and
  its digest checked: issue #1's third Done-when line. `kit update` there, once 0.1.1 exists, closes § Not checked.
