#!/usr/bin/env bats
# welcome.bats - kit welcome: its steps, its checks as JSON, their fixes, the text guide, and the hand-off to kittui
# ----------------------------------------------------------------------------------------------------------------------
# Each test runs in a fresh repository, with no toolchain file unless the test writes one: its tools are ambient. The
# tools a check looks for are fakes in a directory first on PATH: a failing stand-in hides a real tool. A fake brew
# is there too, so brew is the system toolchain on any machine; a test that needs none chooses ambient for it.

bats_require_minimum_version 1.5.0

load helper

setup() {
  hide-system-toolchains
  fresh-repo >/dev/null
  fakes="${BATS_TEST_TMPDIR}/fakes"
  mkdir -p "${fakes}"
  # brew is found, so it is the system toolchain; it is never run, except for --prefix where a test asks
  fake-script brew 'exit 1'
  export PATH="${fakes}:${PATH}"
}

# fake-script <tool> <body> writes a bash script named <tool> into the fakes directory, first on PATH.
fake-script() {
  printf '#!/usr/bin/env bash\n%s\n' "$2" > "${fakes}/$1"
  chmod +x "${fakes}/$1"
}

# checks prints the checks as JSON, from kit welcome --checks --json.
checks() {
  "${kit_root}/bin/kit" welcome --checks --json
}

# check <id> prints one check of kit welcome --checks --json.
check() {
  checks | jq --compact-output --arg id "$1" '.[] | select(.id == $id)'
}

# fake-gh writes a gh that runs, and is logged in to github.com once its login has run: auth login touches a file,
# and auth status succeeds while the file is there. Every call is recorded in gh.calls. With LOGIN_FAILS set, the
# login changes nothing.
fake-gh() {
  # shellcheck disable=SC2016  # a fake's body expands when it runs
  fake-script gh '
printf "%s\n" "$*" >> "${BATS_TEST_TMPDIR}/gh.calls"
case "$*" in
  ("--version") echo "gh version 2.103.2" ;;
  ("auth status --hostname github.com") [[ -f "${BATS_TEST_TMPDIR}/logged-in" ]] ;;
  ("auth login --hostname github.com") [[ -n "${LOGIN_FAILS:-}" ]] || touch "${BATS_TEST_TMPDIR}/logged-in" ;;
  (*) exit 1
esac'
}

# outline prints the outline of the text guide on stdin: each page's heading, and ✅ or ❌ for each check, a run
# of the same glyph once.
outline() {
  grep --extended-regexp '^(# |✅|❌)' | sed -E 's/^(✅|❌).*/\1/' | uniq | paste -s -d '|' -
}

# under prints "<heading>|<glyph> <word>" for each check in the text guide on stdin: the heading of the page above
# it, and the check's glyph and first word.
under() {
  awk '/^# / { heading = $0 } /^(✅|❌)/ { print heading "|" $1 " " $2 }'
}

# ambient-system chooses ambient for the system role, so no toolchain names a command for a missing tool.
ambient-system() {
  "${kit_root}/bin/kit" toolchains choose ambient --system >/dev/null 2>&1
}

@test "the checks are a JSON array of {id, step, ok, line, fix, interactive, more, runs}, in order, each id once" {
  run --separate-stderr checks
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output '[.[] | keys | sort == ["fix", "id", "interactive", "line", "more", "ok", "runs", "step"]] | all' \
    <<<"$output")" = true ]
  [ "$(jq --raw-output 'map(.id) | (unique | length) == length' <<<"$output")" = true ]
  # what kit runs on and the toolchains, then the repository, then where you are; no kit.yml here, so no check of
  # what kit installs
  [ "$(jq --raw-output '.[] | "\(.id) \(.step)"' <<<"$output")" = "$(printf '%s\n' 'tool-bash toolchains' \
    'tool-git toolchains' 'tool-jq toolchains' 'tool-yq toolchains' \
    'toolchain-project toolchains' 'toolchain-system toolchains' 'repo-git setup' 'kit-installed setup' \
    'where setup')" ]
  [ "$(jq --raw-output '.[] | select(.id == "kit-installed") | [.ok, .fix] | @tsv' <<<"$output")" = \
    $'false\tkit install' ]
  [ "$(jq --raw-output '.[] | select(.id == "where") | .line' <<<"$output")" = \
    "you are at the desk, your own checkout; work on an issue goes in a worktree" ]
}

