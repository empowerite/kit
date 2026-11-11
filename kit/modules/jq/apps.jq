# apps.jq - the app verbs' documents: a forge call's objects as rows, kit's own reads, a stage as text (jq module)
# ----------------------------------------------------------------------------------------------------------------------
# import "apps" as apps; with jq -L "${kit_jq}". A stage's document is {rows, errors}, and limited where a capped call
# stopped. A row is an object with key, the row's own within its stage, a value per field, null where it has none, and
# links, where a value leads: a page on a forge, or row:<stage>/<key>, the row of another stage.
#
# see: modules/bash/apps.sh

# fold(f) reads the objects a forge call printed, slurped, and gives the document: each object through f, a row, and
# limited from a {"limited": N} object, which a capped call prints last.
def fold(f):
  (map(select(type == "object" and keys == ["limited"]) | .limited) | first) as $limited
  | {rows: [.[] | select((type == "object" and keys == ["limited"]) | not) | f], errors: []}
    + (if $limited == null then {} else {limited: $limited} end);

# merge reads an array of documents and gives one: the rows joined in order, the errors joined, and limited the least
# of those that hold one.
def merge:
  [.[].limited | numbers] as $limited
  | {rows: map(.rows[]), errors: map(.errors[])} + (if $limited == [] then {} else {limited: ($limited | min)} end);

# toolchain reads a row kit toolchains --json prints and gives it keyed by its provider and role.
def toolchain: {key: "\(.provider) \(.role)"} + . + {links: {}};

# tool reads a row kit tools --json prints and gives it keyed by where it is pinned and its name.
def tool: {key: "\(.from) \(.name)"} + . + {links: {}};

# settings($schema) reads kit.yml as JSON, null for none, and gives a row per key the schema names, in its order: the
# value kit.yml sets, from set, else the schema's default, null where it names none, from default.
def settings($schema):
  (if type == "object" then . else {} end) as $set
  | [$schema.properties | to_entries[] | .key as $k
     | {key: $k, setting: $k}
       + if $set | has($k) then {value: $set[$k], from: "set"} else {value: .value.default, from: "default"} end
       + {links: {}}];

# milestone reads a milestone the forge's milestones call prints and gives it keyed by its number, its page the link of
# its title.
def milestone: {key: .number} + del(.url) + {links: {title: .url}};

# iteration reads an iteration the forge's iterations call prints and gives it keyed by its id, current where now falls
# between its start and its end, its page the link of its title.
def iteration:
  {key: .id} + del(.url)
  + {current: (.start != null and .end != null and .start <= now and now <= .end), links: {title: .url}};

# shown reads a row's value and gives it as text prints it: nothing for none, a list's values apart by commas, an
# object as JSON.
def shown:
  if . == null then ""
  elif type == "array" then map(if type == "string" then . else tojson end) | join(", ")
  elif type == "object" then tojson
  else tostring end;

# text reads a stage's rows and gives its lines as text: a line of the fields' names, in the order the rows first hold
# them, less key, links, and member, then a line per row, each column as wide as its widest, two spaces apart, the
# last unpadded; a stage with no rows gives none.
def text:
  . as $rows
  | (reduce ($rows[] | keys_unsorted[]) as $k ([]; if index([$k]) then . else . + [$k] end)
     | map(select(IN("key", "links", "member") | not))) as $names
  | [$rows[] | [.[$names[]] | shown]] as $cells
  | [range($names | length) as $k | ([$names[$k], $cells[][$k]] | map(length) | max)] as $widths
  | def line:
      [range(length) as $k | .[$k] + (if $k < length - 1 then " " * ($widths[$k] - (.[$k] | length) + 2) else "" end)]
      | join("") | sub(" +$"; "");
  if ($rows | length) == 0 then empty else ($names | line), ($cells[] | line) end;

# ----------------------------------------------------------------------------------------------------------------------
