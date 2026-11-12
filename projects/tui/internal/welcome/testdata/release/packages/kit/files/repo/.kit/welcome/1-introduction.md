<!-- kit:generated -- kit welcome's first step, the introduction, from the kit package -->
<!-- see: "Changing what kit adds" in .kit/1st.md, to replace it -->
# Welcome

kit keeps a repository trustworthy while people and agents work in it side by side.

## What kit is

kit is your house rules as a versioned release, and the checks that hold the work to them. It places the rules
where people and agents both look: `.kit/1st.md`, which every agent reads first, a rule for each kind of file, and
the files each agent harness reads. Its commands do what the rules describe, so the right way is also the short way.
Take a new release, and the rules and checks move with it, in every repository that takes it.

## Who it is for

- People who work with coding agents, and want an agent's "done" to mean what theirs means.
- Teams that keep one set of rules across many repositories, versioned rather than copied.
- The agents themselves: kit tells an agent where the rules are, which command does each step, and why a check
  refused, in a line it can act on.

## What it solves

For people:

- Rules drift between repositories, branches, and colleagues. kit delivers the same rules and checks to each, and
  `kit check` says when a delivered file drifted.
- Where does the work stand? `kit flow` draws it in one picture, from the backlog to a release.
- A new machine, or a new clone, misses a tool or a login. This guide checks for them, and offers each fix.

For agents:

- An agent holds none of your house rules unless they are where it looks. kit puts them there, in each harness's
  own place.
- An agent can say "done" when it is not: a test skipped, a change linked to no issue, a commit that breaks the
  policy. `kit ci`, `kit lint`, and `kit policy` refuse that work, for an agent as for a person.
- An agent improvises git where no command fits. `kit work`, `kit commit`, `kit rebase`, and `kit submit` are the
  commands, and each refuses what would harm the work.

## This guide

- Toolchains: the four tools kit runs on, and the tool and package managers kit found here.
- Setup: the linters, each code forge's login, and this repository.
- Customize: what teams change first, and where.
- Terms: the words kit uses.

kit's applications, its screens and what each one shows, are a tutorial that ships with kit: `kit tutorials
applications`.

The next two steps are checks. A failed check shows the command that fixes it. Nothing runs until you say so.

`kit welcome --checks`
