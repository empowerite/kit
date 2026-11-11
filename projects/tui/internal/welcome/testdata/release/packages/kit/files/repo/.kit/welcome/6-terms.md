<!-- kit:generated -- kit welcome's sixth step, the terms, from the kit package -->
<!-- see: "Changing what kit adds" in .kit/1st.md, to replace it -->
# Terms

The words kit uses, one or two sentences each, and the commands that show each one.

<!-- each `## ` heading names the terms that share one meaning, comma-separated; each line of one backticked command
under it is an example, the first naming the command it groups under -->

## desk

Your root checkout, where you cloned the repository. It is yours; agents never change it.

`kit flow desk`

## worktree

A checkout of its own, under `worktrees/`, on one issue's branch. `kit work` makes one for each issue.

`kit flow worktrees`

`git worktree list`

## code forge

The service that hosts the repository, its issues, its pull requests, and its builds: GitHub or GitLab. kit talks to
it through its command-line tool, `gh` or `glab`.

`kit flow incoming --no-fetch`

## remote, primary

A remote is another copy of the repository, as git names it. The primary is the remote kit reads pull requests,
builds, and issues from: `primary` in `kit.yml`, else `origin`. `kit work` and `kit submit` push to `origin`.

`git remote --verbose`

## package

A set of files kit delivers, such as rules, styles, or a CI job. A repository takes the packages it wants.

`kit packages`

## kind

A kind of file, such as bash or markdown. Its paths and its first line say which files are of it.

`kit kinds`

## rule

What a file of one kind must do to be trusted, and the linter that checks it. Rules live in `.kit/rules/`.

`kit context README.md`

## style

How a file of one kind looks. A style package places it in `.kit/styles/`.

`kit context README.md`

## template

The shape a new file of a kind starts from. `kit new file` makes a file from one, and `kit lint` holds files to it.

`kit help new`

## region

A part of a file you own that a command writes, from a `kit:generate` line to a `kit:end` line. The rest of the file
stays yours.

`kit generate --check`

## gate

The repository's own CI command, such as `just ci`. The delivered CI job runs it on each pull request into the
default branch, and `kit gate` runs it on your machine.

`kit gate`

## stage

One part of the picture `kit flow` draws, such as the backlog or the builds. The `flow` setting in `kit.yml`
picks which are drawn.

`kit flow --stages`
