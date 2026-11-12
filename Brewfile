# Brewfile - the few tools Homebrew installs on a macOS desk; see: https://docs.brew.sh/Brew-Bundle-and-Brewfile
#
# The kit toolchain pins the tools the gate runs; see: tools: in kit.yml. Homebrew installs the rest:
#
#   - what kit stands on before it chooses a toolchain, the system role in docs/toolchains.md
#   - what puts the kit toolchain's tools on PATH
#
# Run `brew bundle install` once per machine. A Brewfile pins a formula by name only, so its version moves with
# Homebrew. git and curl come from the machine.
#
# On macOS, brew fills in for the tools the kit toolchain does not pin: kit tools lists, installs, and updates bash and
# direnv through it. On Linux, kit does not read this file. See docs/toolchains/using-brew.md#brew-fills-in.
#
# Why each tool is here:
#   bash     runs kit, which needs 4.4 or later; macOS ships 3.2
#   direnv   runs .envrc on cd, which puts kit/bin and the kit toolchain's tools on PATH, and hydrates .kit/
#   jq       reads every verb's arguments, before any toolchain is chosen
#   yq       reads kit.yml, before any toolchain is chosen

brew "bash"
brew "direnv"
brew "jq"
brew "yq"
