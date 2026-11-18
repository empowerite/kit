# kit: the foundations (design, 2026-11-17)

Status: the owner approves this design by merging the pull request that brings it. Nothing is built from it until
then.

## Intent

The proof of concept built usable tools and proved little; see [the retrospective](poc-retrospective.md). Three of
its hypotheses failed: one screen that shows
where the work is and what to do next, verification that stays fast, and agents that build a coherent system when many
work at once. The three share a cause. Nothing in the proof of concept says, in one place, what a piece of work is,
what state it is in, and what can happen to it next. The screens, the tests, and the agents each worked that out again.

The new kit starts from that one statement. Its purpose does not change, and [the charter](../charter.md) says all
of it: rules
written once for any harness, the rules that govern a file, templates that make a file and check one, the checks, a
toolchain, the path of work, where the work stands, every role's own questions, many repositories as one, evidence,
identity and trust, a release and not a copy, standing a repository up, and a newcomer's start.

This design covers what all of that stands on, and two of those purposes, the path of work and where it stands, in
two slices. It does not design the whole of kit. Each other purpose gets a design and a slice of its own, in the
order the owner sets.

## What the owner decided

These came from the owner in conversation, and the design holds them as given:

1. The new kit is a new repository. The proof of concept becomes `kit-poc`, a reference, and nothing is ported
   unless a slice needs it.
2. kit's commands are written in Go, structured carefully and methodically.
3. The terminal UI is in from the first slice, because it is how the owner feels how kit works.
4. The first screen is "your move": where the work is, where the person acts, and what to expect next.
5. Work begins with one issue. Another issue joins that work only as a sub-issue of it that the same commit
   addresses, as work discovered along the way and done with it so that no debt is left, or as a blocker rolled
   into it.
6. From "your move", a person can drill down to any level of detail, and get from any fact to the forge or whatever
   else is its source of truth.
7. kit stays driven by data: the schemas of its commands and of what they print, the development lifecycle, and
   the like are data, not Go.
8. An act travels one of two lanes, a person's or a bot's, and a bot never acts as a person. One bot can serve many
   people; a bot for each person is not needed.
9. A person's act needs proof that the person is there.
10. Until kit has the tools that make a bot, kit serves the work a person does, and no agent acts through it.
11. A bot's session runs for ten hours and more. A token that ends after an hour with nothing to renew it is not
    acceptable.
12. Not every review will be a person's, so kit does not fix which acts are a person's decisions.
13. Nothing is made by hand that kit will later make. Whoever uses kit has no Terraform repository of empowerite's,
    so kit itself makes what it needs at the forge, and the person's part is simple, natural, and painless.
14. kit is not opinionated about tools or processes, except where they affect correctness. It works with a forge, a
    tracker, a tool manager, and a harness that are already in place.
15. The history of the default branch is a matter of correctness, not of taste: it is linear and signed, a change
    lands on it as one squashed commit, and it is never rewritten.
16. Gradual reveal, consistency, and a path from novice to super user are tenets of the approach.

