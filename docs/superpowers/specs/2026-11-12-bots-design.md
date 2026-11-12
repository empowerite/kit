# Bots: design

Status: step 1 built, 2026-11-12; steps 2 to 8 designed. see: issue #258

## Intent

A person hands work to agents: a Claude Code session, a Codex run, a script. Today an agent acts through the person's
own login, so a label it sets, a comment it writes, and a pull request it opens all read as the person's. Nobody can
tell what the person decided from what their agent did, and the person cannot tell at a glance what is waiting on them
and what is waiting on their agents.

A bot is the forge account a person's agents act through: on GitHub, a GitHub App's `<slug>[bot]` account. kit helps
the person set one up with the permissions and the repository scope its jobs need, and then makes three things work:

1. Handing an issue to the bot is the trigger for an agent to pick it up.
2. The bot reviews code: the person's changes, at their request, and the person reviews the bot's.
3. Every issue and change is in someone's court, and kit says whose: the person's, the bot's, someone else's, or
   nobody's.

## Terms

- **Bot**: a forge account that acts for one person, or for an organization's people, under its own login. On GitHub
  it is a GitHub App's bot account, `empowerite-jig[bot]`. It is the account of what
  [the entities design](2026-11-09-entities-design.md#actors) calls an Agent. It is not that design's Bot, a service
  account that acts for no person, such as Dependabot; this design calls that one a service bot.
- **The person**: whoever the bot acts for. On a desk, the account the forge's CLI is signed in as; in CI, the
  person the bot's work was handed by.
- **Agent session**: one run of a harness acting as the bot. One bot has any number of sessions.
- **Hand**: give an issue or a change to a bot, as its next job.
- **Mark**: a label kit sets where the forge cannot record a hand natively.
- **Court**: who holds the next move on an issue or a change: `you`, `bot`, `others`, or `none`, always from the
  caller's view.
- **Job**: what a bot is set up to do: `work` on issues, `review` changes, and `workflows`, editing CI files.

## Goals

- Set up a bot in one command, from a terminal: register the App, install it on the repositories it works in, store
  its key, and prove the chain by minting a token. Adopt an App that already exists the same way.
- Ask for exactly the permissions the chosen jobs need, and say when an App has more or fewer.
- Run any command as the bot, for as long as it runs, past the hour a GitHub installation token lasts.
- Hand an issue to the bot with `kit assign`, and take it back with `kit unassign`, on every forge.
- Let an agent find its queue, claim an issue, ask the person a question, and hand back, each with one verb.
- Ask the bot for a review, and let the bot post one, with `kit review`.
- Give every issue row and change document a court, filter Flow by it, and show it in kittui.
- Pick up handed work on the desk, with a watcher the person starts, and in CI, with a workflow a package delivers.

## Non-goals

- kit is not the agent. Which harness does the work, with which prompt and model, is the person's command; kit hands
  it the brief and the identity.
- kit runs no service and receives no webhook. The desk watcher polls; CI reacts to the forge's own events.
- GitHub's agent apps, which can be assigned issues natively, are a partner preview with a waitlist. When anyone can
  register one, the GitHub provider gains native assignment and the mark goes away there.
- A bot that merges. A human merges, as [the workflow](../../../.kit/workflow.md) says.
- Several bots for one person on one forge, routed by role. One bot per person per forge serves every job.

## Anti-goals

- An act of the bot that reads as the person's, or one of the person's that reads as the bot's.
- A key that opens more than the repositories the bot works in, or a token with permissions no job needs.
- A private key in a repository, in `kit.yml`, in a log, or on the screen.
- State of kit's own about who holds what. The court is computed from what the forge holds, every time.
- A bot's approval counted in place of the person's review of the bot's own work.

## What exists now

- The empowerite organization registers `empowerite-jig` (App 4906610), installed on `empowerite/jig` alone, with
  actions, contents, issues, pull requests, and workflows to write, and checks and statuses to read. Its key is in the
  login keychain as service `jig-app`, account `4906610`, hex-encoded by `security`. Commits in jig are already
  authored by `empowerite-jig[bot]`.
- jig's superseded jigbot RFC, `jig/docs/rfcs/superseded/jig-014-jigbot.md`, holds the prior thinking: a session acts
  as an App and never as the person; one App per organization; `jig app create` through GitHub's manifest flow, with
  a listener on localhost; commits stay the person's, with the session named in a trailer. jig dropped the per-person
  App because an organization registers at most a hundred Apps.
- `infrastructure/bash/provision-github-app.sh` adopts an App by hand: it asks for the ID and the key, derives the
  installation, mints a token to prove the chain, and only then stores anything.
- In kit, `kit assign --to=<login>` hands an issue to any login, through the tracker seam. The forge seam's `bots`
  call and `kit people agents` name bots that acted, and `open-prs` already reads a bot requested as a reviewer.
  `forge:me` asks the CLI who is signed in. kit holds no token and touches no keychain.
- Welcome's checks are bash, in `modules/bash/welcome.sh`; a check that holds but has a fix is an offer. Flow's
  issue rows are built in `modules/jq/underway.jq`, change documents in `modules/jq/forge.jq`. The only notion of a
  turn today is Flow's colour: yellow when a human owes a decision, blue when a machine or a reviewer does.

GitHub's limits, which shape everything below:

| | A GitHub App's bot |
| --- | --- |
| Assigned an issue | no: the assignees call refuses it, by REST and by GraphQL |
| Requested as a reviewer | no |
| Posts a review that approves, comments, or requests changes | yes, with pull requests to write |
| Opens issues and pull requests, comments, labels, and pushes | yes, within its permissions |
| Its token | an installation token, minted from the App's key, which lasts one hour |
| Registered or installed by API | no: both are pages in a browser; the manifest flow fills in the first |

## Decided

From the request (issue #258):

- A forge App is the bot's identity: GitHub first, the other forges after.
- kit sets up the App's permissions and scope.
- Handing an issue to the bot triggers its pickup.
- The bot reviews code.
- kit keeps track of what is in the person's court and what is in the bot's.

## Design

### The bot on each forge

A provider declares what its forge can do for a bot, as features of the forge seam:

| Feature | GitHub | GitLab | Azure DevOps |
| --- | --- | --- | --- |
| `bot-account` | a GitHub App | a project or group access token's bot user | a service principal |
| `bot-new` | yes, through the manifest flow | yes, by API | later |
| `bot-assign`, the bot as a native assignee | no | to verify | to verify |
| `bot-review-request`, the bot as a requested reviewer | no | to verify | to verify |
| `bot-review`, the bot posts a review | yes | yes | yes |

Where `bot-assign` is missing, a hand is a mark; where it is there, a hand is an assignment, and kit reads either the
same way.

### The bots file

The bots a person has set up live in `kit/bots.yml` under `XDG_CONFIG_HOME`, else `~/.config`, as
[workspaces](2026-11-08-workspaces-design.md) keep theirs. It holds public facts only:

```yaml
bots:
  - name: empowerite-jig      # the App's slug; the bot's login is empowerite-jig[bot]
    host: github.com
    app: 4906610              # the App ID
    owner: empowerite         # the account that registered the App
    jobs: [work, review]      # what it is set up to do
```

A repository may name its bot in `kit.yml`, as `bot: empowerite-jig`, so every desk and every CI run knows the login
behind the repository's marks without the bots file. The installation that reaches a repository is looked up with the
App's key, never stored.

The key lives in the operating system's secret store, under service `kit-bot`, account `<host>/<app>`: the macOS
keychain through `security`, or the Secret Service through `secret-tool` on Linux. kit stores the key base64-encoded,
so `security` never hands it back as hex. Where neither store exists, `kit bots` refuses and names the store to
install. In CI the key comes from the environment, `KIT_BOT_KEY`, filled from a secret.

### `kit bots`

`kit bots` lists the person's bots: name, host, App, owner, the repositories each installation reaches, and the
jobs its permissions allow. Its forms:

| Form | Does |
| --- | --- |
| `kit bots --new=<name> [--owner=<account>] [--jobs=<job>…] [--repos=<repository>…]` | registers and installs an App, stores its key, and proves it |
| `kit bots --add=<name> --app=<id> --key=<source> [--jobs=<job>…]` | adopts an App that exists, from its key: a PEM file, or `keychain:<service>/<account>` |
| `kit bots --check [--bot=<name>]` | compares the App's permissions with its jobs, and its repositories with the current one |
| `kit bots --repos=<name>` | opens the installation's page to change which repositories it reaches, and waits |
| `kit bots [--bot=<name>] token [--json]` | prints an installation token for the current repository, for a pipe; `--json` prints `{token, expires_at}` |
| `kit bots [--bot=<name>] run <command>…` | runs a command as the bot; every word after `run` is the command's, and none is parsed |
| `kit bots --remove=<name>` | forgets a bot and deletes its key from the store; the App stays on the forge |

A list option repeats or joins its values with `:`, `--jobs=work:review`. Without `--bot`, the bot is the one
`kit.yml` names, else the only one on the primary remote's host.

Every form prints the key never. `--add` proves the key before it stores anything, as the infrastructure script
does: it mints a JWT (a JSON Web Token signed with the key), reads the App, finds an installation, and mints a
token with the forge seam's `bot-proof`, a token narrowed to no repository that holds `metadata: read` alone, where no
repository is reached, then revokes it. The forge seam's calls for a bot, at contract 6, are `bot-app`,
`bot-installation`, `bot-installations`, `bot-proof`, `bot-token`, `bot-repositories`, and `bot-revoke`.

The jobs ask for these permissions, and nothing else:

| Job | GitHub App permissions |
| --- | --- |
| always | metadata: read |
| `work` | contents, issues, and pull requests: write; checks, statuses, and actions: read |
| `review` | pull requests: write; contents and checks: read |
| `workflows` | workflows: write |

`--check` names each permission the App lacks for a job, and each one it holds that no job needs: `empowerite-jig`
holds actions to write, which no job needs. It exits 1 when a job lacks a permission, since that job would fail when
the bot acts, and when the installation does not reach this repository; a permission no job needs is advice, and exits
0. A change of permissions is the App owner's, on the App's page, and every installation accepts it on its own page;
kit opens both and waits.

### `kit bots --new`

1. kit computes the manifest from the jobs: the name, the owner's URL, no webhook, no user authorization, the
   permissions above, and a redirect to a listener on `127.0.0.1`.
2. A non-interactive helper in kittui's binary, `libexec/kittui`, holds the listener: bash cannot. It serves a page
   that posts the manifest to `github.com/settings/apps/new`, or `github.com/organizations/<owner>/settings/apps/new`,
   with a random `state`, and opens it in the browser.
3. The person clicks Create. GitHub redirects to the listener with a code; the helper checks the state, exchanges the
   code at `POST /app-manifests/{code}/conversions`, and hands kit the ID, the slug, and the key. kit keeps the key
   and drops the client secret and the webhook secret, which no job needs.
4. kit opens the App's install page, with the repositories `--repos` names, and polls until the installation
   appears.
5. kit proves the chain, as `--add` does, writes the bots file, and prints what the bot can now do.

`--owner` defaults to the person's own account. A person's own App acts for them alone, its owner is a fact GitHub
publishes, and it does not count against an organization's hundred. An organization's owner approves its installation
on the organization's repositories once. `--owner=<organization>` registers one App for the organization's people
instead, as jig chose.

### Acting as the bot

`kit bots run <command>…` runs the command with:

- `GH_TOKEN` set to a fresh installation token, and a `gh` on `PATH` that mints a new one when the old is within ten
  minutes of its hour, so a session that runs all afternoon keeps working. Tokens live only in a directory of the
  session's own, which `file:scratch` makes, readable by the person alone, removed when the command ends, and are
  revoked then.
- `KIT_BOT` set to the bot's name, and `KIT_BOT_FOR` to the person's login.
- git's credential helper for the forge's host answering with the same token, so `git push` and `kit submit` push as
  the bot.

Under `KIT_BOT`, kit itself changes in four places:

- `forge:me` answers the bot's login, and a new call, `forge:principal`, the person's.
- `kit work` assigns the person, never the bot, and marks the issue for the bot where it must.
- `kit commit` keeps the person as the author and the committer, so signing and `require_signing` keep working, and
  adds a trailer, `Agent: empowerite-jig[bot]`. Pushes, pull requests, comments, labels, and reviews are the bot's.
- `kit ready` requests the person's review on the bot's pull request.

### Handing work to the bot

- `kit assign <ref>… --to=bot` hands each issue to the person's bot on that issue's forge: `--to=bot` names it,
  `--to=<slug>[bot]` names any. Without `bot-assign`, kit keeps the person as an assignee and adds the mark
  `bot:<slug>`, creating the label the first time. `kit unassign <ref>… --from=bot` takes the hand back.
- An issue's state for its bot comes from what the forge holds: handed and not underway is queued; underway, with a
  branch and a draft pull request, is working; with the mark `bot:asks`, it waits on the person.
- `kit bots --queue[=<name>]` lists the issues handed to the bot, queued, oldest first, across the workspace.
- `kit bots --claim=<ref>`, as the bot, runs `kit work <ref>` and comments that the bot picked it up for the person.
- `kit bots --ask=<ref> --body=<text>`, as the bot, comments the question and adds `bot:asks`. The person answering,
  by a comment or by handing it again, takes `bot:asks` off: a hand does it at once, a comment at the next pickup.
- `kit bots --brief=<ref>` prints what an agent session starts from: the issue, its comments, the worktree, and the
  rules to follow, `.kit/1st.md` and the worker contract.

The tracker seam gains `label` and `unlabel` for the marks, so a hand works on any tracker that keeps labels: a tag on
Azure Boards, a label on Jira.

### Reviews

`kit review` is a verb of its own:

| Form | Does |
| --- | --- |
| `kit review <n> --request=<login>…` | asks for a review; `--request=bot` asks the person's bot |
| `kit review <n> --approve \| --comment \| --request-changes [--body=<text> \| --body-file=<file>]` | posts a review as the caller |
| `kit review <n> --notes=<file>` | adds line comments to the review, from a JSON list of path, line, and body |

A request of a login is the forge's own. A request of a bot without `bot-review-request` is the mark `bot:<slug>` on
the pull request: on an issue the mark means work it, on a change the bot does not author it means review it. Posting
its review takes the mark off. The bot never approves a pull request it authored, which GitHub refuses anyway, and
`kit bots --check` says whether the branch's rules count a bot's approval.

The forge seam gains `pr-request-review`, `pr-review`, and `pr-comment`, with their features.

### Court

Every issue row and change document gains `court`, one of `you`, `bot`, `others`, or `none`, and `why`, a few words.
The caller is `forge:principal`, so a bot's session and its person see the same court, and the bot is any login in
the person's bots file or the repository's `bot:`. Another person's bot is `others`. A bot's login reads
`<slug>[bot]` in GitHub's REST API, `<slug>` in its GraphQL API, and `app/<slug>` in `gh`'s output, and the court
treats all three as the bot; this was verified against empowerite/jig#139. The first rule that matches wins:

A change:

| When | Court | Why |
| --- | --- | --- |
| merged or closed | none | done |
| by the bot, draft, failing, or conflicted | bot | working |
| by the bot, the person's latest review requests changes, no push since | bot | answering review |
| by the bot, ready | you | to review |
| marked for the bot, the bot has not reviewed since the last push | bot | reviewing |
| by the person, draft, failing, conflicted, or changes requested | you | working |
| by the person, approved and mergeable | you | to merge |
| by the person, otherwise | others | waiting on review |
| the person's review is requested | you | to review |
| otherwise | others | theirs |

An issue with an open change takes its change's court. Otherwise:

| When | Court | Why |
| --- | --- | --- |
| closed | none | done |
| handed to the bot, with `bot:asks` | you | to answer |
| handed to the bot | bot | queued |
| assigned to the person | you | to do |
| assigned to anyone else | others | theirs |
| unassigned | none | open |

The rules live in one jq module, `modules/jq/court.jq`, which both builders call. Flow gains `--court=<court>…`,
and kittui a Court column, 🧑 you, 🤖 bot, 👥 others, and two presets on the issues and the changes stages, Your Court
and Bot's Court.

### Picking up

On the desk, `kit bots --watch[=<name>] [--every=<seconds>] [--max=<sessions>]` polls through Flow's snapshot, 60
seconds by default. For each issue newly in the bot's queue, and each change newly marked for its review, it claims
the work and starts the command `kit.yml` names, else the bots file, under `kit bots run`, in the worktree:

```yaml
bots:
  run: claude -p "$(kit bots --brief="${KIT_REF}")"
  review: claude -p "$(kit bots --brief="${KIT_REF}" --review)"
```

`--max` caps the sessions at once, one by default. The watcher prints what it started and what each session's exit
was, and starts nothing it cannot run as the bot.

In CI, a package, `github/bot`, delivers a workflow that runs on `issues: labeled` and `pull_request: labeled` with
a `bot:` mark. It mints the token from `KIT_BOT_KEY`, an Actions secret, and `KIT_BOT_APP`, a variable, and runs the
same claim and the same command.

### Welcome

- `bot-<host>`, an offer, holds where no bot is set up and offers `kit bots --new`.
- Where a bot is set up, the check fails, each with its remedy, when the key is missing from the store, when the
  installation does not reach this repository, or when the permissions lack a job's need.

## The order of the work

1. `kit bots`: the bots file, `--add` from a key file or a keychain item, the secret stores, `token`, `run` with
   the refreshing `gh`, `--check`, `--remove`, `forge:me` and `forge:principal` under `KIT_BOT`, the commit trailer,
   and the Welcome check. Proved against `empowerite-jig`.
2. `kit bots --new`: the manifest helper in kittui, the install, and `--repos`.
3. Handing: `kit assign --to=bot`, `kit unassign --from=bot`, the tracker's `label` and `unlabel`, `--queue`,
   `--claim`, `--ask`, and `--brief`.
4. Court: `court.jq`, `kit flow --court`, kittui's column and presets.
5. Reviews: `kit review`, and the forge calls under it.
6. Picking up on the desk: `kit bots --watch`.
7. Picking up in CI: the `github/bot` package.
8. GitLab's provider, then Azure DevOps'.

## My calls

- The word is bot, the person's word, and the forge's own (`[bot]`). The entities design's Bot, which acts for no
  person, becomes a service bot there.
- On GitHub a hand is a label, `bot:<slug>`, with the person kept as an assignee, because a GitHub App cannot be
  assigned. One label covers both jobs, by what it is on.
- `bot:asks` is one shared label, not one per bot, since it marks the issue's state, not the bot.
- `--owner` defaults to the person's own account, against jig's one App per organization: the person asked for a bot
  acting on their behalf, and a person's App does not count against the organization's hundred. To verify before
  step 2: that a person's App counts against the person's own limit only.
- Commits stay the person's, with an `Agent:` trailer, as jig chose, so signing keeps working. Committing as the bot
  through GitHub's API, which GitHub signs, is left for later.
- The key is stored base64-encoded in the operating system's store, under kit's own service name, never in a file.
  Adopting the `jig-app` item copies it there.
- The court is computed every time from the forge, never stored.
- `kit review` is a new verb, not a form of `kit changes`, because it acts and `kit changes` reads.
- The watcher is a desk loop the person starts, not a service, and CI uses the forge's own events.

## Left to the builder

- Whether GitLab's and Azure DevOps' bot accounts can be assignees and requested reviewers, which sets their
  features.
- Whether `PUT /user/installations/{id}/repositories/{id}` accepts the token `gh` holds; where it does not, `--repos`
  opens the page, as designed.
- Whether a branch ruleset counts an App's approval, for `--check` to report.
- The manifest helper's name as a kittui subcommand, and its timeout.
- The brief's exact text.
- The glyphs for the Court column, held to two cells.
