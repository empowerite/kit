# Charter

kit keeps the work of AI coding agents trustworthy. Agents take the same shortcuts people do: they call work done
when it is not, skip tests, and rewrite other people's branches. kit writes the rules down, puts them where every
agent reads them, checks that they were followed, and carries the work from an issue to a release.

This page says what kit is for and the principles it holds. Only the owner changes it.

## What kit is for

1. **Rules, written once, for any harness.** A repository's rules live in the repository, in one place. Every agent
   harness, and every person, is pointed at the same rules. kit depends on no one harness, and a rule is never kept
   in a session's memory or in one vendor's format.
2. **The rules that govern a file.** Before a file is made or changed, an agent asks which rules govern it, and reads
   those alone. A kind of file is defined once, and a rule names the kinds it governs.
3. **Templates that make a file and check one.** One template, with holes in it, does both: it makes a new file, and
   it checks that a file still has that shape. It is the guardrail for files that have no schema, such as scripts,
   recipes, and workflows.
4. **Checks that hold agents and people to the rules.** What an agent ships is verified, not trusted: by the
   repository's own gate, by each rule's linter, by the policy for names, and by a check that what kit delivered is
   still as kit delivered it. The checks run the same at a desk and in CI.
5. **A toolchain.** A repository pins the tools its work runs with, each locked by version and by digest for every
   platform, and kit puts them on the `PATH`, so a desk and CI run the same tools. A repository that has a tool
   manager keeps it, and kit works with it.
6. **The path of work.** kit takes work from an issue to a merged change, and on to a release, with the same few
   commands, wherever the repository is hosted and whatever tracks its issues.
7. **Where the work stands.** kit shows where each piece of work is, whose move it is, and what comes next: to a
   person on a screen, and to a script or an agent as data.
8. **Every role's own questions.** A developer asks what is waiting on them. A platform engineer, a security
   engineer, a person who answers for cost or for a team's results, and an auditor each ask something else. kit asks
   each question of the work and not of the tool, so the answer has the same shape on any forge and any tracker. An
   organization defines its own roles, and a person's views follow the person.
9. **Many repositories as one.** A workspace is what kit looks at: one repository, or a named group of them. The
   picture of the work, and each question about it, spans a group as it spans one.
10. **Evidence.** kit shows the evidence that the rules held: what was checked and what it answered, what the forge
    enforces, what was waived and by whom, and a record of each act kit took and for whom. An auditor, who checks
    the work from outside it, reads that evidence as of a date.
11. **Identity and trust.** A person's decisions are the person's, and an agent's work is a bot's. kit runs nothing
    in a clone until a person trusts it.
12. **A release, not a copy, and changed without a fork.** kit reaches a repository as a pinned release. An update
    is a change like any other, and a delivered file that someone edits in place fails the checks. A repository adds
    rules of its own, and replaces what kit delivers, through packages of its own. kit changes a file it shares in
    the least intrusive way the file allows, and what kit puts in, kit can take out.
13. **Standing a repository up.** What kit needs at the forge and on the machine, kit makes.
14. **A newcomer's start.** A newcomer starts with a guide that checks the machine and each repository, and offers
    the fix for what it finds, to copy or to run. Short lessons teach the rest, one page gives the words kit uses,
    and a form builds each command and shows it.

## Principles

### What kit insists on, and what it leaves alone

- kit is not opinionated about tools or processes, except where they affect correctness. It works with the forge,
  the tracker, the tool manager, the agent harness, and the gate that a repository already has.
- Where kit insists, correctness depends on it, and kit says why. The history of the default branch is the first
  case. It is linear and signed, a change lands on it as one commit, and it is never rewritten, because each commit
  on it must be a state that was verified.
- kit does not impose taste. A rule says what work must do to be trusted. How a file looks is a style, and a style
  is optional.
- kit works with what is there. A part of kit that needs what a repository lacks, such as a forge or a terminal,
  stands aside, and the rest still works.

### Trust

- Verify; do not trust. "Done", "passing", and "verified" are said only of something that ran and succeeded. A skip
  is reported as a skip. Being wrong is recoverable; misreporting is not.
- kit says what it verifies and what it only advises. A status is never a guess.
- The forge and git hold the truth. kit keeps nothing it could not read again from them, and every fact kit shows
  leads back to its source.
- Every change shows its plan before it runs.

### People and agents

- A bot never acts as a person. A person's decision needs proof that the person is there.
- The desk, a person's own checkout, is the person's. An agent works in a worktree.
- Work begins with one issue.
- What a person may do by hand, at a command line or in a browser, the person may approve kit to do, while there.
  What must not happen with nobody there does not happen in CI.

### The repository's own

- What a team decides is data, not code: its lifecycle, its policy, and its schemas. kit's own defaults are data
  too.
- Rules live in the repository, where every agent and every person reads them.

### The experience

- Gradual reveal. kit shows first what a person needs first. The rest is there when it is asked for, and nothing has
  to be learned before the first useful thing is done.
- Consistency. One word names one thing. A field means the same wherever it appears, and a command, an option, and
  a screen behave the same on every forge and in every part of kit.
- A path from novice to super user. Each thing a person learns leads to the next. A screen shows the command it
  runs, so the screen teaches the command line. The command line prints data, so it teaches scripting. What kit
  delivers, and how kit works, can each be extended without a fork.
- Nothing is made by hand that kit will later make. A step a person must take by hand is a defect.
- The person's part is simple, natural, and painless. Where a forge insists on a person, kit prepares everything and
  asks for one confirmation.
- Everything a screen shows, a script or an agent can read as data.

### How kit is built

- A design comes before the code, and only the owner changes it.
- No known wrong case is left for later.
- Verification stays fast: tests run in memory, only for what a change reaches, within a budget the gate enforces.
- A release is published in CI and nowhere else.
