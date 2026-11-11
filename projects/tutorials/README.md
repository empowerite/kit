# kit tutorials

`projects/tutorials` is the site of kit's tutorials: one web page for each feature, tagged by who it is for. It is a
TypeScript, React, and Vite site, and every page is an MDX file. `kit tutorials` shows the top-level tutorials as
buttons, and a button opens its page in the browser. Nothing on the site runs a command for the reader.

Each build is for one line of kit, at one base path:

- `/` when `npm run dev` serves it
- `/kit/tutorials/main/` and `/kit/tutorials/<pre-release>/` on this repository's GitHub Pages
- `/kit/tutorials/<major>.<minor>/` on this repository's GitHub Pages, one site for every patch release in the line

## Running it

Node comes from `mise.toml`. In this directory, install the packages once, then serve the site:

```bash
npm ci
npm run dev
```

In a repository with kit, point kit's buttons at that server instead of the published site:

```bash
KIT_TUTORIALS_BASE=http://localhost:5173/ kit tutorials
```

`npm run check` runs what the gate, `bin/ci`, runs here: `tsc`, the tests, and a build for the base
`/kit/tutorials/main/`. The build checks what it built and fails, naming the file, on a page that throws, such as one
with a `<Since>` whose version is not `X.Y.Z`; on a page that renders empty; on a route with no `dist/<path>/index.html`;
and on an `href` or `src` in `dist/` that resolves outside the base or names no file. A `TUTORIALS_BASE` that is not a
path, such as a URL, fails it too.

A build reads two variables:

| Variable | Default | What it is |
| --- | --- | --- |
| `TUTORIALS_BASE` | `/` | The base path the site is served at, Vite's `base`, written with its slashes added |
| `TUTORIALS_LINE` | `main` | The line of kit the site is for: `main`, `0.4`, or `0.5.0-rc.1` |

`package.json` denies `fsevents` its install script, through `allowScripts`: the package ships its macOS binary
built, and only the file watchers of Vite and Rollup load it.

## Content

Every MDX file under `src/content/` is a page, served at a path relative to the base:

| File | Served at | Kind |
| --- | --- | --- |
| `src/content/<id>/index.mdx` | `<id>/` | A top-level tutorial, when its front matter says `top: true` |
| `src/content/<id>/<page>.mdx` | `<id>/<page>/` | A deeper page that a tutorial links to |

Two more pages are in each site's top navigation:

- `src/whats-new.mdx`, served at `whats-new/`: the story of the line's minor release, with links to the tutorials it
  added or changed.
- `src/changelog/<version>.mdx`, one entry for each release, written by hand as part of cutting it. `changelog/`
  shows every entry, newest first.

Each page is pre-rendered to `dist/<path>/index.html`, so a deep link works on any host with no rewrites. A link
between pages is written from the page it is on, as on the web: `[marks](marks/)` on `kit-tutorials/` and
`[the changelog](../changelog/)` on any page one level down, never with a leading `/`. The link keeps the reader's
`?kit=` on the next page. An address that is no page shows "No such page", and the build writes the same page as
`dist/404.html`, which Pages serves for any address with no file.

## Front matter

Every page opens with YAML front matter between two `---` lines. [frontmatter.schema.json](frontmatter.schema.json)
holds it to these keys, and the gate checks every page against it:

| Key | Type | Required | What it is |
| --- | --- | --- | --- |
| `title` | string | always | The page's heading and title |
| `top` | boolean | no, `false` | `true` for a top-level tutorial, one kit lists |
| `summary` | string | when `top` | The sentence on the tutorial's button |
| `glyph` | string | when `top` | The emoji on the tutorial's button |
| `audience` | list of strings | when `top` | Who it is for, at least one id from `audiences.yml` |
| `order` | integer | no, `1000` | Where it stands among the tutorials, lowest first |

[audiences.yml](audiences.yml) is the closed list of audiences, each an `id`, the `label` kit shows, and the `color`
kittui draws its pill on, as `#rrggbb`, dark enough for white text, in the order kit shows them. Adding an audience is
an edit there; `bin/tutorials catalogue` refuses one without a color shaped `#rrggbb`.

## Marking what a patch release added

A passage that a patch release made true is wrapped in `<Since>`, which a page needs no import for:

```mdx
<Since version="0.4.9">
`kit flow` takes `--quota`.
</Since>
```

kit opens every page with its own version in the address, as `?kit=0.4.7`. A passage newer than the reader's kit
says so, "Needs kit 0.4.9; you have 0.4.7.", and how to upgrade: install that version or later, then run
`kit update`. A pre-release of the passage's own version, `0.4.9-rc.1`, has the passage. Without `?kit=`, or with
`unreleased` or anything that is not a version, the passage is marked plainly, "Since 0.4.9".
