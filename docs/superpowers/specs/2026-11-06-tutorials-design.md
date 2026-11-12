# kit tutorials: design

Status: approved, 2026-11-06.

## Intent

kit has many features and has never been released. `kit welcome` gets a newcomer through their first hour; nothing
goes deeper. kit needs focused training: one tutorial per feature, each tagged with who it is for (just starting,
UI, maintainer, and so on), so a reader can find their own way through a large feature set.

The training is reached from inside the product, `kit tutorials` and the 🎓 Tutorials item in kittui's app menu, but
the tutorials themselves are web pages. kit shows a pretty index of big buttons; a button opens its tutorial in the
browser.

Quality matters more than speed. There is no rush for the release.

## Goals

- A new project, `projects/tutorials`, that holds the tutorials and builds them into a website.
- One copy of that site per minor version of kit, at `https://kit.empowerite.dev/tutorials/<major>.<minor>/`, so
  every kit links to the tutorials written for its line.
- Tutorials written for every patch in their line: what arrived in a patch says so, and the reader is told when
  their kit is older than a passage.
- A "What's new" page and a changelog in each minor line's site.
- A catalogue, kept in kit, listing the top-level tutorials: what kit needs to draw its index. `kit generate`
  writes it from the tutorials project, so the two cannot drift.
- `kit tutorials`: the index, in kittui on a terminal and as text elsewhere. Its buttons open tutorials in the
  browser, and it can be filtered by audience tag.
- v0.1.0 ships the whole mechanism and at least one tutorial, done well.

## Non-goals

- Nothing in a tutorial runs against the learner's own setup. A learner may not be in a throwaway environment, so
  kit never runs a tutorial's commands for them. Tutorials can show commands; the learner decides where to run them.
- No sandbox for v0.1.0. Kit building a throwaway repository to run a tutorial in is a possible later step, and this
  design must not make it harder, but it is not built now.
- kit does not show the tutorials' contents. It shows titles, summaries, and tags, and opens the page.
- kit does not list the deeper pages (see Two kinds of pages). They are reached by links from top-level tutorials.
- *My call:* a consumer's packages cannot add or replace tutorials, as they can `kit welcome`'s pages. The catalogue
  is one fixed file. This can be opened up later.

## Anti-goals

- The catalogue and the site disagree: a button that opens a missing page, or a tutorial the site has that kit never
  shows. The generator and `kit check` exist to prevent this.
- A kit that links to tutorials for a different minor line.
- A reader on an older patch who follows a passage that needs a newer one, without being told.
- `kit tutorials` that is just a list of URLs. It should look inviting: big buttons, readable summaries, visible tags.
- A published release whose links break. A release's links are permanent, since a release is.

## What has been decided

### `projects/tutorials`

- A TypeScript, React, and Vite site. Tutorials are MDX files.
- Each MDX file starts with a front-matter block of metadata. The front matter is the single source of each
  tutorial's title, summary, and audience tags; the catalogue is generated from it.

### Two kinds of pages

- **Top-level tutorials.** Focused, one per feature, tagged by audience. These, and only these, are in the
  catalogue, and kit shows them.
- **Deeper pages.** Detail that a top-level tutorial links to. They are on the site but not in the catalogue.

The front matter marks which kind a page is.

### One site per minor line

- The tutorials ignore the patch version: kit 0.4.7 opens `https://kit.empowerite.dev/tutorials/0.4/`, the same
  site every 0.4.x release opens.
- The site lives on a domain empowerite owns, not a host's own domain, so it can move hosts without breaking the
  links built into published releases.
- Each release rebuilds its minor line's site from its own tag. A fix to a tutorial ships in the next patch release,
  like any other fix; nothing is edited by hand after a release, and what is online always matches a tag.
- A line's site changes only when a release in that line is cut, so once 0.5.0 is out, `/0.4/` stays as 0.4's last
  patch left it, with no extra machinery.

### Tutorials that know their patch

- A passage that a patch release made true is marked with the version it arrived in, by an MDX component, as
  `<Since version="0.4.9">`.
- kit puts its exact version in every link it opens, as `/tutorials/0.4/<path>?kit=0.4.7`. A page that knows the
  reader's version flags the passages newer than it, says which version they need, and says how to upgrade. A page
  opened without it, from a browser's history say, shows the marks plainly.
- Each line's site has, in its top navigation, **What's new in 0.4**, the minor release's story with links to the
  tutorials it added or changed, and **Changelog**, each 0.4.x release, newest first, with what changed and which
  tutorials it touched.
- *My call:* the changelog is written by hand, one MDX entry per release, as part of cutting it; it is not scraped
  from commits or the forge's release notes. `kit release` refuses a version with no entry.

### The catalogue

- It lists the top-level tutorials. It is everything kit knows about the site.
- `kit generate` writes it from the tutorials' front matter, through a `kit:generate` region, as the kit's other
  generated content is written ([docs/generate.md](../../generate.md)). `kit check` then fails when the catalogue is
  stale.