@test "welcome's steps are introduction, toolchains, setup, applications, terms, customize" {
  run --separate-stderr "${kit_root}/bin/kit" welcome --text
  [ "$status" -eq 1 ]
  # each step's page in order, the checks of steps 2 and 3 under their pages, then what is left
  [ "$(outline <<<"$output")" = "$(printf '%s|' '# Welcome' '# Toolchains' ✅ '# Setup' \
    ✅ ❌ ✅ '# Applications' '# Terms' '# Customize' '# What is left' | sed 's/|$//')" ]
  # bash, git, jq, and yq, then one check for each role, then the rest
  [ "$(under <<<"$output")" = "$(printf '%s\n' '# Toolchains|✅ bash' '# Toolchains|✅ git' \
    '# Toolchains|✅ jq' '# Toolchains|✅ yq' '# Toolchains|✅ no' '# Toolchains|✅ brew' \
    '# Setup|✅ this' '# Setup|❌ kit.yml' '# Setup|✅ you')" ]
}

@test "welcome offers the shellenv line when brew is at a prefix but not on PATH, and shows no check otherwise" {
  # on PATH: no check
  run --separate-stderr check brew-path
  [ -z "$output" ]
  # off PATH, with none at a prefix: no check
  rm "${fakes}/brew"
  path-without brew
  export KIT_BREW_PREFIXES="${BATS_TEST_TMPDIR}/opt/bin/brew"
  run --separate-stderr check brew-path
  [ -z "$output" ]
  # off PATH, at a prefix: the check fails and names the line
  mkdir -p "${BATS_TEST_TMPDIR}/opt/bin"
  printf '#!/usr/bin/env bash\nexit 1\n' > "${BATS_TEST_TMPDIR}/opt/bin/brew"
  chmod +x "${BATS_TEST_TMPDIR}/opt/bin/brew"
  run --separate-stderr check brew-path
  [ "$(jq --raw-output '[.step, .ok, .line, .fix, .interactive] | @tsv' <<<"$output")" = \
    "$(printf 'toolchains\tfalse\tbrew is installed at %s/opt but not on your PATH\t' "${BATS_TEST_TMPDIR}"
      printf 'eval "$(%s/opt/bin/brew shellenv)"\tfalse' "${BATS_TEST_TMPDIR}")" ]
  [[ "$(jq --raw-output .more <<<"$output")" == *profile* ]]
  # it comes right after yq, before the toolchains' own checks
  run --separate-stderr checks
  [ "$(jq --raw-output 'map(.id) | .[0:6] | join(" ")' <<<"$output")" = \
    "tool-bash tool-git tool-jq tool-yq brew-path toolchain-project" ]
  # --fix does not run it: kit cannot change the shell it was started from
  run --separate-stderr "${kit_root}/bin/kit" welcome --fix brew-path
  [ "$status" -eq 2 ]
  [[ "$stderr" == *"must run in your own shell"* ]]
  # the check says so, and the guide names the line without offering to run it
  run --separate-stderr check brew-path
  [ "$(jq --raw-output .runs <<<"$output")" = false ]
  [ "$(check tool-git | jq --raw-output .runs)" = true ]
  run --separate-stderr "${kit_root}/bin/kit" welcome --text
  [[ "$output" == *'- brew-path: eval "$('*"kit does not run it"* ]]
}

