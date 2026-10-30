# toolchain.jq - a tool manager's answers, read for the tools the repository pins (jq module)
# ----------------------------------------------------------------------------------------------------------------------
#
# import "toolchain" as toolchain; with jq -L "${kit_jq}".
#
# see: modules/bash/toolchain.sh, modules/bash/toolchain-brew.sh

# brew_outdated($names) reads brew outdated --json=v2 and gives, for each formula or cask named in $names, one name
# per line, "<name>\t<installed>\t<current>". Brew reports every outdated package on the machine; only the
# Brewfile's count.
def brew_outdated($names):
  ($names | split("\n") | map(select(. != ""))) as $wanted
  | (.formulae + .casks)[]
  | select(.name as $n | $wanted | index($n))
  | "\(.name)\t\(.installed_versions | last)\t\(.current_version)";
