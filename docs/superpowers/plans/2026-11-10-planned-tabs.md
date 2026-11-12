# Every Planned Tab Reads kit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Each of kittui's 20 planned tabs reads a kit stage: forge contract 4's new calls on GitHub, GitLab, and Azure
DevOps, six verbs that print them as `{rows, errors}` with `--stages` and `--marks`, and a catalog entry per tab.

**Architecture:** `kit/modules/bash/apps.sh` is the one runner the six verbs share: their stages, `--stages`,
`--marks`, `--json`/`--jq`/`--text`, the group over `across.sh`, and the handoff to kittui on a terminal. Each verb's
stages live in `kit/modules/bash/apps/<verb>.sh`; each forge read is a seam call in `forge.sh`, a function in each
provider, and a fold in `kit/modules/jq/forge.jq`. kittui draws each stage from one `fields.json` entry, as the #138
plan's Tasks 1 to 4 make possible.

**Tech Stack:** bash, jq, bats with `tests/fakes/fake-tool`; Go 1.27 (Bubble Tea v2).

**Spec:** [the design](../specs/2026-11-10-planned-tabs-design.md); the #138 plan,
[2026-11-10-apps-tabs-read-kit.md](2026-11-10-apps-tabs-read-kit.md), which the design revises; issues #162 and #138.

## Global Constraints

- kit commit only, never kit submit or push. kit's commits start `#162 kit: …` or `#162 forge: …`, kittui's `#138
  kittui: …`. `export PATH=$PWD/kit/bin:$PATH` before `kit commit`.
- Peers share the worktree (kit-68 holds `internal/flow`, `flow.sh`, `kit-flow`, `glyphs.json`, and
  `docs/integrations/git.md` until it says #161 landed). Stage named paths only; guard with `test "$(git diff
  --cached --name-only)" = "<your paths, sorted>" && kit commit --message …`, no paths to `kit commit`; never leave
  anything staged; `git show --stat HEAD` after.
- Everything under `kit/` is written for a consumer; a path in it is written from `kit/`: `modules/bash/apps.sh`.
- Every option with a value is `--opt=value`; lines are 120 cells at most.
- A verb's header comment is its help page and its usage line its grammar (`kit syntax --check`); each new verb gets
  `tests/syntax.bats`' fixture, `internal/syntax/testdata/<verb>.json`, refreshed by the command it prints.
- The forge contract's rules in `forge.sh`'s header hold for every new call: whole or failing, one ❌ line with the
  remedy then what could not be read, 3 and nothing printed for a feature lacked.
- kittui: plain clicks, a menu line's shortcut at its right, no circled glyphs, two-cell emoji in menus.
- Go tests run without `-update`; goldens are regenerated only on a tree with no peer work under `projects/tui/`.
  Coverage in `projects/tui/` stays at 98.0 or more.
- Bats run as `KIT_TOOLCHAIN=mise kit/bin/kit tools run bats tests/<file>.bats`; `kit gate` before each commit.

## Review Focus

1. A forge that has the feature but answers 404 with "disabled" for this repository, as secret scanning off: the stage
   is `{rows: [], errors: [{feature, off: true}]}`, exit 0, never a failure. Tasks 1 and 8.
2. A caller refused one alerts kind (403) while allowed another: findings draws the other, the refused one under
   `errors` with its line, exit 1. Task 8.
3. A cap reached: deployments with 60, teams with 25: the document holds `limited: 50` or `limited: 20`, the rows are
   the newest, and kittui's count reads `50+`. Tasks 6, 7, 11.
4. `kit people --json` in a repository whose primary has no forge: every stage `{rows: [], errors: []}`, exit 0, and
   `--marks` all null. Task 2's framework, tested in Task 7.
5. `kit --workspace=<group> audit findings --json` with one member refused: the others' rows, each with `member`, and
   the refused member under `errors`. Task 2.

---

## Task 1: Forge contract 4 declares the new features