@test "welcome checks bash, git, jq, and yq with the toolchains, each fixed by the system toolchain" {
  local real_jq
  real_jq=$(command -v jq)
  fake-script yq 'exit 1'
  # the system toolchain names its command for a missing one
  run --separate-stderr check tool-yq
  [ "$(jq --raw-output '[.step, .ok, .line, .fix, .more] | @tsv' <<<"$output")" = \
    $'toolchains\tfalse\tyq is not installed; kit reads its YAML through it\tbrew install yq\t' ]
  # still the system toolchain's, with project toolchains here: kit needs these to look for one
  printf 'brew "yq"\n' > Brewfile
  run --separate-stderr check tool-yq
  [ "$(jq --raw-output .fix <<<"$output")" = "brew install yq" ]
  rm Brewfile "${fakes}/yq"
  # no system toolchain: no fix, and the tool's site to read
  ambient-system
  fake-script jq "[[ \"\$*\" != --version ]] || { echo jq-1.6; exit 0; }; exec '${real_jq}' \"\$@\""
  run --separate-stderr check tool-jq
  [ "$(jq --raw-output '[.step, .ok, .line, .fix, .more] | @tsv' <<<"$output")" = \
    $'toolchains\tfalse\tjq 1.6 is too old; kit needs jq 1.7 or later\t\thttps://jqlang.org/download/' ]
  run --separate-stderr check toolchain-system
  [ "$(jq --raw-output '[.ok, .line] | @tsv' <<<"$output")" = \
    $'true\tthe system toolchain is ambient, as chosen: kit uses what is on PATH' ]
}

@test "with apt the system toolchain, welcome names its package, or the site for a tool Debian has none for" {
  local sudo='sudo '
  (( EUID != 0 )) || sudo=''
  rm "${fakes}/brew"
  path-without brew
  fake-script apt-get 'exit 1'
  run --separate-stderr check toolchain-system
  [ "$(jq --raw-output '[.ok, .line] | @tsv' <<<"$output")" = \
    $'true\tapt is the system toolchain; it names the fix for a missing tool' ]
  # Debian's yq is not mikefarah's, so there is no command, and the site to read
  fake-script yq 'exit 1'
  run --separate-stderr check tool-yq
  [ "$(jq --raw-output '[.fix, .more] | @tsv' <<<"$output")" = $'\thttps://github.com/mikefarah/yq#install' ]
  # with no project toolchain, a missing linter is apt's to install
  rm "${fakes}/yq"
  "${kit_root}/bin/kit" install >/dev/null 2>&1
  fake-script shellcheck 'exit 1'
  run --separate-stderr check linter-shellcheck
  [ "$(jq --raw-output '[.fix, .runs] | @tsv' <<<"$output")" = "${sudo}apt-get install shellcheck"$'\ttrue' ]
}

@test "welcome names kit check's one finding with its own remedy, and offers a fix only where the finding names one" {
  "${kit_root}/bin/kit" install hooks/lefthook >/dev/null 2>&1
  # the one finding: lefthook.yml lacks the include, which no command fixes
  run --separate-stderr check kit-check
  [ "$(jq --raw-output '[.ok, .fix] | @tsv' <<<"$output")" = $'false\t' ]
  [[ "$(jq --raw-output .line <<<"$output")" == "lefthook.yml: put extends: [.kit/lefthook.yml] in lefthook.yml"* ]]
  # a finding that names a command offers that command
  printf '# lefthook.yml - the git hooks\nextends: [.kit/lefthook.yml]\n' > lefthook.yml
  rm .gitignore
  run --separate-stderr check kit-check
  [ "$(jq --raw-output .ok <<<"$output")" = false ]
  [ "$(jq --raw-output .fix <<<"$output")" = "kit update" ]
  [[ "$(jq --raw-output .line <<<"$output")" == ".gitignore "* ]]
}

