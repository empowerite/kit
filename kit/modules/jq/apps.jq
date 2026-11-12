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

# checked($path; $branch) reads a row kit tools --json prints in the checkout at $path, on $branch, "" for none, and
# gives it as tool does, keyed by the checkout too, and saying the checkout's folder as its worktree and its branch.
def checked($path; $branch):
  tool | .key = "\($path) \(.key)" | . + {worktree: ($path | split("/") | last), branch: (if $branch == "" then null else $branch end)};

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

# review reads a change the forge's open-prs call prints and gives a row per reviewer, keyed by the change's number and
# the reviewer's login: the reviewer's state, when they gave it, and the change's unresolved threads, on each row.
def review:
  . as $change
  | .reviewers[]?
  | { key: "\($change.number) \(.login)", change: $change.number, title: $change.title, login, state, at,
      unresolved: $change.unresolved, links: {} };

# check reads a change the forge's open-prs call prints and gives a row per check, keyed by the change's number and the
# check's name: the latest run of each name, by at, so a rerun supersedes the run before it.
def check:
  . as $change
  | (.runs // []) | group_by(.name) | map(sort_by(.at) | last)[]
  | { key: "\($change.number) \(.name)", change: $change.number, title: $change.title, name, verdict, at, took,
      links: {} };

# merged reads a change the forge's merged-prs call prints and gives it keyed by its number, its page the link of its
# title.
def merged: {key: .number} + del(.url) + {links: {title: .url}};

# linked reads a row's links and keeps those that lead somewhere.
def linked: .links |= with_entries(select(.value != null));

# artifact reads an artifact the forge's artifacts call prints and gives it keyed by its id, its page the link of its
# name.
def artifact: ({key: .id} + del(.url) + {links: {name: .url}}) | linked;

# environment reads an environment the forge's environments call prints and gives it keyed by its name, its page the
# link of its name.
def environment: ({key: .name} + del(.url) + {links: {name: .url}}) | linked;

# deployment reads a deployment the forge's deployments call prints and gives it keyed by its id: its environment
# links to that row of the environments stage, row:environments/<name>, its sha to the commit's page, and its state to
# its log or page.
def deployment:
  ({key: .id} + del(.url, .commit)
   + {links: {environment: (if .environment == null then null else "row:environments/\(.environment)" end),
              sha: .commit, state: .url}})
  | linked;

# person reads one with access the forge's collaborators call prints and gives it keyed by the login.
def person: {key: .login} + . + {links: {}};

# team reads a team the forge's teams call prints and gives it keyed by the team, its page the link of the team.
def team: ({key: .team} + del(.url) + {links: {team: .url}}) | linked;

# agent reads a bot the forge's bots call prints and gives it keyed by its login, its page the link of the login.
def agent: ({key: .login} + del(.url) + {links: {login: .url}}) | linked;

# keyed(f) gives the key of the values f gives, those that are not null, apart by a space.
def keyed(f): [f | select(. != null) | tostring] | join(" ");

# attestation reads an attestation the forge's attestations call prints and gives it keyed by its digest and its
# kind, its page the link of its subject.
def attestation: ({key: keyed(.digest, .kind)} + del(.url) + {links: {subject: .url}}) | linked;

# kept reads an object a forge call prints and gives it as it is, for a stage that reads it twice, as Trust's alerts
# are read once for the findings and the waivers.
def kept: .;

# alert($tool) reads an alert one of the forge's alerts calls prints, the call of the feature $tool, and gives it
# keyed by the tool and its number, the tool its own field, its page the link of its rule; who dismissed it, and why,
# are the waivers'.
def alert($tool):
  ({key: "\($tool) \(.number)", tool: $tool} + del(.url, .by, .reason) + {links: {rule: .url}}) | linked;

# control reads a control the forge's rulesets call prints and gives it keyed by its id, its page the link of the
# control, and its source, a ruleset or a protection, as its kind, so source says the forge, as every row's does.
def control: ({key: .id} + del(.url, .source) + {kind: .source, links: {control: .url}}) | linked;

# waiver reads a waiver the forge's bypasses call prints, or one dismissal gives, and gives it keyed by what it waives
# and for whom, or for those hidden, <waives> hidden: what it waives links to that row, of the findings stage for a
# dismissal and of the controls stage for a bypass, row:<stage>/<key>, and its page is the link of its reason.
def waiver:
  ({key: (if .hidden != null then "\(.waives) hidden" else keyed(.waives, .actor) end)} + del(.url)
   + {links: {waives: (if .kind == "dismissal" then "row:findings/\(.waives)" else "row:controls/\(.waives)" end),
              reason: .url}})
  | linked;

# dismissal($tool) reads an alert one of the forge's alerts calls prints, the call of the feature $tool, and gives the
# waiver of a dismissed one, as waiver does: waives <tool> <number>, by who dismissed it, and the reason; an open one
# gives none.
def dismissal($tool):
  select(.state == "dismissed")
  | {waives: "\($tool) \(.number)", actor: null, kind: "dismissal", reason: (.reason // null), by: (.by // null),
     until: null, hidden: null, url}
  | waiver;

# rule reads a rule kit policy --rules --json prints and gives it keyed by the rule.
def rule: {key: .rule} + . + {links: {}};

# placed($where) reads a rule kit policy --rules --json prints and gives it as rule does, where $where says it is in
# force: {source, organization, repository, branch}, its scope branch, as one checkout's policy is.
def placed($where): rule as $rule | {key: $rule.key} + $where + {scope: "branch"} + ($rule | del(.key));

# scope_levels are the places a rule's row can be in force at, the widest first: each is the row's field of its name.
def scope_levels: ["source", "organization", "repository", "branch"];

# scoped reads rows of one rule that agree and gives the one row that stands for them all, at the narrowest place
# that holds them: the levels before the first whose values differ among them are kept, the rest left null, and scope
# names the last one kept, workspace where even the sources differ. Rows at one place, the same in every level, give
# the first as it is, its scope its own.
def scoped:
  . as $rows
  | ([range(scope_levels | length) | select(. as $i | $rows | map(.[scope_levels[$i]]) | unique | length > 1)]
     | first) as $first
  | if $first == null then $rows[0]
    else $rows[0]
      | reduce range($first; scope_levels | length) as $i (.; .[scope_levels[$i]] = null)
      | .scope = (if $first == 0 then "workspace" else scope_levels[$first - 1] end)
      # a row wider than a repository is no member's
      | if $first < 3 and has("member") then .member = null else . end
    end;

# folded($i) reads rows of one rule, all at one place of level $i - 1, and gives them merged where they agree: all
# of them as one row, by scoped, where their pattern, from, and engine are the same; else each place of level $i's,
# folded in turn.
def folded($i):
  if (map([.pattern, .from, .engine]) | unique | length) == 1 then [scoped]
  elif $i == (scope_levels | length) then .
  else group_by(.[scope_levels[$i]]) | map(folded($i + 1)) | add end;

# scopes reads a stage's policy rows, each placed, and gives them with the rows that agree merged, as folded merges
# them from the widest place down, the rules in the order they first come, each keyed by the rule and where it is in
# force. Merged rows fold again the same, so a workspace's members, each folded, fold once more over all of them.
def scopes:
  (reduce .[].rule as $rule ([]; if index([$rule]) then . else . + [$rule] end)) as $order
  | group_by(.rule) | sort_by(.[0].rule as $rule | $order | index($rule)) | map(folded(0)) | add // []
  | map(.key = ([.rule, (.source, .organization, .repository, .branch | values)] | join(" ")));

# shown reads a row's value and gives it as text prints it: nothing for none, a list's values apart by commas, an
# object as JSON.
def shown:
  if . == null then ""
  elif type == "array" then map(if type == "string" then . else tojson end) | join(", ")
  elif type == "object" then tojson
  else tostring end;

# text reads a stage's rows and gives its lines as text: a line of the fields' names, in the order the rows first hold
# them, less key, links, member, and where each row is from, organization, repository, and source, which the rows
# of a member are drawn under, then a line per row, each column as wide as its widest, two spaces apart, the last
# unpadded; a stage with no rows gives none.
def text:
  . as $rows
  | (reduce ($rows[] | keys_unsorted[]) as $k ([]; if index([$k]) then . else . + [$k] end)
     | map(select(IN("key", "links", "member", "organization", "repository", "source") | not))) as $names
  | [$rows[] | [.[$names[]] | shown]] as $cells
  | [range($names | length) as $k | ([$names[$k], $cells[][$k]] | map(length) | max)] as $widths
  | def line:
      [range(length) as $k | .[$k] + (if $k < length - 1 then " " * ($widths[$k] - (.[$k] | length) + 2) else "" end)]
      | join("") | sub(" +$"; "");
  if ($rows | length) == 0 then empty else ($names | line), ($cells[] | line) end;

# ----------------------------------------------------------------------------------------------------------------------