**Files:** Modify `kit/modules/bash/forge.sh` (header, `forge_contract`, `forge_features`, `forge_feature_of`),
`kit/modules/bash/forge/{github,gitlab,ado}.sh` (`:contract`, `:lacks`), `tests/forge-capabilities.bats`,
`docs/architecture.md` (the seams' bullet "3 today").

**Interfaces:** Produces, for every later task:

| Feature (since 4) | Call | Prints, one JSON object a line |
| --- | --- | --- |
| `milestones` | `milestones` | `{number, title, state, due, open, closed, url}`; `state` open or closed, `due` epoch or null |
| `iterations` | `iterations` | `{id, title, start, end, url}`; `start`, `end` epoch; ordered by start |
| `reviewers` | (none: `open-prs` gains `reviewers`) | each change's `reviewers: [{login, state, at}]`, `state` requested, approved, changes, or commented |
| `merged` | `merged-prs` | the 50 newest merged changes: `{number, title, author, branch, merged, by, url}` |
| `artifacts` | `artifacts` | `{id, name, digest, size, run, age, expires, url}`; the newest 100 |
| `environments` | `environments` | `{name, protection, approvers, branches, url}`; `protection` a list of rule kinds |
| `deployments` | `deployments` | `{id, environment, ref, sha, state, actor, age, url}`; the newest 50 |
| `collaborators` | `collaborators` | `{login, name, role, kind}`; `kind` user or bot |
| `teams` | `teams` | `{team, name, role, members, url}`; `members` logins; 20 teams, 100 members each |
| `bots` | `bots` | `{login, acted, last, url}`; `acted` a list of commit, change, review |
| `attestations` | `attestations` | `{subject, digest, kind, signer, age, url}` |
| `code-scanning`, `dependabot`, `secret-scanning` | `code-scanning-alerts`, `dependabot-alerts`, `secret-scanning-alerts` | `{number, rule, severity, state, path, age, url}`, open and dismissed |
| `rulesets` | `rulesets` | `{id, control, target, enforcement, rules, source, url}`; `source` ruleset or protection |
| `bypasses` | `bypasses` | `{waives, actor, kind, reason, by, until, url}`; `kind` bypass or dismissal |

A capped call prints `{"limited": N}` as its last line when it stopped at N. A call of a feature the forge has may
return 3 at run time when the forge says this repository has it off; it prints nothing.

- [ ] **Step 1: Write the failing test** in `tests/forge-capabilities.bats`: `kit flow --capabilities` on each world
  has `contract == 4` and `.features["milestones"] == "none"` (each provider declares 4 and lacks every new feature
  until its task); a provider declaring `contract` 4 and lacking `nonesuch` is refused as today.
- [ ] **Step 2: Run** `tests/forge-capabilities.bats`. Expected: `contract` is 3.
- [ ] **Step 3: Implement.** `forge_contract=4`; the features above with `:4`; the calls in `forge_feature_of`, and
  `forge:<call>() { forge:call <call> "$@"; }` for each; the header's call list, feature table, and the run-time 3.
  Each provider's `:contract` prints 4 and `:lacks` adds every new feature.
- [ ] **Step 4: Run** `tests/forge-capabilities.bats tests/forge-conformance.bats tests/forge-ado.bats`, then `kit
  gate`. Expected: pass.
- [ ] **Step 5: Commit** the five paths and the test as `#162 forge: contract 4 declares milestones, iterations,
  reviewers, merged, artifacts, environments, deployments, people, teams, bots, evidence, alerts, rulesets, and
  bypasses`.

## Task 2: The app verbs' runner, and kit platform

**Files:** Create `kit/modules/bash/apps.sh`, `kit/modules/bash/apps/platform.sh`, `kit/bin/kit-platform`,
`kit/modules/jq/apps.jq`, `tests/platform.bats`, `docs/apps.md`, `projects/tui/internal/syntax/testdata/platform.json`.
Modify `docs/features.md` (§ The terminal UI's verb list).

**Interfaces:** Produces, in `apps.sh`:

| Name | Contract |
| --- | --- |
| `apps:main <verb> <word>…` | The verb's body after `guard:arguments`: `--stages` prints `<verb>_stages`; `--marks` prints `{<stage>: text\|null}`; `<stage> --json` prints the stage's document, `--jq` it formatted, `--text` a table; no stage with `--json` prints `{stages, <stage>: document…}`; on a terminal with no form, `exec`s kittui `<verb> [<stage>]` as `kit-workspace` does. Over `KIT_WORKSPACE`, each stage runs in every member through `across:stage` with `across_verb=<verb>`, each row gaining `member`. |
| `<verb>_stages` | The array a module `apps/<verb>.sh` declares, in tab order. |
| `<verb>:<stage>` | Prints the stage's document `{rows, errors}` (and `limited` when capped); returns 1 after its ❌ line when nothing could be read. |
| `<verb>:mark-<stage>` | Prints the stage's mark, or nothing for null. Absent, the mark is null. |
| `apps:forge <feature> <call> <jq-def> [<arg>…]` | Runs `forge:<call>`. A primary with no forge, or a forge lacking the feature: `{rows: [], errors: []}`. Return 3: `{rows: [], errors: [{feature: <feature>, off: true}]}`. Return 1: the line, exit 1. Else each object through `apps::<jq-def>` from `apps.jq`, which gives `{key, …, links}`, and `limited` from a `{"limited": N}` line. |
| `apps:merge <document>…` | One document of several calls' documents: rows joined in order, errors joined, `limited` the least. |
| `apps:hash <path>…` | The sha256 of the paths' contents that exist, or nothing when none does. |
| `apps:forge-mark <kind>…` | The forge's marks of the kinds (`issues`, `runs`, `releases`, `milestones`) joined by a space, or nothing when any is null. |

`kit platform`'s stages, from `apps/platform.sh`:

| Stage | Source | Key | Fields | Mark |
| --- | --- | --- | --- | --- |
| `services` | the forge (`forge:provider`, `forge_settings`, `forge:require`), each tracker prefix (`tracker.sh`), the toolchain, dependencies (`dependencies.sh`), policy (`policy:engine`) | `.capability` | `capability, service, provider, chosen` (inferred or set), `ready` (whether its tool is there and signed in) | `apps:hash kit.yml` |
| `toolchains` | `kit toolchains --json` | `.provider, .role` | its own: `provider, role, usable, active, chosen, fills` | `apps:hash` of the toolchain's pin files and kit's choice |
| `tools` | `kit tools --json` | `.from, .name` | its own: `name, pinned, installed, from` | as toolchains |
| `settings` | every key of `kit.yml` the settings schema names, its value, `set` or `default` | `.setting` | `setting, value, from` | `apps:hash kit.yml` |

`kit tools --json` refusing, as with no toolchain chosen, is tools' ❌ line and exit 1; toolchains still draws.

- [ ] **Step 1: Write the failing tests** in `tests/platform.bats`: `kit platform --stages` prints the four in order;
  `kit platform toolchains --json` in a repository with `mise.toml` has a row keyed `mise project`, `links` `{}`;
  `kit platform services --json` with a github origin and the fake gh signed in has `{capability: "hosting", service:
  "github", chosen: "inferred", ready: true}`; `kit platform --marks` is an object of four, `settings` changing after
  `kit.yml` changes; `kit --workspace=<group of two> platform settings --json` gives each row a `member`; the forge's
  services in a repository with no remote say `service: null`.
- [ ] **Step 2: Run** `tests/platform.bats`. Expected: `kit platform` is no verb.
- [ ] **Step 3: Implement** `apps.sh`, `apps.jq`, `apps/platform.sh`, and `kit-platform`, whose header describes the
  stages, `--stages`, `--marks`, the forms, the document, each stage's fields, and the exits: 0 drawn, 1 a stage that
  could not be read, 2 a usage error. `docs/apps.md` says the same for the six verbs, a section each, filled in as
  each lands.
- [ ] **Step 4: Run** `tests/platform.bats tests/syntax.bats tests/help.bats tests/completion.bats`, refresh the syntax
  fixture as `tests/syntax.bats` prints, then `kit gate`.
- [ ] **Step 5: Commit** as `#162 kit: kit platform lists the services, toolchains, tools, and settings kit runs with,
  each a stage with a mark`.

## Task 3: kit policy --rules

The #138 plan's Task 7, Steps 1 to 4, as written there: `kit policy --rules [--json | --jq]`, `policy:listed`, its
bats test, the header, `docs/policy.md`, and the syntax fixture. Commit as `#162 kit: kit policy --rules lists each
rule, its pattern, and where it is set`.

## Task 4: kit issues: milestones and iterations

**Files:** Create `kit/modules/bash/apps/issues.sh`, `kit/bin/kit-issues`, `tests/issues.bats`, its syntax fixture.
Modify `forge/{github,gitlab,ado}.sh`, `kit/modules/jq/forge.jq`, `tests/forge-conformance.bats`, `tests/forge-ado.bats`,
`docs/integrations/{gh,glab,az}.md` (the read tables' kit status), `docs/apps.md`, and each forge's `marks`.

**Interfaces:** Consumes Tasks 1 and 2. Each provider reads as the design's table says:

| Provider | `milestones` | `iterations` |
| --- | --- | --- |
| github | `gh api --paginate --slurp 'repos/{owner}/{repo}/milestones?state=all&per_page=100'` | one `gh api graphql` over `repository.projectsV2(first: 20) { nodes { url fields(first: 50) { nodes { ... on ProjectV2IterationField { configuration { iterations { id title startDate duration } completedIterations { id title startDate duration } } } } } } }`; none without a linked project |
| gitlab | `glab api 'projects/:id/milestones?per_page=100'`, paged | `glab api 'projects/:id/iterations?include_ancestors=true&per_page=100'`, paged; 403 or 404 from a tier without them is off |
| ado | lacks `milestones` | `az boards iteration team list --team "${team}" --project … --org … --output json`, the team being `forge_settings.ado.team`, else the project's default team |

Each forge's `marks` gains `milestones`: GitHub `repos/{owner}/{repo}/milestones?state=all&sort=updated&per_page=1`'s
ETag, GitLab the newest `updated_at`, Azure DevOps none.

| Stage | Call | Key | Links | Mark |
| --- | --- | --- | --- | --- |
| `milestones` | `milestones` | `.number` | `title`: the milestone's page | `apps:forge-mark milestones issues` |
| `iterations` | `iterations` | `.id` | `title`: the project or team page; `current` true for the one containing now | null |

- [ ] **Step 1: Write the failing tests.** In `tests/forge-conformance.bats`, "every provider lists its milestones in
  one shape": one world milestone, number 2 "v0.1", due 2026-12-01T00:00:00Z, 3 open and 2 closed issues, gives
  `{number: 2, title: "v0.1", state: "open", due: 1796083200, open: 3, closed: 2}` plus `url` on github and gitlab, and
  Azure DevOps answers unsupported; "every provider lists its iterations": "Sprint 1" from 2026-11-02 for 14 days on
  github and ado, 2026-11-02 to 2026-11-15 on gitlab, gives `start: 1793577600, end: 1794787199`; a GitLab 403 on
  iterations returns 3. In `tests/issues.bats`: `kit issues --stages` is `milestones` then `iterations`; `kit issues
  milestones --json` has the row keyed `2`; `kit issues backlog --json` exits 2 naming `kit flow backlog`; on ado,
  `kit issues milestones --json` is `{"rows":[],"errors":[]}`, exit 0.
- [ ] **Step 2: Run** both files. Expected: the calls exit 3 everywhere, `kit issues` is no verb.
- [ ] **Step 3: Implement** the folds `github_milestones`, `gitlab_milestones`, `github_iterations`,
  `gitlab_iterations`, and `ado_iterations` in `forge.jq`; the provider functions; drop the two features from each
  provider's `lacks` where implemented; `apps/issues.sh`, `kit-issues`, `docs/apps.md`'s section.
- [ ] **Step 4: Run** `tests/forge-conformance.bats tests/forge-ado.bats tests/issues.bats tests/syntax.bats
  tests/help.bats`, then `kit gate`.
- [ ] **Step 5: Commit** as `#162 kit: kit issues lists the forge's milestones and iterations, each with its mark`.

## Task 5: kit changes: reviews, checks, and merged

**Files:** Create `kit/modules/bash/apps/changes.sh`, `kit/bin/kit-changes`, `tests/changes-app.bats`, its syntax
fixture. Modify the three providers, `forge.jq` (`pr`, `ado_pr`, the GitLab change), `tests/forge-conformance.bats`,
`tests/forge-ado.bats`, `docs/integrations/{gh,glab,az}.md`, `docs/apps.md`.

**Interfaces:** `open-prs`' documents gain `reviewers` (feature `reviewers`) and `runs`, the `[{name, verdict, at,
took}]` `checks($now)` already reads, kept beside `checks`. GitHub adds `reviewRequests(first: 20) { nodes {
requestedReviewer { ... on User { login } ... on Team { slug } } } }` and `latestReviews(first: 20) { nodes { author {
login } state submittedAt } }` to the open changes' query: no request more. GitLab reads each open merge request's
`approvals`, one request each, beside the discussions it reads already; Azure DevOps maps each reviewer's vote, 10
and 5 approved, -5 and -10 changes, 0 requested.

| Stage | Source | Key | Fields | Mark |
| --- | --- | --- | --- | --- |
| `reviews` | `forge:open-prs`' `reviewers` | `.change, .login` | `change, title, login, state, at, unresolved` | `apps:forge-mark issues runs` |
| `checks` | `forge:open-prs`' `runs`, the latest of each name | `.change, .name` | `change, title, name, verdict, at, took` | as reviews |
| `merged` | `merged-prs`: GitHub `pullRequests(states: MERGED, last: 50, orderBy: {field: UPDATED_AT, direction: DESC})`, GitLab `merge_requests?state=merged&order_by=merged_at&per_page=50`, Azure DevOps `az repos pr list --status completed --top 50` | `.number` | the call's | as reviews |

- [ ] **Step 1: Write the failing tests.** The world's change 3 asks bob and has ada's comment: `reviewers` is
  `[{login: "bob", state: "requested", at: null}, {login: "carol", state: "approved", at: <epoch>}]` on every provider
  (the fixtures add carol's approval); `merged-prs` gives change 2, merged by ada; `kit changes checks --json` has
  rows `3 a` and `3 b`, `a`'s verdict the rerun's `SUCCESS`; GitHub's open changes still cost one GraphQL read
  (`reads github` is 1).
- [ ] **Steps 2 to 4:** run them (no `reviewers`), implement, run the five bats files and `kit gate`.
- [ ] **Step 5: Commit** as `#162 kit: kit changes lists each change's reviews, its checks, and the changes merged
  last`.

## Task 6: kit delivery: artifacts, environments, and deployments

**Files:** as Task 5's, with `apps/delivery.sh`, `kit-delivery`, `tests/delivery.bats`, and `docs/integrations/cd.md`
and `ci.md` (planned becomes the call).

| Provider | `artifacts` | `environments` | `deployments` |
| --- | --- | --- | --- |
| github | `gh api 'repos/{owner}/{repo}/actions/artifacts?per_page=100'`, one page, `limited` when `total_count` is more | `gh api 'repos/{owner}/{repo}/environments'`, `protection_rules[].type` and `reviewers`, `deployment_branch_policy` | `gh api graphql` on `repository.deployments(last: 50) { nodes { databaseId environment ref { name } commitOid state creator { login } createdAt latestStatus { state logUrl environmentUrl } } }` |
| gitlab | `glab api 'projects/:id/jobs?scope[]=success&per_page=100'`, the jobs with `artifacts_file` | `glab api 'projects/:id/environments?per_page=100'` and `protected_environments` | `glab api 'projects/:id/deployments?order_by=created_at&sort=desc&per_page=50'` |
| ado | `az pipelines runs artifact list --run-id` over the 20 newest runs | `az rest … distributedtask/environments` | `environmentdeploymentrecords` of each environment, 20 environments, the newest 50 |

| Stage | Key | Links | Mark |
| --- | --- | --- | --- |
| `artifacts` | `.id` | `name`: its page | `apps:forge-mark runs` |
| `environments` | `.name` | `name`: its page | `apps:forge-mark runs` |
| `deployments` | `.id` | `environment`: `row:delivery/environments/<name>`; `sha`: the commit's page | `apps:forge-mark runs` |

- [ ] **Step 1: Write the failing tests**: one artifact `kit-0.1.0.tar.gz`, digest `sha256:ab…`, from run 103; the
  environment `production`, required reviewers ada, branches `main`; deployment 9 of `release/0.1.0` to production,
  `success`, by ada. Each provider gives the same rows (Azure DevOps' digest null). 60 deployments on GitHub's fixture
  give 50 rows and `limited: 50`.
- [ ] **Steps 2 to 5:** as Task 4's; commit as `#162 kit: kit delivery lists the artifacts, environments, and
  deployments the forge holds`.

## Task 7: kit people: people, teams, and agents

**Files:** as Task 5's, with `apps/people.sh`, `kit-people`, `tests/people.bats`.

| Provider | `collaborators` | `teams` | `bots` |
| --- | --- | --- | --- |
| github | `gh api --paginate --slurp 'repos/{owner}/{repo}/collaborators?per_page=100'`, `role_name`; `type == "Bot"` is a bot | `gh api 'repos/{owner}/{repo}/teams?per_page=20'`, each `orgs/{org}/teams/{slug}/members?per_page=100`; a user-owned repository answers none | one `gh api graphql`: the default branch's `history(first: 100) { nodes { author { user { login __typename } } committedDate } }`, and the open and 50 newest merged changes' authors and reviews' authors, those of `__typename Bot` or a login ending `[bot]` |
| gitlab | `glab api 'projects/:id/members/all?per_page=100'`, paged; `access_level` to its role name; `bot: true` | `glab api 'projects/:id'`'s `shared_with_groups`, each `groups/:id/members?per_page=100` | the members with `bot: true`, `last` their `last_activity_on` |
| ado | `az devops user list` filtered to the project's members | `az devops team list --project …`, each `az devops team list-member` | lacks |

| Stage | Key | Mark |
| --- | --- | --- |
| `people` | `.login` | null |
| `teams` | `.team` | null |
| `agents` | `.login` | null |

- [ ] **Step 1: Write the failing tests**: ada admin, bob write, `dependabot[bot]` a bot; team `core`, members ada and
  bob; agents lists `dependabot[bot]` with `acted: ["change"]`. 25 teams give 20 and `limited: 20`. In a repository
  with no remote, `kit people --json` prints every stage empty, exit 0, and `kit people --marks` three nulls.
- [ ] **Steps 2 to 5:** as Task 4's; commit as `#162 kit: kit people lists who has access, their teams, and the bots
  that acted`.

## Task 8: kit audit: evidence, findings, controls, waivers, and policy

**Files:** as Task 5's, with `apps/audit.sh`, `kit-audit`, `tests/audit.bats`.

| Provider | `attestations` | `*-alerts` | `rulesets` | `bypasses` |
| --- | --- | --- | --- | --- |
| github | `gh api 'repos/{owner}/{repo}/attestations/<digest>'` for each asset digest of the 5 newest releases (`gh release view --json assets`'s `digest`) and the 20 newest artifacts' | `code-scanning/alerts`, `dependabot/alerts`, `secret-scanning/alerts`, each `?state=open,dismissed&per_page=100` paged; 404 with "disabled" or "no analysis found" is 3, 403 the remedy `gh auth refresh --scopes security_events` | `repos/{owner}/{repo}/rulesets?includes_parents=true` and each one's rules, and `branches/{default}/protection` | each ruleset's `bypass_actors`, and the dismissed alerts' `dismissed_by`, `dismissed_reason`, and `dismissed_comment` |
| gitlab | lacks | `vulnerability_findings` for `dependabot` and `code-scanning` by `report_type`, 403 or 404 off; secret-scanning lacks | `protected_branches` and `push_rule` | lacks |
| ado | lacks | lacks | `az repos policy list --branch <default>` | lacks |

| Stage | Source | Key | Mark |
| --- | --- | --- | --- |
| `evidence` | `attestations` | `.digest, .kind` | `apps:forge-mark runs releases` |
| `findings` | `apps:merge` of the three alerts calls, each row's `tool` its call's | `.tool, .number` | `apps:forge-mark runs` |
| `controls` | `rulesets` | `.id` | null |
| `waivers` | `bypasses` | `.waives, .actor` | null |
| `policy` | `kit policy --rules --json` | `.rule` | `apps:hash kit.yml .kit/policy` |

- [ ] **Step 1: Write the failing tests**: one attestation of `sha256:ab…`, kind `https://slsa.dev/provenance/v1`, by
  the release workflow; a code-scanning alert 4 open and a dependabot alert 1 dismissed by ada as `tolerable_risk`,
  which waivers lists too; secret scanning answering 404 "Secret scanning is disabled on this repository." gives
  findings an `errors` entry `{feature: "secret-scanning", off: true}` and exit 0; dependabot answering 403 gives its
  remedy under `errors`, code scanning's row still drawn, exit 1; the default branch's ruleset `main` with two rules
  and a bypass for the `release` team.
- [ ] **Steps 2 to 5:** as Task 4's; commit as `#162 kit: kit audit lists the forge's attestations, alerts, rulesets,
  and their bypasses, and the policy's rules`.

## Task 9: The catalog reads kit (the #138 plan's Tasks 1 to 4)

Wait for kit-68's message that #161 landed, then run the #138 plan's Tasks 1 to 4 as written there, against
kit-68's `rowRecords`, which reads any `{key, member, links, …}` row by catalog field id. Its commits keep their
`#138 kittui:` subjects.

## Task 10: Every planned tab is a catalog entry

**Files:** Modify `projects/tui/internal/flow/{fields.json,apps.go,view.go,menu.go,watch.go}`, `testdata/fake-kit`, and
the tests below. Create `testdata/<verb>-<stage>.json` for each stage, from Tasks 2 and 4 to 8's bats fixtures' output.

**Interfaces:** Consumes Task 9's `Schema.Command`, `Takes`, `Staged`, `Key`, `Empty`, and `Records`. `apps.go`'s
stages become `<verb>/<stage>`: Issues `backlog, underway, issues/milestones, issues/iterations`; Changes `changes,
changes/reviews, changes/checks, changes/merged`; Delivery `releases, builds, delivery/artifacts,
delivery/environments, delivery/deployments`; People, Trust (`audit/…`), and Platform all theirs. Each entry is
`command: ["<verb>", "<stage>", "--json"]`, `record: ".rows[]"`, `staged: true`, `takes: ["workspace"]`. A document
with `limited` makes the tab's count `N+`; an `errors` entry with `off` makes the empty line, dim, `<feature> is off
here`. `planned`, `ofPlanned`, and the planned line in `view.go` go.

- [ ] **Step 1: Write the failing tests** in `apps_test.go`: `TestEveryTabReadsAKitStage`, every stage of every app
  `sourced`; `TestATabOffHereSaysSo`, `audit/findings` answered with secret scanning off draws `secret-scanning is off
  here`; `TestACappedTabCountsWithAPlus`, `delivery/deployments` with `limited: 50` shows `50+`;
  `TestATabWithANullMarkIsNeverStale`, `people/teams` offers no Watch.
- [ ] **Step 2: Run** `go test ./internal/flow -run 'TestEveryTab|OffHere|Capped|NullMark' -count=1`. Expected: the
  tabs are planned.
- [ ] **Step 3: Implement**; fake-kit answers each `<verb> <stage> --json`, `--stages`, and `--marks` from the fixtures.
- [ ] **Step 4: Regenerate the goldens** as the Global Constraints say, review `testdata`'s diff, run `go test ./...`
  and the coverage command. Expected: 98.0 or more.
- [ ] **Step 5: Commit** as `#138 kittui: every planned tab reads its kit stage, says when the forge has it off, and
  counts a capped list with a plus`.

## Task 11: The verbs open their apps

**Files:** Modify `projects/tui/cmd/kittui/main.go`, `main_test.go`, `internal/flow/apps.go` (`AppOf`), `tests/tui.bats`,
`projects/tui/README.md`, `docs/features.md`, `docs/apps.md`.

**Interfaces:** `AppOf(verb, words)` maps `issues`, `changes`, `delivery`, `people`, `audit`, and `platform` to their
apps, a tab among the words opening it maximized, kit flow's tabs included: `AppOf("issues", ["backlog"])` is Issues
on backlog. kittui's `main` takes the six verbs as it takes `workspace`.

- [ ] **Step 1: Write the failing tests**: `TestAnAppsVerbOpensItOnTheTabNamed` in `main_test.go`, `issues backlog`
  giving `shell.Issues` maximized on backlog and `audit` giving Trust on evidence; in `tests/tui.bats`, `kit people`
  on a terminal runs `kittui native-native people`.
- [ ] **Steps 2 to 4:** run, implement, run `go test ./cmd/kittui ./internal/flow -count=1` and `tests/tui.bats`.
- [ ] **Step 5: Rewrite** `projects/tui/README.md`'s apps section for the seven apps, each tab and the verb it reads;
  `docs/features.md` names no planned tab.
- [ ] **Step 6: Commit** the kit paths as `#162 kit: each app's verb opens kittui on its app`, then `projects/tui` as
  `#138 kittui: each app's verb opens it on the tab named, and the README names every tab's source`.
