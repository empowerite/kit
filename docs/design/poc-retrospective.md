# The proof of concept: a retrospective (2026-11-17)

kit was first built as a proof of concept (PoC), kept at [empowerite/kit-poc](https://github.com/empowerite/kit-poc).
This page says what it proved and what it did not. Its verdicts are a reading of the issue trail, the repository, and
CI, not measurements, unless a number is given, and nothing here was run against a clean machine. The issue numbers
are kit-poc's.

## What the PoC was for, and what it was not

The PoC was asked to show that kit's idea works: write down the rules an agent must follow, put them where any
harness reads them, check them in CI, and take work from an issue to a merged pull request with the same few
commands on any forge, seen in one terminal UI. It was not asked to be a product, but it was built like one: 79k
lines of Go, 34k of shell, 37k of bats, and 33k of Markdown in 24 days (1,582 commits, 393 of them on 2026-11-12).

## Numbers

| Measure | Value |
| --- | --- |
| Issues opened, 2026-10-24 to 2026-11-17 | 621; 616 open, 5 closed |
| Opened on 2026-11-12 and 11-13 | 361 (the review fan-out) |
| Issues by area, by title (approximate) | kittui 234, kit verbs and forges 118, docs 46, welcome 28, CI/release/gate 16, workspaces 11, tests 7, other 161 |
| kittui issues about chrome (glyphs, widths, menus, labels, rules) vs. about work (select, act, read, refresh) | about 97 vs. about 73 of 272 mentioning kittui or Flow, overlapping |
| Design documents | 13 specs, 20 plans, 16 reviews, each about one feature or one review; none about the whole |
| CI run time for a pull request | about 25 minutes (the `kit-ci` step took 1,489 s); of the last 15 runs, 6 failed or cancelled |
| Pre-releases published | rc.3 to rc.7, 2026-10-24 to 10-31; none since |
| Pull request #2 | open; nothing it closes is closed |

## Hypotheses and verdicts

| # | Hypothesis | Verdict | Evidence |
| --- | --- | --- | --- |
| H1 | Written rules, placed where agents read them, checked in CI, keep agents honest | Untested | The mechanism exists. The gate does not count skipped tests (#556), some assertions cannot fail (#555), and a test passes whatever the forge tool answers (#559). No record shows the checks catching an agent's shortcut. |
| H2 | The same few commands take work from issue to merged PR on any forge | GitHub only | ADO and GitLab each carry defects (#439, #443, #467, #499). The loop has not been closed end to end on the real repository: #2 is unmerged. |
| H3 | One screen shows where the work is, what to do, and what comes next | **Disproven** | See "Selection then mutation". |
| H4 | Every fact the UI shows is JSON a script or an agent can read | Supported | kittui reads only `kit ... --json`; fixtures and `same-keys` tests hold both sides. This is the best-proven part. |
| H5 | Any repository installs a kit release and keeps current | Untested | Five pre-releases exist; update semantics are told two ways (#497, #543) and merge keeps stale keys (#489). |
| H6 | Verification stays fast enough to run on every change | **Disproven** | About 25 minutes per run, the whole suite each time. |
| H7 | Everything stands up from scratch with no GUI | Untested | Not attempted. Repository settings, secrets, and the tap were set by hand. |
| H8 | Trust is explicit: an untrusted clone or flow is refused | Partly built | `kit trust` and the loader exist; the security model is a set of rules found by issues (#488), not a design. |
| H9 | Agent fan-out builds a coherent system | **Disproven as practised** | See "Patterns". |

## Selection then mutation

This is the owner's first complaint, and the issue trail confirms it. The review of 2026-11-12 found:

- In every app's rows, ⏎ does nothing; with the cursor on an issue and nothing selected, every action is dim, and
  starting work on one issue takes 12 keys (#373).
- No plain click selects a row or opens its menu, so no action on rows has a mouse path (#394).
- Assign to Me, Start Local Work for Each, and Packages' Add run at once, with no question (#355).
- An action on another repository's issue ran on this repository's issue of the same number (#304).
- The Issues app offers Add to Pull Request, which can never be enabled there (#505).

The cause is in the design, not the code. What an action acts on is read from the selection alone (`selectedRows`), so
there is no rule for "the row under the cursor", no preview of what will change, and no statement of what happens next.
The screen was built from tables and menus outward; the act (target, effect, confirmation, result) was never a unit.

## Patterns behind the issues

1. **Slices without a system.** Work came in slices (#609 "slice 1 of ..."). Each slice had a spec; the whole did
   not. So decisions flip: circled tab digits (#415, then #566 "again"), labels ending in half circles (#568
   "again"). Nobody owned the answer.
2. **Polish before the core worked.** About 36% of the kittui issues are about glyph widths, fonts, menus, and rules,
   while the core loop (see, choose, act, read the result) was not yet usable.
3. **Review output outran fixing.** One fan-out produced 361 issues in two days and 436 medium findings (#561). An
   issue for every ask, no triage, and no close until the merge give a tracker that cannot say what matters.
4. **Docs and code drift.** Five issues say a page or diagram "says what the code does not do" (#532, #536, #544 and
   others); both sides are hand-written. #597 proposes data-driven docs.
5. **The suite is the unit of everything.** 94 bats files and 199 Go test files run as one gate on every push, on a
   macOS desk and on a Linux runner that differ (#616). Nothing says which tests a change can break.
6. **Security by accretion.** Trust, tokens, forge Apps, and CI permissions were each handled when a case arose. A
   person approves a change on the desk, yet repository settings and the tap still ask for GUI work.

## What to carry forward

- What kit is for. The PoC's purposes stand, and [the charter](../charter.md) states them: rules written once for
  any harness, the rules that govern a file, templates that make a file and check one, a toolchain, a release and
  not a copy, and the rest. This page judges how they were built, not whether they are wanted.
- H4's shape: JSON as the interface, a UI that adds no facts of its own.
- The contract snapshots (verbs, grammars, tools): a change to the surface is a reviewable diff.
- The forge seam as an idea, with GitHub as its only proven provider.
- The entity model work (#597, `docs/entities`), as a start, once it is judged against the screens.
- Findings that name a root cause are design input; findings about a single glyph or width are not.

## What not to carry forward

- A UI built before its interaction model: act-on-cursor-or-selection, preview, confirm, result.
- Hand-written docs about behaviour that code could emit.
- One monolithic gate; a 600-issue backlog as the plan; "no merge until done" as the only closing event.

## What the owner decided next

1. kit starts again, in a new repository, from a charter and a design. The PoC stays at kit-poc, with its issues,
   as a reference, and nothing is carried over unless a design calls for it.
2. The terminal UI is in from the first slice that has work to show, since it is how the owner feels how kit works.
3. Nothing kit needs is made by hand. kit makes it, with the person's approval.
4. The first proof is one workflow from end to end: see the work, start an issue, submit, watch the checks, and
   merge.
