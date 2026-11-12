# Tutorials

The tutorials are short lessons, each written for some audiences: a newcomer, someone working in kittui, a
maintainer. Some ship with the kit, pages read on this machine; the others are a site online, one page for each.
`kit tutorials` lists them and opens one.

```sh
kit tutorials                         # on a terminal, kittui's tutorials screen
kit tutorials --text                  # the list, as text, even on a terminal
kit tutorials --tag=newcomer          # only the tutorials for that audience
kit tutorials --tag=ui --tag=newcomer # the tutorials for either
kit tutorials kit-tutorials           # opens one: kittui on a terminal, else the browser
kit tutorials applications            # one in the kit: kittui on a terminal, else its page as text
kit tutorials --json                  # one JSON object, for a program
```

## What it does

- On a terminal, with no `--text`, `--json`, or `--jq`, it opens kittui's tutorials screen: a card for each tutorial,
  under the table's control line, as every kittui table has. The control line groups, sorts, and filters the cards by
  any field, the audience among them, and keeps them in presets: `By audience` puts the cards under a heading for each
  audience, a tutorial for two under both, and `In the kit or online` under one for each place. The `--tag` values
  that name an audience filter the cards to the tutorials for any of them, as the text, `--json`, and `--jq` forms
  keep them; the filter is the table's, to change or drop. Without a kittui, it prints the list.
- Each card says where its tutorial is, on its top edge: 📦 for one in the kit, 🌐 for one online; the `where` field
  holds the same, to group or filter the cards by. Enter or a click on a card opens it: one online in the browser,
  and one in the kit in kittui, its page in place of the cards. The arrows, page up, page down, and the wheel scroll
  the page; esc, or a click on the 🔙 line over it, goes back to the cards.
- Without a terminal, or with `--text`, it prints a block for each tutorial: its glyph and title, its summary, its
  audiences, and its link, or, for one in the kit, the command that reads it. With none to show, it says so in one
  line.
- On a terminal, a tutorial named opens kittui's screen on its card. With `--text`, or in a pipe, one in the kit
  prints its title and its page; for one online, its title and link print and it opens in the browser, so a machine
  with no browser has the link.
- `--json` prints one object. `--jq` prints the same, formatted by jq for a person.
- Nothing here runs a tutorial's commands. A tutorial tells you what to type; you type it.

## The JSON

```json
{"version":"0.4.7","base":"https://empowerite.github.io/kit/tutorials/0.4/",
 "audiences":[{"id":"newcomer","label":"Just starting","color":"#15803d"}],
 "tutorials":[{"id":"kit-tutorials","title":"Reading kit's tutorials","summary":"…","glyph":"🎓",
               "audience":["newcomer"],"where":"online",
               "url":"https://empowerite.github.io/kit/tutorials/0.4/kit-tutorials/?kit=0.4.7","file":""},
              {"id":"applications","title":"kit's applications","summary":"…","glyph":"🧭",
               "audience":["newcomer","ui"],"where":"kit","url":"",
               "file":"/opt/kit/modules/tutorials/applications.md"}]}
```

A tutorial's `where` is `kit` for one that ships with the kit, whose `file` is the absolute path of its page, or
`online` for one on the site, whose `url` is its link; the other is empty.

`--tag` filters the tutorials and leaves the audiences whole, so a screen can show every filter. An audience's `color`,
as `#rrggbb`, is the one kittui draws its pill on; it is empty where the catalogue gives none.

## Where tutorials live

The online tutorials are the MDX files of the tutorials site. The tutorials in the kit are markdown pages in the
release, `modules/tutorials/<id>.md`. Each top-level page of the site, and each page in the kit, describes its
tutorial in its front matter: a title, a summary, a glyph, and its audiences. `kit generate` reads them into the
catalogue, `modules/tutorials.yml`, in the release, an online one with its path on the site and one in the kit with its
file. Edit a page, then run `kit generate`; the catalogue is not edited by hand. `KIT_TUTORIALS_CATALOGUE`, when
set, names another catalogue file in its place.

A tutorial that needs no network ships in the kit. The tour of kittui's screen does: `screen`, then one page for each
part, `menu-bar`, `tab-bar`, `control-line`, `rows`, `status-bar`, `legend`, and `search`.

## Writing a tutorial

Readers are impatient. Write so a page can be scanned, not read:

- A summary is one or two short sentences. kittui shows it whole, and every card grows to the longest one.
- Short sentences. One idea each.
- Bullets over paragraphs. Put the key or the glyph first, in backticks, then what it does: `` `=`  the kit menu ``.
- A small text picture of the part, in a fenced `text` block, before the words.
- Headings name what the reader is looking for: Keys, Reading a tab, The presets menu.
- Name only what is on the screen, and check it against kittui before it ships.

## How a link is chosen

The tutorials are one site, `https://empowerite.github.io/kit/tutorials/`, and a version is a line of it. A tutorial's
link is the base, then the page's path, then `?kit=<version>`. The base follows the version of the kit that
runs, so the page you open describes the kit you have:

| `kit version` | Base |
| --- | --- |
| `unreleased`, a checkout | `https://empowerite.github.io/kit/tutorials/main/` |
| a pre-release, as `0.5.0-rc.1` | `https://empowerite.github.io/kit/tutorials/0.5.0-rc.1/` |
| a release, `X.Y.Z` | `https://empowerite.github.io/kit/tutorials/X.Y/` |

`KIT_TUTORIALS_BASE`, when set, replaces the base, as for a site served on this machine. The `?kit=` part lets a page
tell you that your kit is too old for a passage.

## Exit status

0 when it printed or opened; 1 when the catalogue, or the page of a tutorial in the kit, is missing; 2 on a usage
error, such as a tutorial or an audience that does not exist.

see: [modules/bash/tutorials.sh](../kit/modules/bash/tutorials.sh), [bin/kit-tutorials](../kit/bin/kit-tutorials)