@test "welcome checks each dependency provider whose manifest is here: its lock, its tool, and what is installed" {
  "${kit_root}/bin/kit" install >/dev/null 2>&1
  printf '{"devDependencies": {"left-pad": "^1.3.0"}}\n' > package.json
  fake-script npm 'exit 1'
  # a manifest without its lock pins nothing, and kit can run no fix
  run --separate-stderr check dependencies-npm
  [ "$(jq --raw-output '[.step, .ok, .fix] | @tsv' <<<"$output")" = $'setup\tfalse\t' ]
  [[ "$(jq --raw-output .line <<<"$output")" == "package.json is here without package-lock.json"* ]]
  printf '{"packages": {"node_modules/left-pad": {"version": "1.3.0"}}}\n' > package-lock.json
  # locked, but not installed: kit tools install fixes it
  run --separate-stderr check dependencies-npm
  [ "$(jq --raw-output '[.ok, .line, .fix] | @tsv' <<<"$output")" = \
    $'false\tnpm has not installed every package package-lock.json pins\tkit tools install' ]
  mkdir -p node_modules
  cp package-lock.json node_modules/.package-lock.json
  run --separate-stderr check dependencies-npm
  [ "$(jq --raw-output '[.ok, .line] | @tsv' <<<"$output")" = $'true\tnpm installed every package package-lock.json pins' ]
  # with no npm on PATH, kit names that
  rm "${fakes}/npm"
  path-without npm
  run --separate-stderr check dependencies-npm
  [ "$(jq --raw-output '[.ok, .line] | @tsv' <<<"$output")" = $'false\tpackage-lock.json is here, but npm is not on PATH' ]
  # the check comes before the linters, and there is none for uv, whose manifest is not here
  run --separate-stderr checks
  [ "$(jq --raw-output 'map(.id) | (index("dependencies-npm") < (map(startswith("linter-")) | index(true)))' \
    <<<"$output")" = true ]
  [ "$(jq --raw-output 'map(.id) | index("dependencies-uv")' <<<"$output")" = null ]
}

@test "a provider is found whichever the list names first" {
  "${kit_root}/bin/kit" install >/dev/null 2>&1
  printf '{"devDependencies": {"left-pad": "^1.3.0"}}\n' > package.json
  printf '{"packages": {"node_modules/left-pad": {"version": "1.3.0"}}}\n' > package-lock.json
  printf '[project]\nname = "here"\nversion = "0.1.0"\n' > pyproject.toml
  printf 'version = 1\n' > uv.lock
  fake-script npm 'exit 0'
  fake-script uv 'exit 0'
  local run
  for run in {1..20}
  do
    run --separate-stderr check dependencies-npm
    [ "$status" -eq 0 ]
    [ -n "$output" ]
    [[ "$(jq --raw-output .line <<<"$output")" != *"is not on PATH"* ]] || { echo "run ${run}: $output"; return 1; }
  done
}

@test "welcome holds a linter npm installed, off PATH, since kit lint looks in its directory" {
  "${kit_root}/bin/kit" install >/dev/null 2>&1
  printf '{"devDependencies": {"shellcheck": "^0.1.0"}}\n' > package.json
  printf '{"packages": {}}\n' > package-lock.json
  fake-script npm 'exit 1'
  # the shellcheck on PATH does not run, as a missing one
  fake-script shellcheck 'exit 1'
  mkdir -p node_modules/.bin
  printf '#!/bin/sh\n' > node_modules/.bin/shellcheck && chmod +x node_modules/.bin/shellcheck
  run --separate-stderr check linter-shellcheck
  [ "$(jq --raw-output '[.ok, .line] | @tsv' <<<"$output")" = \
    $'true\tshellcheck is installed in node_modules/.bin; kit lint runs it for .kit/rules/bash.md' ]
}

