---
title: The control line
summary: The line over the rows says how they are grouped, and the preset they are set as.
glyph: 📊
audience: [ui]
order: 13
---

```text
  ▾ source ▾ assignees ▾ 3          📇 Fields  Default ▼
```

## Left: the groups

- `▾ source`  the rows are grouped by source. Click the name for its settings.
- `▾`  open. Click to close that level.
- `▸`  closed. Click to open it.
- `3`  how many rows

With no groups, it is the rows' `▾` and their count alone.

## Right: the preset

- `Default ▼`  the preset the table is set as. Click it, or `⏎` on it, for the presets menu.
- A name struck through means you changed the table since.
- Each tab opens as you last left it, your changes kept, even unsaved ones.
- 📇 **Fields**  every field: show, hide, sort, group, filter. `f` on the preset opens it too.
- Before it: how many rows are selected, and a chip per filter. Click a chip to change its filter.

## The presets menu

- **Save**  keep your changes in this preset
- **Save As…**  keep them as a new preset
- **Manage…**  every preset: star it, order your favorites, rename or delete one of yours
- The presets, to switch to: your favorites first, then your team's, then the built-in ones, then yours

Save is dim on the built-in presets, and they cannot be renamed or deleted. Click a preset's ☆, or press `*` on it, to
make it a favorite; ★ marks one. Your team's presets are files in the repository, under `.kit/presets`; in Manage…, Share
with team moves one of yours there, for you to commit, and Copy to mine copies one of theirs to yours. A repository
without kit installed has no team presets.

## Keys

- `↓` from the tabs reaches the line. `↓` again enters the rows.
- `⇥` `⇧⇥`  move between the names, the count, and the preset
- `→` `←`  open and close the level under the cursor
- `⏎`  on a name: its settings. On the preset: the presets menu.
