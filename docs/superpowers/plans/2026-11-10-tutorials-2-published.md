# Tutorials 2: the site is published and its links resolve Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every link `kit tutorials` opens leads to a page: the site is built and published for `main` and for each
`release/*` tag, its build refuses broken pages, and kit and kittui say only what happened.

**Architecture:** GitHub Pages serves this repository's `gh-pages` branch, one directory for each copy of the site,
`tutorials/<line>/`. A repository-owned workflow builds one copy with a read-only token and hands `dist/` to a second
job, which replaces that directory in one commit. A branch of directories is chosen over the artifact deploy
(`actions/deploy-pages`), which replaces the whole site, so every run would rebuild or fetch every other copy; on a
branch the other copies are untouched by construction, and retiring a pre-release is one `git rm`.
`kit/modules/bash/tutorials.sh` writes the site's address once, and the workflow asks it.

**Tech Stack:** bash, jq, git, bats; TypeScript, React, Vite, vitest; Go, Bubble Tea; GitHub Actions and Pages.

**Spec:** [the tutorials design](../specs/2026-11-06-tutorials-design.md), lines 67 to 87 and 104 to 121, with
parts 2 and 3 of issue #86. Its scheme is kept as written: "kit 0.4.7 opens
`https://kit.empowerite.dev/tutorials/0.4/`, the same site every 0.4.x release opens"; "a version with a suffix, as
`0.5.0-rc.1`, links to its pre-release's copy on Pages; `unreleased`, a checkout, links to `main`'s copy"; and "kit
puts its exact version in every link it opens, as `/tutorials/0.4/<path>?kit=0.4.7`". Only the host is open. The
changelog gate, lines 88 and 89, is left to the Release readiness plan: `kit release` and `bin/release verify` are
that plan's, and `kit/` names no repository.

**Review:** [the findings](../reviews/2026-11-10-code-review-findings.md) of the review of 2026-11-10, cited
here by area and number: Serve 8 is item 8 under its Serve heading.

## Decisions for the owner

1. Where a release's line is served. The spec: "The site lives on a domain empowerite owns, not a host's own domain,
   so it can move hosts without breaking the links built into published releases." Options: (a) the spec's two
   hosts, which adds a Cloudflare deploy, a secret, and DNS before 0.1.0; (b) one site on GitHub Pages,
   `https://empowerite.github.io/kit/tutorials/<line>/`, with no DNS and nothing outside this repository; (c) that
   Pages site under `kit.empowerite.dev`, one CNAME record. Written on (b): it is live the day Pages is on. Moving
   to (c) is one line in `tutorials.sh`, the record, and a rebuild of each copy; GitHub redirects the old address.
2. What "not reachable" means in kittui. Default: before it asks the browser, kittui asks the page for its head, for
   three seconds at most, and gives the link when nothing answers. The other option makes no request and claims
   less, "Asked the browser for …"; it costs nothing offline and never doubts a proxy that works.

## Global Constraints

- kit commit only, never kit submit or push; every subject starts with `#86`, the issue for this plan.
- `export PATH=$PWD/kit/bin:$PATH` before `kit commit`.
- Peers share the worktree: stage named paths only, guard that the index holds only them, never leave anything staged.
- Everything under `kit/` is written for a consumer, and a path in it is written from `kit/`: `modules/bash/x.sh`.
- Lines are 120 cells at most.
- A verb's header comment is its help page and its usage line is its grammar (`kit syntax --check` holds them): a
  new flag is added to the header's usage line and described there, and `docs/` says the same.
- `.github/workflows/tutorials.yml` is this repository's own file, as `release.yml` is; `kit-ci.yml` is an installed
  copy and is not touched. Every `uses:` is `owner/repo@<40-hex>  # v<version>`, which `kit lint` holds.
- `projects/tui` stays at or above 98% statement coverage. No test reaches the network or opens a browser.
- Bats runs as `KIT_TOOLCHAIN=mise kit/bin/kit tools run bats <file>`; vitest as `npx vitest run` in
  `projects/tutorials/`, after `npm ci --no-audit --no-fund`; Go as `go test ./internal/tutorials/` in
  `projects/tui/`. A test block below gives each test's name and assertions; a comment gives what it sets up.

## Review Focus

