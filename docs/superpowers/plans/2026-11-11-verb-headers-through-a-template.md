# Verb headers read through a template Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A verb's header has one schema, a template, so `kit lint` names every verb a change to it leaves behind,
and kit help, kit syntax, kit serve, and kittui read the header through that template instead of sed and awk.

**Architecture:** The kit package's bash rule gains a `verb` template for `**/bin/kit-*`, whose header is one labeled
section: a `Usage:` block, the description, an `Arguments:` block, `Exits`, and `see:`. `help.sh` reads every value
of a verb's header in one jq pass through `template::match` (#227), and `syntax:arguments` reads the `Arguments:`
block alone. All 52 headers are rewritten to the template by a converter, in the same commit as the readers.

**Tech Stack:** bash 5, jq 1.7 (`kit/modules/jq/template.jq`), bats, the kit's own `kit lint` and `kit syntax`.

**Spec:** issue #231, with the decisions taken in its session: the template lives in the kit package's bash rule,
house overrides it with its lines of dashes, and every header is one labeled section (the "Labeled sections" layout).

## Global Constraints

- The kit package's template has no line of dashes; house's copy at the same path adds one under the banner and one
  at the end, as house's `script` does. A file under `kit/` names no consumer, person, or repository.
- The readers use the release's own template, `${kit_root}/packages/kit/files/repo/.kit/templates/bash/verb`, never a
  repository's `.kit/`, and drop each line that is `# -+` before matching, as `help:header` does today.
- Labels are the exact lines `# Usage:` and `# Arguments:`; usage lines and argument lines are indented with `#` and
  three spaces.
- Every verb has a usage line; one that takes no arguments has `kit <verb>`. Every verb has a `see:` line.
- A header that does not match the template is refused, never read loosely: the reader names the verb and the
  template line, `kit-<verb>: its header leaves the verb template at line <n>`, and returns 1.
- Shared worktree: stage only your own files with `git hash-object -w` and `git update-index --cacheinfo`, guard every
  commit with the recorded HEAD and the exact staged list, never unstage another session's files, and commit with
  `kit commit --message` and no paths. Subjects start with `#231`.

## Review Focus

- A verb a peer changes after the conversion ran: its header is in the old shape. Expect `kit lint` to refuse it at
  commit, naming the template line; the converter is rerun on HEAD right before the commit (Task 2, Step 9).
- A description paragraph whose first line starts with `#` and three spaces: the `Usage:` block would take it.
  Expect the converter to refuse such a header rather than write it (Task 2, Step 4 test).
- `kit-version` and the four other verbs with no usage line: expect `kit <verb>` under `Usage:` and an unchanged
  `kit usage <verb>` (Task 2, Step 7 diff).
- kit-audit's argument descriptions: today the row-shape list overwrites them. Expect `kit syntax audit` to give
  `evidence` the attestation description, `the attestations of the 5 newest releases' assets and the 20 newest
  artifacts` (Task 2, Step 6 test).
- A body line that equals `# Usage:` or `# Arguments:`, as in a heredoc: the reserved rule refuses the file. Expect
  `kit lint --all-files` to pass after the conversion, and any such line to be reworded (Task 2, Step 8).

---

### Task 1: A block also ends at a reserved line

A description is a block with prefix `#` and a space, which every header line starts with; it must stop at `# Arguments:`.

**Files:**

- Modify: `kit/modules/jq/template.jq` (`block`, `match`, the header comment)
- Modify: `docs/templates.md` (§ Blocks, § Where a hole ends)
- Test: `tests/jq/template.test`

**Interfaces:**

- Produces: `block($prefix; $reserved)`: the block's lines end before the first line, without its newline, that is
  in `$reserved`, as well as where its indentation ends. `match` passes its `reserved` list.

- [ ] **Step 1: Write the failing test** in `tests/jq/template.test`:

```text
# A block ends at one of the template's own lines too, so a block whose prefix every line has still ends at a label.
import "template" as template; template::parts | template::match("# a\n#\n#   b\n#\n# Arguments:\n#   --x  y\n")
"# §description\n#\n# Arguments:\n#   §arguments\n"
{"ok":true,"values":{"description":"a\n\n  b\n","arguments":"--x  y\n"}}
```

