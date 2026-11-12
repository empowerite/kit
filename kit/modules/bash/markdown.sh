# shellcheck shell=bash
# markdown.sh - what the pages kit prints as text do with markdown only kittui follows (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# The pages under .kit/welcome/ and modules/tutorials/ are read by kittui, which draws a link and follows a click on
# it, and printed as text by kit welcome and kit tutorials, which have no link to follow. markdown:unlink gives the text
# what kittui draws: a link to kittui, [label](kittui:<where>), is its label alone, and a link to a web address,
# [label](https://…), is its label and the address once, in parentheses. Any other link is left as written.
#
# see: projects/tui/internal/markdown

# markdown:unlink filters markdown on stdin to stdout, each link in it drawn as the text does: label, or label (address).
markdown:unlink() {
  sed -E \
    -e 's/\[([^]]+)\]\(kittui:[^) ]*\)/\1/g' \
    -e 's/\[([^]]+)\]\((https?:\/\/[^) ]+)\)/\1 (\2)/g'
}
# ----------------------------------------------------------------------------------------------------------------------
