# kit tutorials, part 1: the site, the catalogue, and the index — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `projects/tutorials` builds a versioned tutorials site, `kit generate` keeps a catalogue of its top-level
tutorials in `kit/`, and `kit tutorials` (text, JSON, and a kittui screen behind the 🎓 menu item) shows them as big
buttons that open each tutorial in the browser.

**Architecture:** MDX front matter is the single source of each tutorial's metadata. A repository script,
`bin/tutorials catalogue`, reads it with yq and writes the body of a `kit:generate` region in
`kit/modules/tutorials.yml`. `kit tutorials` reads that file, builds each link from kit's own version, and prints text
or JSON or execs kittui, whose new `internal/tutorials` screen reads `kit tutorials --json`.

**Tech Stack:** bash, jq, yq, bats (the verb and generator); Go with Bubble Tea v2 (kittui); TypeScript, React, Vite,
MDX, Vitest, Node from mise (the site).

**Spec:** [docs/superpowers/specs/2026-11-06-tutorials-design.md](../specs/2026-11-06-tutorials-design.md)

**Not in this plan:** publishing to GitHub Pages and Cloudflare, `kit release`'s changelog gate, and the Terraform in
empowerite/infrastructure (part 2); the content of v0.1.0's first tutorial (part 3). This plan builds the site with
one fixture tutorial so every path is exercised.

## Global Constraints

- Links, decided by `kit version`:
  - `unreleased` → `https://empowerite.github.io/kit/tutorials/main/`
  - a version with a suffix, as `0.5.0-rc.1` → `https://empowerite.github.io/kit/tutorials/0.5.0-rc.1/`
  - `X.Y.Z` → `https://kit.empowerite.dev/tutorials/X.Y/`
  - `KIT_TUTORIALS_BASE`, when set, replaces the base (a desk running `npm run dev`, and the tests).
  - A tutorial's link is `<base><path>?kit=<version>`.
- The app's menu reads Welcome, Tutorials, Flow, Help (commit 358c7bd); Tutorials is dim today and becomes enabled.
- kittui: every action works with a plain click or a key; a command that cannot act stays visible and dim.
- kittui's tests keep at least 98% of its statements covered (`bin/ci`).
- Nothing in kit or the site runs a tutorial's commands.
- Everything under `kit/` reads as it will in the tarball; paths in it are written from `kit/`.
- Audiences are a closed list in `projects/tutorials/audiences.yml`: `newcomer` "Just starting", `ui` "UI",
  `maintainer` "Maintainer". Adding one is an edit there.
- Commits: `export PATH="$PWD/kit/bin:$PATH"`, then `kit commit --message "#86 <scope>: <sentence>"
  --message "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>" <paths>`, naming only your own paths. Never
  push or `kit submit`. Other sessions edit this worktree: before touching `projects/tui/internal/shell`,
  `internal/screens`, or `cmd/kittui`, run `git status` and ListAgents and message a peer with changes there.

## Review Focus

1. A narrow terminal (60×20) or more cards than fit: the cards reflow to fewer columns and the pane scrolls with the
   cursor; no card is cut through a glyph. (Task 4: `tutorials-narrow.golden`, `TestTheCursorScrollsThePane`.)
2. An empty catalogue, or a tag no tutorial has: the screen says so in the pane and the tabs still work; `kit
   tutorials` prints one dim line and exits 0. (Task 3: `@test "no tutorials"`; Task 4: `tutorials-empty.golden`.)
3. No browser opener (SSH, a headless Linux box): kittui's notice shows the URL to copy; `kit tutorials <id>` prints
   it. (Task 3: `@test "without an opener, the link prints"`; Task 4: `TestABrowserThatFailsShowsTheLink`.)
4. Every version shape (`unreleased`, `0.1.0-rc.7`, `0.4.12`, `1.0.0`) on the kit side, and a missing, malformed, or
   `unreleased` `?kit=` on the site side. (Task 3: `@test "the base follows the version"`; Task 1: `sinceState`.)
