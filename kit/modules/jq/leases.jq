# leases.jq - what a lease holds, which leases a claim overlaps, and which paths a lease leaves out (jq module)
# ----------------------------------------------------------------------------------------------------------------------
# import "leases" as leases; with jq -L "${kit_jq}". A lease is the JSON blob at refs/kit/leases/<key>, read with
# read($key; $text) into {key, readable, issue, paths, holder, host, branch, worktree, taken, renewed, expires}. A path
# is relative to the root, as git ls-files prints it, and holds itself and everything under it: * and ? match within
# one segment, […] one character of a set, and ** any number of segments. The time, $now, is seconds since the epoch.
#
# see: modules/bash/leases.sh, docs/superpowers/specs/2026-11-12-leases-design.md

# glob_chars is true when a segment holds a character a pattern gives a meaning to.
def glob_chars: test("[*?\\[]");

# regex gives the regular expression that matches every path a pattern holds: what it matches, and everything under.
def regex:
  rtrimstr("/")
  | [scan("\\*\\*/|\\*\\*|\\*|\\?|\\[!?\\]?[^\\]]*\\]|.")
    | if . == "**/" then "(?:.*/)?"
      elif . == "**" then ".*"
      elif . == "*" then "[^/]*"
      elif . == "?" then "[^/]"
      elif startswith("[") and length > 2 then "[" + (.[1:-1] | sub("^!"; "^") | gsub("\\\\"; "\\\\")) + "]"
      else gsub("(?<c>[.+()|{}^$\\\\\\[\\]])"; "\\\(.c)") end]
  | "^" + join("") + "(?:/.*)?$";

# holds($path) is true when the pattern holds the path.
def holds($path): . as $pattern | $path | test($pattern | regex);

# segment_apart($a; $b) is true when two segments can match no name in common: two literals that differ, a literal
# the other's pattern does not match, or two patterns whose literal starts or ends differ.
def segment_apart($a; $b):
  def ends: [(capture("^(?<s>[^*?\\[]*)").s), (capture("(?<e>[^*?\\]]*)$").e)];
  if ($a | glob_chars | not) and ($b | glob_chars | not) then $a != $b
  elif ($a | glob_chars | not) then ($a | test("^" + ($b | regex | ltrimstr("^") | rtrimstr("(?:/.*)?$")) + "$") | not)
  elif ($b | glob_chars | not) then segment_apart($b; $a)
  else ($a | ends) as [$as, $ae] | ($b | ends) as [$bs, $be]
    | (($as | startswith($bs)) or ($bs | startswith($as)) | not)
      or (($ae | endswith($be)) or ($be | endswith($ae)) | not) end;

# overlap($a; $b) is true unless two patterns can be proven to hold no path in common, segment by segment: ** or one
# running out of segments first overlaps, so the judgement errs toward overlapping.
def overlap($a; $b):
  def segments: rtrimstr("/") | split("/") | map(select(. != ""));
  def walk($x; $y):
    if ($x | length) == 0 or ($y | length) == 0 then true
    elif $x[0] == "**" or $y[0] == "**" then true
    elif segment_apart($x[0]; $y[0]) then false
    else walk($x[1:]; $y[1:]) end;
  walk($a | segments; $b | segments);

# refused gives why a path cannot be claimed, or nothing: it is empty, starts with /, or holds a .. segment.
def refused:
  if . == "" then "a path is not empty"
  elif startswith("/") then "'\(.)' starts with /; a path is relative to the repository's root"
  elif split("/") | any(. == "..") then "'\(.)' holds a .. segment; a path stays inside the repository"
  else empty end;

# tidy gives a path as a claim keeps it: without a leading ./ or a trailing /.
def tidy: sub("^(\\./)+"; "") | rtrimstr("/") | if . == "" or . == "." then "**" else . end;

