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
