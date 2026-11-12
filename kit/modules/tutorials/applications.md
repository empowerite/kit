---
title: kit's applications
summary: kit has an application for each side of the work, from Flow, the trip of the work from an issue to a release, to Packages.
glyph: 🧭
audience: [newcomer, ui]
order: 2
---

Each app shows one part of your work. Click an app's name to open it. ⌫ brings you back here. You can also open
any app from the = menu, or run its command.

## 🧭 Flow

All your work, from an issue to a release, with one tab for each step:

1. [Issues](kittui:flow/issues): every open issue. 💤 marks one no one has started, in the backlog, and 🛫 one
   someone is working on, underway. `kit work <N>` starts issue N. Its presets show the Backlog alone, what is
   Underway, or the issues by milestone, iteration, or swimlane.
2. [Changes](kittui:flow/changes): pull requests. A person merges each one.
3. [Builds](kittui:flow/builds): CI runs outside any pull request, such as those on the default branch.
4. [Incoming](kittui:flow/incoming): what the last fetch brought in.
5. [Releases](kittui:flow/releases): the versions you tagged.

Open [Flow](kittui:flow), or run `kit flow`.

## 🎫 Issues

The work to do: every open issue, in the backlog or underway, then the milestones and the iterations.

Open [Issues](kittui:issues), or run `kit issues`.

## 🔀 Changes

Pull requests, with their reviews and checks, and the ones already merged.

Open [Changes](kittui:changes), or run `kit changes`.

## 📘 Repositories

Your repositories, with their worktrees, branches, commits, tags, and remotes. Your desk is the checkout you cloned.
It is yours, and agents never change it. Each issue gets its own worktree.

Open [Repositories](kittui:repositories), or run `kit workspace`.

## 🚚 Delivery

Releases, the builds that made them, and where they are deployed.

Open [Delivery](kittui:delivery), or run `kit delivery`.

## 👥 People

The people, teams, and agents who do the work.

Open [People](kittui:people), or run `kit people`.

## 🔏 Trust

The evidence that the work follows the rules, and what fails them.

Open [Trust](kittui:trust), or run `kit audit`.

## 🧰 Platform

What kit runs on here: services, toolchains, tools, and settings.

Open [Platform](kittui:platform), or run `kit platform`.

## 🧩 Packages

The packages kit delivers to this repository.

Open [Packages](kittui:packages), or run `kit packages`.

## Need more help?

`kit welcome` walks you through your first hour with kit. `kit help` lists every command.
