#!/usr/bin/env bats
# modules.bats - every function says where it is defined, and a release is split by concern into modules of its own
# ----------------------------------------------------------------------------------------------------------------------

load helper

setup() {
  fresh-repo
}

@test "every function in a module is named <module>:<name>, so that each says where it is defined" {
  local file module name stray=''
  while IFS= read -r file
  do
    module=${file#"${kit_root}/modules/bash/"}
    module=${module%.sh}
    [[ "${module}" != 1st ]] || continue
    while IFS= read -r name
    do
      # Two names are a convention outside the kit: super, which an override calls by that name, and _kit, which
      # the shell's completion registers.
      [[ "${name}" == super || "${name}" == _kit ]] && continue
      [[ "${name}" == "${module}:"* ]] || stray+="${module}: ${name}"$'\n'
    done < <(sed -n 's/^\([A-Za-z0-9_:\/-]*\)() {$/\1/p' "${file}")
  done < <(find "${kit_root}/modules/bash" -name '*.sh' | LC_ALL=C sort)
  [ -z "${stray}" ] || { printf '%s' "${stray}"; return 1; }
}

@test "no verb or module reads a command's lines through < <(…), which loses its status, unless it cannot fail" {
  local pure='jq|awk|sed|printf|cat|find|compgen|for|while|list:without|use:find|use:modules|help:[a-z-]+'
  pure+='|region:markers|flow:(label|stage-args|fetch-diff)|context:(governed-by|embedded|read-kinds)'
  pure+='|commit:placed|welcome:[a-z-]+|serve:(watched|changed|resources)|new:seam-calls'
  pure+='|tracker:body-refs'
  local kept="< <\\((${pure})[ )]|(^|[ ;])read -r [^<]*< <\\(.*\\) \\|\\| (return|exit|continue)|^[^:]+:[0-9]+: *#"
  # list:fill's own read, whose command's status travels in band, after the lines, and is read back from there
  kept+='|/modules/bash/1st\.sh:[0-9]+: +if ! readarray -t -d "\$\{list_fill_delimiter\}" list_fill_items < <\('
  run bash -c 'grep --recursive --line-number "< <(" "$1/bin" "$1/modules/bash" | grep --invert-match -E "$2"' \
    scan "${kit_root}" "${kept}"
  [ -z "$output" ] || { printf '%s\n' "$output"; return 1; }
}

@test "no module assigns a forge or tracker read unguarded" {
  local read='=\$\((forge:(pr-|issue-|branch-for-issue|repo|me)|tracker:(of|at) |git:default-branch|forge/gitlab:api )'
  # A line that ends in a backslash joins the next, so a guard on the continuation counts.
  run bash -c 'find "$1/modules/bash" -name "*.sh" -exec awk '"'"'
      FNR == 1 { held = "" }
      /\\$/ { held = held substr($0, 1, length($0) - 1); next }
      { print FILENAME ":" FNR ": " held $0; held = "" }
    '"'"' {} + | grep -E "$2" | grep --invert-match -E "\|\||(^|[^a-z])(el)?if "' scan "${kit_root}" "${read}"
  [ -z "$output" ] || { printf '%s\n' "$output"; return 1; }
}

@test "no pipeline ends in grep --quiet" {
  # grep --quiet exits at its first match, the command before it dies of SIGPIPE, and pipefail makes the match a
  # failure. A line that ends in a pipe or a backslash joins the next, so a grep on the continuation counts.
  run bash -c 'find "$1/bin" "$1/modules/bash" -type f -exec awk '"'"'
      FNR == 1 { held = "" }
      /^[[:space:]]*#/ { next }
      /\\$/ { held = held substr($0, 1, length($0) - 1); next }
      /\|[[:space:]]*$/ { held = held $0 " "; next }
      { print FILENAME ":" FNR ": " held $0; held = "" }
    '"'"' {} + | grep -E "(^|[^|])\| *grep [^|]*(--quiet|-[A-Za-z]*q[A-Za-z]*)( |$)"' scan "${kit_root}"
  [ -z "$output" ] || { printf '%s\n' "$output"; return 1; }
}

@test "every || true says why it swallows" {
  # A statement's || true or || : ends in a comment, or the line before the statement is one. A line that ends in a
  # backslash, a pipe, or && joins the next, so the comment above the statement's first line counts.
  run bash -c 'find "$1/bin" "$1/modules/bash" -type f -exec awk '"'"'
      FNR == 1 { held = ""; prev = "" }
      held == "" { before = prev }
      { prev = $0 }
      /^[[:space:]]*#/ { next }
      /(\\|\||&&)[[:space:]]*$/ { held = held $0 " "; next }
      {
        line = held $0
        held = ""
        if (line !~ /\|\| *(true|:)([ ;)}]|$)/) next
        if (line ~ /\|\| *(true|:)([ ;)}][^#]*)?[[:space:]]#/ || before ~ /^[[:space:]]*#/) next
        print FILENAME ":" FNR ": " line
      }
    '"'"' {} +' scan "${kit_root}"
  [ -z "$output" ] || { printf '%s\n' "$output"; return 1; }
}

@test "the release is read, rendered, placed, and recorded by one module each, and use release loads none of them but resolve" {
  local module function
  while IFS=' ' read -r module function
  do
    [ -f "${kit_root}/modules/bash/${module}.sh" ] || { echo "no module ${module}"; return 1; }
    run kit-bash "use ${module} && declare -F ${module}:${function}"
    [ "$status" -eq 0 ] || { echo "no ${module}:${function}"; return 1; }
  done <<'LIST'
release fetch
release bases
settings declare
settings validate
resolve active
resolve reach
render all
region write
place content
place retire
materialize all
materialize hydrate
manifest write-all
manifest recorded
LIST
  # release:finish reports with the packages' metadata, so use release brings resolve; nothing that writes comes with it
  run kit-bash 'use release && declare -F | sed "s/^declare -f //" | grep -E "^(settings|render|region|place|materialize|manifest):"'
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  run kit-bash 'use file && printf "abc" | file:digest && printf "abc" > f && file:digest f'
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" ]
  [ "${lines[1]}" = "${lines[0]}" ]
  run kit-bash 'list:among b a b c && ! list:among z a b c && list:without b a b c | paste -s -d " " -'
  [ "$status" -eq 0 ]
  [ "$output" = "a c" ]
}
