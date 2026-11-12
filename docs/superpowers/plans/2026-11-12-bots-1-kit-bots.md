# Bots, step 1: `kit bots` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A `kit bots` verb that adopts a GitHub App as the person's bot, keeps its key in the operating system's
secret store, mints installation tokens narrowed to the current repository, runs any command as the bot for as long as
it runs, and checks the App's permissions against its jobs; with kit itself telling the bot from the person it acts for.

**Architecture:** The App's API calls are GitHub's, so they are forge seam calls under a new feature, `bot-accounts`,
at contract 6, which the GitHub provider implements with `curl` and an `openssl`-signed JWT; GitLab and Azure DevOps
stay at contract 5 and so lack it. `modules/bash/secrets.sh` is the secret store; `modules/bash/bots.sh` holds the
bots file, the choice of bot, the jobs, and the run; `bin/kit-bots` parses and prints. Under `kit bots run`, a `gh`
shim and a git credential helper in a private session directory hand out a fresh token on every call.

**Tech Stack:** bash 5, jq, mikefarah yq 4 (through `yaml.sh`), curl, openssl, macOS `security` or `secret-tool`, bats.

**Spec:** `docs/superpowers/specs/2026-11-12-bots-design.md`, sections "The bots file", "`kit bots`", "Acting as the
bot", and "Welcome". This plan is the spec's step 1 of 8; `--new`, handing, court, reviews, and pickup are later plans.

## Global Constraints

- The bots file: `${XDG_CONFIG_HOME:-$HOME/.config}/kit/bots.yml`, `{bots: [{name, host, app, owner, jobs}]}`; `app` a
  number, `jobs` a list of `work`, `review`, `workflows`, default `[work, review]`.
- The key: service `kit-bot`, account `<host>/<app>`, value the PEM base64-encoded on one line. macOS: `security`,
  with the secret passed on stdin to `security -i`, never in argv; Linux: `secret-tool`, the secret on stdin.
- Permissions by job, verbatim from the spec: always `metadata: read`; `work`: `contents`, `issues`, `pull_requests`
  write, `checks`, `statuses`, `actions` read; `review`: `pull_requests` write, `contents`, `checks` read;
  `workflows`: `workflows` write. Write covers read.
- A token is narrowed to the current repository, minted with `{"repositories": ["<name>"]}`, refreshed when within
  600 seconds of `expires_at`, and revoked with `DELETE /installation/token` when `run` ends.
- The API base: `https://api.github.com` for github.com, `https://<host>/api/v3` otherwise.
- Under `run`: `KIT_BOT` (the bot's name), `KIT_BOT_LOGIN` (`<name>[bot]`), `KIT_BOT_FOR` (the person's login, read
  before the switch).
- The commit trailer: `Agent: <KIT_BOT_LOGIN>`.
- The key is never printed, logged, or put in argv; a token is printed only by `kit bots token`.
- Exits: 0 done; 1 a failure (no key, no installation, the forge refused); 2 a usage error or an unknown or ambiguous
  bot.
- Every file under `kit/` names paths from `kit/`. New files start from `kit new file <path>`. Verb headers follow
  the others: banner with `(kit verb: workflow)`, usage lines in the grammar of `modules/bash/syntax.sh` (no `--`, no
  optional `=value`), `Arguments:`, one `Exits` line, `see:`.
- Commits: `export PATH="$PWD/kit/bin:$PATH"`, then `kit commit` with the subject starting `#258`, the
  `Co-Authored-By` trailer as a second `--message`. The worktree is shared with peer sessions: stage only your own
  files by exact path, guard the index (`test "$(git diff --cached --name-only)" = "<your files, sorted>" && kit
  commit …`), never `git add` a directory, never touch another session's staged or unstaged hunks.

## Review Focus

1. A key handed over as `security`'s hex form, with CRLF line ends, or as PKCS#8 (`BEGIN PRIVATE KEY`): accepted and
   stored as the same PEM. Pinned in Tasks 1 and 3.
2. An `--app` that does not match the key, so `GET /app` answers 401: refused before anything is stored, the file and
   the store untouched. Pinned in Task 3.
