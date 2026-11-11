---
title: The screen at a glance
summary: The parts of every kittui screen, top to bottom, and the five keys to start with.
glyph: 📺
audience: [newcomer, ui]
order: 10
---

Every kittui screen stacks the same parts. An app, such as Flow, looks like this:

```text
  = kit flow · - owner/repo                 🔎 /search   ← menu bar
  ➊╶────────────────╮➁╶─────────────────╮                ← tab bar
  │ 🎫 backlog  3 : │  🛫 underway  4   │
  ╵                 ╰───────────────────╰──
  ▾ source 3                          Default ▼          ← control line
      id   age  title                                    ← header
      ·────·────·──────────────                          ← rule
    ▸ #45   2h  🎫 kit flow is the…                      ← rows
    ▸ #46   2d  🎫 A noun lists…
 = menu - workspace / search ? legend │ ⇥·⇧⇥ tab         ← status bar
```

## The parts

- **Menu bar**, the first line: the kit menu, the screen's name, the workspace, and the search.
- **Tab bar**: one tab per stage. Apps only.
- **Control line**: how the rows are grouped, and the preset they are set as.
- **The view**: the header, its rule, and the rows. Most of the screen.
- **Status bar**, the last line: the keys that work right now.
- **Legend**: floats over the rows and names every glyph on the screen. Apps only.

## Five keys to start

- `=`  the kit menu: every screen, Settings…, Quit
- `/`  search
- `?`  the legend
- `esc`  back out one step. It never quits.
- `⌃Q`  quit

Every action also works with a plain click.

## Read next

- `kit tutorials menu-bar`
- `kit tutorials tab-bar`
- `kit tutorials control-line`
- `kit tutorials rows`
- `kit tutorials status-bar`
- `kit tutorials legend`
- `kit tutorials search`
