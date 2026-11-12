#!/usr/bin/env bats
# forge.bats - each remote's forge, the primary remote, and the provider: the primary's forge, refused by name if unknown
# ----------------------------------------------------------------------------------------------------------------------

load helper

setup() {
  fresh-repo
}

# provider prints the provider the seam picks here.
provider() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use forge && forge:provider'
}

@test "an origin at github.com names the github provider, and one at gitlab.com the gitlab one, by https or ssh" {
  git remote add origin https://github.com/empowerite/kit.git
  run provider
  [ "$status" -eq 0 ]
  [ "$output" = "github" ]
  git remote set-url origin git@github.com:empowerite/kit.git
  run provider
  [ "$output" = "github" ]
  git remote set-url origin git@gitlab.com:empowerite/kit.git
  run provider
  [ "$output" = "gitlab" ]
}

@test "an origin elsewhere, or none, is refused naming the setting and the forges the kit knows" {
  git remote add origin https://codeberg.org/empowerite/kit.git
  run provider
  [ "$status" -eq 1 ]
  [[ "$output" == *"set forge in kit.yml; origin's host 'codeberg.org' names no forge the kit knows"* ]]
  [[ "$output" == *"(ado github gitlab, or none)"* ]]
  # a local path has no host, so it is named as a path
  git remote set-url origin "${BATS_TEST_TMPDIR}/origin.git"
  run provider
  [ "$status" -eq 1 ]
  [[ "$output" == *"origin is a local path, so it does not name a forge; set forge in kit.yml"*"(ado github gitlab, or none)"* ]]
  # no origin at all says so, and how to add one
  git remote remove origin
  run provider
  [ "$status" -eq 1 ]
  [[ "$output" == *"this repository does not have a remote named origin, so kit cannot tell its forge;"* ]]
  [[ "$output" == *"run: cd "*" && git remote add origin <url>, or set forge in kit.yml"* ]]
  [[ "$output" != *"'none'"* ]]
}

@test "the forge setting wins, and an unknown one is refused by name" {
  git remote add origin https://codeberg.org/empowerite/kit.git
  mkdir .kit && printf 'forge: github\n' > kit.yml
  run provider
  [ "$status" -eq 0 ]
  [ "$output" = "github" ]
  printf 'forge: sourcehut\n' > kit.yml
  run provider
  [ "$status" -eq 1 ]
  [[ "$output" == *"'sourcehut' is not a forge the kit knows; the forges are: ado github gitlab"* ]]
}

# forge-of <remote> prints the forge the seam names for the remote here, or nothing.
forge-of() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use forge && forge:of "$1"' _ "$1"
}

# in-forge <function> <argument>… runs one of the forge module's functions here.
in-forge() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use forge && "$@"' _ "$@"
}

@test "a URL's host drops the user and the port; a path and a file:// URL have none" {
  run in-forge forge:url-host https://ada@git.example.test:8443/team/kit.git
  [ "$output" = "git.example.test" ]
  run in-forge forge:url-host ssh://git@gitlab.com/team/kit.git
  [ "$output" = "gitlab.com" ]
  run in-forge forge:url-host git@github.com:empowerite/kit.git
  [ "$output" = "github.com" ]
  run in-forge forge:url-host /srv/git/kit.git
  [ -z "$output" ]
  run in-forge forge:url-host file:///srv/git/kit.git
  [ -z "$output" ]
  run in-forge forge:url-host ../kit.git
  [ -z "$output" ]
}