# read($key; $text) gives the lease the blob's text holds, or {key, readable: false} when it is not a JSON object with
# issue a string, paths an array of strings, and expires a number.
def read($key; $text):
  ($text | try fromjson catch null) as $doc
  | if ($doc | type) == "object" and ($doc.issue | type) == "string" and ($doc.expires | type) == "number"
      and ($doc.paths | type) == "array" and ($doc.paths | all(type == "string"))
    then {key: $key, readable: true, issue: $doc.issue, paths: $doc.paths, holder: ($doc.holder // null),
      host: ($doc.host // null), branch: ($doc.branch // null), worktree: ($doc.worktree // null),
      taken: ($doc.taken // $doc.expires), renewed: ($doc.renewed // $doc.taken // $doc.expires),
      expires: $doc.expires}
    else {key: $key, readable: false} end;

# live($now) is true of a lease that has not expired, and of one that does not read, which holds until it is released.
def live($now): (.readable | not) or .expires > $now;

# own is true of a branch's own lease: the one whose key the name of the branch it is on starts with, <key>-.
def own: .key as $key | .readable and (.branch | type) == "string" and (.branch | startswith("\($key)-"));

# compete($a; $b) is true when two leases are judged for overlap: they are of different issues, and neither is the
# branch's own lease of the branch both are on, which the other nests in. One that does not read competes with all.
def compete($a; $b):
  $a.key != $b.key
  and (($a.readable and $b.readable and $a.branch != null and $a.branch == $b.branch and (($a | own) or ($b | own)))
    | not);

# conflicts($claim; $now) gives each lease of the input array that refuses the claim, {key, paths, branch}: a live one
# that competes with it and holds a path that may overlap one of the claim's. One that does not read overlaps all.
def conflicts($claim; $now):
  .[] | select(live($now) and compete(.; $claim + {readable: true}))
  | select((.readable | not) or any(.paths[] as $p | $claim.paths[] as $q | overlap($p; $q); .));

# outside($paths) gives each path of the input array that none of the patterns holds.
def outside($paths): .[] | . as $path | select(any($paths[]; holds($path)) | not);

# body_paths gives the paths an issue's body names under a heading whose text is Paths, one per list item, until the
# next heading: the item's text in backticks when it has some, else its first word, without a : , or ; after it. A path
# no claim takes is left out.
def body_paths:
  [split("\n")[] | sub("\r$"; "")]
  | reduce .[] as $line ({in: false, paths: []};
      if $line | test("^ {0,3}#{1,6}\\s") then .in = ($line | test("^ {0,3}#{1,6}\\s+paths\\s*#*\\s*$"; "i"))
      elif .in and ($line | test("^\\s*([-*+]|[0-9]+[.)])\\s+\\S")) then
        ($line | sub("^\\s*([-*+]|[0-9]+[.)])\\s+"; "")) as $item
        | .paths += [($item | capture("`(?<p>[^`]+)`").p) // ($item | split(" ")[0] | sub("[:,;]+$"; ""))]
      else . end)
  | .paths | map(select([refused] | length == 0) | tidy) | unique;

# span($seconds) gives a length of time in its coarsest unit: 45s, 12m, 3h, 2d.
def span($seconds):
  ($seconds | if . < 0 then -. else . end | floor) as $s
  | if $s < 60 then "\($s)s" elif $s < 3600 then "\($s / 60 | floor)m" elif $s < 86400 then "\($s / 3600 | floor)h"
    else "\($s / 86400 | floor)d" end;

# when($now) gives when a readable lease expires, as kit leases prints it: until 15:04, in 52m; or expired 12m ago.
def when($now):
  if .expires > $now then "until \(.expires | strflocaltime("%H:%M")), in \(span(.expires - $now))"
  else "expired \(span($now - .expires)) ago" end;

# line($now) gives a lease's lines as kit leases prints them: the issue, who holds it, where, and when it expires, then
# its paths; or, for one that does not read, the command that releases it.
def line($now):
  if .readable then
    "\(.issue)  \(.holder // "someone")\(if .host then " on \(.host)" else "" end), \(.branch // "a detached HEAD"),"
    + " \(when($now))\n    \(.paths | join("  "))"
  else "refs/kit/leases/\(.key) does not read as a lease, so it holds every path until it is released: kit leases"
    + " release \(.key)" end;

# refusal($now) gives why a lease refuses a claim, as one clause: who holds what, where, and until when.
def refusal($now):
  if .readable then
    "\(.issue) holds \(.paths | join(" ")), taken by \(.holder // "someone") on \(.branch // "a detached HEAD"),"
    + " \(when($now))"
  else "refs/kit/leases/\(.key) does not read as a lease, so it holds every path; release it: kit leases release"
    + " \(.key)" end;
