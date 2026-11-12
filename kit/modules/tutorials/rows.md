---
title: The rows
summary: The rows stand under a header and its rule, and they sort, group, fold, and select.
glyph: 📋
audience: [ui]
order: 14
---

```text
      id   assignees   age  labels           title
      ·────×·──────────¹───·────────────────·──────────
    ▸ #45               2h  kind:feat… +2    🎫 kit flow…
    ▸ #46  jeffrey…     2d  bug              🎫 A noun l…
```

## The header and its rule

- Click a column's **name** for its menu: sort, group, flatten, hide, options.
- The **rule** under a name says how it sorts. Click it to turn the sort.
- `·`  can sort, and does not
- `¹` `₁`  sorts up (raised) or down (lowered). The small number is its place among the sorts: `²` or `₂` is the second.
- `×`  a list, as labels. Click to give each value its own row.
- `÷`  a list given a row per value. Click to put it back on one row.

## Rows

- `▸`  closed. `→` or a click opens it, to show all it holds.
- `▾`  open. `←` or a click closes it.
- A heading groups the rows under it, and folds the same way.

## Keys

- `↑` `↓`  move
- `→` `←`  open and close, or step in and out
- `␣`  select. On a heading: its whole group.
- `[` `]`  previous and next group
- `{` `}`  first and last group
- `>` `<`  open or close a level everywhere

A selected heading shows `▣` when all its rows are selected, and `◩` when some are.