### `kit tutorials`

- Reads the catalogue and shows big buttons, filterable by audience tag. Choosing one opens the tutorial in the
  browser.
- The 🎓 Tutorials item in kittui's app menu, dim today (`projects/tui/internal/shell/shell.go`), opens this screen.

### Permanent copies and temporary ones

- **Permanent:** a release's line, at `https://kit.empowerite.dev/tutorials/<major>.<minor>/`. Its links are built
  into a release that never changes, so they live on a domain empowerite owns.
- **Temporary:** pre-releases and `main`, on this repository's GitHub Pages, as
  `https://empowerite.github.io/kit/tutorials/0.5.0-rc.1/` and `…/tutorials/main/`. Their links need not last,
  since the pre-release is deleted and `main` moves.
- kit picks the base from its own version: a version with a suffix, as `0.5.0-rc.1`, links to its pre-release's copy
  on Pages; `unreleased`, a checkout, links to `main`'s copy on Pages; any other version links to its line on
  `kit.empowerite.dev`.
- The site is built the same way for both; only its base path differs.

*My calls:*

- Every push to `main` deploys `main`'s copy. So a checkout's buttons almost always find their pages, and no desk
  needs Node to run kit.
- A pre-release's copy is removed when the pre-release is retired, by `retire-pre-releases.yml`. A pre-release never
  writes to its line's permanent copy; only 0.5.0 itself writes `/0.5/`.

## Constraints from the existing codebase

- `kit/` is the release. Everything in it is written for a consumer, and paths in it read as they do in the tarball.
  The catalogue ships in `kit/`, so the tarball needs nothing new.
- kittui reads what it shows from kit as JSON and never parses kit's text. The tutorials screen reads the catalogue
  through `kit tutorials --json`, as the flow screen reads `kit flow <stage> --json`.
- kittui already opens pages with `open` on macOS and `xdg-open` elsewhere (`browse` in
  `projects/tui/internal/flow/flow.go`); `kit/modules/bash/new.sh` has the shell equivalent, `new:browse`. Reuse them.
- In kittui every action needs a plain click or a key: iTerm swallows ⌃, ⌥, and right clicks. Commands that cannot
  act right now stay visible and dim, never hidden.
- A verb is a script in `kit/bin/kit-<verb>` with the standard header, usage, and exit codes. It has bats tests in
  `tests/`, and its kittui screen has golden tests.
- `kit generate` runs its command in bash from the repository root. `kit check` runs every region's command, so the
  generator runs in CI and in the pre-commit hook.
- `kit version` prints the running kit's version, or `unreleased` in a checkout without a version file.
- Releases are `release/<version>` tags; pre-releases are `release/<version>-<suffix>` and are deleted when their
  branch is gone ([README.md](../../../README.md) § Release). The release workflow is
  `.github/workflows/release.yml`; retiring pre-releases is `.github/workflows/retire-pre-releases.yml`.
- `empowerite/infrastructure` manages this repository's settings and the `empowerite.dev` domain on Cloudflare,
  through Terraform. The DNS name and the hosting for `kit.empowerite.dev` are made there, not here.

## Left to the builder

These are open; a sensible builder can settle them. Items marked *my call* are proposals, not decisions the user
made.

- **Front-matter fields.** *My call:* `title`, `summary`, `audience` (a list of tags), `order`, and `top` (true for a
  top-level tutorial). The tags come from a closed list defined in the tutorials project and checked, so a typo
  cannot create a new audience.
- **Catalogue format and place.** JSON has no comments, so it cannot hold a `kit:generate` region. *My call:* a YAML
  file under `kit/` (YAML has `#` comments; kit already uses yq), whose body is the region. `kit tutorials --json`
  turns it into JSON.
- **The generator.** *My call:* a script that reads each MDX file's front matter with `yq --front-matter=extract`, so
  `kit check` does not need Node or `npm install` to run it.
- **Hosting under the domain.** *My call:* Cloudflare Pages, which `empowerite/infrastructure` already plans. A
  deploy of one line must leave every other line's directory as it is; how (one project per line, an assembled
  deploy, or object storage behind the domain) is the builder's.
- **Publishing to GitHub Pages.** A deploy of one temporary copy must leave the others as they are, and retiring a
  pre-release removes only its own; how (a `gh-pages` branch of directories, or an assembled Actions deploy) is the
  builder's.
- **The screen's layout.** Card size, how tags are shown and filtered, keyboard order, and the text fallback
  (`kit tutorials --text`, or no terminal) follow kittui's existing screens.
- **Arguments.** *My call:* `kit tutorials [<id>]` opens one tutorial directly, and `--tag <tag>` filters.
- **The first tutorial's subject.** Which feature v0.1.0's tutorial teaches.