1. A push to `main` and a tag pushed at the same moment: the second push to `gh-pages` is refused, and `publish`
   applies its one directory to the branch as it then stands. (Task 4, "a push that is refused is tried again".)
2. An address that is no page, typed or from a bookmark to a page a later patch removed: Pages answers with the
   site's not-found page and a link to the tutorials, not a blank frame. (Task 3, `entry` and `NotFound`; Task 4.)
3. `TUTORIALS_BASE` without its slashes, or a whole URL: the first is given them, the second refused. (Task 2.)
4. An offline desk, or a copy not yet published, as a checkout before the first deploy or a retired pre-release:
   kittui gives the link and says the site did not answer. (Task 6, `TestOnlyAPageThatAnswersIsOpened`.)
5. A pre-release of the next line, or an old tag run again: each writes its own directory, never its line's, and
   `retire` removes only a pre-release whose tag is gone. (Task 4, the `site` and `retire` tests.)

---

## Task 1: `kit tutorials` links to one site, and exits as its header says

**Files:** Modify `kit/modules/bash/tutorials.sh:10-16,23-39,77`, `kit/bin/kit-tutorials:2-28,71-75`,
`tests/tutorials.bats`, `docs/tutorials.md:11,22,29-32,50-54`, `README.md:123-124`, and
`projects/tutorials/README.md:7-11`.

**Interfaces:** Produces `tutorials:site`, which prints `https://empowerite.github.io/kit/tutorials/`, the one place
the host is written; `tutorials:line <version>`, which prints `main` for `unreleased`, the version for one with a
suffix, and `<major>.<minor>` for a release; and `tutorials:base <version>`, `KIT_TUTORIALS_BASE` or `<site><line>/`.

- [ ] **Step 1: Write the failing tests.** In `tests/tutorials.bats`, set
  `site=https://empowerite.github.io/kit/tutorials` above `setup`, write `${site}` for every
  `https://kit.empowerite.dev/tutorials`, replace the first test, and add the other two (Command line 23):

```bash
# @test "the line and the base follow the version, on one site"
run bash -c "source '${kit_root}/modules/bash/1st.sh'; use tutorials
  for v in unreleased 0.1.0-rc.7 0.4.12; do tutorials:line \"\${v}\"; tutorials:base \"\${v}\"; done"
[ "$output" = "$(printf '%s\n' main "${site}/main/" 0.1.0-rc.7 "${site}/0.1.0-rc.7/" 0.4 "${site}/0.4/")" ]
# @test "a missing catalogue is 1, whatever is named": for word in --text, --json, and first
KIT_TUTORIALS_CATALOGUE=/nonesuch run --separate-stderr "${kit_root}/bin/kit" tutorials "${word}"
[[ "$status" -eq 1 && "$stderr" == *"no catalogue of tutorials at /nonesuch"* ]]
# @test "the grammar gives a named tutorial no --json"
[ "$("${kit_root}/bin/kit" syntax tutorials --match -- --json first | jq .status)" -eq 2 ]
[ "$("${kit_root}/bin/kit" syntax tutorials --match -- --text --tag ui first | jq .status)" -eq 0 ]
```

- [ ] **Step 2: Run `tests/tutorials.bats`.** Expected: the first fails with `tutorials:line: command not found`, the
  second on `first`, status 2, the third on status 0, and four more on the old host.
- [ ] **Step 3: Change the module and the verb.** Write the three functions. `tutorials:choices` returns 1 where the
  catalogue is missing, so the guard asks nothing and `tutorials:json` refuses with 1. The usage becomes two lines,
  `kit tutorials [--text | --json | --jq] [--tag <audience>…]` and `kit tutorials [--text] [--tag <audience>…]
  <tutorial>`, and lines 71 to 75 go. Line 2 reads `lists kit's tutorials, and opens one, in kittui or the browser`.
  The docs name the one host, and say what the header says (Documents 4): on a terminal a tutorial named opens
  kittui's screen on its card; with `--text`, or in a pipe, its title and link print and it opens in the browser.
- [ ] **Step 4: Run it again**, then `kit syntax --check`. Expected: all pass.
- [ ] **Step 5: Commit** the six files, subject `#86 kit: the tutorials are one site, and exit 1 without a catalogue`.

## Task 2: The build checks what it built

