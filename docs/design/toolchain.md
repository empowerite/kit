# kit: the toolchain (design, 2026-11-21)

Status: the owner approves this design by merging the pull request that brings it. Nothing is built from it until
then.

## Intent

The charter's fifth purpose: a repository pins the tools its work runs with, each locked by version and by digest for
every platform, and kit puts them on the `PATH`, so a desk and CI run the same tools. A repository that has a tool
manager keeps it, and kit works with it.

Nothing holds that today. kit's own gate runs Go alone. `go.mod` pins Go (`toolchain go1.27.4`), and Go fetches
itself and checks what it fetched against its own checksum database. The scripts are kept free of shellcheck's
findings by habit, at a desk, with whatever shellcheck the desk has, and CI runs no shellcheck. empowerite's other
repositories that have a tool manager each use mise with a lock, and some entries in those locks have a version and
no digest: in infrastructure, `npm:bats`, `npm:markdownlint-cli2`, `npm:wrangler`, and `ubi:terror/just-lsp`; in
enchiridion, three `npm:` tools and two `pipx:` tools. Nothing checks that a lock covers every platform, or that a
desk runs what CI runs.

This design says where the pins and their lock live, how kit makes the lock and fetches what it names, how a tool
reaches the `PATH`, how kit works with mise, what kit says when a tool manager is used in a way that breaks
correctness, and what kit does in a repository that has not adopted it. Its slice makes this repository the first
user: its gate runs shellcheck, pinned by kit, at a desk and in CI.

## What the owner decided

These came from the owner in conversation on 2026-11-21, and the design holds them as given:

1. kit never installs mise.
2. kit runs in a repository that has not adopted kit. It infers the forge and the tool settings from what is there,
   and does less, since there is no `kit.yml`. It may place files under `.kit/`, so long as nothing is checked in.
3. In a repository that has adopted kit, kit is the tool manager where no other one is in use.
4. Where kit finds a tool manager used in a way that does not deliver correctness, it tells the person fully how to
   resolve it.
5. Where there is nothing to work from, kit says so and offers two fixes: to adopt kit, or to set up the lock in the
   tool manager already there.
6. `.kit/` is gitignored selectively, as in the proof of concept: kit names each path it placed, and nothing is
   excluded wholesale. What kit placed is read-only, a strong hint that kit owns it.
7. mise is the one tool manager kit works with for now.
8. kit knows where a tool's releases are from a list of tools it carries, as data. aqua's registry may become a
   later source of entries for that list.
9. shellcheck is the first tool this repository pins, and lychee follows close behind.
10. kit supports direnv and `.envrc`.
11. This milestone builds what the toolchain needs from the foundations, in its final form, as far as this slice
    uses it: plan, approve, and apply; data files with their schemas; and presence.