@test "welcome fixes a missing tool with kit tools install, or the system toolchain's command when ambient" {
  "${kit_root}/bin/kit" install >/dev/null 2>&1
  fake-script shellcheck 'exit 1'
  fake-script actionlint 'exit 0'
  # ambient for the project: the system toolchain's command
  run --separate-stderr check linter-shellcheck
  [ "$(jq --raw-output '[.step, .ok, .line, .fix, .more] | @tsv' <<<"$output")" = \
    $'setup\tfalse\tshellcheck is not installed; kit lint runs it for .kit/rules/bash.md\tbrew install shellcheck\t' ]
  run --separate-stderr check toolchain-project
  [ "$(jq --raw-output '[.step, .ok, .fix] | @tsv' <<<"$output")" = $'toolchains\ttrue\t' ]
  [ "$(jq --raw-output .line <<<"$output")" = \
    "no toolchain is usable for the project, so its tools are ambient: kit uses what is on PATH" ]
  # ambient for the system too: no fix, and the tool's site
  ambient-system
  run --separate-stderr check linter-shellcheck
  [ "$(jq --raw-output '[.fix, .more] | @tsv' <<<"$output")" = \
    $'\thttps://github.com/koalaman/shellcheck#installing' ]
  # one project toolchain: it installs what it pins; brew reads a Brewfile on macOS alone
  export OSTYPE=darwin24
  printf 'brew "shellcheck"\n' > Brewfile
  run --separate-stderr check linter-shellcheck
  [ "$(jq --raw-output '[.fix, .more] | @tsv' <<<"$output")" = $'kit tools install\t' ]
  run --separate-stderr check toolchain-project
  [ "$(jq --raw-output '[.ok, .line] | @tsv' <<<"$output")" = \
    $'true\tbrew is the project toolchain; kit tools lists what it pins' ]
  # several, none chosen: the toolchains step asks first, and setup has no fix until then; brew fills in beside them
  touch mise.toml toolchain.Dockerfile
  fake-script mise 'exit 0'
  fake-script docker 'exit 0'
  run --separate-stderr check toolchain-project
  [ "$(jq --raw-output '[.ok, .fix, .interactive, .more] | @tsv' <<<"$output")" = \
    $'false\tkit toolchains choose mise\tfalse\tor: kit toolchains choose docker' ]
  [ "$(jq --raw-output .line <<<"$output")" = "several toolchains are usable for the project: mise, docker; choose one" ]
  # what is left keeps the other toolchain's command beside the fix
  run --separate-stderr "${kit_root}/bin/kit" welcome --text
  [[ "$output" == *$'\n- toolchain-project: kit toolchains choose mise; or: kit toolchains choose docker\n'* ]]
  run --separate-stderr check linter-shellcheck
  [ "$(jq --raw-output '[.fix, .more] | @tsv' <<<"$output")" = \
    $'\thttps://github.com/empowerite/kit/blob/main/docs/toolchains.md' ]
  # the toolchains check's fix records the first; the others' commands stand beside it
  run --separate-stderr "${kit_root}/bin/kit" welcome --fix toolchain-project
  [ "$status" -eq 0 ]
  [ "$output" = "✅ mise is the project toolchain; kit tools lists what it pins" ]
  # with mise and docker gone, brew is the only one again, and the recorded mise gives way to it
  rm mise.toml toolchain.Dockerfile
  run --separate-stderr check linter-shellcheck
  [ "$(jq --raw-output .fix <<<"$output")" = "kit tools install" ]
  # installed by the toolchain, in a directory not on PATH: no install, the directory is what is missing
  # shellcheck disable=SC2016  # a fake's body expands when it runs
  fake-script brew '[[ "$*" != --prefix ]] || echo "${BATS_TEST_TMPDIR}/prefix"'
  mkdir -p "${BATS_TEST_TMPDIR}/prefix/bin"
  printf '#!/usr/bin/env bash\nexit 0\n' > "${BATS_TEST_TMPDIR}/prefix/bin/shellcheck"
  chmod +x "${BATS_TEST_TMPDIR}/prefix/bin/shellcheck"
  run --separate-stderr check linter-shellcheck
  [ "$(jq --raw-output '[.ok, .line, .fix] | @tsv' <<<"$output")" = \
    $'false\tshellcheck is installed, but not on PATH; kit tools path names its directory\t' ]
}

@test "a forge host among the remotes gets two checks: its tool installed, and logged in, the login's fix interactive" {
  git remote add origin https://github.com/acme/widget.git
  git remote add mirror git@github.com:acme/widget-mirror.git
  # gh missing: the login cannot be checked, and has no fix
  fake-script gh 'exit 1'
  run --separate-stderr checks
  [ "$(jq --raw-output '[.[] | .id | select(startswith("forge-"))] | join(" ")' <<<"$output")" = \
    "forge-cli-github.com forge-login-github.com" ]
  [ "$(jq --raw-output '.[] | select(.id == "forge-cli-github.com") | [.ok, .line] | @tsv' <<<"$output")" = \
    $'false\tgh is not installed; kit reaches github.com through it' ]
  [ "$(jq --raw-output '.[] | select(.id == "forge-login-github.com") | [.ok, .fix, .more] | @tsv' \
    <<<"$output")" = $'false\t\thttps://cli.github.com' ]
  # gh installed, not logged in
  fake-gh
  run --separate-stderr check forge-login-github.com
  [ "$(jq --raw-output '[.ok, .line, .fix, .interactive] | @tsv' <<<"$output")" = \
    $'false\tgh is not logged in to github.com\tgh auth login --hostname github.com\ttrue' ]
  # logged in
  touch "${BATS_TEST_TMPDIR}/logged-in"
  run --separate-stderr checks
  [ "$(jq --raw-output '[.[] | select(.id | startswith("forge-")) | .ok] | all' <<<"$output")" = true ]
  [ "$(jq --raw-output '.[] | select(.id == "forge-login-github.com") | .line' <<<"$output")" = \
    "gh is logged in to github.com" ]
}