**Files:** Create `projects/tutorials/src/build.ts`, `src/build.test.ts`. Modify `vite.config.ts:28-39,43,54-58`,
`package.json:7`, `README.md:28-31`, `bin/ci:13`. Delete `scripts/verify-dist.mjs` and
`src/content/kit-tutorials/dollar-signs.mdx`, a fixture shipped as a page (Infrastructure 32).

**Interfaces:** Produces, in `src/build.ts`, which only `vite.config.ts` and the tests import:

- `siteBase(value: string | undefined): string`: `/` for nothing; else one leading and one trailing slash, runs of
  slashes made one; it throws for a value with `:`, `?`, `#`, or a space (Infrastructure 19).
- `filled(shell: string, route: { path: string; title: string }, page: string): string`: the shell with the title,
  escaped, and the page, both put in by function; it throws when the page has no `<article`.
- `verify(dist: string, base: string, paths: string[]): Promise<string[]>`: the problems, each naming its file. A
  path names `<path>index.html`, or itself when it ends in `.html`. Every `href` and `src` in every `.html` under
  `dist` that is not a fragment, `//…`, or a URL with a scheme is resolved against its page's address under the
  base, its query and fragment dropped, and must be inside the base and name a file (Infrastructure 16, 18).

- [ ] **Step 1: Write the failing tests** in `src/build.test.ts`. `dist(files)` writes the files, less those that
  are `null`, under a fresh `mkdtemp` directory and returns it; `shell` is `index.html`'s title and `<!--page-->`.

```ts
// test "verify passes a whole dist, and names each break"
const page = (body = "") => `<link href="/b/assets/site.css"><div id="root"><article>${body}</article></div>`;
const whole = { "index.html": page('<a href="/b/a/?kit=0.1.0#top">a</a>'), "a/index.html": page(), "assets/site.css": "" };
const broken = async (change: Record<string, string | null>) =>
  (await verify(dist({ ...whole, ...change }), "/b/", ["", "a/"])).join("\n");
expect(await broken({})).toBe("");
expect(await broken({ "assets/site.css": null })).toContain('href="/b/assets/site.css" has no file');
expect(await broken({ "a/index.html": '<div id="root"></div>' })).toContain("dist/a/index.html: an empty page");
expect(await broken({ "index.html": null })).toContain("no dist/index.html");
expect(await broken({ "a/index.html": page('<a href="/b/gone/">x</a>') })).toContain('href="/b/gone/" has no file');
expect(await broken({ "a/index.html": page('<img src="/out.png">') })).toContain('src="/out.png" is outside /b/');
// test "siteBase gives a base its slashes, and refuses what is not a path"
expect([undefined, "/k/main", "k/main/", "//k//main//"].map(siteBase)).toEqual(["/", "/k/main/", "/k/main/", "/k/main/"]);
expect(() => siteBase("https://x.test/k/")).toThrow("TUTORIALS_BASE is a path");
// test "filled keeps a page's dollar signs, and refuses an empty page"
expect(filled(shell, { path: "x/", title: "$&" }, "<article>echo $$ $& $'x'</article>")).toContain("<article>echo $$ $& $'x'</article>");
expect(() => filled(shell, { path: "x/", title: "X" }, "")).toThrow("x/: the page rendered empty");
```

- [ ] **Step 2: Run `npx vitest run src/build.test.ts`.** Expected: fail, `./build` does not resolve.
- [ ] **Step 3: Write `src/build.ts`, and use it in `vite.config.ts`.** `base` is
  `siteBase(process.env.TUTORIALS_BASE)`; `prerender` fills each route through `filled`, puts the route's path
  before the message of a render that throws, and then throws `verify`'s problems over the routes' paths. `check`
  in `package.json` ends at `TUTORIALS_BASE=/kit/tutorials/main/ vite build`. The README and `bin/ci:13` say so.
- [ ] **Step 4: Run `npm run check`.** Expected: it passes, and `dist/kit-tutorials/dollar-signs/` is gone.
- [ ] **Step 5: Commit** them, subject `#86 tutorials: the build refuses an empty page, a dead link, and a bad base`.

## Task 3: Links from the page, a not-found page, and `<Since>`

