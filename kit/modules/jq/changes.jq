# changes.jq - which open changes the changes stage of kit flow lists (jq module)
#
# import "changes" as changes; with jq -L "${kit_jq}".
#
# see: modules/bash/changes.sh

# concerning($refs) keeps a change document, as forge.jq gives it, whose own number is one of $refs, or that closes an
# issue numbered one of them.
def concerning($refs):
  select((.number as $n | $refs | index($n)) or (.closes // [] | any(.number as $n | $refs | index($n))));