@test "--fix runs the check's fix, checks again, and exits 0 when it now holds and 1 when it does not" {
  git remote add origin https://github.com/acme/widget.git
  fake-gh
  run --separate-stderr env LOGIN_FAILS=1 "${kit_root}/bin/kit" welcome --fix forge-login-github.com
  [ "$status" -eq 1 ]
  [ "$output" = $'❌ gh is not logged in to github.com\n   gh auth login --hostname github.com' ]
  grep --quiet --line-regexp 'auth login --hostname github.com' "${BATS_TEST_TMPDIR}/gh.calls"
  run --separate-stderr "${kit_root}/bin/kit" welcome --fix forge-login-github.com
  [ "$status" -eq 0 ]
  [ "$output" = "✅ gh is logged in to github.com" ]
  [[ "$stderr" == *"→ gh auth login --hostname github.com"* ]]
}

@test "--fix refuses with 2 an unknown check, one that holds, and one kit knows no fix for, and runs nothing" {
  fake-gh
  touch "${BATS_TEST_TMPDIR}/logged-in"
  git remote add origin https://github.com/acme/widget.git
  run "${kit_root}/bin/kit" welcome --fix tool-nothing
  [ "$status" -eq 2 ]
  [[ "$output" == *"no check is named 'tool-nothing'; the checks are: tool-bash "* ]]
  run "${kit_root}/bin/kit" welcome --fix forge-login-github.com
  [ "$status" -eq 2 ]
  [[ "$output" == *"forge-login-github.com holds already, so there is nothing to fix"* ]]
  run grep --quiet 'auth login' "${BATS_TEST_TMPDIR}/gh.calls"
  [ "$status" -eq 1 ]
  # gh gone: its login has no fix, only a page to read
  fake-script gh 'exit 1'
  run "${kit_root}/bin/kit" welcome --fix forge-login-github.com
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit knows no fix for forge-login-github.com; read: https://cli.github.com"* ]]
  run "${kit_root}/bin/kit" welcome --fix
  [ "$status" -eq 2 ]
  [[ "$output" == *"--fix needs a value; run: kit help welcome"* ]]
}

@test "the text guide prints each page with its step's checks, then what is left, and exits 1 on a failure" {
  fake-script yq 'exit 1'
  run --separate-stderr "${kit_root}/bin/kit" welcome --text
  [ "$status" -eq 1 ]
  [ "${lines[0]}" = "# Welcome" ]
  [[ "$output" == *$'\n❌ yq is not installed; kit reads its YAML through it\n   brew install yq\n'* ]]
  [[ "$output" == *$'# What is left\n\n- tool-yq: brew install yq\n'* ]]
  [[ "$output" != *"kit:generated"* ]]
  # nothing ran: the guide prints the fixes and runs none
  run --separate-stderr "${kit_root}/bin/kit" welcome
  [ "$status" -eq 1 ]
}