**Files:** under `projects/tutorials/`. Modify `src/reader.tsx:20-33`, `src/routes.tsx:37,116-133`,
`src/main.tsx:19-23`, `src/version.ts:10,27`, `src/components/Since.tsx:13`, `src/whats-new.mdx:5`,
`src/changelog/0.1.0.mdx:5`, `vite.config.ts`, `README.md:59-61,91-93`, `src/version.test.ts:12`, and
`src/components/Since.test.tsx`. Create `src/entry.ts` and the three test files the block names.

**Interfaces:** Produces `PageLink`, MDX's `a`, whose relative `href` resolves against the page it is on; `SiteLink`
stays relative to the base, for the layout and the home page; `NotFound`, the route `*`; and
`entry(prerendered: boolean, pathname: string, base: string): "hydrate" | "render" | "leave"`.

- [ ] **Step 1: Write the failing tests:**

```tsx
// src/reader.test.tsx, test "a page's link resolves against the page, the layout's against the base, and both keep ?kit="
// in <MemoryRouter initialEntries={["/kit-tutorials/?kit=0.4.7"]}>; href(name) is that link's href attribute
<PageLink href="marks/">marks</PageLink> <PageLink href="../changelog/#0.4.7">changelog</PageLink>
<PageLink href="https://example.test/">out</PageLink> <SiteLink href="whats-new/">new</SiteLink>
expect(["marks", "changelog", "out", "new"].map(href)).toEqual(["/kit-tutorials/marks/?kit=0.4.7",
  "/changelog/?kit=0.4.7#0.4.7", "https://example.test/", "/whats-new/?kit=0.4.7"]);
// src/routes.test.tsx, renamed from .ts, test "an address that is no page says so": <App /> at /nonesuch/?kit=0.4.7
expect(screen.getByRole("heading", { name: "No such page" })).toBeTruthy();
expect(screen.getByRole("link", { name: "All tutorials" }).getAttribute("href")).toBe("/?kit=0.4.7");
// src/entry.test.ts: a built page hydrates, dev's empty root renders, Pages' 404 for another copy stands as built
expect(entry(true, "/kit/tutorials/main/x/", "/kit/tutorials/main/")).toBe("hydrate");
expect(entry(false, "/", "/")).toBe("render");
expect(entry(true, "/kit/tutorials/0.1/gone/", "/kit/tutorials/main/")).toBe("leave");
// src/version.test.ts, in place of line 12 (Infrastructure 17); then Since.test.tsx's first test
expect(sinceState("0.5.0", "0.5.0-rc.1")).toBe("met");
expect(sinceState("0.5.1", "0.5.0-rc.1")).toBe("newer");
expect(screen.getByText(/To upgrade, install kit 0\.4\.9 or later, then run/)).toBeTruthy();
```

- [ ] **Step 2: Run `npx vitest run`.** Expected: the five files fail.
- [ ] **Step 3: Write them.** `main.tsx` calls `entry(root.firstElementChild !== null, location.pathname,
  import.meta.env.BASE_URL)`. `prerender` also writes `dist/404.html` from `render("404.html")`, and gives `verify`
  that path. The spec's line 83, "says which version they need, and says how to upgrade", is a second paragraph in
  `<Since>` when the reader's kit is older: `To upgrade, install kit <version> or later, then run <code>kit
  update</code>.` The links in `whats-new.mdx` and `changelog/0.1.0.mdx` become `../kit-tutorials/`, and the README
  says a link is written from its page, as on the web.
- [ ] **Step 4: Run `npm run check`.** Expected: it passes, and `dist/404.html` holds "No such page".
- [ ] **Step 5: Commit** them, subject `#86 tutorials: links resolve from their page, and no page says so`.

## Task 4: `bin/tutorials` refuses a page it cannot place, and gains `site`, `publish`, and `retire`

**Files:** Modify `bin/tutorials:2-17,44-50,73-79`, `tests/tutorials-catalogue.bats`, `README.md:275`, its `bin/` row.

**Interfaces:** Consumes `tutorials:line` and `tutorials:base` (Task 1). Produces:

- `bin/tutorials site <ref>` prints four `key=value` lines, as `GITHUB_OUTPUT` reads them, for `main` or a
  `release/<version>` tag, and refuses any other ref with 1. It alone sources `kit/modules/bash/1st.sh`, as
  `tests/tutorials.bats`' first test does, and it ignores `KIT_TUTORIALS_BASE`.
