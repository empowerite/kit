# workspace.jq - kit workspace's rows, made from what git prints (jq module)
# ----------------------------------------------------------------------------------------------------------------------
# import "workspace" as workspace; with jq -L "${kit_jq}". Each stage's function reads what git printed in one
# repository, as text, its fields apart by the unit separator, and gives the stage's rows, an array. A row is an object
# with key, the row's own within its stage, a value per field, by the field's id, null where it has none, and links,
# where a value leads: a page on a forge, or row:<stage>/<key>, the row of another stage. An age is in seconds before
# $now, which the caller gives.
#
# see: modules/bash/workspace.sh, bin/kit-workspace

# fields reads a line git printed and gives its fields, apart by the unit separator.
def fields: split("\u001f");

# lines reads text and gives each of its lines that is not empty.
def lines: split("\n")[] | select(length > 0);

# some reads a string and gives it, or null for an empty one.
def some: if . == "" then null else . end;

# web($forge; $member) gives a repository's page, https and its member, host and path; null with no forge or member.
def web($forge; $member): if $forge == null or $member == null then null else "https://\($member)" end;

# page($remote; $kind; $name) gives the page of a remote's branch, as tree, or of a commit, as commit, on its forge:
# a remote is {forge, web}; null for one with no forge or no page, and for a forge kit knows no pages of.
def page($remote; $kind; $name):
  if $remote.web == null or $name == null then null
  elif $remote.forge == "github" then "\($remote.web)/\($kind)/\($name)"
  elif $remote.forge == "gitlab" then "\($remote.web)/-/\($kind)/\($name)"
  else null end;

# row($stage; $key) gives a link to the row of another stage, by its key.
def row($stage; $key): if $key == null then null else "row:\($stage)/\($key)" end;