Everything else below is a proposal, and the list under [Calls made here](#calls-made-here) names the larger ones.

## Goals

1. **The same tools.** A desk and CI run the same bytes. Where kit is the tool manager, kit checks each tool against
   the lock's digest every time before it runs. Where mise is, kit checks mise's lock, and says that mise checks
   the bytes when it installs them, since `mise.lock` holds the digest of a download and not of the program.
2. **Every platform.** A lock covers macOS and Linux, each on arm64 and x86-64, the four platforms kit builds for. A
   lock that lacks a platform or a digest fails the check.
3. **What is there.** Where mise is in use, kit works through it and changes none of its files. Where no tool manager
   is in use, kit is one. A repository that has not adopted kit gains no change that git sees.
4. **Nothing by hand.** Adding a tool, making the lock, fetching, and reaching the `PATH` are each one command.
5. **The fix, in full.** Each finding says what is wrong, why it breaks "the same tools", and the fix, as a command
   to copy where there is one.

## Not in this design

- Tool managers other than mise: asdf, aqua, Nix, devbox, a `Brewfile`, and the rest. Each gets an issue of its own.
  Until then kit names one it finds and says that it does not read it yet.
- aqua's registry as a source of entries for kit's list.
- A language's own dependencies: `go.sum`, `package-lock.json`, and the like. Their own tools lock them.
- Signatures and attestations of a release, beyond digests.
- Windows.
- Trust in full, which is the charter's eleventh purpose. This design takes from it only what running a tool needs.
- The foundations' machinery beyond what this slice uses: other operations, the organization's and the person's
  layers, and `kit explain`.
- Pins kept up to date by a bot. A new version is a change like any other, made with the command that adds a tool.

## What must not happen

- kit installs mise, or any other tool manager.
- Where kit is the tool manager, a tool runs whose bytes kit did not check against the lock.
- A repository that has not adopted kit gains a change that git sees.
- kit edits a file of another tool manager's, such as `mise.toml` or `mise.lock`.
- A check passes a lock that lacks a platform, or an entry that lacks a digest.
- A test in the pull request's gate reaches the network or starts a process (from the foundations).
- The gate takes kit's word that a tool ran.

## Constraints

- The foundations hold: Go 1.27, one binary, macOS and Linux, and only the adapters start a process, open a
  connection, or touch a file.
- The four platforms: `darwin/arm64`, `darwin/amd64`, `linux/arm64`, `linux/amd64`.
- kit has no release yet. This repository builds kit from its own source, with the Go that `go.mod` pins, and the
  gate uses the kit it built.
- Scripts are bash, and run under bash 3.2.
- The gate stays within its 3 minutes.

## The design

### The pins and the lock

A repository that has adopted kit has a `kit.yml` at its root. This design gives it one key, `tools:`, a tool's
name and its version:

```yaml
tools:
  shellcheck: 0.11.0
```

`kit.lock`, beside it, is the lock. Only kit writes it, and it is checked in, since CI must read it. For each tool it
holds the version, the entry of kit's list it was made from, and for each of the four platforms: the address of the
file kit downloaded, that file's sha256, the program's path inside it, and the program's own sha256.

The check holds `kit.yml` and `kit.lock` to each other. Every tool in `kit.yml` is in the lock at that version, with
all four platforms and both digests, and the lock holds nothing else. A digest changed by hand fails when kit next
fetches, since the file it downloads no longer matches.

### kit's list of tools

kit carries a list of the tools it knows how to fetch, as a data file inside the binary, like its other defaults.
`kit data show tools` prints it. An entry says where the tool's releases are, which file of a release is for each
platform, what kind of archive that file is, where the program is inside it, and where the tool publishes its own
checksums, if it does:

```yaml
shellcheck:
  github: koalaman/shellcheck
  tag: v{version}
  files:
    darwin/arm64: shellcheck-v{version}.darwin.aarch64.tar.gz
    darwin/amd64: shellcheck-v{version}.darwin.x86_64.tar.gz
    linux/arm64: shellcheck-v{version}.linux.aarch64.tar.gz
    linux/amd64: shellcheck-v{version}.linux.x86_64.tar.gz
  program: shellcheck-v{version}/shellcheck
```

lychee's entry also names the `.sha256` file that lychee publishes beside each archive.

- **The archives kit reads are `.tar.gz` and `.zip`**, which Go's own library opens. shellcheck offers `.tar.gz`
  beside `.tar.xz`, and lychee ships `.tar.gz` only, so kit needs no library outside Go's.
- **A repository can add an entry of its own.** Under `tools:` in `kit.yml`, a tool's value is its version, or an
  entry in the same shape with a `version:` of its own. The repository's entry wins over kit's, as the foundations'
  layers do. That way a tool kit does not know needs no new release of kit.
- **aqua's registry can later become one more source of entries.** Nothing in the lock or the commands then changes.

### Making the lock

- `kit tools add <name>[@<version>]` writes the tool into `kit.yml` and its entry into `kit.lock`. With no version it
  takes the tool's newest release. A new version of a tool already pinned is the same command.
- `kit tools lock` makes `kit.lock` again from `kit.yml`.
- For each platform, kit downloads the file and takes its sha256. That digest must agree with the digest GitHub keeps
  for the release's file, and with the tool's own published checksum where there is one. Any disagreement refuses,
  and the refusal names the file and the digests that differ.
- **A digest that nothing confirms is refused.** GitHub keeps a digest only for files uploaded since mid-2025, and
  not every tool publishes checksums. Where neither is there, kit refuses, and says that `--unconfirmed` pins the
  file on kit's own download alone. The lock records, for each file, what confirmed its digest, and `kit tools`
  shows a file that nothing confirmed.
- **An entry names a GitHub release, for now.** A tool released elsewhere, such as at `releases.hashicorp.com`, is a
  kind of entry of its own, and an issue of its own.
- Each one is an act with a plan, as the foundations' plan, approve, and apply has it: kit shows which lines of
  `kit.yml` and `kit.lock` change, and the files it will download. `--plan` prints the plan and stops, and with
  `--json` prints it as data. Nothing is written until it is approved. With no bot there is one lane, so the act is
  the person's, and the approval is the person's presence, bound to the plan's hash. An agent can ask for the plan,
  and cannot apply it.

### Fetching, and the cache

- **The cache is the person's, one for the machine.** kit keeps each tool there in a folder named for the digest of
  the file it came from, with the program extracted and read-only. Every repository and worktree that pins the same
  file shares one copy. A deleted cache only means kit fetches again.
- **`kit tools fetch`** fetches what the lock names for this platform. Every command that runs a tool fetches first.
- **Fetching changes nothing that git sees,** and puts on the machine only what a trusted lock already names. So it
  needs the clone's trust and not presence, and it runs in CI with nobody there. Changing the pins is the act that
  needs presence.
- **Before a tool runs,** kit checks the program's sha256 against the lock, every time. Hashing a program of a few
  megabytes takes milliseconds.
- **What kit places under `.kit/`** is a link for each tool, in `.kit/bin/`, and the environment file that direnv
  loads, `.kit/env`. Each is named in `.kit/.gitignore`, which kit writes and which names itself too. The rest of
  `.kit/` is the repository's own.
- **A link in `.kit/bin/` leads to kit, not to the program.** Run by a tool's name, kit checks that program's digest
  against the lock, and then runs it in its own place, with the same arguments. So a tool run from a shell, an
  editor, or direnv's `PATH` is checked as surely as one run by `kit exec`. Each run costs kit's start and the hash,
  a few milliseconds.

### The `PATH`

- **`kit exec -- <command>`** runs a command with the pinned tools first on the `PATH`, once they are fetched and
  checked.
- **`kit env`** prints the environment for a shell: `kit env --shell=bash` and `--shell=zsh` print `export` lines,
  `--json` prints them as data.
- **direnv.** `kit env --shell=direnv` prints direnv's own words, `PATH_add` and `watch_file`, with `kit.yml` and
  `kit.lock` watched, so direnv loads the environment again when a pin changes. Where a repository that has adopted
  kit has an `.envrc`, `kit tools add` puts the one line that calls it into `.envrc`, as a region of kit's, which
  kit can take out again. kit makes no `.envrc` where there is none, since whether to use direnv is the repository's
  choice.
- **kit reads direnv's state, and never takes it as consent.** `direnv status --json` says whether direnv has allowed
  the `.envrc`. kit uses that to say where things stand, such as "direnv has not allowed this `.envrc`: `kit trust`
  covers it".

### Trust

kit fetches nothing and runs no tool in a clone until the person trusts it.

- **What the trust covers:** `kit.yml`'s tools and `kit.lock`; for mise, `mise.toml` and `mise.lock`; and `.envrc`
  where there is one. kit records the trust for the clone and all of its worktrees, by a digest of what it covers,
  as the proof of concept did. It holds several digests at once, so each worktree's own pins can be trusted. When
  any of those files changes, as when a pull brings a new pin, kit asks again, and shows what changed: which tool,
  which version, and which address.
- **One act.** `kit trust` shows all of it, and asks once. To trust is a person's decision, so it takes the person's
  presence. On a yes kit records its own trust and, where there is an `.envrc`, runs `direnv allow` for the person,
  while the person is there.
- **mise trusts by itself what `mise exec` runs.** Outside its paranoid mode, `mise exec` trusts the repository's
  configuration without asking, and mise's own trust is of a file's path, not its contents. So kit's trust, which
  covers `mise.toml` and `mise.lock` by their digest, is what stands in front of mise, and kit asks it before
  `kit exec` runs `mise exec`.
- **Why direnv's approval is not taken as kit's.** direnv approves the bytes of `.envrc`. When `.envrc` calls kit,
  what then runs is named by `kit.yml` and `kit.lock`, which direnv does not see: a change to a pin leaves direnv's
  approval standing.
- **A script of the repository's that a person runs,** such as `bin/gate`, passes `--trust=checkout`. To run the
  repository's script is the person's own act, at the command line, as is passing the option. It lasts for that one
  run, and kit records nothing. There is no environment variable for it, so no file that sets one can grant trust.
- **What trust does not hold.** An agent with a shell can pass `--trust=checkout` too, as it can run any program.
  Trust keeps a clone from running what its person never saw; it does not hold an agent that has a shell, which the
  foundations' "What kit cannot hold" already says.

### What this milestone builds of the foundations

The owner chose to build these now, in their final form, as far as this slice uses them, so that the slice of
manual work builds on them and not on something it must replace:

- **Plan, approve, apply,** with the operations this slice needs and no more: write a file of the repository's,
  place a file under `.kit/`, download into the cache, and record a trust. Each can say what it would do before it
  does it. The plan has a hash, the approval binds to it, and apply refuses when the facts changed since the plan.
- **Data files with their schemas:** kit's list of tools, inside the binary, and the shapes of `kit.yml`'s
  `tools:` and of `kit.lock`. kit checks a file when it loads it and names the file and the line of what is wrong.
  `kit data show <name>` prints one of kit's own. The layers are kit's defaults and the repository's `kit.yml`;
  the organization's and the person's layers come with the slice that first needs them.
- **Presence:** Touch ID, or the account's password, on macOS, as the foundations propose. Where a machine has no
  such proof, as on Linux for now, kit says so and marks the approval as weaker. No agent can give it.
- **What kit prints, as data:** `--json` for every command here, each with a schema and a version.
- **The commands' data:** each command's words, options, and help come from data, as the foundations ask, for the
  commands this slice adds.

### Working with mise

kit finds mise where the repository has mise's configuration. It reads `mise.toml` and `mise.lock` itself, as the
checkout holds them, since that is what CI sees.

- **It lists** each tool mise pins, with its version and whether the lock holds a digest for each of the four
  platforms.
- **It runs** a tool through the mise on the machine: `kit exec -- <command>` runs `mise exec -- <command>`. Where no
  mise is installed, kit says so, says that kit does not install it, and names mise's own way to install it.
- **It checks**, and each finding names its fix:
  - mise keeps no lock: turn it on in `mise.toml` and run `mise lock`.
  - the lock is not checked in.
  - a tool in `mise.toml` has no entry in `mise.lock`: run `mise lock`.
  - an entry lacks one of the four platforms: `mise lock` with those platforms.
  - an entry has a version and no digest, as mise's `npm:`, `pipx:`, and `ubi:` entries do. mise installs such a
    tool by its version alone, and a package's own dependencies are chosen when it is installed, so two machines can
    run different code. The fix names a source that keeps a digest, where the tool has one, such as mise's `aqua:`
    or `github:` for a tool that publishes programs. For a package of a language, it is the language's own lock, in
    the repository, such as `package.json` with `package-lock.json`.
- **It changes nothing of mise's.** kit prints each fix. It does not edit `mise.toml` or `mise.lock`.
- **It informs, and does not block.** `kit tools check` fails on a finding. `kit exec` still runs the tool through
  mise, and prints the findings first, on standard error, with the command that lists them in full. Where mise is the
  tool manager, the fix is the repository's to make.
- **One tool manager at a time.** Where mise is in use, kit is not one: a `tools:` key in `kit.yml` then fails the
  check, and the fix is to pin those tools in mise.

### A repository that has not adopted kit

There is no `kit.yml`.

- **With mise,** kit reads, lists, checks, and runs through mise as above. It places nothing for tools.
- **With another tool manager,** kit names it, says that it does not read it yet, and runs nothing through it.
- **With none,** kit says that it has nothing to work from, and offers the two fixes: adopt kit, which
  `kit tools add` does by writing `kit.yml`; or set up the lock in the tool manager already there.

### Go

`go.mod`'s `toolchain` line is Go's pin, and Go checks what it fetches against its own checksum database. `kit tools`
lists Go as pinned there, so the list of what a repository's work runs with is whole, and kit pins Go nowhere else.
That is also how this repository starts: Go builds kit, and kit fetches the rest.

### What kit prints

- `kit tools` prints each tool: where it is pinned (kit, mise, or `go.mod`), its version, and whether it is fetched
  and checked on this machine.
- `kit tools check` prints the findings, each with what is wrong, why, and the fix. It ends in failure when there is
  one. For example:

  ```text
  mise.lock: npm:markdownlint-cli2 0.23.1 has a version and no digest
    why  mise installs it from npm by version, and npm chooses its dependencies when it installs,
         so a desk and CI can run different code
    fix  pin it in package.json, with package-lock.json checked in, and run it with npx
  ```

- Each prints the same facts as JSON with `--json`, with a schema and a version, as the foundations ask.

### This repository, the first user

- `kit.yml` pins shellcheck, and `kit.lock` locks it.
- `bin/gate` builds kit, runs `kit tools check`, and fetches. It then runs shellcheck over every script of the
  repository, and refuses on a finding. CONTRIBUTING's list of what the gate refuses gains that line.
- **The gate does not take kit's word.** `bin/gate` runs shellcheck at the path kit names, after the gate's judge,
  `tools/gate`, has checked that program's sha256 against `kit.lock` itself. A change to kit's own code cannot then
  make shellcheck seem to run when it did not.
- CI needs no step of its own for shellcheck. It keeps kit's cache between runs, keyed on `kit.lock`. The runner's
  own shellcheck is never the one that runs: GitHub's Ubuntu 24.04 image has shellcheck 0.9.0, two releases behind.
- lychee follows as an issue of its own in this milestone, once the slice is in.

### The Go code

- A package `toolchain` holds kit's list of tools, the pins, the lock, the check, and the plan for a change to them.
  It reads only through ports. It is not called `tools`, since `tools/` already holds the gate's judges, and one word
  names one thing.
- The ports gain what the toolchain needs: downloads, the records of a release, and running a program.
  `adapters/os` downloads and runs, `adapters/github` reads a release's records, and `adapters/fake` holds all of
  them in memory.
- The table of layers, `tools/layers`, gains `toolchain`, which may import `model`, `data`, `ports`, and `plan`, and
  which `doc` and `cli` may import. The import rule holds it as it holds the rest.

### Testing

- **In the pull request's tier,** in memory: making a lock from a fake release; each disagreement of digests (with
  GitHub's, with a published checksum, with the lock); a lock that lacks a platform; mise's files as data, among them
  copies of the shapes of infrastructure's and enchiridion's locks; `kit.yml` and mise both pinning tools; no tool
  manager; another tool manager; the trust's digest changing when a pin changes; and `.kit/.gitignore`.
- **In the nightly tier,** for real, on macOS and Linux: fetching shellcheck and lychee, and running each; and a
  repository with mise in place, in which kit lists mise's tools, runs one through `mise exec`, and leaves
  `git status` clean.

### Done when

As #5 has it: this design is approved; this repository's gate runs with pinned tools at a desk and in CI; a
repository with another tool manager in place works unchanged; and nothing it needed was made by hand.

- **"Works unchanged"** means that kit lists, checks, and runs in such a repository, and that `git status` is clean
  after. Findings with their fixes are kit working. Fixing infrastructure's and enchiridion's locks is their own
  work, in their own repositories, and not this milestone's.
- **kit at another desk, before a release.** kit has no release yet, so another repository's desk gets kit with
  `go install github.com/empowerite/kit/cmd/kit@<commit>`, which Go checks against its own checksum database. The
  release is the charter's twelfth purpose, and its own design.

## Approaches considered

| Question | Picked | Dropped, and why |
| --- | --- | --- |
| Who fetches where no tool manager is in use | kit | kit installs mise and drives it: the owner ruled it out. It also picks a tool for the repository, and is a second program to fetch and check first. |
| Where kit learns a tool's releases | kit's own list, as data, with aqua's registry a later source | aqua's registry now: its format is large and keeps changing, and reading part of it leaves known wrong cases. Addresses in `kit.yml`: every repository writes the same ones by hand. |
| direnv's trust | one act in kit, which also runs `direnv allow` | Taking direnv's approval as kit's: it covers the bytes of `.envrc`, and not `kit.yml` or the lock. |
| Tool managers, for now | mise | mise with asdf, aqua, and Nix: each reads pins and locks its own way, and asdf keeps no lock at all. |
| Keeping `.kit/` out of git | `.kit/.gitignore`, naming each path kit placed | A wholesale exclusion, or `.git/info/exclude`: the owner ruled it out. |

## Calls made here

The owner did not decide these; they are proposals to check:

1. kit installs no tool manager of any kind, not only mise.
2. The lock is `kit.lock`, beside `kit.yml`, and holds two digests for each platform: the downloaded file's and the
   program's.
3. A digest must agree with GitHub's digest of the release's file, and with the tool's published checksum where
   there is one.
4. kit reads `.tar.gz` and `.zip` alone, so it needs no library outside Go's.
5. A repository adds an entry to kit's list in `kit.yml`, and its entry wins.
6. The commands: `kit tools`, `kit tools add`, `kit tools lock`, `kit tools fetch`, `kit tools check`, `kit exec`,
   `kit env`, `kit trust`, and `kit data show tools`.
7. The cache is one for the machine, a folder for each digest, read-only.
8. kit checks a program's digest each time before it runs it.
9. `kit tools add` puts kit's line into an `.envrc` that exists, as a region, and makes no `.envrc` where there is
   none.
10. To trust is a person's decision and takes presence. The trust is recorded for the clone and its worktrees with
    a digest of what it covers, and a change to any of it asks again.
11. A script of the repository's passes `--trust=checkout`, and no environment variable grants trust.
12. kit reads mise's files itself and runs mise only to run a tool.
13. An `npm:`, `pipx:`, or `ubi:` entry in `mise.lock` fails the check. In infrastructure that is four entries, and in
    enchiridion five.
14. Where mise is in use, a `tools:` key in `kit.yml` fails the check.
15. `go.mod`'s `toolchain` line is Go's pin, and kit pins Go nowhere else.
16. `tools/gate` checks shellcheck's digest itself, so the gate does not take kit's word.
17. lychee is an issue of its own in this milestone, after the slice.
18. A link in `.kit/bin/` leads to kit, which checks the program's digest and runs it, so a tool run outside kit is
    checked too.
19. A digest that neither GitHub nor the tool confirms is refused, unless `--unconfirmed` is given, and the lock
    records what confirmed each one.
20. An entry names a GitHub release, for now.
21. Under mise, `kit exec` runs and prints the findings first; only `kit tools check` fails on them.
22. Trust is held by digest, several at once, and a pull that brings a new pin asks again.
23. `--trust=checkout` lasts one run and is recorded nowhere.
24. "Works unchanged" is met by kit listing, checking, and running in such a repository with `git status` clean,
    findings and all; other repositories' locks are their own work.
25. Before a release, another desk gets kit with `go install` at a commit.
26. The package is `toolchain`, not `tools`, and its row in the table of layers is the one given above.

## Left to the builder

- How kit asks for Touch ID. Apple's framework for it is reached through cgo, and `bin/build` builds kit without cgo
  for all four platforms, so the builder finds a way that keeps one build for every platform, or else builds kit
  for macOS on a Mac, in CI as at a desk.
- The exact shape of `tools:` in `kit.yml`, of `kit.lock`, and of an entry of kit's list, and their schemas.
- Which of mise's configuration files kit reads. mise has a setting, `locked`; the builder confirms in mise's own
  documentation what it refuses, and where it makes CI refuse a tool its lock does not name, kit checks for it.
- Where the cache is on each system.
- For lychee, which Linux file to take, the one built for musl or for glibc.
- The words of the region kit puts in `.envrc`.
- The key under which CI keeps kit's cache.
- For lychee's issue: whether the gate checks links that leave the repository, since the gate reaches no network
  in its tests.