- `bin/tutorials publish <directory> <dist>` clones `gh-pages` from `TUTORIALS_PAGES`, by default origin's URL, or
  starts the branch where there is none; replaces `<directory>` whole; keeps an empty `.nojekyll`; for
  `tutorials/main`, copies `<dist>/404.html` to the root; commits as `github-actions[bot]` when anything changed; and
  pushes. A refused push starts again from a fresh clone, five times at most. It exits 2 for a directory not shaped
  `tutorials/<name>`, and 1 for a dist with no `index.html` or a fifth refusal.
- `bin/tutorials retire` removes each `tutorials/<version>-<suffix>` whose tag `release/<version>-<suffix>` is not in
  `TUTORIALS_PAGES`, in one commit, printing `retired <directory>` for each.

- [ ] **Step 1: Write the failing tests**, with `site` as Task 1 sets it and four helpers. `broken <page> <text>`
  runs `catalogue` over a copy of the `ok` fixture with that one more page, its text written by `printf '%b'`.
  `pages` makes a bare repository and exports `TUTORIALS_PAGES`. `built <text>` makes and prints
  `${BATS_TEST_TMPDIR}/dist-<text>`, whose `index.html` and `404.html` say `<text>`. `online <path>` prints a file
  of the branch, by `git --git-dir="${TUTORIALS_PAGES}" show "gh-pages:$1"`, and `listed` is `ls-tree` there.

```bash
# @test "a page the catalogue cannot place is refused, naming the file and why" (Infrastructure 30): status 1 each time
broken alpha/bare.mdx 'No front matter here.\n'
[[ "${stderr}" == *"alpha/bare.mdx: a page starts with front matter that has a title"* ]]
broken gamma/index.mdx '---\ntitle: G\ntop: "true"\nsummary: s\nglyph: g\naudience: [ui]\n---\n'
[[ "${stderr}" == *'gamma/index.mdx: top is true or false, not "true"'* ]]
broken 'My Page/index.mdx' '---\ntitle: G\ntop: true\nsummary: s\nglyph: g\naudience: [ui]\n---\n'
[[ "${stderr}" == *"My Page/index.mdx: a tutorial's id is lower-case letters, digits, and hyphens"* ]]
# @test "site says which copy a ref builds, and a pre-release never names its line's"
run "${repo_root}/bin/tutorials" site release/0.4.7
[ "$output" = "$(printf '%s\n' line=0.4 directory=tutorials/0.4 base=/kit/tutorials/0.4/ url="${site}/0.4/")" ]
KIT_TUTORIALS_BASE=http://localhost:5173/ run "${repo_root}/bin/tutorials" site release/0.5.0-rc.1
[[ "$output" == "line=0.5.0-rc.1"*"directory=tutorials/0.5.0-rc.1"*"base=/kit/tutorials/0.5.0-rc.1/"* ]]
run "${repo_root}/bin/tutorials" site main
[[ "$output" == "line=main"*"directory=tutorials/main"* ]]
run --separate-stderr "${repo_root}/bin/tutorials" site feature/x
[[ "$status" -eq 1 && "$stderr" == *"feature/x has no copy of the tutorials: main and a release tag have one"* ]]
# @test "publish replaces one copy whole, leaves the others, and main's gives the site its 404 page"
# pages; old.html is added to "$(built one)"; tutorials/main is published from one, three, and three; tutorials/0.1 from two
[ "$(online tutorials/main/index.html)$(online tutorials/0.1/index.html)$(online 404.html)" = threetwothree ]
[ "$(listed -r --name-only gh-pages | grep -c -e '^\.nojekyll$' -e old.html)" -eq 1 ]
[ "$(git --git-dir="${TUTORIALS_PAGES}" rev-list --count gh-pages)" -eq 3 ]
# then `publish ../x "$(built one)"` has status 2, and `publish tutorials/0.2 "${BATS_TEST_TMPDIR}"` has status 1
# @test "a push that is refused is tried again on the branch as it then stands"
# pages; tutorials/main is published; then the bare repository gets an executable update hook that refuses once:
printf '#!/bin/sh\n[ -e refused ] && exit 0\ntouch refused\nexit 1\n' > "${TUTORIALS_PAGES}/hooks/update"
"${repo_root}/bin/tutorials" publish tutorials/0.1 "$(built two)"
[[ -e "${TUTORIALS_PAGES}/refused" && "$(online tutorials/0.1/index.html)" == two ]]
# @test "retire removes the copy of each pre-release whose tag is gone, and nothing else"
# pages; main, 0.1, 0.1.0-rc.7, and 0.1.0-rc.8 are published; the bare repository gets the tag release/0.1.0-rc.8
run "${repo_root}/bin/tutorials" retire
[[ "$status" -eq 0 && "$output" == *"retired tutorials/0.1.0-rc.7"* ]]
[ "$(listed --name-only gh-pages tutorials/ | paste -s -d ' ' -)" = "tutorials/0.1 tutorials/0.1.0-rc.8 tutorials/main" ]
```