@test "each remote's forge: the forges map, then the host, then the hosts gh or glab logged in to, else none" {
  git remote add origin https://github.com/empowerite/kit.git
  git remote add upstream https://git.example.test/team/kit.git
  git remote add lab https://code.example.test/team/kit.git
  git remote add plain https://elsewhere.test/team/kit.git
  git remote add local "${BATS_TEST_TMPDIR}/kit.git"
  git config url.https://gitlab.com/.insteadOf gl:
  git remote add short gl:team/kit.git
  run forge-of origin
  [ "$output" = github ]
  run forge-of short
  [ "$output" = gitlab ]
  run forge-of upstream
  [ -z "$output" ]
  # gh's login file, at its default place under XDG_CONFIG_HOME, names the host
  mkdir -p "${XDG_CONFIG_HOME}/gh"
  printf 'git.example.test:\n    user: ada\n    git_protocol: https\n' > "${XDG_CONFIG_HOME}/gh/hosts.yml"
  run forge-of upstream
  [ "$output" = github ]
  # GH_CONFIG_DIR wins over the default place, and glab's file names its own hosts
  mkdir -p "${BATS_TEST_TMPDIR}/gh" "${BATS_TEST_TMPDIR}/glab"
  printf 'github.com:\n    user: ada\n' > "${BATS_TEST_TMPDIR}/gh/hosts.yml"
  printf 'hosts:\n    code.example.test:\n        api_host: code.example.test\n' > "${BATS_TEST_TMPDIR}/glab/config.yml"
  GH_CONFIG_DIR="${BATS_TEST_TMPDIR}/gh" run forge-of upstream
  [ -z "$output" ]
  GLAB_CONFIG_DIR="${BATS_TEST_TMPDIR}/glab" run forge-of lab
  [ "$output" = gitlab ]
  run forge-of lab
  [ -z "$output" ]
  run forge-of plain
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run forge-of local
  [ -z "$output" ]
  # the forges map wins over the host
  printf 'forges: {plain: gitlab, origin: gitlab}\n' > kit.yml
  run forge-of plain
  [ "$output" = gitlab ]
  run forge-of origin
  [ "$output" = gitlab ]
  # for the primary, the forge setting wins over the forges map
  printf 'forge: github\nforges: {plain: gitlab, origin: gitlab}\n' > kit.yml
  run forge-of origin
  [ "$output" = github ]
  run forge-of plain
  [ "$output" = gitlab ]
}

@test "on a runner, the server's host names the primary's forge, and a real host nothing names is never none" {
  unset GITHUB_SERVER_URL CI_SERVER_HOST SYSTEM_COLLECTIONURI
  git remote add origin https://ghe.example.com/o/r.git
  GITHUB_SERVER_URL=https://ghe.example.com run forge-of origin
  [ "$output" = github ]
  GITHUB_SERVER_URL=https://github.com run forge-of origin
  [ -z "$output" ]
  CI_SERVER_HOST=ghe.example.com run forge-of origin
  [ "$output" = gitlab ]
  SYSTEM_COLLECTIONURI=https://ghe.example.com/tfs/DefaultCollection/ run forge-of origin
  [ "$output" = ado ]
  # nothing names the host: the primary has a forge the kit cannot tell, so it is not taken for none
  run in-forge forge:none
  [ "$status" -eq 1 ]
  GITHUB_SERVER_URL=https://ghe.example.com run in-forge forge:none
  [ "$status" -eq 1 ]
  # a local path, and a primary with no remote, have no host, and no forge
  git remote set-url origin "${BATS_TEST_TMPDIR}/origin.git"
  run in-forge forge:none
  [ "$status" -eq 0 ]
  git remote remove origin
  run in-forge forge:none
  [ "$status" -eq 0 ]
}

