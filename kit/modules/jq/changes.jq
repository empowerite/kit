# changes.jq - which open changes kit changes lists (jq module)
#
# import "changes" as changes; with jq -L "${kit_jq}".
#
# see: modules/bash/changes.sh

# closing($issues) keeps a change document, as forge.jq gives it, when it closes one of the issues numbered $issues.
def closing($issues):
  select(.closes // [] | any(.number as $n | $issues | index($n)));