- [ ] **Step 2: Run `tests/tutorials-catalogue.bats`.** Expected: the first fails, the page skipped and the status
  0, and the other four fail with `usage: bin/tutorials catalogue`.
- [ ] **Step 3: Write them.** In `catalogue`, the first two refusals stand before line 47's `continue`, and the id's
  after line 50, held to `^[a-z0-9][a-z0-9-]*$`. `publish` and `retire` need git alone. The header and the usage
  name all four subcommands; the test file's line 2 and README's `bin/` row name `bin/tutorials`.
- [ ] **Step 4: Run it again**, then `kit lint bin/tutorials` and `kit generate --check kit/modules/tutorials.yml`.
  Expected: all pass.
- [ ] **Step 5: Commit** the three files, subject `#86 tutorials: bin/tutorials publishes and retires a copy`.

## Task 5: The workflow

**Files:** Create `.github/workflows/tutorials.yml`. Modify `.github/workflows/retire-pre-releases.yml:1-8,28-31`,
`README.md` § Release, `projects/tutorials/README.md` (a section, Publishing).

- [ ] **Step 1: Write `.github/workflows/tutorials.yml`**, with a blank line between its top-level keys, jobs, and
  steps, and the closing line of dashes, as `release.yml` has them:

```yaml
# tutorials.yml - builds one copy of the tutorials site and publishes it as a directory of the gh-pages branch
# ----------------------------------------------------------------------------------------------------------------------
# A push to main publishes tutorials/main/; a release tag, release/0.1.0, its line's, tutorials/0.1/; a pre-release
# tag, release/0.1.0-rc.8, its own. GitHub Pages serves the branch. The build runs npm's packages, so its job only
# reads; the job that writes runs none of them. A newer push to the same ref replaces an older run, and two refs at
# once both land, since bin/tutorials publish tries a refused push again. see: bin/tutorials
name: tutorials
on:
  push:
    branches: [main]
    tags: ['release/*']
  workflow_dispatch:
permissions: {}
concurrency:
  group: tutorials-${{ github.ref }}
  cancel-in-progress: true
jobs:
  build:
    runs-on: ubuntu-latest
    timeout-minutes: 15
    permissions:
      contents: read
    outputs:
      directory: ${{ steps.site.outputs.directory }}
    steps:
      - name: checkout
        uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1  # v7.0.1
        with:
          persist-credentials: false
      - name: tools
        uses: jdx/mise-action@c2a87611a18de5b3828c5652fe268e992400cb5c  # v4.3.0
      - name: site
        id: site
        run: bin/tutorials site "${GITHUB_REF_NAME}" >> "${GITHUB_OUTPUT}"
      - name: build
        working-directory: projects/tutorials
        env:
          TUTORIALS_BASE: ${{ steps.site.outputs.base }}
          TUTORIALS_LINE: ${{ steps.site.outputs.line }}
        run: npm ci --no-audit --no-fund && npm run build
      - name: upload
        uses: actions/upload-artifact@cf430e030ddbb5b0abf93d22962f4752f3646cd9  # v7.0.2
        with:
          name: site
          path: projects/tutorials/dist/
          if-no-files-found: error
          retention-days: 1
  publish:
    needs: build
    runs-on: ubuntu-latest
    timeout-minutes: 10
    permissions:
      contents: write
    steps:
      - name: checkout
        uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1  # v7.0.1
        with:
          persist-credentials: false
      - name: download
        uses: actions/download-artifact@9000827ccba6bdab643e8b6fd33ac0654aef8333  # v8.0.2
        with:
          name: site
          path: ${{ runner.temp }}/site
      - name: publish
        env:
          GH_TOKEN: ${{ github.token }}
          DIRECTORY: ${{ needs.build.outputs.directory }}
        run: gh auth setup-git && bin/tutorials publish "${DIRECTORY}" "${RUNNER_TEMP}/site"
```

