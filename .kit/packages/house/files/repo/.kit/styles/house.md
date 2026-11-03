---
# kit:generated -- this repository's own style, from the house package
# see: .kit/packages/house/package.yml
kind:
  - text
---
# House style

Applies to every file of kind `text` in this repository, on top of the style for the file's kind.

1. A file with a banner has a line of dashes to column 120 under it, and another at the end of the file. The
   templates in `.kit/templates/` hold the exact shape, and `kit lint` checks it.
2. The line under the dashes is never a bare `#`. The comment paragraph starts on the next line, right under the
   dashes. A bare `#` still separates two paragraphs further down.
