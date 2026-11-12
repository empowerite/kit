# terminal.jq - the terminal kit runs in, judged from its settings and its answers, as kit welcome's checks (jq module)
# ----------------------------------------------------------------------------------------------------------------------
# import "terminal" as terminal; with jq -L "${kit_jq}". Every terminal kit knows, each finding, and each probe live in
# terminals.json beside this module. Its keys:
#
#   docs        the page that lists every finding; a check's more is its terminal's section, docs#anchor
#   probes      each probe, in the order its checks come: need, one of speed, clicks, keys, widths, and color; held,
#               the line where the terminal answered as kittui needs; failed, where it did not
#   terminals   each terminal by its key, the word in its checks' ids: name; anchor, its section of the docs; and
#               findings, each with id, need, bad, a jq expression over the reader's object that is true when the
#               setting breaks the need, line, held, and fix; or unread, the line its heading says where kit has no
#               reader for it; and probes, by probe, fix, the setting that turns the feature on, or none, the line
#               that says what kittui does where no setting can
#
# A line, a held line, and a fix name what the reader read in braces: {profile}. jq cannot evaluate a string, so bash
# evaluates each bad and hands the answers to checks.
#
# see: modules/bash/terminal.sh, modules/bash/welcome.sh

import "terminals" as $terminals;

# data gives terminals.json.
def data: $terminals::terminals[0];

# fill($o) reads a string and gives it with each {key} that $o has replaced by its value.
def fill($o):
  reduce ($o | to_entries[] | select(.value | type != "object" and type != "array")) as $e
    (.; gsub("\\{" + $e.key + "\\}"; $e.value | tostring));

# measured_badly($probe) is true when the probes object says the terminal does not give kittui what $probe asks. A
# width the terminal did not report was not measured, so it is not bad.
def measured_badly($probe):
  if $probe == "kitty-keyboard" then .kitty_keyboard == false
  elif $probe == "emoji-width" then (.emoji_width // 2) != 2
  elif $probe == "ambiguous-width" then (.ambiguous_width // 1) != 1
  elif $probe == "color" then .color == "fewer"
  else false end;

# checks($d; $key; $read; $bad; $probes; $kept) gives the checks of the terminal $key of the table $d: its heading,
# then each finding, then each probe, each {id, ok, line, fix, more}. $read is the reader's object, {unreadable} where
# it could not read, or null for a terminal kit has no reader for; $bad, each finding's answer by its id; $probes,
# what the terminal answered, or null where kit did not ask; $kept, the ids kept, each after @ and its profile where
# the reader gave one.
def checks($d; $key; $read; $bad; $probes; $kept):
  $d.terminals[$key] as $t
  | "\($d.docs)#\($t.anchor)" as $more
  | ({name: $t.name} + (if ($read | type) == "object" then $read else {} end)) as $o
  | (($read | type) == "object" and ($read | has("unreadable") | not)) as $read_ok
  | (if $read_ok and $read.profile then "@\($read.profile)" else "" end) as $at
  | def kept($c):
      if ($c.ok | not) and (($kept | index($c.id + $at)) != null)
      then $c + {ok: true, line: "kept: \($c.line)", fix: null, more: "kit welcome --unkeep=\($c.id)"}
      else $c end;
    [if $read_ok then $t.findings[] else empty end
     | . as $f
     | ($bad[$f.id] == true) as $failed
     | kept({id: "terminal-\($key)-\($f.id)", ok: ($failed | not),
             line: (if $failed then $f.line else $f.held end | fill($o)),
             fix: (if $failed then $f.fix | fill($o) else null end), more: $more})] as $findings
  | [if $read_ok then $t.findings[] | select($bad[.id] == true) | .need else empty end] as $failed_needs
  | ($o + ($probes // {})) as $po
  | [if $probes == null then empty else $d.probes | keys_unsorted[] end
     | . as $p
     | $d.probes[$p] as $probe
     | ($t.probes[$p] // {}) as $row
     | "terminal-probe-\($p)" as $id
     | if $probes.answered == false
       then {id: $id, ok: true, line: "\($t.name) did not answer kit's question", fix: null, more: $more}
       elif ($probes | measured_badly($p) | not) then {id: $id, ok: true, line: ($probe.held | fill($po)), fix: null,
                                                        more: $more}
       elif $read_ok and ($probe.need | IN($failed_needs[])) then empty
       elif $row.none then {id: $id, ok: true, line: $row.none, fix: null, more: $more}
       elif $row.fix then {id: $id, ok: false, line: ($probe.failed | fill($po)), fix: ($row.fix | fill($po)),
                           more: $more}
       else {id: $id, ok: false, line: ($probe.failed | fill($po)) + "; no setting kit knows explains it", fix: null,
             more: $more} end
     | if $p == "emoji-width" and (.ok | not) and $probes.grapheme_width == false
       then .line += ", and it does not measure by grapheme" else . end
     | kept(.)] as $answers
  | ($findings + $answers) as $body
  | {id: "terminal-\($key)", ok: true,
     line: (if $read == null then "\($t.name): \($t.unread)"
            elif $read_ok | not then "\($t.name): kit could not read its settings: \($read.unreadable)"
            else $t.name + (if $read.profile then ", profile \($read.profile)" else "" end)
                 + (if all($body[]; .ok and (.line | startswith("kept: ") | not))
                    then ": nothing in it slows or breaks kittui" else "" end) end),
     fix: null, more: $more}
  | [., $body[]];

# checks($key; $read; $bad; $probes; $kept) gives the checks of the terminal $key of terminals.json.
def checks($key; $read; $bad; $probes; $kept): checks(data; $key; $read; $bad; $probes; $kept);