Everything else below is a proposal, and the list under [Calls made here](#calls-made-here) names the larger ones.

## Goals

1. **One document.** For any piece of work, kit says its state, whose move it is and why, which acts are possible
   now, and what follows each.
2. **One gesture for every change.** Choose a target, see the plan, approve it, apply it, read the result. The
   gesture is the same in the terminal UI, on the command line, and for an agent.
3. **Drill down, and back to the source.** Every item opens into its parts, and every fact names where it lives.
4. **Driven by data.** A repository or an organization changes the lifecycle, the entities, the commands' schemas,
   the policy, and the views without compiling Go.
5. **Verification in minutes.** A pull request's gate takes 3 minutes or less, the loop on save takes seconds, and
   the gate itself fails when either is exceeded.
6. **A process agents can fan out under.** Many agents work at once without the design drifting.
7. **No bot acts as a person.** A person's decision is the person's in every record, and an agent's work is a bot's.
8. **Nothing by hand.** kit makes what it needs, in one command, and asks a person only for what the forge insists
   a person do.

## Not in this design

Each of these is real work for kit, and each gets its own design later:

- A broker that mints tokens for a team, and enrollment through an identity provider. These slices build the size
  for one person, who is their own broker.
- Reviews by a bot, and how an organization lets one stand where a person's stood.
- Standing up more than kit itself needs, such as an organization's settings and the secrets of other tools. That
  builds on the same plan mechanism and the same `kit setup`.
- Rules written once for any harness: the kinds of file, the rules by kind, `kit context`, the adapters that point
  each harness at the same rules, and the packages that deliver them.
- Templates that make a file and check one.
- The checks that hold a repository to its rules, and the release that a repository pins.
- The toolchain: the tools a repository pins, locked for every platform, and on the `PATH`.
- Forges and trackers other than GitHub. The first slice defines their interfaces and builds GitHub alone.
- Providers that run as programs of their own, so that how kit works is extended without a fork.
- A workspace of several repositories, and the other screens of the proof of concept's UI, its menus, and its
  presets.
- The newcomer's guide and the lessons. The tenets ask for them: gradual reveal, and a path from novice to super
  user.
- Roles other than the developer's, and the evidence an auditor reads.
- Releases, and the Homebrew tap.

## What must not happen

- Go code that names a state or an act of the default lifecycle. Then the lifecycle is no longer data.
- A screen that holds a fact or an act that kit's commands do not print.
- A second branch of a thousand commits, or a tracker nobody can read.
- A test in the pull request's gate that starts a process, reads the network, or reads the person's own tools.
- Polish of the screen's look before the first slice's scenario works.
- A bot that acts as a person, or a person's act that runs with nobody there.
- A thing made by hand with a promise that kit will make it later. That is the worst kind of debt.

## Constraints

- Go 1.27, one module, one binary, `kit`, which holds the terminal UI too. It runs on macOS and Linux.
- The forge and git hold the truth. kit keeps nothing about the work that it could not read again from them; what it
  keeps is a cache.
- Rules carried from the proof of concept, which the owner set there:
  - In the default lifecycle, to merge is a person's decision. An agent works in a worktree, never on the person's
    own checkout.
  - An option with a value is written `--option=value`.
  - In the terminal UI every act has a path of plain left clicks, as well as a key. An act that cannot run now is
    drawn dim, with the reason, and is not left out. No circled glyphs.
  - No known wrong case is left for later. Go code is held to 100% test coverage.

## The model

### Entities, addresses, and sources

Everything kit knows is an entity: an issue, a work, a change (a pull request), a check, a job, a review, a commit, and
so on. An entity has a kind, an id, fields, relations to other entities, and a source.

- **Address.** Every entity has one, made of kinds and ids: `issue/31`, `work/31`, `change/14/check/lint/job/2`.
  `kit show <address>` prints the entity, and `kit show <address> --json` prints it as data.
- **Relations** are what a person drills into. A work has its issues, its change, and its commits; a change has its
  checks and its reviews; a check has its jobs; a job has its steps; a step has its log.
- **Source.** Every entity says where its facts live (an address at the forge, a git ref, a file's path), which
  system answered, and when kit read it. From any entity, one act opens the source itself: the forge's page in the
  browser, or the file in the pager.

The entities, their fields, and their relations are data files with a schema, as the proof of concept's entity model
began to be.

### A work

A work is the unit kit moves: one branch, one worktree, and one change. The change lands on the default branch as
one commit.

- A work begins with exactly one issue, its lead. `kit start <issue>` makes it.
- Another issue joins a work only through `kit attach <issue> --as=<relation>`, and kit refuses an attach without a
  relation. The relations are:
  - `subissue`: a sub-issue of the lead that the same change addresses. kit refuses it for an issue the forge does
    not hold as a sub-issue of the lead.
  - `discovered`: work found along the way and done with the lead, so that no debt is left.
  - `blocker`: something that blocks the lead and is rolled into it.
- The change closes the lead and every issue attached. Each attached issue shows under the work with its relation.
- To start several issues at once is to start several works, one for each.
- A change that kit did not start, such as one opened by hand or by a bot, is still a work, and `kit next` shows it
  and counts it. Its lead is the issue its branch's name starts with, where the change closes that issue. A change
  that closes no issue is a work with no lead, marked so, and `kit attach` gives it one. An issue it closes with no
  relation recorded shows as attached and `unsorted`, until a person gives it a relation. Whether the gate refuses a
  change with no lead is policy, and is not in the first slice.
- Where the attachments are recorded is the builder's to choose, within the constraint above: they are read back
  from the forge and git, and a person at the forge can see them.

The list of relations is data, so an organization can add one or take one away.

### The lifecycle is data

One file says how a kind of work moves. It holds:

- **States**, each with the rule that derives it from the facts. A state is never stored; it is computed on every
  read.
- **Court**: who holds the next move (`you`, `others`, `nobody`, and later `bot`), and why, in a few words. The rules
  are tried in order and the first that holds wins. The proof of concept's
  [bots design](https://github.com/empowerite/kit-poc/blob/1-the-first-release-v0-1-0/docs/superpowers/specs/2026-11-12-bots-design.md)
  has a table of such rules to start from.
- **Acts**, each with: its name; the kind of entity it acts on; its lane, a decision or work; the rule that says when
  it can run; the words that say why it cannot; what it does, as a list of operations; what follows, in words; and
  the states in which it is the default.

An illustration of the shape, not its final form:

```yaml
relations: [subissue, discovered, blocker]
states:
  - name: backlog
    when: "!has(work)"
  - name: started
    when: "has(work) && (!has(change) || change.draft)"
court:
  - when: "change.checks.exists(c, c.state == 'failed')"
    court: author
    why: "CI failed"
acts:
  - name: start
    on: issue
    lane: work
    when: "state == 'backlog'"
    why_not: "work on it has started"
    does:
      - assign: {to: caller}
      - branch: {from: default}
      - worktree: {}
    then: "it moves to started; nothing is pushed until you submit"
    default_in: [backlog]
```

The rules are written in CEL, the Common Expression Language. It is typed and checked when the file loads, it cannot
change anything, it always ends, and it is a Go library.

### Operations

What an act does is a list of operations: assign an issue, make a branch, make a worktree, push, open a change, mark
it ready, merge it. The operations are Go, a fixed and typed vocabulary. Data arranges them and gives them their
values; data cannot define a new one. Each operation can say what it would do before it does it.

### Plan, approve, apply

Every act runs the same way:

1. **Plan.** kit works out the steps: what changes, in which system (the forge, git, the files), whether the step can
   be undone, and the command line that does the same. Nothing has changed yet.
2. **Approve.** kit shows the plan. A person's act is approved by the person's presence, which kit binds to the
   plan's hash. Work on the bot's lane needs no presence: whoever asked gets the plan and applies it by its hash.
   `--plan` prints the plan and stops, and with `--json` prints it as data.
3. **Apply.** kit runs the steps in order. If the facts changed since the plan, it refuses, and says to plan again.
   If a step fails, kit stops, says which steps are done and which are not, and names the command that goes on or
   that undoes. It does not claim to roll back what it cannot.
4. **Result.** kit prints each step's outcome, the work's new state, whose move it now is, and what follows.

An approval binds to a plan's hash, so what was approved is what runs. This is attended automation: `kit setup` and
the making of a bot run the same way as `kit start`.

### Two lanes

The lanes follow the owner's paper, One Integration, Many Hands.

- **Every act has a lane, and the lifecycle's data says which.** A decision is a person's: in the default lifecycle,
  to merge. Work is what prepares: to start, to attach, to push, to open a change, to comment. An organization's
  layer can move an act from one lane to the other, so a review need not stay a person's.
- **A decision travels as the person, with proof of presence.** kit asks the machine to prove that the person is
  there, and binds the proof to the plan's hash. No agent can give it.
- **Work travels as the bot, where there is one.** The bot is the forge's identity for software, on GitHub an App,
  and what it does shows at the forge under the bot's name. kit records the person it worked for. A person or an
  agent can ask for work, and neither needs presence for it.
- **With no bot there is one lane.** Every act that changes something is then the person's and needs presence, so
  kit serves manual work alone. An agent can read what kit prints and can ask for a plan, and it can apply nothing.
- **A bot never acts as a person.** kit never gives an agent's session the person's token, and never runs a decision
  on the bot's.
- **A bot's session has no limit on its length.** GitHub ends a bot's token after an hour. kit mints the next one
  from the bot's key before then, for git and for every call, for as long as the session runs. A session of ten
  hours never sees a token end.
- **A bot holds no signing key.** On the bot's lane `kit submit` does not push the commits made on the machine. It
  makes each of them again at the forge, through its API, in order, with its message and its tree. GitHub signs a
  commit made so and marks it verified, as the bot's. There is no key to make, to keep, or to lose, and kit never
  signs with a person's key. The commits keep their steps and their words; their hashes, authors, and dates are the
  forge's, and kit moves the branch on the machine onto them.
- **A person's commits are the person's.** On the person's lane `kit submit` pushes the commits as the person made
  them, where the forge will take them. Where the repository asks for signatures and a commit has none, kit lands it
  through the API as above, signed by GitHub as the person's, and says so.
- **A change's steps stay in its pull request.** `kit submit` keeps each step there as a commit. Nothing that was
  pushed is rewritten: a work is brought up to date by a merge of the default branch into it, never by a rebase.
- **The default branch is linear, signed, and never rewritten.** kit holds this as a matter of correctness, not of
  taste. Each commit on it is one change that a person decided to merge, with the gate's answer for exactly that
  state: a search for a fault lands only on states that passed, and to undo a change is to revert one commit.
  `kit setup` sets the rule, and `kit merge` squashes. On GitHub nothing else fits: a linear branch takes a squash
  or a rebase, GitHub cannot sign the commits a rebase makes, and it has no merge that keeps a branch's own commits.
- **The rules are the default branch's.** The lifecycle and the policy that govern an act are the default branch's,
  as the remote has them. A worktree's copy governs nothing until it merges, so an agent cannot loosen a rule for
  its own change.
- **What kit cannot hold.** kit governs what goes through kit. An agent on the same machine can still use a
  credential the machine holds, such as a `gh` that is signed in, or a key that signs a commit with nobody there.
  `kit setup` reports such credentials, and the rule on the default branch at the forge is what stands behind kit.

### kit makes what it needs

- `kit setup` makes a repository ready for kit, or makes the repository: the rule on its default branch, and
  whatever else a slice's acts depend on. It is an act like any other, with a plan. It is safe to run again, and
  then changes only what differs.
- A slice that needs something to exist, at the forge or on the machine, includes the act that makes it. No page of
  the docs tells a person to make something by hand.
- Some acts a forge takes only from a person signed in at its website, and never from a token: on GitHub, to
  register an App and to install one. The page posts to the website with the person's own session, so kit could do
  it only by taking that session over, which is a bot acting as a person. kit opens the page with everything filled
  in, waits, and goes on by itself. The person's part is one confirming click, inside the one command.
- kit depends on no tool an organization may have for this, such as Terraform.

### What kit prints

- `kit next` prints the picture for the repository: how many works are in each state and where they are stuck, then
  the works under whose move they are, each with its state, the reason, and its default act.
- `kit show <address>` prints one entity: its place in the lifecycle, the acts possible now and those that are not
  with the reason, what follows each, its relations, and its source.
- Each act is a command of its own: `kit start`, `kit attach`, `kit submit`, `kit ready`, `kit merge`.

Each of them prints text at a terminal and the same facts as JSON with `--json`. The JSON has a schema and a version.

## The terminal UI

The UI draws what `kit next` and `kit show` print, and sends acts. It holds no rule of the lifecycle. It opens on
"your move":

```text
 kit · empowerite/kit                                    / search   ? keys
 backlog 41 ─▸ started 6 ─▸ in review 3 ─▸ building 2 ─▸ merged 5
                 └ 2 stuck over 3 days

 YOUR MOVE  4
 ▸ #31  kit commit warns twice    ada asked for your review, 3h   ⏎ open
   #12  Lint the docs             CI failed: lint                 ⏎ see why
   #46  A noun left out           top of backlog, unassigned      ⏎ start
   #2   The first release         ready, CI green                 ⏎ merge
 WAITING ON OTHERS  3     RUNNING  2     (folded)
 ──────────────────────────────────────────────────────────────────────────
 #31  issue ● assigned ● branch ● PR ● CI ● review ◉ merge ○
 Now   ada asked for your review 3 hours ago
 Act   ⏎ open at GitHub   . all acts
 Then  once you approve at GitHub, it can merge: CI is already green
```

An act shows its plan before anything changes. With two issues selected, start makes two works:

```text
 Start 2 works: #46, #45
 Runs     kit start 46 45
 Changes  assigns you to both · makes 2 branches · makes 2 worktrees
 Then     each moves to started; nothing is pushed until you submit
 ⏎ approve with Touch ID    e edit the command    esc cancel
```

Drilling down keeps the way back in view, and every level reaches its source:

```text
 kit · empowerite/kit ▸ #12 Lint the docs ▸ change 14 ▸ checks ▸ lint
 lint   failed 4 minutes ago   job 2 of 3      read from GitHub 20s ago
   step  markdownlint    12 errors
 ▸ docs/apps.md:41  MD013 line length
   docs/apps.md:57  MD013 line length
 o open this job at GitHub    ← back    r read again
```

- **Target.** An act applies to the rows selected, or to the row under the cursor when none is. ⏎ runs the row's
  default act, and `.` lists every act.
- **Drill down.** → opens the entity under the cursor into its relations, and ← goes back. The first line always
  says where the person is.
- **Source.** `o` opens the source of whatever is under the cursor. A fact older than a set age says how old it is,
  and `r` reads it again.
- **The look is frozen for the first slice:** this one screen, one-cell characters, no menu bar, and no presets.

## What is data, and who can change it

| What | Read by | Changed by |
| --- | --- | --- |
| The entities, their fields and relations | `kit show`, the UI's drill-down, the docs | organization, repository |
| The lifecycle: states, court, acts, relations | `kit next`, every act, the UI | organization, repository |
| The commands: words, options, help, what each prints | the command line, completion, the UI's forms, the docs, and later the agents' tools | kit; an act in the lifecycle is a command |
| The policy: names of branches, subjects of commits | the acts that make them | organization, repository |
| The views: what a screen lists and in what order | the UI | organization, repository, person |

- **Layers.** kit's own defaults are data files inside the binary, and `kit data show <name>` prints one. An
  organization's files come over them, then the repository's, then the person's, which reach the views alone. The
  later layer wins.
- **Explain.** `kit explain <state, act, or key>` says which layer's file decided it, so a change is never a
  mystery.
- **Schemas.** Every data file has a JSON Schema. kit checks a file when it loads it and names the file and the line
  of what is wrong.
- **One source for a command.** From a command's data come its parsing, its help, its completion, the schema of
  what it prints, and its page in the docs. Nobody writes those by hand.
- **Proof.** kit's tests run the engine on a second lifecycle, a toy with other states and acts. The default
  lifecycle is tested as data, through scenarios. The Go under the engine never names a state or an act.

Beyond data, a new forge or tracker is a Go adapter behind an interface. The charter asks for more: that how kit
works can be extended without a fork, by a provider that runs as a program of its own, without compiling kit. These
slices draw the interfaces so that it can come, and it gets a design of its own.

## The structure of the Go code

One module. The packages stand in layers, and a test fails an import that goes the wrong way.

| Package | Holds | May import |
| --- | --- | --- |
| `model` | entities, addresses, sources | nothing of kit's |
| `data` | the files inside the binary, the layers, the schemas, explain | `model` |
| `expr` | the rules' language | `model` |
| `lifecycle` | states, court, and acts, from data | `model`, `data`, `expr` |
| `ports` | the interfaces: forge, git, files, clock, browser, presence, secrets | `model` |
| `plan` | operations, plans, apply | `model`, `lifecycle`, `ports` |
| `doc` | what `kit next` and `kit show` print | `model`, `lifecycle`, `plan`, `ports` |
| `cli` | the command line, from the commands' data | `doc`, `plan`, `data` |
| `tui` | the screens | `doc`'s types and one interface that runs acts |
| `adapters/github`, `adapters/git`, `adapters/os` | the real systems | `ports`, `model` |
| `adapters/fake` | a forge, a git, and files in memory | `ports`, `model` |
| `cmd/kit` | wires the adapters in | everything |

- Only the adapters start a process, open a connection, or touch a file. Time comes from the clock.
- The UI reads only what `doc` prints, as the proof of concept's UI read only kit's JSON.
- Packages stay small, since Go runs again only the tests of a package whose inputs changed.

## Testing

| Tier | Runs | What | Budget |
| --- | --- | --- | --- |
| Save | on every save and before a commit | the tests of the packages a change reaches | 10 seconds |
| Pull request | on every push | every package, from Go's cache; the scenarios; the schemas; the import rule; coverage | 3 minutes |
| Nightly | each night and before a release | the real binary against real git and a sandbox repository at GitHub, on macOS and Linux; the fake held to the real forge | none |

- **In the first two tiers no test starts a process or reaches the network.** They run against the fakes.
- **Scenarios.** A scenario is a world (issues, changes, checks, and a git history, as data), a list of acts, and the
  documents kit must print after each. It runs in memory. The first slice's acceptance is one.
- **The fake is held to the real thing.** One suite of tests runs against the fake forge on every pull request and
  against GitHub each night. When they differ, the fake is wrong.
- **The sandbox is kit's own making.** `kit setup` makes the sandbox repository. In CI kit runs as the workflow's
  own identity, which is a bot's lane, and the sandbox's own layer of the lifecycle marks every act as work, so the
  nightly tier runs the scenario with nobody there. No credential for it is made by hand.
- **Screens.** A screen's test gives a document and reads the lines drawn, at one width, with one more test for a
  narrow terminal. Keys are tested on the screen's state without drawing. There are no tests of every key at every
  width.
- **Budgets are tests.** A test that takes over a second fails, unless it is in the nightly tier. The pull request's
  gate fails when it takes over 3 minutes.
- **Coverage.** 100%, measured in the pull request's tier. Where the rule and a budget collide, the test is
  designed again; neither gives way.

The budgets are targets until the first slice exists. In the proof of concept, the Go tests took 2 minutes 40 seconds
cold and 0.7 seconds when nothing had changed, and one package of whole-screen tests took 2 of those minutes.

## How the work is done

- **The spine.** The charter, this design, the data's schemas, the table of packages, and a log of decisions. Only
  the owner changes the spine. An agent proposes a change to it and does not make one, and a question the log has
  decided is opened again only by the owner.
- **A slice is a workflow.** Each slice ends in a scenario that the owner runs in the UI on a real repository. A
  slice is done when its scenario passes, the gate is green, and the owner has used it.
- **Contract first, then many agents.** One session writes a slice's interfaces, its fakes, and its scenario. Only
  then do other agents build behind those interfaces, each on one issue, in its own worktree and branch.
- **Small merges to a default branch that is always green.** Each issue is one small pull request. A person arms it
  to merge, and it lands when the gate passes. An issue closes when its pull request merges.
- **Review per change.** A change is reviewed against its slice's design when it is made. A review of the whole
  happens at the end of a slice, and it reports its ten most important causes, not every finding.
- **Findings are sorted before they are issues.** A finding goes to a list. It becomes an issue only if it is in the
  slice's scope or names a cause; the rest stay on the list with a reason.
- **Limits that stop the line.** A slice has at most 10 open issues, and the gate has its budget. Over either, no new
  work starts until it is back under.
- **What the gate catches is written down.** Each time the gate refuses an agent's change for a shortcut, a line
  records it. That is the evidence the first hypothesis lacked.
- **Nothing by hand.** A slice includes the acts that make what it needs. Something a person would have to make by
  hand is a defect of the slice, and never a note for later.

## The order of the work

The owner set this order on 2026-11-18. Each step is a milestone.

1. Foundations: the charter, this design, and the spine.
2. Toolchain.
3. Rules where agents look.
4. Kit for manual work, which this design calls the first slice.
5. The bot, which this design calls the second slice.

The toolchain and the rules each get a design of their own. The charter's other purposes each hold an issue until
the owner gives it a milestone: templates, the checks, the release and its packages, the newcomer's start, every
role's questions, the workspace, evidence, and the path of work past the merge.

## The first slice: kit for manual work

This slice is kit for manual work. A person runs `kit setup` on a GitHub repository, or to make one, and approves its
plan. On that repository the person can then:

1. Run `kit next`, or open the UI, and see the works by whose move they are.
2. Start a work from an issue in the backlog, and attach a second issue to it with a relation.
3. Commit in the worktree with git, and `kit submit`: kit pushes the commits, or lands them at the forge where the
   repository asks for signatures they lack, and opens the change as a draft.
4. See the checks run, drill into one that failed down to its log, and open it at GitHub.
5. Mark the change ready, and merge it once what the forge requires of a merge is met: its checks, and its reviews
   where the repository's rules ask for them. To merge is a decision.

Reviewing stays at the forge in this slice. A change that waits for a review has "open at GitHub" as its default
act, and no act of kit approves one. In a repository whose rules ask for no review, as one person's does, a ready
change with green checks can merge.

Of the data, the first slice builds:

- kit's own defaults and the repository's layer over them, for the lifecycle alone, with the schema check,
  `kit data show`, and `kit explain` for a state and an act.
- The acts as commands, from the lifecycle's data: their parsing and their help.
- The name of a branch, as one rule in the lifecycle's file.
- The test that runs the engine on the toy lifecycle.

It leaves these for later, and keeps room for them: the organization's layer and the person's, completion, the docs'
pages from data, the views as data, and the rest of the policy. Where an organization's files live belongs to the
design for delivery.

Every act shows its plan first, and every act that changes something is approved by the person's presence. There is
no bot yet, so no agent acts through kit. The slice is done when this scenario passes in memory, the nightly tier
passes it against a sandbox repository, the pull request's gate is under its budget, the owner has run it and
accepted it, and nothing it needed was made by hand.

## The second slice: the bot

This slice lets agents work, and keeps them from deciding.

1. `kit bot create` makes the bot in one command. It registers the App with the permissions the lifecycle's work
   needs, keeps its key in the machine's keychain, installs it on the repository, and proves the chain by minting a
   token. The person confirms twice, once for an organization and never again: the App, which kit has filled in,
   and its install, with the account and the repositories already chosen. A repository added later needs no click.
   GitHub Enterprise Cloud has an API for the install, and kit uses it where it can.
2. Work travels the bot's lane from then on, whoever asked, and kit records the person the bot worked for. A commit
   lands as the bot's, signed by GitHub, with that person named in its trailers.
3. An agent's session is given the bot's tokens, for git and for `gh`, and kit renews them for as long as the
   session runs.
4. `kit setup` adds to the rule on the default branch what keeps a bot from deciding.

The slice is done when an agent's session does the scenario's work as the bot for longer than an hour, is refused a
merge, and the person merges with presence; and nothing in it was made by hand.

## Approaches considered

| Question | Picked | Dropped, and why |
| --- | --- | --- |
| The screen | kit prints the state, the court, and the acts; the UI draws them | Mending the proof of concept's tables: it shortens the keys and leaves no picture. Prompts for each command with no screen: no picture either. |
| Fast verification | Go, tests in memory against fakes, small packages, three tiers with budgets | Keeping bash with a map of which tests a change reaches: each test still costs seconds, and a wrong map skips tests in silence. Splitting into repositories: it moves the cost and repeats the CI and release work. |
| Many agents | a spine, contract first, small merges, limits | Adding only a sorting of findings: without a spine the drift goes on. One agent at a time: coherent and slow, and it does not test kit's own claim. |
| Customizing | data files in layers, rules in CEL, a fixed vocabulary of operations in Go | Everything in Go with settings: the lifecycle is no longer the organization's. A scripting language or plug-ins for acts: a plan could no longer say what will happen before it happens. |
| Telling an agent from a person | two lanes chosen by the act: presence for a person's, a bot's identity for work | A variable the agent sets: a statement and no proof. Agents on the person's token: an agent's act is the person's in every record. A bot for each person: more to register, and not needed to keep a bot from acting as a person. |
| What kit needs at the forge | kit makes it, through `kit setup` and a plan | Steps by hand in the docs, automated later: the debt the owner refuses. An organization's Terraform: whoever uses kit does not have it. |

## Calls made here

The owner did not decide these; they are proposals to check:

1. One binary, `kit`, holds the UI. The proof of concept shipped two programs.
2. The rules' language is CEL. The proof of concept used jq for court and OPA for policy; policy is out of this
   design, and OPA can stay for it.
3. The GitHub adapter speaks to GitHub's API itself, with the token `gh` holds, and does not run `gh`.
4. `kit submit` opens the change as a draft at the first push, so a work has a change from early on.
5. The names: `kit next`, `kit show`, `kit start`, `kit attach`, `kit submit`, `kit ready`, `kit merge`, and the
   relations `subissue`, `discovered`, and `blocker`.
6. The budgets: 10 seconds, 3 minutes, one second for a test, and 10 open issues for a slice.
7. A person's own files change views only, never the lifecycle or the policy.
8. 100% coverage holds for every package, as the proof of concept held its Go.
9. In CI kit runs as the workflow's own identity, on the bot's lane, and the sandbox's layer marks every act as
   work.
10. The lifecycle that governs an act is the default branch's, never the worktree's.
11. Reviewing is not an act of kit in the first slice, and merge asks only what the forge's rules require.
12. A change kit did not start is a work, shown and counted. With no issue it has no lead, and an issue it closes
    with no relation is `unsorted`.
13. The first slice builds one layer over kit's defaults, for the lifecycle alone, and leaves the other layers, the
    views, completion, and the docs' pages for later.
14. Presence is Touch ID or the account's password on macOS. Where a machine has no such proof, kit says so and
    marks the approval as weaker.
15. In the first slice the person's token is the one `gh` holds. With the bot's App it becomes a token the App
    issues to the person, which kit keeps.
16. The names `kit setup` and `kit bot create`.
17. The first bot is the size for one person: its key in that person's keychain, with no broker.
18. Work travels the bot's lane even when a person asked for it, as the owner's paper has it.
19. On the bot's lane each commit is made again at the forge through GitHub's `createCommitOnBranch`, which signs
    it: the steps and the messages are kept, and the hashes are not. On the person's lane kit pushes the person's
    commits as they are, and uses the same call only where a signature is required and missing.
20. A work is brought up to date by a merge of the default branch into it, so nothing pushed is rewritten, and the
    squash at the merge folds that merge away.

## Left to the builder

- The exact shape of each data file and its schema, and the states and court rules of the default lifecycle.
- Where a work's attachments are recorded, within the constraint that the forge and git hold them.
- The client for GitHub's API, the cache, and how stale a fact may be before the UI says so.
- The layout of the screens within the sketches, and the keys beyond those named.
- How scenarios and worlds are written as data, and the test helpers.
- Where the lists of findings, of decisions, and of what the gate caught are kept.
- How presence is proved on each system, and how the proof is bound to a plan's hash.
- How an agent's session is given the bot's tokens, and how they are renewed under git and `gh`.
- Which rule at GitHub keeps a bot from deciding. GitHub's docs do not say whether a bot's approval counts toward a
  rule that asks for approvals, so the builder tests it on the sandbox.
- How the nightly tier reaches the sandbox with no secret made by hand.
- Whether `createCommitOnBranch` carries a file's executable bit and a symbolic link. Where it does not, the builder
  uses GitHub's lower API for commits, and tests on the sandbox that GitHub still signs the commit as the bot's.
- A commit with two parents, as a merge of the default branch into a work makes. `createCommitOnBranch` may make
  only commits of one parent, so the builder finds the route, GitHub's own update of a branch or its lower API, and
  tests that the result is signed.