3. A repository the installation does not reach (404 on `/repos/{owner}/{repo}/installation`): `token` and `run` fail
   naming `kit bots --check` and the App's installation page. Pinned in Task 4.
4. A `run` whose command fails, or is interrupted: the token is revoked, the session directory removed, and the
   command's exit status passed through. Pinned in Task 5.
5. Two bots on one host, with neither `--bot` nor `bot:` in `kit.yml`: refused naming both, exit 2. Pinned in Task 3.

---

### Task 1: The secret store

**Files:**

- Create: `kit/modules/bash/secrets.sh`
- Create: `tests/fakes/fake-secrets` (linked as `security` and `secret-tool`; keeps items as files under
  `${FAKE_DIR}/secrets/`, reads the secret from `security -i`'s stdin command and from `secret-tool store`'s stdin,
  and answers `find-generic-password -w` as `security` does, hex when the value holds a newline)
- Test: `tests/secrets.bats`

**Interfaces:**

- Produces:
  - `secrets:backend` prints `keychain` or `secret-service`; with neither, errors `install secret-tool (libsecret) so
    kit can keep a key; kit keeps none in a file` and returns 1.
  - `secrets:put <service> <account>` stores stdin, base64-encoded, replacing any item.
  - `secrets:get <service> <account>` prints the stored value decoded; returns 1, silent, when there is none.
  - `secrets:delete <service> <account>`; succeeds when there was none.
  - `secrets:foreign <service> <account>` prints an item another tool stored, as it stored it: un-hexed when
    `security` answers hex.

- [ ] **Step 1: Write the failing tests**
  - `"secrets:put then secrets:get gives back a PEM with its newlines"`: put a three-line PEM, get it, compare.
  - `"the secret never reaches argv"`: after put, no line of `${FAKE_DIR}/security.calls` contains the PEM or its
    base64.
  - `"secrets:get of no item returns 1 and prints nothing"`.
  - `"secrets:foreign un-hexes what security hex-encodes"`: an item stored by the fake as raw PEM comes back as PEM.
  - `"with neither security nor secret-tool on PATH, secrets:backend names secret-tool and returns 1"`.
  - `"secret-tool is the backend where security is missing"`: put and get through `secret-tool`.
- [ ] **Step 2: Run `kit/bin/kit tools run bats tests/secrets.bats`; expect every test to fail.**
- [ ] **Step 3: Implement the five functions in `kit/modules/bash/secrets.sh`.** `security -i` reads one command per
  line; base64 holds no space or quote, so the line is safe.
- [ ] **Step 4: Run the tests; expect them to pass.**
- [ ] **Step 5: Commit** `#258 secrets.sh: a key kept in the macOS keychain or the Secret Service, never in argv`.

### Task 2: The forge seam's `bot-accounts`

**Files:**

- Modify: `kit/modules/bash/forge.sh` (header: the calls and the feature table; `forge_contract=6`;
  `forge_features+=(bot-accounts:6)`; `forge_feature_of` for the five calls; thin `forge:bot-*` wrappers beside
  `forge:me`)