- [ ] **Step 2: Add a step to `retire-pre-releases.yml`**, after `retire` and with the same `env`: its name is
  `retire the tutorials`, and it runs `gh auth setup-git && bin/tutorials retire`. The header gains a sentence on it.
- [ ] **Step 3: Write the two README passages**: which ref publishes which copy, where it is served, and that a
  pre-release's copy goes with its tag.
- [ ] **Step 4: Run `kit lint .github/workflows/tutorials.yml .github/workflows/retire-pre-releases.yml`.**
  Expected: actionlint and `pinned` pass. The four SHAs are their tags' commits, read from GitHub on 2026-11-10.
- [ ] **Step 5: Commit** the four files, subject `#86 tutorials: a workflow publishes each copy to gh-pages`.

Nothing in either workflow file runs before the branch is pushed, since this one starts only from `main` or a
`release/*` tag. Unexercised until then: mise on the runner, the artifact between the jobs, `gh auth setup-git` with
the job's token, the push to `gh-pages`, Pages' own build, and the served addresses. The checklist is their test.

## Task 6: kittui's screen says what happened, and takes the keys a list takes

**Files:** Modify `projects/tui/internal/tutorials/tutorials.go:1-26,47-51,98-118,173-178,193-214,328-350,438-441`,
`view.go:67-80,120-124`, `helpers_test.go:46-61`, `tutorials_test.go`, and the five goldens' last line.

**Interfaces:** Produces `reachable(ctx context.Context, url string) error`, a HEAD within `reachLimit`, three
seconds, nil for a status under 400; `Model.reach`, which `New` sets to it and `fake` to a recorder (`browser` gains
`asked []string` and `unreachable error`); `errNoLink`; and `browsedMsg.silent`. `open` gives `errNoLink` for an
empty URL, then `reach`'s error with `silent` set, then `browse`'s, and asks no further after the first.

- [ ] **Step 1: Write the failing tests** (Apps 23, 24):

```go
func TestOnlyAPageThatAnswersIsOpened(t *testing.T) {
    c := eight()
    c.Tutorials[0].URL = ""
    m, b := fake(t)
    m = drive(m, at120, catalogue(c), enter)
    if len(b.asked)+len(b.urls) != 0 || lines(m)[29] != " Tutorial 1 has no link in the catalogue" {
        t.Errorf("no link: asked %q, opened %q, said %q", b.asked, b.urls, lines(m)[29])
    }
    b.unreachable = errors.New("404 Not Found")
    m = drive(m, tabKey, enter)
    if len(b.urls) != 0 || lines(m)[29] != " No answer from https://example.test/t2/?kit=unreleased" {
        t.Errorf("no answer: opened %q, said %q", b.urls, lines(m)[29])
    }
}
```

  It ends: after `m.note.FadeMsg()` and a resize to 30×20, line 19 still starts with `No answer from`, after its
  leading space. Four more.
  `TestHomeEndAndThePageKeysMoveTheCursor`: with `withEight` at 60×24, one card a row and two rows a page, the keys
  PgDn, PgDn, End, PgDn, PgUp, Home, and PgUp leave the cursor on 2, 4, 7, 7, 5, 0, and 0, the cards keeping the
  keyboard; after esc, End leaves the cursor on 0 and the bar with the keyboard.
  `TestReachableAsksForThePagesHead`, against an `httptest` server: nil for a path that answers 200; an error for
  one that answers 404, for the server once closed, and for the URL `::`; and every request a HEAD.
  `TestWhatTheCatalogueLacksIsSaid`: over `testdata/fake-kit`, the words `--tag=wizard nonesuch` leave
  `Not in the catalogue: audience wizard, tutorial nonesuch` on line 29, after its leading space, until a key, and
  `--tag=ui kittui-tour` leaves tab 2, cursor 0, and no notice. In `TestTheWordsAreTheTagsAndTheTutorialNamed`, the
  row `"--tag=a t1": {[]string{"a"}, "t1"}`.