5. A tutorial author's mistakes: an unknown audience, a top-level page without a summary or glyph, two pages with
   one id, a top-level page nested deeper than `<id>/index.mdx`, a malformed `<Since>`: each fails the generator or
   the build, naming the file. (Task 2: the generator's refusals; Task 1: `Since` throws on a bad version.)

---

### Task 1: The site, `projects/tutorials`

**Files:**

- Create: `projects/tutorials/package.json`, `package-lock.json`, `tsconfig.json`, `vite.config.ts`, `index.html`
- Create: `projects/tutorials/audiences.yml`, `projects/tutorials/frontmatter.schema.json`
- Create: `projects/tutorials/src/main.tsx`, `src/routes.tsx`, `src/Layout.tsx`
- Create: `projects/tutorials/src/version.ts`, `src/components/Since.tsx`
- Create: `projects/tutorials/src/content/kit-tutorials/index.mdx` (the fixture), `src/whats-new.mdx`,
  `src/changelog/0.1.0.mdx`
- Create: `projects/tutorials/scripts/verify-dist.mjs`, `projects/tutorials/README.md`
- Test: `projects/tutorials/src/version.test.ts`, `src/components/Since.test.tsx`, `src/Layout.test.tsx`
- Modify: `mise.toml` (pin `node = "24"`, with its line in the "Why each tool is here" list), `mise.lock`,
  `.gitignore` (`/projects/tutorials/node_modules/`, `/projects/tutorials/dist/`), `bin/ci` (header list and body)

**Interfaces:**

- Produces: the content layout Task 2 reads. A top-level tutorial is `src/content/<id>/index.mdx`, served at
  `<id>/`; a deeper page is `src/content/<id>/<page>.mdx`, served at `<id>/<page>/`.
- Produces: front matter, validated by `frontmatter.schema.json` (draft 2020-12): `title` string, required on every
  page; `top` boolean, default false; when `top` is true, `summary` string, `glyph` string, and `audience` array of
  at least one string are required; `order` integer, default 1000.
- Produces: `audiences.yml`, a YAML list of `{id, label}` in display order, with the three in Global Constraints.
- Produces: the build reads `TUTORIALS_BASE` (default `/`) as Vite's `base`, and `TUTORIALS_LINE` (default `main`),
  the line the site is for, as `0.4`, `main`, or `0.5.0-rc.1`.
- Produces: `sinceState(since: string, reader: string | null): "plain" | "met" | "newer"` in `src/version.ts`.

- [ ] **Step 1: Write the failing tests**

`version.test.ts`, `test("sinceState compares the reader's kit with the passage's version")`:
`sinceState("0.4.9", "0.4.7") === "newer"`, `sinceState("0.4.9", "0.4.9") === "met"`,
`sinceState("0.4.9", "0.4.12") === "met"`, `sinceState("0.4.9", null) === "plain"`,
`sinceState("0.4.9", "unreleased") === "plain"`, `sinceState("0.4.9", "garbage") === "plain"`,
`sinceState("0.5.0", "0.5.0-rc.1") === "newer"`, and `sinceState("0.4", "0.4.7")` throws.

`Since.test.tsx`: rendered with `?kit=0.4.7` in the location, `<Since version="0.4.9">x</Since>` shows the text
"Needs kit 0.4.9; you have 0.4.7." and the children (how to upgrade is part 3's words); with no `?kit=`, it
shows "Since 0.4.9" and the
children, and no "Needs".

`Layout.test.tsx`: with `TUTORIALS_LINE` `0.4`, the top navigation has the links "What's new in 0.4" and "Changelog";
with `main`, "What's new" and "Changelog".

- [ ] **Step 2: Run them to see them fail**

Run: `cd projects/tutorials && npx vitest run`
Expected: FAIL, the modules do not exist.

- [ ] **Step 3: Implement the site**

Vite with `@mdx-js/rollup`, `remark-frontmatter`, and `remark-mdx-frontmatter`; React Router routes made from
`import.meta.glob("./content/**/*.mdx")`. Every route is pre-rendered to `dist/<path>/index.html` (a library such as
`vite-react-ssg`), so deep links work on both hosts with no server rewrites. Links between pages are relative to the
base, never absolute. `Since` throws on a version that is not `X.Y.Z`, failing the build on a typo. The fixture
tutorial is about the tutorials themselves (`top: true`, `audience: [newcomer]`, glyph 🎓) and uses `<Since>` once.
`README.md` says how to run it (`npm run dev`, then `KIT_TUTORIALS_BASE=http://localhost:5173/ kit tutorials`), the
front matter, and the content layout.

- [ ] **Step 4: Add `npm run check` and wire it into the gate**

`package.json`'s `check` runs `tsc --noEmit`, `vitest run`, `TUTORIALS_BASE=/kit/tutorials/main/ vite build`, and
`node scripts/verify-dist.mjs /kit/tutorials/main/`. `verify-dist.mjs` fails, naming the file, when a page under
`src/content/` has no `dist/<path>/index.html`, or when an `href` or `src` in `dist/` starts with `/` but not with the
base. `bin/ci` gains, after lychee: ajv over every MDX file's front matter against `frontmatter.schema.json`, with the
same awk extraction it uses for rules, then `(cd projects/tutorials && npm ci --no-audit --no-fund && npm run check)`.

- [ ] **Step 5: Verify**

Run: `cd projects/tutorials && npm run check`
Expected: every test passes, the build succeeds, and verify-dist prints nothing and exits 0.

- [ ] **Step 6: Commit** — subject `#86 tutorials: a site of MDX tutorials, built for one line of kit`.

### Task 2: The catalogue, generated

**Files:**

- Create: `bin/tutorials`
- Create: `kit/modules/tutorials.yml`
- Test: `tests/tutorials-catalogue.bats`, with fixtures under `tests/fixtures/tutorials/`

**Interfaces:**

- Consumes: Task 1's content layout, front matter, and `audiences.yml`.
- Produces: `bin/tutorials catalogue` prints the region's body, YAML, to stdout. `TUTORIALS_PROJECT` (default
  `projects/tutorials`) names the project, so the tests point it at fixtures.
- Produces: `kit/modules/tutorials.yml`, whose body is one region, named `catalogue`, whose command is
  `bin/tutorials catalogue`, holding:

```yaml
audiences:
  - {id: newcomer, label: Just starting}
tutorials:
  - {id: kit-tutorials, path: kit-tutorials/, title: …, summary: …, glyph: 🎓, audience: [newcomer], order: 1000}
```

  Tutorials are the `top: true` pages, sorted by `order`, then `id`.

- [ ] **Step 1: Write the failing tests** in `tests/tutorials-catalogue.bats`:
  - `@test "the catalogue lists the top-level tutorials in order"`: a fixture with two top pages (orders 20 and 10)
    and one deeper page prints the two, order 10 first, and no deeper page.
  - `@test "an unknown audience is refused, naming the file"`: exit 1; stderr names `<id>/index.mdx` and the audience.
  - `@test "a top-level page needs a summary and a glyph"`: one test each, exit 1, naming the file and the field.
  - `@test "a top-level page lives at <id>/index.mdx"`: `top: true` in `a/b.mdx` exits 1, naming it.
  - `@test "the catalogue in kit is current"`: `kit generate --check kit/modules/tutorials.yml` exits 0.

- [ ] **Step 2: Run them to see them fail**

Run: `bats tests/tutorials-catalogue.bats`
Expected: FAIL, `bin/tutorials` does not exist.

- [ ] **Step 3: Implement `bin/tutorials`**

The repository's script header style (as `bin/ci`). For each `*.mdx` under `${TUTORIALS_PROJECT}/src/content/`:
`yq -p yaml --front-matter=extract -o json '.' <file>`; refusals are `❌ <file>: <reason>` on stderr and exit 1;
output through `yq -P` so the region is stable. Two pages may not share an id.

- [ ] **Step 4: Create `kit/modules/tutorials.yml`** with a header comment saying what it is and that `kit generate`
writes it, the region, and nothing else; run `kit generate kit/modules/tutorials.yml`.

- [ ] **Step 5: Verify**

Run: `bats tests/tutorials-catalogue.bats && kit generate --check`
Expected: all pass; no stale file.

- [ ] **Step 6: Commit** — `#86 kit: the catalogue of top-level tutorials, generated from their front matter`.

### Task 3: The verb, `kit tutorials`

**Files:**

- Create: `kit/bin/kit-tutorials`, `kit/modules/bash/tutorials.sh`
- Test: `tests/tutorials.bats`
- Modify: `docs/` — add `docs/tutorials.md` (what the verb does, where tutorials live, how links are chosen) and
  link it from wherever `docs/` lists the verbs' pages

**Interfaces:**

- Consumes: `kit/modules/tutorials.yml` (path in code: `${kit_root}/modules/tutorials.yml`); `release:version`;
  `new:browse <url>`; `tui:binary`.
- Produces: the usage line `kit tutorials [--text | --json | --jq] [--tag <audience>]… [<tutorial>]`, group `help`.
- Produces: `kit tutorials --json`, one object, the shape kittui decodes:
  `{"version": "0.1.0", "base": "https://kit.empowerite.dev/tutorials/0.1/", "audiences": [{"id", "label"}],
  "tutorials": [{"id", "title", "summary", "glyph", "audience": [...], "url"}]}`, filtered by every `--tag` given.
- Produces: `KIT_TUTORIALS_CATALOGUE`, when set, names the catalogue file in place of the module's, for the tests.
- Produces in `tutorials.sh`: `tutorials:base <version>` (the rule in Global Constraints), `tutorials:json [<tag>…]`,
  `tutorials:choices <key> [--kind | <answer>…]` answering `<tutorial>` and `<audience>`, both closed.

- [ ] **Step 1: Write the failing tests** in `tests/tutorials.bats`, with `KIT_TUTORIALS_BASE` unset unless named:
  - `@test "the base follows the version"`: `tutorials:base` of `unreleased`, `0.1.0-rc.7`, `0.4.12`, and `1.0.0`
    gives the four bases in Global Constraints (`…/main/`, `…/0.1.0-rc.7/`, `…/tutorials/0.4/`, `…/tutorials/1.0/`).
  - `@test "--json carries each tutorial's link"`: with a version file of `0.4.7`, every `url` is
    `https://kit.empowerite.dev/tutorials/0.4/<path>?kit=0.4.7`.
  - `@test "KIT_TUTORIALS_BASE replaces the base"`.
  - `@test "--tag keeps the tutorials for that audience"`, and two `--tag`s keep either's.
  - `@test "an unknown tutorial or audience is a usage error"`: exit 2.
  - `@test "a tutorial named opens in the browser"`: a fake `open` and `xdg-open` record the URL; output names the
    title and the URL.
  - `@test "without an opener, the link prints"`: no opener on PATH; exit 0; the URL is on stdout.
  - `@test "the text lists glyph, title, summary, audiences, and link"`: `--text`, one block per tutorial.
  - `@test "no tutorials"`: an empty catalogue (point the module at a fixture) prints one dim line and exits 0.
  - `@test "on a terminal, it opens kittui's screen"`: under `pty`, as `tests/welcome.bats` does, with a fake kittui
    that records its arguments: `tutorials`, then each `--tag`.

- [ ] **Step 2: Run them to see them fail** — `bats tests/tutorials.bats`; FAIL, no verb.

- [ ] **Step 3: Implement `kit/bin/kit-tutorials` and `kit/modules/bash/tutorials.sh`**, following `kit-welcome`
for the header, the `guard:arguments` call, and the terminal hand-off (`exec "${binary}" tutorials --tag <t>…`), and
`help:choices` for the choices' row format. Every argument has its description, so `kit syntax --check` passes.

- [ ] **Step 4: Verify** — `bats tests/tutorials.bats && kit syntax --check && kit lint kit/bin/kit-tutorials
kit/modules/bash/tutorials.sh docs/tutorials.md`; all pass.

- [ ] **Step 5: Commit** — `#86 kit: kit tutorials lists the tutorials and opens one in the browser`.

### Task 4: The screen, `internal/tutorials`

**Files:**

- Modify: `projects/tui/internal/kit/kit.go` — the catalogue's types and `Runner.Tutorials`
- Create: `projects/tui/internal/tutorials/tutorials.go`, `view.go`
- Create: `projects/tui/internal/tutorials/testdata/fake-kit`, `testdata/catalogue.json`, `testdata/empty.json`
- Test: `projects/tui/internal/tutorials/tutorials_test.go`, `golden_test.go`, `helpers_test.go`;
  `projects/tui/internal/kit/kit_test.go`

**Interfaces:**

- Consumes: Task 3's JSON.
- Produces in `internal/kit`: `type Catalogue struct { Version, Base string; Audiences []Audience; Tutorials
  []Tutorial }`, `type Audience struct { ID, Label string }`, `type Tutorial struct { ID, Title, Summary, Glyph
  string; Audience []string; URL string }` (JSON tags as Task 3), and `func (r Runner) Tutorials(ctx
  context.Context, tags []string) (Catalogue, error)`, which runs `kit tutorials --json [--tag <t>]…`.
- Produces in `internal/tutorials`: `func Open(r kit.Runner, tags []string) Model`, `func New(r kit.Runner, tags
  []string) Model`, and on `Model`: `Init`, `Update`, `View`, `Interrupted() bool`, `Close()`, `Noticing() bool`,
  and a `browse func(string) error` seam defaulting to the same opener `flow.browse` uses.

The screen: the title line with the app's menu glyph, like welcome's; a tab bar (`internal/tabs`) of "All" then each
audience, each tab counting its tutorials, a tag given on the command line selecting its tab; under it, the pane: a
grid of cards, 38 cells wide with their border, 6 lines tall: the glyph and the title in bold, the summary wrapped to
three lines with an ellipsis when it runs over, and the audiences' labels dim on the last line. Columns are as many as
fit, at least one; the pane scrolls with the cursor. The card under the cursor is drawn in the menu's bright cyan.
Keys: ←→↑↓ move between cards; ⏎ opens; the bar takes 1–9, and ↑ from the first row; = opens the app's menu; ctrl-c
quits with 130 and alt-esc with 0, as welcome does. A click on a card opens it; a click on a tab selects it. Opening
says "🎓 Opened `<title>` in the browser" in place of the keys for five seconds or until a key; when the browser fails,
"Open `<url>` in a browser" stays until a key. With no tutorials, or none under the tab, the pane says "No tutorials for
`<label>` yet" (or "No tutorials yet" under All).

- [ ] **Step 1: Write the failing tests**
  - `kit_test.go` `TestTutorialsDecodesTheCatalogue`: a fake kit printing `testdata/catalogue.json` gives two
    tutorials with their URLs; a kit that exits 2 gives an `*ExitError`.
  - `golden_test.go` `TestGoldenViews`: `tutorials.golden` (All, 120×30), `tutorials-ui.golden` (`press('3')`),
    `tutorials-narrow.golden` (60×20), `tutorials-empty.golden` (`empty.json`), and `tutorials-opened.golden`
    (`enter` on the first card).
  - `tutorials_test.go`: `TestArrowsMoveBetweenCardsAndRows`, `TestTheCursorScrollsThePane` (at 60×20 with eight
    cards, the cursor's card is always drawn whole), `TestEnterAndAClickOpenTheCardsURL` (the seam records the URL
    with `?kit=`), `TestABrowserThatFailsShowsTheLink`, `TestATagFromTheCommandLineSelectsItsTab`,
    `TestTheAppMenuSwitchesAndQuits`, and `TestCtrlCIsInterrupted`.

- [ ] **Step 2: Run them to see them fail** — `cd projects/tui && go test ./internal/kit/ ./internal/tutorials/`.

- [ ] **Step 3: Implement** `Runner.Tutorials`, then the package, following `internal/welcome` for the model's
shape, its app's menu, its keys bar, and its tests' helpers (`testkit.Runner(t, "tutorials")`, `testkit.Drive`,
`testkit.Golden`). Package comment in the style of welcome's: what the screen shows and every key.

- [ ] **Step 4: Verify** — `go test -update ./internal/tutorials/`, read every golden by eye at its width, then
`go test -cover ./internal/tutorials/ ./internal/kit/`; all pass, tutorials at least 98% covered.

- [ ] **Step 5: Commit** — `#86 kittui: the tutorials screen, a card per tutorial, tabbed by audience`.

### Task 5: Wiring the screen into kittui

**Files:**

- Modify: `projects/tui/cmd/kittui/main.go` (`open`, `needsTerminal`, `run`'s finish switch),
  `cmd/kittui/main_test.go`
- Modify: `projects/tui/internal/screens/screens.go` (`open`), `internal/screens/screens_test.go` (line 79's
  refusal becomes a screen shown)
- Modify: `projects/tui/internal/shell/shell.go` (`enabled`, and the comments at lines 30–31 and 100–101 that say
  kit has no tutorials), `internal/shell/shell_test.go` (the enabled list: Tutorials is true off its own screen)
- Modify: `projects/tui/README.md` (five screens; the tutorials screen in the list and the layout)

**Interfaces:**

- Consumes: Task 4's `tutorials.Open(r, tags)`, `Model.Interrupted()`.
- Produces: `kittui tutorials [--tag <t>]…` opens on the screen; the menu's Tutorials item switches to it from any
  screen; without a terminal kittui says `kit tutorials needs an interactive terminal; run: kit tutorials --text`.

- [ ] **Step 1: Coordinate** — `git status` and ListAgents; message any peer with changes in these files and agree
  who commits first.
- [ ] **Step 2: Write the failing tests** — `main_test.go` `TestTutorialsOpensItsScreen` and the no-terminal line;
  `screens_test.go` `TestTheMenuOpensTheTutorials`; the updated enabled list in `shell_test.go`.
- [ ] **Step 3: Run them to see them fail** — `cd projects/tui && go test ./cmd/... ./internal/screens/
  ./internal/shell/`.
- [ ] **Step 4: Implement** the three cases, as `welcome` is wired.
- [ ] **Step 5: Verify the whole gate** — `kit gate`; it ends with `✅ ci passed`. A failure in a package another
  session has uncommitted work in is theirs: say so in the report rather than touching it.
- [ ] **Step 6: Commit** — `#86 kittui: the menu's Tutorials opens the tutorials screen`.
