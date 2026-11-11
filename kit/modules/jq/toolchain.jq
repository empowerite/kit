# toolchain.jq - a tool manager's answers, read for the tools the repository pins (jq module)
# ----------------------------------------------------------------------------------------------------------------------
# import "toolchain" as toolchain; with jq -L "${kit_jq}".
#
# see: modules/bash/toolchain.sh, modules/bash/toolchain/brew.sh, modules/bash/toolchain/devcontainer.sh

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

# nix_query($pairs) joins the attributes a reference carries after the ?, from [name, value] pairs that have a value.
def nix_query($pairs):
  [$pairs[] | select(.[1] != null and .[1] != false) | "\(.[0])=\(.[1] | if . == true then 1 else . end)"]
  | if length == 0 then "" else "?" + join("&") end;

# nix_refs gives the inputs that can be asked for a newer revision, "<name>\t<reference>\t<locked revision>". The
# reference carries the attributes the lock's original does: ref, dir, host, and submodules. Left out are an input
# pinned to a revision in flake.nix, one with no revision locked, one from a path, and one that is not a flake
# (flake = false), which nix flake metadata cannot read.
def nix_refs:
  nix_inputs
  | select((.node.original.rev // null) == null and (.node.locked.rev // null) != null)
  | select(.node.flake != false)
  | .node.original as $o
  | (if ($o.type == "github" or $o.type == "gitlab" or $o.type == "sourcehut")
     then "\($o.type):\($o.owner)/\($o.repo)" + (if $o.ref then "/\($o.ref)" else "" end)
       + nix_query([["dir", $o.dir], ["host", $o.host]])
     elif $o.type == "git"
     then "git+\($o.url)" + nix_query([["ref", $o.ref], ["dir", $o.dir], ["submodules", $o.submodules]])
     elif $o.type == "tarball" and ($o.dir // null) == null then $o.url
     elif $o.type == "indirect" then $o.id + (if $o.ref then "/\($o.ref)" else "" end) + nix_query([["dir", $o.dir]])
     else null end) as $ref
  | select($ref != null)
  | "\(.name)\t\($ref)\t\(.node.locked.rev)";

# kit_wanted($platforms) reads the tools under tools: in kit.yml, each {name, version, url, bin, profile, os, arch,
# platforms}, and gives one line per tool and platform it is locked for: "<name>\t<version>\t<platform>\t<profile>\t
# <url>\t<bin>". $platforms, space-separated, are the platforms a tool with no platforms: of its own is locked for. In
# the url and the bin, {version}, {platform}, {os}, and {arch} are filled in, os: and arch: renaming kit's linux,
# macos, x64, and arm64 for the tool's own.
def kit_wanted($platforms):
  .[] as $tool
  | ($tool.platforms | if length > 0 then . else ($platforms | split(" ")) end)[] as $platform
  | ($platform | split("-")) as [$os, $arch]
  | def fill: gsub("\\{version\\}"; $tool.version) | gsub("\\{platform\\}"; $platform)
      | gsub("\\{os\\}"; $tool.os[$os] // $os) | gsub("\\{arch\\}"; $tool.arch[$arch] // $arch);
    [$tool.name, $tool.version, $platform, $tool.profile, ($tool.url | fill), ($tool.bin | fill)] | join("\t");

# devcontainer_pins reads devcontainer-lock.json and gives one line per locked feature, "<id>\t<pinned>\t<installed>".
# The lock records the version each feature resolved to, and nothing of what was built, so the two columns agree.
def devcontainer_pins:
  (.features // {}) | to_entries[]
  | "\(.key)\t\(.value.version // "")\t\(.value.version // "")";

# devcontainer_outdated reads devcontainer outdated --output-format json and gives, for each feature whose current
# version differs from the latest, "<id>\t<current>\t<latest>". A feature with no latest is not reported.
def devcontainer_outdated:
  (.features // {}) | to_entries[]
  | select(.value.latest != null and .value.current != .value.latest)
  | "\(.key)\t\(.value.current // "")\t\(.value.latest)";