- Modify: `kit/modules/bash/forge/github.sh` (`forge/github:contract` prints `6`; the five calls; `forge/github:jwt`
  and `forge/github:app-api` as the provider's own)
- Test: `tests/forge-bots.bats`; `tests/forge-capabilities.bats` for the lack on GitLab and Azure DevOps

**Interfaces:**

- Consumes: nothing from Task 1; the PEM arrives as a file path.
- Produces, each taking `<app> <pem-file>` first:
  - `forge:bot-app <app> <pem>` prints `{slug, owner, permissions, html_url}` from `GET /app`.
  - `forge:bot-installation <app> <pem> <owner/repo>` prints `{id, permissions, repository_selection, html_url}`
    from `GET /repos/{owner}/{repo}/installation`; returns 1 on 404 with no output.
  - `forge:bot-token <app> <pem> <installation> <repo>` prints `{token, expires_at}`, narrowed to `<repo>`.
  - `forge:bot-repositories <token>` prints the full names the token reaches, one per line.
  - `forge:bot-revoke <token>`.
  - `forge/github:jwt <app> <pem>` prints an RS256 JWT, `iat` 60 seconds back, `exp` 540 ahead, `iss` the App ID.

- [ ] **Step 1: Write the failing tests**, with a test key from `openssl genrsa` and `fake-tools curl`:
  - `"forge/github:jwt signs what the key's public half verifies"`: split the JWT, verify with `openssl dgst
    -verify`, and read `iss` back as the App ID.
  - `"forge:bot-token asks for the one repository and prints the token and its expiry"`: the curl call's body holds
    `"repositories":["kit"]`.
  - `"forge:bot-installation returns 1 on a 404"`.
  - `"the API base is api.github.com for github.com and /api/v3 for another host"`.
  - `"GitLab and Azure DevOps lack bot-accounts"`, in `forge-capabilities.bats`: the call returns 3.
- [ ] **Step 2: Run them; expect failures.**
- [ ] **Step 3: Implement** the calls with `curl --fail-with-body --silent`, the JWT with `openssl dgst -sha256
  -sign`, base64url by `tr '+/' '-_' | tr -d '='`. Missing `curl` or `openssl` errors naming the tool to install.
  Calls that change the forge (`bot-token`, `bot-revoke`) go through `forge:changing` only where it applies to the
  snapshot; they do not, so they call `forge:call` directly.
- [ ] **Step 4: Run `tests/forge-bots.bats tests/forge-capabilities.bats tests/forge-conformance.bats`; expect pass.**
- [ ] **Step 5: Commit** `#258 forge seam: bot-accounts at contract 6, a GitHub App's JWT, installation, and token`.

### Task 3: `bots.sh` and `kit bots`, `--add`, `--remove`

**Files:**

- Create: `kit/modules/bash/bots.sh`
- Create: `kit/bin/kit-bots` (from `kit new file kit/bin/kit-bots`)
- Modify: `kit/modules/bash/settings.sh` (top-level key `bot`), `kit/packages/kit/files/repo/.kit/kit.schema.json`
  (`bot`: a string, the name of the repository's bot)
- Test: `tests/bots.bats`

**Interfaces:**

- Consumes: Task 1's `secrets:*`; Task 2's `forge:bot-*`.
- Produces:
  - `bots:file` prints the file's path; `bots:json` prints `{bots: [...]}`, empty when the file is missing.
  - `bots:choose [<name>]` prints one bot's JSON: the named one; else `bot` in `kit.yml`; else the only bot on the
    primary remote's host. Unknown, none, or two on the host: an error naming them, exit 2.
  - `bots:key <bot-json> <file>` writes the PEM to `<file>`, mode 600; errors `no key for <name> in the <backend>;
    run: kit bots --add=<name> --app=<app> --key=<file>` and returns 1.
  - `bots:add <name> <app> <source> [<job>…]`: `<source>` a PEM file or `keychain:<service>/<account>`.
  - `bots:remove <name>`.
  - `bots:jobs-need <job>…` prints the permissions object the jobs need.

The verb's usage lines:

```text
kit bots [--text | --json | --jq]
kit bots --add=<name> --app=<id> --key=<source> [--jobs=<job>…]
kit bots --remove=<name>
kit bots --check [--bot=<name>] [--text | --json | --jq]
kit bots [--bot=<name>] token
kit bots [--bot=<name>] run <command>…
```

The list's JSON is `{file, bots: [{name, host, app, owner, jobs, key}]}`, `key` true when the store holds it.

- [ ] **Step 1: Write the failing tests:**
  - `"kit bots lists none where the file is missing"`, text and JSON.
  - `"--add proves the key before it stores anything"`: routes for `/app` and `/repos/*/installation` answer, the
    file gains `{name, host: github.com, app, owner, jobs: [work, review]}`, the store gains the key, and a token was
    minted and revoked.
  - `"--add with an App ID the key does not sign for stores nothing"`: `/app` answers 401; file and store unchanged;
    exit 1, naming the App ID.
  - `"--add takes keychain:jig-app/4906610, hex as security gives it"`, and a PKCS#8 key with CRLF line ends.
  - `"--add refuses a slug that differs from --add's name"`: the name is the slug, so the login is right.
  - `"--remove forgets the bot and deletes its key"`.
  - `"two bots on one host and no choice: exit 2, naming both"`; `"bot: in kit.yml chooses"`.
- [ ] **Step 2: Run them; expect failures.**
- [ ] **Step 3: Implement** `bots.sh` and the verb's `list`, `--add`, and `--remove`. Write the file with `yq
  --inplace` through `yaml.sh`, so comments stay. The PEM goes to a `file:temp` file, mode 600, removed on return.
- [ ] **Step 4: Run `tests/bots.bats tests/help.bats tests/exits.bats tests/arguments.bats tests/syntax.bats`.**
- [ ] **Step 5: Commit** `#258 kit bots: adopts a GitHub App as your bot, proving its key before keeping it`.

### Task 4: `--check` and `token`

**Files:**

- Modify: `kit/modules/bash/bots.sh`, `kit/bin/kit-bots`
- Test: `tests/bots.bats`

**Interfaces:**

- Produces:
  - `bots:check <bot-json>` prints `{name, installation, repository, reached, lacks: {<job>: [<permission>]},
    extra: [<permission>]}`; `reached` false when this repository is not in the installation.
  - `bots:token <bot-json>` prints `{token, expires_at}` for the current repository; refuses outside a repository.

- [ ] **Step 1: Write the failing tests:**
  - `"--check names what a job lacks and what no job needs"`: an installation with `actions: write` and no
    `checks` gives `extra: ["actions: write"]` and `lacks: {work: ["checks: read"]}`.
  - `"--check says where the installation does not reach this repository"`: 404, the text names the installation
    page, exit 1.
  - `"token prints only the token"`; `"token outside a repository refuses, exit 1"`.
  - `"token where the installation does not reach names kit bots --check"`.
- [ ] **Step 2: Run them; expect failures.**
- [ ] **Step 3: Implement.** `extra` is each permission the installation holds that `bots:jobs-need` of the bot's
  jobs does not, `metadata` aside; a write held where read is needed is extra as `<permission>: write`.
- [ ] **Step 4: Run the tests; expect pass.**
- [ ] **Step 5: Commit** `#258 kit bots --check and token: the App's permissions against its jobs, and a token`.

### Task 5: `run`

**Files:**

- Modify: `kit/modules/bash/bots.sh`, `kit/bin/kit-bots`
- Test: `tests/bots.bats`

**Interfaces:**

- Produces: `bots:run <bot-json> <command>…` runs the command and returns its status. It makes a session directory,
  mode 700, under `${XDG_RUNTIME_DIR:-${TMPDIR:-/tmp}}`, holding `token.json`, `bin/gh`, and
  `bin/git-credential-kit-bot`. `bin/gh` re-mints through `kit bots --bot=<name> token` when within 600 seconds of
  expiry, then execs the `gh` that was first on `PATH` with `GH_TOKEN` set. The credential helper answers `get` for
  the forge's host with `username=x-access-token` and the token. git sees it through `GIT_CONFIG_COUNT` with two
  entries for `credential.https://<host>.helper`: an empty one, which clears the person's helpers, then the helper.
  A `trap` on EXIT, INT, and TERM revokes the token and removes the directory.

- [ ] **Step 1: Write the failing tests:**
  - `"run gives the command GH_TOKEN, KIT_BOT, KIT_BOT_LOGIN, and KIT_BOT_FOR"`: the command prints them.
  - `"run's gh mints a new token when the old is within ten minutes of expiry"`: a first token expiring in 300
    seconds; the command calls `gh api user` and the fake gh sees the second token.
  - `"git push under run asks the helper, which answers with the token"`: `git credential fill` in the command.
  - `"run passes the command's exit status through, revokes the token, and removes the directory"`: `false`, then
    the curl calls end with the `DELETE`, and the directory is gone.
  - `"run revokes when the command is interrupted"`: the command kills its parent shell with TERM.
- [ ] **Step 2: Run them; expect failures.**
- [ ] **Step 3: Implement** `bots:run`, reading `KIT_BOT_FOR` with `forge:me` before the switch.
- [ ] **Step 4: Run the tests; expect pass.**
- [ ] **Step 5: Commit** `#258 kit bots run: a command acts as the bot for as long as it runs`.

### Task 6: kit tells the bot from its person

**Files:**

- Modify: `kit/modules/bash/forge.sh` (`forge:me` prints `KIT_BOT_LOGIN` when set; new `forge:principal` prints
  `KIT_BOT_FOR` when set, else `forge:me`), `kit/modules/bash/tracker/forge.sh` (`tracker/forge:me` calls
  `forge:principal`), `kit/modules/bash/flow.sh:682` and `kit/bin/kit-flow:364` (`--mine` and `--me` read
  `forge:principal`), `kit/modules/bash/commit.sh` (`commit:commit` adds the trailer with `git interpret-trailers
  --in-place --if-exists addIfDifferent --trailer "Agent: ${KIT_BOT_LOGIN}"` when `KIT_BOT_LOGIN` is set)
- Test: `tests/bots.bats`, `tests/commit.bats`, `tests/flow.bats`

**Interfaces:**

- Produces: `forge:principal` prints the person behind the caller.

`issue:join`'s author check keeps `forge:me`: the bot joins only a change the bot opened.

- [ ] **Step 1: Write the failing tests:**
  - `"forge:me is the bot's login under KIT_BOT_LOGIN, and forge:principal the person"`.
  - `"kit assign with no --to assigns the person under a bot"`, in `tests/bots.bats`, with the fake gh's route for
    `issue edit 7 --add-assignee ada`.
  - `"kit commit under a bot adds one Agent trailer"`, in `tests/commit.bats`, and none without one.
  - `"kit flow --me prints the person under a bot"`, in `tests/flow.bats`.
- [ ] **Step 2: Run them; expect failures.**
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run `tests/bots.bats tests/commit.bats tests/flow.bats tests/tracker.bats tests/work.bats`.**
- [ ] **Step 5: Commit** `#258 kit: under a bot, me is the bot and the principal its person; commits name the Agent`.

### Task 7: The Welcome check

**Files:**

- Modify: `kit/modules/bash/welcome.sh` (`welcome:plan-folder` adds `bot` where the primary remote's forge is
  GitHub; `welcome:judge` dispatches it to a new `welcome:bot`)
- Test: `tests/welcome.bats`

**Interfaces:**

- Consumes: `bots:choose`, `bots:key`, `bots:check`.
- Produces: check `bot`, step `setup`:
  - no bot for the host: holds, line `no bot acts for you here; set one up to hand it issues`, fix
    `kit bots --add=<name> --app=<id> --key=<file>`, `runs: false`, more the design's GitHub App page,
    `https://docs.github.com/en/apps/creating-github-apps`.
  - key missing: fails, fix the same `--add`, `runs: false`.
  - not reached, or a job lacks a permission: fails, fix `kit bots --check`.
  - otherwise: holds, `<name>[bot] acts for you here, for <jobs>`.

- [ ] **Step 1: Write the failing tests**, one per line above.
- [ ] **Step 2: Run them; expect failures.**
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run `tests/welcome.bats`; expect pass.**
- [ ] **Step 5: Commit** `#258 kit welcome: a bot check, offering one where none acts for you`.

### Task 8: Docs, the gate, and `empowerite-jig`

**Files:**

- Modify: `docs/features.md` (a Bots section after Workspaces), `docs/integrations/gh.md` (the App's token, the
  keychain), `docs/superpowers/specs/2026-11-12-bots-design.md` (Status: step 1 built)

- [ ] **Step 1: Write the docs.** `kit lint` passes over them.
- [ ] **Step 2: Run `kit gate`; expect it to pass.** Report any failure that is not this plan's, with its output.
- [ ] **Step 3: Prove it against the real App.** Expected: `--add` stores the key and prints what the bot can do;
  `--check`, in a checkout of empowerite/jig, says it is reached and names `actions: write` as extra; `run` prints
  `empowerite/jig`; afterwards the token is revoked, and `curl` with it answers 401.

  ```bash
  kit bots --add=empowerite-jig --app=4906610 --key=keychain:jig-app/4906610
  kit bots --check --bot=empowerite-jig
  kit bots --bot=empowerite-jig run gh api /installation/repositories --jq '.repositories[].full_name'
  ```

- [ ] **Step 4: Commit** `#258 docs: kit bots`.
