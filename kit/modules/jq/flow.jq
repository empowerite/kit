# flow.jq - the change documents kit flow draws, kept by their authors (jq module)
# ----------------------------------------------------------------------------------------------------------------------
# import "flow" as flow; with jq -L "${kit_jq}".
#
# see: modules/bash/flow.sh

# by_authors($logins; $except) keeps a change document opened by one of $logins, or with $except, by none of them.
def by_authors($logins; $except):
  select((.author as $a | $logins | index($a) != null) != $except);
