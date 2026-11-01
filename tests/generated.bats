#!/usr/bin/env bats
# generated.bats - every file a package delivers under .kit/ opens with a kit:generated line and a see: line
# ----------------------------------------------------------------------------------------------------------------------
#
# The two lines are in the file's own comment syntax, after a frontmatter's opening --- where there is one. A JSON
# file, which has no comments, carries both in a "$comment" key. A template is the named exception: its text is the
# shape of the files kit new makes from it, and the lines would pass into each of them.

load helper

@test "every file a package delivers under .kit/, but a template, says it is generated, by what, and how to stop or replace it" {
  local root file package first second missing=0
  root=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  while IFS= read -r file
  do
    package=${file#"${root}"/kit/packages/}
    package=${package#"${root}"/.kit/packages/}
    package=${package%%/files/repo/*}
    if [[ "${file}" == *.json ]]
    then
      first=$(jq --raw-output '."$comment" // ""' "${file}")
      [[ "${first}" == "kit:generated -- "*", from the ${package} package; see: "* ]] && continue
    else
      # the first two lines that are not a frontmatter's opening ---
      first=$(awk 'NR == 1 && $0 == "---" { next } { print; exit }' "${file}")
      second=$(awk 'BEGIN { n = 0 } NR == 1 && $0 == "---" { next } { if (++n == 2) { print; exit } }' "${file}")
      if [[ "${first}" =~ ^(#|//|\<!--)\ kit:generated\ --\ .+,\ from\ the\ ${package}\ package( -->)?$ \
        && "${second}" =~ ^(#|//|\<!--)\ see:\ .+ ]]
      then
        continue
      fi
    fi
    echo "no kit:generated and see: lines: ${file#"${root}"/}"
    missing=$(( missing + 1 ))
  done < <(find "${root}"/kit/packages/*/files/repo/.kit "${root}"/kit/packages/*/*/files/repo/.kit \
    "${root}"/.kit/packages/*/files/repo/.kit -type f ! -path '*/templates/*' 2>/dev/null | sort)
  [ "${missing}" -eq 0 ]
}

# ----------------------------------------------------------------------------------------------------------------------