- [ ] **Step 2: Run the package's tests.** Expected: they do not compile, `b.asked` undefined.
- [ ] **Step 3: Write it.** The page keys move `rows()*columns()` cards, held to the first and the last. The keys
  bar's cards segment gains `shell.Bind("⇞⇟", "page", "pgup", "pgdown")` and takes `home` and `end`, as welcome's
  does. `parse` reads `--tag=<audience>`. `take` leaves the "Not in the catalogue" notice, not done, after the first
  catalogue. `bottom` clips to the line a notice whose first word is not `Open`. The package comment says all of it.
- [ ] **Step 4: Run `go test ./internal/tutorials/ -update`, read the goldens' diff**, the keys bar's line alone,
  then `go test -cover ./...`. Expected: all pass, and the package's coverage is no lower than it was.
- [ ] **Step 5: Commit** them, subject `#86 kittui: the tutorials screen opens only a page that answers`.

## Task 7: The first real tutorial

**Files:** Create `projects/tutorials/src/content/first-change/index.mdx`, `first-change/refusals.mdx`. Modify
`src/whats-new.mdx`, `src/changelog/0.1.0.mdx`, `kit/modules/tutorials.yml` (by `kit generate`).

The spec leaves the subject to the builder. It is the loop every other feature hangs on, which `kit welcome` names
and never walks; a tour of kittui, the other candidate, needs screenshots this plan does not build.

- [ ] **Step 1: Write `first-change/index.mdx`**, "Your first change with kit", `top: true`, `audience: [newcomer]`,
  `glyph: 🚀`, `order: 2`, to this outline. Every command's output is pasted from one run in a scratch repository.

  1. Who it is for and what it needs: `kit welcome` finished, and one open issue, a throwaway one for a first try.
  2. Where the work stands: `kit flow`, then `kit flow backlog`, and what a stage is.
  3. Start: `kit work <N>` makes the branch, the worktree under `worktrees/`, and the draft pull request.
  4. Change one file there; `kit context <path>` names the rules that apply to it.
  5. Commit: `git add <path>`, then `kit commit --message "#<N> docs: …"`, and what kit checked first.
  6. Submit: `kit submit` pushes; `kit flow changes` and `kit flow builds` show the pull request and its check.
  7. `kit gate` runs the same gate at the desk, before the runner does.
  8. `kit rebase`, for when the default branch moved under the work.
  9. Merge on the forge, then `kit tidy`; `kit flow` no longer shows the work.
  10. A link to `refusals/`, the deeper page: each refusal the loop gives, its line as kit prints it, and its fix.

- [ ] **Step 2: Run `kit generate kit/modules/tutorials.yml`, then `npm run check`.** Expected: the catalogue lists
  two tutorials, and the build finds every link, the two new ones from What's new and the changelog among them.
- [ ] **Step 3: Commit** them, subject `#86 tutorials: the first tutorial, from an open issue to a merged change`.

## The owner's checklist, once

1. Push, then start the first run: merge to `main`, or `kit release 0.1.0-rc.8` on the branch, whose tag carries the
   workflow and also publishes that pre-release. After `gh run watch`, `git ls-remote origin gh-pages` shows the
   branch.
2. Turn Pages on: `gh api --method POST repos/empowerite/kit/pages -f 'source[branch]=gh-pages' -f 'source[path]=/'`.
3. Check: `gh api repos/empowerite/kit/pages --jq .status` prints `built`; `curl --silent --output /dev/null
   --write-out '%{http_code}\n' https://empowerite.github.io/kit/tutorials/main/` prints 200; an address under it that
   is no page prints 404, with "No such page" in its body; and `kit tutorials --text` prints links that open.
4. Record the Pages source in `empowerite/infrastructure`, `terraform/projects/kit`, which manages this repository.
5. For decision 1 (c) alone: a CNAME record, `kit.empowerite.dev` to `empowerite.github.io`, in infrastructure; `gh
   api --method PUT repos/empowerite/kit/pages -f cname=kit.empowerite.dev`; `tutorials:site` and the tests' `site`
   name the new address; then `gh workflow run tutorials.yml --ref <ref>` for `main` and each tag still online.
