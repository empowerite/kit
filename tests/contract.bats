#!/usr/bin/env bats
# contract.bats - the real tools answer in the shapes the fakes stand in for, offline, where the desk has them
# ----------------------------------------------------------------------------------------------------------------------
# tools.bats and toolchains.bats drive every provider through tests/fakes/fake-tool, with fixtures shaped like the
# tool's answers. Only the real tool can prove those shapes hold. mise is on every desk and runner, so these tests run
# it in a scratch project and assert the fields the mise provider reads. Each skips, naming why, where mise is absent
# or has no tool installed. mise runs offline throughout, so nothing reaches the network.

bats_require_minimum_version 1.5.0

load helper

setup() {
  command -v mise >/dev/null || skip "mise is not on PATH, so its answers cannot be checked"
  fresh-repo >/dev/null
  # a tool the desk's mise has installed, and its version: the pin the scratch project makes
  read -r tool version < <(MISE_OFFLINE=1 mise ls --installed --json 2>/dev/null \
    | jq --raw-output 'to_entries | map(select(.value | length > 0)) | first // empty | "\(.key) \(.value[0].version)"')
  [[ -n "${tool:-}" ]] || skip "mise has no tool installed here, so there is nothing to pin"
  printf '[tools]\n"%s" = "%s"\n' "${tool}" "${version}" > mise.toml
  # trusted for these calls alone, so the desk's list of trusted configs is left as it is
  export MISE_TRUSTED_CONFIG_PATHS=${repo} MISE_OFFLINE=1
}

@test "mise ls --json --current answers an object keyed by tool, each a list with version, requested_version, installed" {
  run mise ls --json --current
  [ "$status" -eq 0 ]
  # toolchain::mise_pins reads these fields, as tools.bats's ls.json fixtures give them
  jq --exit-status --arg tool "${tool}" --arg version "${version}" '
    type == "object" and (.[$tool] | type == "array" and length > 0)
    and (.[$tool][0] | (.version == $version) and (.requested_version | type == "string")
      and (.installed == true))' <<<"$output"
}

@test "mise outdated --json answers an object keyed by tool, each with its current and its latest version" {
  # a pin that is not installed is outdated even offline, so the answer has a row to read
  printf '[tools]\n"%s" = "0.0.1"\n' "${tool}" > mise.toml
  run mise outdated --json
  [ "$status" -eq 0 ]
  # toolchain/mise:outdated reads .current, or none when null, and .latest, as the outdated.json fixtures give them
  jq --exit-status --arg tool "${tool}" '
    type == "object" and (.[$tool] | has("current") and (.latest | type == "string"))' <<<"$output"
}

@test "mise bin-paths answers one existing directory per line" {
  run mise bin-paths
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -gt 0 ]
  for line in "${lines[@]}"
  do
    [ -d "${line}" ]
  done
}

@test "kit tools, through the real mise, lists the pin with the version mise installed" {
  run env KIT_TOOLCHAIN=mise "${kit_root}/bin/kit" tools --json
  [ "$status" -eq 0 ]
  jq --exit-status --slurp --arg tool "${tool}" --arg version "${version}" '
    map(select(.name == $tool)) | length == 1 and (.[0] | .pinned == $version and .installed == $version
      and .from == "mise")' <<<"$output"
}

# ----------------------------------------------------------------------------------------------------------------------
