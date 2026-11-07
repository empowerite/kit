# Tutorials

The tutorials are short lessons on a site, one page for each, each written for some audiences: a newcomer, someone
working in kittui, a maintainer. `kit tutorials` lists them and opens one.

```sh
kit tutorials                         # on a terminal, kittui's tutorials screen
kit tutorials --text                  # the list, as text, even on a terminal
kit tutorials --tag newcomer          # only the tutorials for that audience
kit tutorials --tag ui --tag newcomer # the tutorials for either
kit tutorials kit-tutorials           # opens one in the browser
kit tutorials --json                  # one JSON object, for a program
```

## What it does

- On a terminal, with no `--text`, `--json`, or `--jq`, it opens kittui's tutorials screen: a button for each tutorial,
  and the audiences as filters. Each `--tag` starts with that filter on. Without a kittui, it prints the list.
- Without a terminal, or with `--text`, it prints a block for each tutorial: its glyph and title, its summary, its
  audiences, and its link. With none to show, it says so in one line.
- A tutorial named opens in the browser. Its title and link print first, so a machine with no browser has the link.
- `--json` prints one object. `--jq` prints the same, formatted by jq for a person.
- Nothing here runs a tutorial's commands. A tutorial tells you what to type; you type it.

## The JSON

```json
{"version":"0.4.7","base":"https://kit.empowerite.dev/tutorials/0.4/",
 "audiences":[{"id":"newcomer","label":"Just starting","color":"#15803d"}],
 "tutorials":[{"id":"kit-tutorials","title":"Reading kit's tutorials","summary":"…","glyph":"🎓",
               "audience":["newcomer"],"url":"https://kit.empowerite.dev/tutorials/0.4/kit-tutorials/?kit=0.4.7"}]}
```

`--tag` filters the tutorials and leaves the audiences whole, so a screen can show every filter. An audience's `color`,
as `#rrggbb`, is the one kittui draws its pill on; it is empty where the catalogue gives none.

## Where tutorials live

The pages are the MDX files of the tutorials site. Each top-level page describes its tutorial in its front matter: a
title, a summary, a glyph, and its audiences. `kit generate` reads them into the catalogue, `modules/tutorials.yml`, in
the release. Edit a page, then run `kit generate`; the catalogue is not edited by hand. `KIT_TUTORIALS_CATALOGUE`, when
set, names another catalogue file in its place.

## How a link is chosen

A tutorial's link is the base, then the page's path, then `?kit=<version>`. The base follows the version of the kit that
runs, so the page you open describes the kit you have:

| `kit version` | Base |
| --- | --- |
| `unreleased`, a checkout | `https://empowerite.github.io/kit/tutorials/main/` |
| a pre-release, as `0.5.0-rc.1` | `https://empowerite.github.io/kit/tutorials/0.5.0-rc.1/` |
| a release, `X.Y.Z` | `https://kit.empowerite.dev/tutorials/X.Y/` |

`KIT_TUTORIALS_BASE`, when set, replaces the base, as for a site served on this machine. The `?kit=` part lets a page
tell you that your kit is too old for a passage.

## Exit status

0 when it printed or opened; 1 when the catalogue is missing; 2 on a usage error, such as a tutorial or an audience
that does not exist.

see: [modules/bash/tutorials.sh](../kit/modules/bash/tutorials.sh), [bin/kit-tutorials](../kit/bin/kit-tutorials)