- [ ] **Step 2: Run it to see it fail:** `jq -L kit/modules/jq --run-tests tests/jq/template.test` reports
  `21 of 22 tests passed`, with this case's `*** Expected`.

- [ ] **Step 3: Implement** `block($prefix; $reserved)`: the run that continues the prefix also stops at a line in
  `$reserved`; the trailing lines that only start the prefix are given back as before.

- [ ] **Step 4: Run the tests:** `22 of 22 tests passed`. Add one sentence to docs/templates.md § Blocks: a block
  ends at one of the template's own lines too.

- [ ] **Step 5: Commit** `#231 templates: a block ends at one of the template's own lines too, so a section with no
  indentation of its own still ends at the next label`.

### Task 2: The verb template, the readers, and the 52 headers

One commit: the readers cannot read the old headers, and the old readers cannot read the new ones. Work in a scratch
copy of HEAD (with `kit/`, without `mise*.toml`) so the live tree keeps working; bring the commit into the live tree
with `git merge-file` per file afterwards.

**Files:**

- Create: `kit/packages/kit/files/repo/.kit/templates/bash/verb`
- Create: `.kit/packages/house/files/repo/.kit/templates/bash/verb`
- Modify: `kit/packages/kit/files/repo/.kit/rules/bash.md` (frontmatter `templates:`, and a paragraph on the verb)
- Modify: `kit/modules/bash/help.sh` (header comment; `help:values` new; `help:purpose`, `help:group`,
  `help:banners`, `help:usage`, `help:description` rewritten; `help:header` removed)
- Modify: `kit/modules/bash/syntax.sh:263` (`syntax:arguments`)
- Modify: `kit/bin/kit-*` (all 52 headers)
- Modify: `projects/tui/internal/syntax/testdata/*.json`, `projects/tui/internal/form/testdata/*.golden` (regenerated)
- Test: `tests/help.bats`, `tests/syntax.bats`, `tests/lint.bats`

**Interfaces:**

- Consumes: `block($prefix; $reserved)` (Task 1), `template::bind`, `template::parts`, `template::match`.
- Produces: `help:values <verb>…`: loads, once per process, `help_value[<verb>:<field>]` for each field `purpose`,
  `group`, `usage`, `description`, `arguments`, `exits`, `see`; returns 1, with the refusal of Global Constraints,
  when a header does not match. `help:usage`, `help:description`, `help:purpose`, `help:group`, `help:banners`
  keep their names and output formats.

- [ ] **Step 1: Snapshot today's output** on HEAD, in the scratch copy: for each verb, `kit help <verb>`,
  `kit usage <verb>`, and `kit syntax <verb>` into `before/<verb>.{help,usage,json}` in the scratch folder.

- [ ] **Step 2: Write the kit package's template** `kit/packages/kit/files/repo/.kit/templates/bash/verb`:

```text
#!/usr/bin/env bash
# «name» - «purpose» (kit verb: «group»)
# Usage:
#   §usage
#
# §description
#
# Arguments:
#   §arguments
#
# Exits «exits…=0 on success, and 2 on a usage error.»
#
# see: «see…=modules/bash/1st.sh»

set -o errexit -o nounset -o pipefail
«body»
```

  House's copy is the same with the line of dashes of house's `script` after the banner and after `«body»`. Add
  `- variant: verb` with `path: "**/bin/kit-*"` to the bash rule's `templates:`, and a paragraph after the seam
  providers' saying the verb template holds the header kit help reads.

- [ ] **Step 3: Write the failing tests** in `tests/help.bats`:
  - `a verb whose header leaves the verb template is refused by kit help, naming the template line`: in a
    `checkout-copy`, write `kit/bin/kit-zz` as the old shape (`# kit zz` under the banner, no labels), and assert
    `kit help zz` exits 1 with `kit-zz: its header leaves the verb template at line 3`.
  - `kit help <verb> shows the description, then the argument block, then Exits, as the header holds them`: assert
    `kit help toolchains` holds `Description` before an indented `--json`, and `--json` before `Exits 0`.

