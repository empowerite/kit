# toolchain.jq - a tool manager's answers, read for the tools the repository pins (jq module)
# ----------------------------------------------------------------------------------------------------------------------
#
# import "toolchain" as toolchain; with jq -L "${kit_jq}".
#
# see: modules/bash/toolchain.sh, modules/bash/toolchain/brew.sh

# brew_outdated($names) reads brew outdated --json=v2 and gives, for each formula or cask named in $names, one name
# per line, "<name>\t<installed>\t<current>". Brew reports every outdated package on the machine; only the
# Brewfile's count.
def brew_outdated($names):
  ($names | split("\n") | map(select(. != ""))) as $wanted
  | (.formulae + .casks)[]
  | select(.name as $n | $wanted | index($n))
  | "\(.name)\t\(.installed_versions | last)\t\(.current_version)";

# mise_pins reads mise ls --json --current and gives one line per tool, "<name>\t<pinned>\t<installed>". The pin is
# the version mise.toml asks for; installed is the version mise resolved it to, or nothing when it is not installed.
def mise_pins:
  to_entries[]
  | .key as $name
  | .value[]
  | "\($name)\t\(.requested_version // .version)\t\(if .installed then .version else "" end)";

# brew_pins($names) reads brew list --versions, as one string, and gives one line per name in $names, one per line:
# "<name>\t<pinned>\t<installed>". A Brewfile pins a version only in a name such as node@22; installed is the
# version brew lists last, or nothing. brew lists a tap's formula, user/tap/name, by its own name.
def brew_pins($names):
  (split("\n") | map(select(. != "") | split(" ") | {key: .[0], value: .[-1]}) | from_entries) as $installed
  | $names | split("\n")[] | select(. != "")
  | (if test("@") then sub("^[^@]*@"; "") else "" end) as $pinned
  | "\(.)\t\($pinned)\t\($installed[.] // $installed[split("/") | last] // "")";

# nix_inputs reads flake.lock and gives each input the flake names itself, with its node. An input that follows another
# is a list in the lock, and is left out.
def nix_inputs:
  .nodes as $nodes
  | $nodes.root.inputs
  | to_entries[]
  | select(.value | type == "string")
  | {name: .key, node: $nodes[.value]};

# nix_pins gives one line per flake input, "<name>\t<pinned>\t<installed>": its locked revision, short. A source with
# no revision, as a tarball, gives the start of its hash.
def nix_pins:
  nix_inputs
  | (.node.locked | (.rev // ((.narHash // "") | ltrimstr("sha256-")))[:7]) as $rev
  | "\(.name)\t\($rev)\t\($rev)";

# nix_refs gives the inputs that can be asked for a newer revision, "<name>\t<reference>\t<locked revision>". An input
# pinned to a revision in flake.nix, or with no revision locked, or from a path, is left out.
def nix_refs:
  nix_inputs
  | select((.node.original.rev // null) == null and (.node.locked.rev // null) != null)
  | .node.original as $o
  | (if ($o.type == "github" or $o.type == "gitlab" or $o.type == "sourcehut")
     then "\($o.type):\($o.owner)/\($o.repo)" + (if $o.ref then "/\($o.ref)" else "" end)
     elif $o.type == "git" then "git+\($o.url)" + (if $o.ref then "?ref=\($o.ref)" else "" end)
     elif $o.type == "tarball" then $o.url
     elif $o.type == "indirect" then $o.id + (if $o.ref then "/\($o.ref)" else "" end)
     else null end) as $ref
  | select($ref != null)
  | "\(.name)\t\($ref)\t\(.node.locked.rev)";