@test "forge: none says a primary on a host has no forge, before its host and a runner's server name one" {
  git remote add origin git@bitbucket.org:empowerite/kit.git
  run in-forge forge:none
  [ "$status" -eq 1 ]
  run provider
  [[ "$output" == *"(ado github gitlab, or none)"* ]]
  printf 'forge: none\n' > kit.yml
  run in-forge forge:none
  [ "$status" -eq 0 ]
  run forge-of origin
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  # a runner's server that matches the host does not outrank the setting, as Forgejo's names itself in GITHUB_SERVER_URL
  git remote set-url origin https://forgejo.example.com/o/r.git
  GITHUB_SERVER_URL=https://forgejo.example.com run forge-of origin
  [ -z "$output" ]
  # a verb that needs the forge says that kit.yml chose none
  run provider
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit.yml sets forge: none"* ]]
  run in-forge forge:capabilities
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output '[.forge, .stand_ins]' <<<"$output")" = '[null,{}]' ]
  # a remote other than the primary is none through forges
  printf 'forges: {origin: none}\n' > kit.yml
  run in-forge forge:none
  [ "$status" -eq 0 ]
}

@test "a primary whose remotes git cannot list is refused, saying the remotes could not be read" {
  git remote add origin https://github.com/empowerite/kit.git
  printf 'primary: origin\n' > kit.yml
  run bash -c 'source "${kit_root}/modules/bash/1st.sh" && use forge
    git() { [[ "$1" != remote ]] || return 1; command git "$@"; }
    forge:primary'
  [ "$status" -eq 1 ]
  said="run: cd ${PWD} && git remote (the remotes could not be read, so primary 'origin' in kit.yml cannot be checked)"
  [[ "$output" == *"${said}"* ]]
  [[ "$output" != *"is not a remote"* ]]
}

@test "the primary is the primary setting, else origin; it comes first among the remotes, and names the provider" {
  git remote add zeta https://gitlab.com/team/kit.git
  git remote add origin https://github.com/empowerite/kit.git
  git remote add upstream https://gitlab.com/team/kit.git
  run in-forge forge:primary
  [ "$output" = origin ]
  run in-forge forge:remotes
  [ "$output" = $'origin\nupstream\nzeta' ]
  run provider
  [ "$output" = github ]
  printf 'primary: upstream\n' > kit.yml
  run in-forge forge:primary
  [ "$output" = upstream ]
  run in-forge forge:remotes
  [ "$output" = $'upstream\norigin\nzeta' ]
  run provider
  [ "$output" = gitlab ]
  # a primary that names no remote is refused before any git command takes it as an argument
  printf 'primary: nowhere\n' > kit.yml
  run provider
  [ "$status" -eq 1 ]
  [[ "$output" == *"primary 'nowhere' in kit.yml is not a remote; the remotes are: origin upstream zeta"* ]]
  printf 'primary: --upload-pack=touch pwned\n' > kit.yml
  run in-forge forge:remotes
  [ "$status" -eq 1 ]
  [ -z "$(find "${BATS_TEST_TMPDIR}" -name pwned)" ]
  printf 'forges: {"-x": github}\n' > kit.yml
  run in-forge forge:of origin
  [ "$status" -eq 1 ]
  [[ "$output" == *"'-x' in forges in kit.yml is not a remote name"* ]]
}

@test "the core verbs need no forge and no gh" {
  mkdir -p "${BATS_TEST_TMPDIR}/shims" && printf '#!/usr/bin/env bash\nexit 127\n' > "${BATS_TEST_TMPDIR}/shims/gh"
  chmod +x "${BATS_TEST_TMPDIR}/shims/gh"
  run env PATH="${BATS_TEST_TMPDIR}/shims:${PATH}" "${kit_root}/bin/kit" install agents/claude
  [ "$status" -eq 0 ]
  run env PATH="${BATS_TEST_TMPDIR}/shims:${PATH}" "${kit_root}/bin/kit" check
  [ "$status" -eq 0 ]
  run env PATH="${BATS_TEST_TMPDIR}/shims:${PATH}" "${kit_root}/bin/kit" context justfile
  [ "$status" -eq 0 ]
  run env PATH="${BATS_TEST_TMPDIR}/shims:${PATH}" "${kit_root}/bin/kit" flow --no-fetch
  [ "$status" -eq 0 ]
  [[ "$output" == *"no origin: nothing fetched, no pull request shown"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
