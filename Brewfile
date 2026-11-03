# Brewfile - the few tools Homebrew installs on a macOS desk; see: https://docs.brew.sh/Brew-Bundle-and-Brewfile
#
# mise pins the tools the gate runs; see: mise.toml. Homebrew installs the rest:
#
#   - what kit stands on before it chooses a toolchain, the system role in docs/toolchains.md
#   - what puts mise's tools on PATH
#
# Run `brew bundle install` once per machine. A Brewfile pins a formula by name only, so its version moves with
# Homebrew. git and curl come from the machine.
#
# On macOS, brew fills in for the tools mise does not pin: kit tools lists, installs, and updates bash, direnv, and
# mise through it. On Linux, kit does not read this file. See docs/toolchains.md#brew-fills-in.
#
# Why each tool is here:
#   bash     runs kit, which needs 4.4 or later; macOS ships 3.2
#   direnv   runs .envrc on cd, which puts kit/bin on PATH and hydrates .kit/
#   jq       reads every verb's arguments, before any toolchain is chosen
#   mise     installs the tools mise.toml and mise.desk.toml pin
#   yq       reads kit.yml, before any toolchain is chosen

brew "bash"
brew "direnv"
brew "jq"
brew "mise"
brew "yq"