@test "with kit installed and every check holding, the text guide says nothing is left and exits 0" {
  "${kit_root}/bin/kit" install >/dev/null 2>&1
  # the kit package's rules run these linters, so a check stands for each; fakes that run stand in
  fake-script shellcheck 'exit 0'
  fake-script actionlint 'exit 0'
  run --separate-stderr checks
  [ "$(jq --raw-output 'map(.id) | join(" ")' <<<"$output")" = \
    "$(printf '%s ' tool-bash tool-git tool-jq tool-yq toolchain-project toolchain-system linter-shellcheck \
      linter-actionlint repo-git kit-installed kit-hydrated kit-check where | sed 's/ $//')" ]
  run --separate-stderr "${kit_root}/bin/kit" welcome --text
  [ "$status" -eq 0 ]
  # what is left comes last
  [[ "$output" == *$'\n# What is left\n\nNothing is left: every check holds.' ]]
  # a placed file gone: hydrate is the fix
  rm .kit/1st.md
  run --separate-stderr check kit-hydrated
  [ "$(jq --raw-output '[.ok, .fix] | @tsv' <<<"$output")" = $'false\tkit hydrate' ]
  run --separate-stderr "${kit_root}/bin/kit" welcome --fix kit-hydrated
  [ "$status" -eq 0 ]
  [ "$output" = "✅ .kit/ is in place: every file kit placed there is present" ]
}

@test "a house package that places a page of the same name replaces the kit's, and a new name adds a page" {
  "${kit_root}/bin/kit" install >/dev/null 2>&1
  mkdir -p .kit/packages/house/files/repo/.kit/welcome
  printf 'description: "our own"\ndepends:\n  - kit\n' > .kit/packages/house/package.yml
  printf '# Our terms\n\nWe say "desk".\n' > .kit/packages/house/files/repo/.kit/welcome/5-terms.md
  printf '# Our team\n\nAsk in #dev.\n' > .kit/packages/house/files/repo/.kit/welcome/7-our-team.md
  "${kit_root}/bin/kit" packages add house >/dev/null 2>&1
  run --separate-stderr "${kit_root}/bin/kit" welcome --text
  [ "$(grep --extended-regexp '^# ' <<<"$output" | paste -s -d '|' -)" = "$(printf '%s|' '# Welcome' \
    '# Toolchains' '# Setup' '# Applications' '# Our terms' '# Customize' \
    '# Our team' '# What is left' | sed 's/|$//')" ]
  [ "$(readlink .kit/welcome/5-terms.md)" = ../packages/house/files/repo/.kit/welcome/5-terms.md ]
}

@test "the pages go by the number their names start with, so a tenth page comes after the ninth" {
  "${kit_root}/bin/kit" install >/dev/null 2>&1
  mkdir -p .kit/packages/house/files/repo/.kit/welcome
  printf 'description: "our own"\ndepends:\n  - kit\n' > .kit/packages/house/package.yml
  printf '# Ten\n' > .kit/packages/house/files/repo/.kit/welcome/10-ten.md
  printf '# Nine\n' > .kit/packages/house/files/repo/.kit/welcome/9-nine.md
  printf '# Notes\n' > .kit/packages/house/files/repo/.kit/welcome/notes.md
  "${kit_root}/bin/kit" packages add house >/dev/null 2>&1
  run --separate-stderr "${kit_root}/bin/kit" welcome --text
  [[ "$(grep --extended-regexp '^# ' <<<"$output" | paste -s -d '|' -)" == *'# Customize|# Nine|# Ten|# Notes|# What is left' ]]
}

@test "on a terminal, kit welcome hands off to kittui; --text, --checks, or no terminal stays in text" {
  mkdir -p "${BATS_TEST_TMPDIR}/fake"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-kittui" "${BATS_TEST_TMPDIR}/fake/kittui"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-go" "${BATS_TEST_TMPDIR}/fake/go"
  export FAKES="${BATS_TEST_TMPDIR}/fake" PATH="${BATS_TEST_TMPDIR}/fake:${PATH}"
  run pty "'${kit_root}/bin/kit' welcome"
  [[ "$output" == *"kittui native-native welcome"* ]]
  [[ "$output" != *"# Welcome"* ]]
  run pty "'${kit_root}/bin/kit' welcome --text"
  [[ "$output" == *"# Welcome"* && "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' welcome --checks"
  [[ "$output" == *"✅ bash "* && "$output" != *"kittui"* ]]
  run --separate-stderr "${kit_root}/bin/kit" welcome </dev/null
  [ "${lines[0]}" = "# Welcome" ]
  [[ "$output" != *"kittui"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
