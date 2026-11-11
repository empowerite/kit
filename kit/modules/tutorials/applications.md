---
title: kit's applications
summary: What each application shows, from Flow, the trip of the work from an issue to a release, to Platform.
glyph: 🧭
audience: [newcomer, ui]
order: 2
---

Each application shows one side of the work in this repository. Open one from the = menu, at the top left of every
screen. Each part below names one, says what it shows, and ends with the command that prints it.

## Flow

The whole trip of the work, from an issue to a release, one stage to a tab, in this order:

1. Backlog: the open issues no one has started.
2. Underway: the issues someone started. `kit work <N>` starts issue N: a branch, a worktree, and a draft pull
   request.
3. Desk: your root checkout, where you cloned the repository. It is yours. Agents never change it.
4. Worktrees: one checkout to an issue, where its work happens. `kit commit` commits there, and `kit submit` pushes.
5. Changes: the pull requests. Mark one ready when its work is done. A person merges it, and nothing reaches the
   default branch any other way.
6. Builds: the CI runs that no pull request owns, such as those on the default branch. `kit gate` runs the same gate
   on your machine.
7. Incoming: what the last fetch brought from each remote.
8. Releases: the versions `kit release <version>` tagged.

`kit flow`

## Issues

The work to do: the backlog, and the issues underway.

`kit flow backlog`

## Changes

The pull requests, each with its build.

`kit flow changes`

## Repositories

This repository, or each one in your workspace: its worktrees, branches, commits, tags, and remotes, and what is
incoming.

`kit workspace`

## Delivery

The releases, and the builds that made them.

`kit flow releases`

## Platform

What kit runs with here: the services it uses, the toolchains, the tools, and the settings.

`kit platform`

## Guides

Welcome is the guide to your first hour with kit. Tutorials, as this one, teach one thing each: some ship with the
kit and open here, and the others are pages online, which open in the browser. Help lists every command, and opens a
form for the one you choose.

`kit help`
