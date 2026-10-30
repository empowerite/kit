# merge.jq - a merge by key: a release's JSON over a repository's, keeping what the repository added (jq module)
# ----------------------------------------------------------------------------------------------------------------------
#
# import "merge" as merge; with jq -L "${kit_jq}". The parameters are values ($name), not filters, so each is
# evaluated once, at the right level.
#
# see: modules/bash/release.sh, docs/packages.md

# apply($cur; $base; $new) applies the release's $new over the current $cur, key by key. For an array, it keeps the
# current entries that were not in the previous release's $base, then adds $new's.
def apply($cur; $base; $new):
  if ($new | type) == "object" then
    reduce ($new | keys[]) as $k ($cur // {}; .[$k] = apply(($cur // {})[$k]; ($base // {})[$k]; $new[$k]))
  elif ($new | type) == "array" then
    ((($cur // []) - ($base // [])) as $kept | $kept + ($new - $kept))
  else $new end;

# remove($cur; $base) removes $base's array entries from $cur, key by key. Scalar values stay.
def remove($cur; $base):
  if ($base | type) == "object" then
    reduce ($base | keys[]) as $k ($cur; if has($k) then .[$k] = remove($cur[$k]; $base[$k]) else . end)
  elif ($base | type) == "array" then ($cur // []) - $base
  else $cur end;

# holds($cur; $new) is true when $cur contains every key, array entry, and value in $new.
def holds($cur; $new):
  if ($new | type) == "object" then
    all($new | keys[]; . as $k | ($cur | type) == "object" and ($cur | has($k)) and holds($cur[$k]; $new[$k]))
  elif ($new | type) == "array" then all($new[]; . as $e | any(($cur // [])[]; . == $e))
  else $cur == $new end;