- [ ] **Step 4: Write the converter**, in the scratch folder only (not committed): `convert <file>` rewrites the
  header between the banner and the first line that is not a comment, from help.sh's paragraphs:
  - usage: every one-line paragraph matching `^kit <verb>( |$)` that ends without `.` or `:`, in order; none gives
    `kit <verb>`;
  - arguments: the first paragraph whose first line matches syntax.sh's argument-line pattern; later such
    paragraphs, as kit-audit's row shapes, stay in the description;
  - exits: the paragraph starting with the word `Exits`; see: the `see:` paragraph, `bin/kit` for kit-version;
  - description: every other paragraph, regions included, in order, joined by `#`;
  - writes the template's layout with the house lines of dashes, and refuses a header whose description would start
    with a line that starts `#` and three spaces.

- [ ] **Step 5: Implement the readers.** `help:values` runs one `jq` over the named verbs' files (`jq -n -R` with
  `input_filename`), binds `name` to `kit-<verb>`, drops `# -+` lines, matches, and prints
  `<verb>\0<field>\0<value>\0` per field; bash reads them into `help_value` with `read -d ''`. `help:banners` loads
  every verb; `help:usage` prints the usage value's non-empty lines; `help:description` prints the description value
  without its `kit:generate`/`kit:end` lines, a blank line, the arguments value with each line indented two spaces,
  a blank line, and `Exits` with the exits value, each newline, `#`, and space read as a newline.
  `syntax:arguments` feeds the arguments value, indented two spaces, to its existing awk.

- [ ] **Step 6: Convert and test.** Run the converter over `kit/bin/kit-*`. Then `kit syntax --check` over every
  verb passes; `kit syntax audit | jq -r .arguments.evidence.description` starts `the attestations of the 5 newest`;
  the Step 3 tests pass.

- [ ] **Step 7: Diff against the snapshot.** Every `usage` file is unchanged. Each `help` or `json` that changed is
  one of the 9 multi-section verbs (audit, changes, context, delivery, flow, issues, people, platform, workspace) or
  a verb whose argument block stood before prose; list them in the commit body. Regenerate the kittui fixtures with
  `kit syntax <verb> > projects/tui/internal/syntax/testdata/<verb>.json` and the form goldens with
  `go test ./internal/form/ -update` in `projects/tui`.

- [ ] **Step 8: Run the gates:** `kit lint --all-files` (the template shapes and shellcheck), `bats --jobs 8 tests/`,
  and `go test ./...` in `projects/tui`. All pass, but for a failure the snapshot on HEAD shows too, named as such.

- [ ] **Step 9: Commit.** Message every peer with a dirty `kit/bin/kit-*` (from `git status`, and `ListAgents`) that
  #231 rewrites every header, and wait for each to commit. Rerun the converter on the new HEAD's verbs, rerun Step 8,
  stage the tested files, and commit `#231 kit help, kit syntax: a verb's header is the verb template's, a Usage:
  block, the description, an Arguments: block, Exits, and see:, read in one jq pass; kit lint names a verb that
  leaves it`. Bring each file into the live tree with `git merge-file <live> <HEAD~1 version> <HEAD version>`.

### Task 3: kit new verb makes the header from the template

**Files:**

- Modify: `kit/modules/bash/new.sh:662` (`new:verb`)
- Test: `tests/new.bats:88`

**Interfaces:**

- Consumes: the `verb` template (Task 2); `kit new file <path> --rule=bash --variant=verb --<hole>=<value>…`.

- [ ] **Step 1: Change the test** at `tests/new.bats:88` to assert the made verb's header holds `# Usage:`,
  `#   kit <name>`, `# Arguments:`, and `# Exits 0 on success, and 2 on a usage error.`, and that `kit lint` passes
  on it.

- [ ] **Step 2: Run it to see it fail:** `bats tests/new.bats --filter "in the kit's own source"`.

- [ ] **Step 3: Implement:** `new:verb` calls `kit new file` with `--variant=verb`, `--purpose=<purpose>`,
  `--group=<group>`, `--usage="kit <name>"`, `--description="kit <name> <purpose>."`, and `--body`; the template's
  defaults give Exits and see.

- [ ] **Step 4: Run** `bats tests/new.bats`: all pass.

- [ ] **Step 5: Commit** `#231 kit new verb: the header comes from the verb template, so a new verb has the shape
  kit lint checks`.

## Not in this plan

- Checking kittui's `syntax/testdata/*.json` at commit: a fixture goes stale when a verb is staged, not the fixture,
  so a lint rule over the fixtures would not run. tests/syntax.bats keeps checking it in the gate.
