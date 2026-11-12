<!-- kit:generated -- kit welcome's fifth step, the terms, from the kit package -->
<!-- see: "Changing what kit adds" in .kit/1st.md, to replace it -->
# Terms — These are the words kit uses, by topic, each with a command that shows it

<!-- each `## ` heading is a topic; each `### ` heading under it names the terms that share one meaning,
comma-separated; each line of one backticked command under it is an example, the first naming the command it
groups under -->

## Where work lives

### repository

The project's files and their history, kept by git.

`git log --oneline`

### code forge

The service that hosts the repository, its issues, its pull requests, and its builds, such as GitHub or GitLab. kit
reaches it through its command-line tool, gh or glab.

`kit platform services`

### remote

Another copy of the repository, such as the one on the code forge.

`git remote --verbose`

### primary

The remote kit reads issues and pull requests from. It is origin, unless `primary` in kit.yml names another.

`kit platform settings`

### workspace

What kit looks at: one repository, or a group of repositories you name. Your groups are yours alone, and never
committed.

`kit workspaces`

### desk

Your root checkout, where you cloned the repository. You read, review, and try things there. Agents never change it.

`kit flow desk`

### worktree

A second checkout of the same repository, on a branch of its own. Each issue's work happens in a worktree of its own,
so the work on two issues never mixes.

`kit flow worktrees`

`git worktree list`

### branch

A line of commits with a name. `kit work` makes one for each issue.

`kit workspace branches`

### default branch

The branch that holds the finished work, such as main. Work reaches it only through a merged pull request.

`git remote show origin`

## How work moves

### issue

One piece of work to do. Work starts from an issue. Every open one is in the backlog or underway.

`kit flow issues`

### tracker

The system that holds the issues: the code forge, or another system, such as Azure DevOps. An issue's reference
names its tracker: `#7` is the code forge's, and `AB#4521` is one that kit.yml declares.

`kit work AB#4521`

### backlog

The open issues that no one has started: no open pull request closes them. 💤 marks one.

`kit flow issues`

### underway

An issue someone started. `kit work` gave it a branch, a worktree, and a draft pull request. 🛫 marks one.

`kit flow issues`

### commit

A saved change, with a message. `kit commit` runs the linters and checks the message first.

`kit commit`

### pull request

A request to merge a branch into the default branch. A person reviews it and merges it. GitLab calls it a merge
request.

`kit flow changes`

### rebase

Moving a branch's commits on top of the newest default branch. `kit rebase` does it, then pushes the branch.

`kit rebase`

### incoming

What the last fetch brought from each remote: new commits, and branches and tags that are new or gone.

`kit flow incoming`

### release

A version of the work. `kit release` marks it with a tag, and the release workflow publishes it.

`kit flow releases`

## Checks

### gate

The repository's own check command, such as `just ci`. `kit gate` runs it on your machine.

`kit gate`

### CI

Continuous integration: the checks the code forge runs each time a branch is pushed, the gate and kit's own. `kit ci`
runs the same checks on your machine.

`kit ci`

### build

One run of CI. The runs that no pull request owns, such as those on the default branch, are the builds.

`kit flow builds`

### CD

Continuous delivery: turning a release into something people use, by building, publishing, and deploying it. The
repository's own workflows do it, and kit reads what came of it.

`kit flow releases`

### linter

A program that reads a file and reports what breaks its rule. `kit lint` runs each rule's linter over its files.

`kit lint`

### check

`kit check` compares every file kit placed with the release, and fails on any difference.

`kit check`

### policy

The patterns that branch names, commit subjects, and pull request titles must match.

`kit policy`

### trust

Your consent for kit to run the commands a clone names. kit asks for it once per clone.

`kit trust`

## Rules for files

### kind

A kind of file, such as bash or markdown. A file's path and first line decide its kinds.

`kit kinds README.md`

### rule

What a file of a kind must do to be trusted, and the linter that checks it.

`kit lint README.md`

### style

How a file of a kind looks. A style package delivers it, and a repository takes the styles it agrees with.

`kit packages`

### context

The rules and the styles that govern a file. Read them before you change the file.

`kit context README.md`

### template

The shape a new file starts from. `kit new file` writes a file from one.

`kit help new`

### region

A part of your file that a command writes, from a `kit:generate` line to a `kit:end` line. The rest of the file stays
yours.

`kit generate --check`

## What kit delivers

### package

A set of files kit delivers, such as rules, styles, or a CI job. A repository takes the packages it wants.

`kit packages`

### bundle

A package that only names other packages, such as empowerite.

`kit packages --active`

### house package

Your team's own package, in `.kit/packages/`. Its file replaces kit's file at the same path.

`kit packages --active`

### hydrate

To place what the active packages deliver under `.kit/`. Each new clone or worktree needs it once.

`kit hydrate`

### kit.yml

The repository's kit settings: the release, the packages, the policy, and the rest.

`kit platform settings`

## Tools

### package manager

Installs software on the machine, such as brew or apt.

`kit toolchains`

### tool manager

Installs and pins the versions of the tools a repository uses, such as mise or asdf.

`kit toolchains`

### toolchain

The set of tools a tool manager installs, such as Go and its linters.

`kit tools`

### ambient

No manager chosen: kit uses the tools already on PATH.

`kit toolchains use ambient`

## kit itself

### application

One of kit's screens, such as Flow or Platform. The = menu opens each one.

`kit flow`

### stage

One tab of an application, such as the issues or the builds. The `flow` setting in kit.yml picks the stages Flow
draws.

`kit flow --stages`

### agent

A program, such as Claude Code, that works in the repository as a person does. It reads `.kit/1st.md` first, and
works in a worktree of its own.

`kit serve`