# track reads what git says of a branch beside its upstream, as [ahead 3, behind 1] or +3 -1, and gives
# [ahead, behind]; null for nothing said, or gone.
def track:
  if . == null or . == "" or . == "[gone]" then null
  else [((capture("(ahead |\\+)(?<n>[0-9]+)") // {n: "0"}) | .n | tonumber),
        ((capture("(behind |-)(?<n>[0-9]+)") // {n: "0"}) | .n | tonumber)] end;

# remotes($primary) reads each remote of a repository, {name, url, push, forge, member, default, branches}, and gives
# its rows: the remote, its forge, the organization and the repository it is, the member's path past the host up to
# its last segment and that segment, the host, its URL, its push URL where it differs, its default branch, how many
# branches it has, and whether it is the primary; the repository leads to its page, and the default branch to its
# row. A remote on this machine has no organization, repository, or host.
def remotes($primary):
  map(web(.forge; .member) as $web
    | (.member | if . then sub("^[^/]+/"; "") | split("/") else null end) as $path
    | {key: .name, remote: .name, forge, organization: ($path | if . then .[:-1] | join("/") else null end),
       repository: ($path | if . then .[-1] else null end),
       host: (.member | if . then sub("/.*$"; "") else null end), url,
       push: (if .push == .url then null else .push end), default, branches,
       primary: (if .name == $primary then "primary" else null end),
       links: {repository: $web, default: (if .default then row("branches"; "\(.name)/\(.default)") else null end)}});

# branches($remotes; $now) reads git for-each-ref's lines for refs/heads and refs/remotes, each refname, symref,
# upstream:remotename, upstream:remoteref, upstream:track, worktreepath, objectname:short, committerdate:unix, and
# subject, and gives a row per branch: each a remote has, with the local branches that pull from it, remote; each local
# branch whose upstream is gone, gone; and each with no upstream on a remote, local. $remotes is each remote's {name,
# forge, web, default}; a remote's name may hold a slash, so the longest that starts a ref names it. A row's ahead and
# behind are its first local branch's, [0, 0] where git says nothing, as it does of one even with it; a remote's branch
# leads to its page, and its commit to its own.
def branches($remotes; $now):
  [lines | fields | {ref: .[0], symref: .[1], remote: .[2], merge: .[3], track: .[4], worktree: .[5], commit: .[6],
                     date: (.[7] | tonumber? // null), subject: .[8]}] as $refs
  | ($remotes | sort_by(-(.name | length))) as $longest
  | [$refs[] | select(.ref | startswith("refs/heads/")) | . + {branch: (.ref | ltrimstr("refs/heads/"))}] as $locals
  | [$refs[] | select((.ref | startswith("refs/remotes/")) and .symref == "")
     | (.ref | ltrimstr("refs/remotes/")) as $rest
     | first($longest[] | select(.name as $name | $rest | startswith($name + "/"))) as $at
     | . + {remote: $at.name, branch: ($rest | ltrimstr($at.name + "/")), at: $at}] as $tracked
  | def pulls($t): $locals | map(select(.remote == $t.remote and .merge == "refs/heads/" + $t.branch));
    def worktrees($ls): $ls | map(.worktree | select(. != ""));
    def age($date): if $date == null then null else $now - $date end;
  [$tracked[] as $t | pulls($t) as $ls
   | {key: "\($t.remote)/\($t.branch)", kind: "remote", branch: $t.branch, remote: $t.remote,
      local: ($ls | map(.branch)), upstream: (if $ls == [] then null else $ls[0].track | track // [0, 0] end),
      default: (if $t.at.default == $t.branch then "default" else null end),
      commit: $t.commit, age: age($t.date), subject: $t.subject, worktree: worktrees($ls),
      links: {branch: page($t.at; "tree"; $t.branch), commit: page($t.at; "commit"; $t.commit),
              worktree: row("worktrees"; worktrees($ls)[0])}}]
  + [$locals[] | select(. as $l | $tracked | any(.remote == $l.remote and "refs/heads/" + .branch == $l.merge) | not)
     | (.remote != "" and .remote != "." and .merge != "") as $upstream
     | {key: (if $upstream then "\(.remote)/\(.merge | ltrimstr("refs/heads/"))" else .branch end),
        kind: (if $upstream then "gone" else "local" end), branch, remote: (if $upstream then .remote else null end),
        local: [.branch], upstream: null, default: null, commit, age: age(.date), subject,
        worktree: worktrees([.]), links: {branch: null, commit: null, worktree: row("worktrees"; worktrees([.])[0])}}];

# status reads git status --porcelain=v2 --branch and gives {upstream, track, changes, untracked}: the upstream, what
# the branch is ahead and behind it, as track does, the paths changed, staged or not, and the paths untracked.
def status:
  [lines] as $lines
  | ($lines | map(select(startswith("# branch.upstream ")) | ltrimstr("# branch.upstream "))[0]) as $upstream
  | {upstream: $upstream,
     track: (if $upstream then ($lines | map(select(startswith("# branch.ab ")) | ltrimstr("# branch.ab "))[0] | track)
             else null end),
     changes: ($lines | map(select(test("^[12u] "))) | length),
     untracked: ($lines | map(select(startswith("? "))) | length)};

# worktrees($statuses; $bases) reads git worktree list --porcelain and gives a row per worktree, the desk first: its
# kind, desk or linked, its path, its branch, or detached, its HEAD's short hash, its upstream and what it is ahead and
# behind it, what it is ahead and behind the default branch, its paths changed and untracked, and whether it is locked
# or prunable. $statuses gives each worktree's git status --porcelain=v2 --branch by its path, and $bases what git
# rev-list --left-right --count <default>...HEAD printed there; a worktree with neither has none of what they say. Its
# branch leads to its row of branches: its upstream's, else its own.
def worktrees($statuses; $bases):
  [split("\n\n")[] | [lines | capture("^(?<k>[A-Za-z]+) ?(?<v>.*)$") | {(.k): .v}] | add | select(. != null)]
  | to_entries
  | map(.key as $n | .value
    | (($statuses[.worktree] // "") | status) as $s
    | (($bases[.worktree] // "") | split("\t") | map(tonumber? // empty)) as $b
    | (.branch // null | if . then ltrimstr("refs/heads/") else null end) as $branch
    | {key: .worktree, kind: (if $n == 0 then "desk" else "linked" end), path: .worktree, branch: $branch,
       detached: (if has("detached") then "detached" else null end), head: (.HEAD // "" | .[0:8] | some),
       upstream: $s.upstream, track: $s.track, base: (if ($b | length) == 2 then [$b[1], $b[0]] else null end),
       changes: (if $statuses[.worktree] then $s.changes else null end),
       untracked: (if $statuses[.worktree] then $s.untracked else null end),
       locked: (if has("locked") then "locked" else null end),
       prunable: (if has("prunable") then "prunable" else null end),
       links: {branch: row("branches"; $s.upstream // $branch)}});

# commits($now; $branches; $unpushed; $heads; $primary) reads git log's lines, each full hash, short hash,
# committer date, author, and subject, and gives a row per commit: its short hash, its subject, its author, its age,
# the issue its subject starts with, as #7, the local branches that hold it, as $branches gives them by hash, pushed or
# local, as $unpushed, an object of the hashes no remote has, says, and the worktrees whose HEAD it is, as $heads gives
# them by hash. A pushed commit leads to its page on the primary, $primary, {forge, web}, and an issue to its page.
def commits($now; $branches; $unpushed; $heads; $primary):
  [lines | fields
   | ((.[4] | capture("^#(?<n>[0-9]+)\\b") | .n) // null) as $issue
   | ($unpushed[.[0]] // false | not) as $pushed
   | {key: .[0], commit: .[1], subject: .[4], author: .[3], age: ($now - (.[2] | tonumber)),
      issue: (if $issue then "#\($issue)" else null end), branches: ($branches[.[0]] // []),
      pushed: (if $pushed then "pushed" else "local" end), worktrees: ($heads[.[0]] // []),
      links: {commit: (if $pushed then page($primary; "commit"; .[0]) else null end),
              issue: (if $issue then page($primary; "issues"; $issue) else null end),
              worktrees: row("worktrees"; ($heads[.[0]] // [])[0])}}];

# hashes reads hashes, one a line, as git rev-list prints them, and gives an object of them, each true.
def hashes: [lines | {(.): true}] | add // {};

# held reads lines of a hash and a branch, apart by a tab, and gives the branches that hold each hash, by the hash.
def held:
  [lines | split("\t") | {sha: .[0], branch: .[1]}] | group_by(.sha) | map({(.[0].sha): map(.branch)}) | add // {};

# heads reads git worktree list --porcelain and gives the worktrees whose HEAD each commit is, by its hash.
def heads:
  [split("\n\n")[] | [lines] | {path: (.[] | select(startswith("worktree ")) | ltrimstr("worktree ")),
                                  head: ((.[] | select(startswith("HEAD ")) | ltrimstr("HEAD ")) // null)}
   | select(.head)]
  | group_by(.head) | map({(.[0].head): map(.path)}) | add // {};

# tags($now) reads git for-each-ref's lines for refs/tags, each refname:short, objecttype, *objectname:short,
# objectname:short, taggername, creatordate:unix, and subject, and gives a row per tag: its name, annotated or
# lightweight, the commit it names, its tagger, an annotated tag's alone, its age, and its subject.
def tags($now):
  [lines | fields
   | {key: .[0], tag: .[0], kind: (if .[1] == "tag" then "annotated" else "lightweight" end),
      commit: ((.[2] | some) // .[3]), tagger: (.[4] | some), age: ((.[5] | tonumber? | $now - .) // null),
      subject: (.[6] | some), links: {}}];

# shown($as) reads a row's value and gives it as text prints it: an age; ahead and behind, as track, ↑3 ↓1, or = for
# neither; a clone, as checkout, or not cloned; worktrees, as worktrees, each by its branch, or its folder; a list, its
# values apart by commas; nothing for none.
def shown($as):
  if . == null then (if $as == "checkout" then "not cloned" else "" end)
  elif $as == "age" then (if . < 120 then "\(.)s" elif . < 7200 then "\(. / 60 | floor)m"
                          elif . < 172800 then "\(. / 3600 | floor)h" else "\(. / 86400 | floor)d" end)
  elif $as == "track" then
    (if . == [0, 0] then "="
     else [(select(.[0] > 0) | "↑\(.[0])"), (select(.[1] > 0) | "↓\(.[1])")] | join(" ") end)
  elif $as == "worktrees" then map(.branch // (.path | split("/") | last)) | join(", ")
  elif type == "array" then join(", ")
  else tostring end;

# columns($stage) gives the columns a stage's text draws, each {id, label, as}, as shown takes it.
def columns($stage):
  {repos: [{id: "repository"}, {id: "forge"}, {id: "branch"}, {id: "ahead"}, {id: "behind"},
          {id: "checkout", as: "checkout"}, {id: "worktrees", as: "worktrees"}, {id: "branches"}, {id: "error"}],
   remotes: [{id: "remote"}, {id: "organization"}, {id: "repository"}, {id: "forge"}, {id: "url"},
             {id: "default"}, {id: "branches"}],
   branches: [{id: "branch"}, {id: "kind"}, {id: "remote"}, {id: "local"}, {id: "upstream", as: "track"},
              {id: "commit"}, {id: "age", as: "age"}, {id: "worktree"}],
   worktrees: [{id: "path"}, {id: "kind"}, {id: "branch"}, {id: "head"}, {id: "upstream"},
               {id: "track", as: "track"}, {id: "base", label: "default", as: "track"}, {id: "changes"},
               {id: "untracked"}],
   commits: [{id: "commit"}, {id: "age", as: "age"}, {id: "author"}, {id: "pushed"}, {id: "branches"},
             {id: "subject"}],
   tags: [{id: "tag"}, {id: "kind"}, {id: "commit"}, {id: "age", as: "age"}, {id: "tagger"}, {id: "subject"}]}[$stage];

# text($stage) reads a stage's rows and gives its lines as text: a line of the columns' names, then a line per row,
# each column as wide as its widest, two spaces apart, the last unpadded; a stage with no rows gives none.
def text($stage):
  . as $rows | columns($stage) as $columns
  | [$columns[] | .label // .id] as $names
  | [$rows[] | [$columns[] as $c | .[$c.id] | shown($c.as)]] as $cells
  | [range($names | length) as $k | ([$names[$k], $cells[][$k]] | map(length) | max)] as $widths
  | def line:
      [range(length) as $k | .[$k] + (if $k < length - 1 then " " * ($widths[$k] - (.[$k] | length) + 2) else "" end)]
      | join("") | sub(" +$"; "");
  if ($rows | length) == 0 then empty else ($names | line), ($cells[] | line) end;
