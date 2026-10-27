#!/usr/bin/env bats
# generate.bats - a :generate: directive opens a region its command fills, to a fixed point, held by --check
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}"
  work=$(mktemp -d)
  cd "${work}"
  git init --quiet --initial-branch=main
  git config user.name t && git config user.email t@example.com && git config commit.gpgsign false
  # A generator in the tree: a script that prints its arguments, one per line, so a region's content is visible.
  printf '#!/usr/bin/env bash\nprintf "%%s\\n" "$@"\n' > lines.sh && chmod +x lines.sh
}

teardown() {
  rm -rf "${work}"
}

@test "until fills the region up to its end marker, which stays, and while replaces every matching line" {
  printf 'before\n<!-- :generate:./lines.sh one two:until:"^<!-- end -->$" -->\nstale\n<!-- end -->\nafter\n' > a.md
  printf '# :generate:./lines.sh x=1 y=2:while:"^[a-z]+=[0-9]+$"\nold=0\nolder=0\nkeep this\n' > b.toml
  run "${kit_checkout}/bin/kit" generate a.md b.toml
  [ "$status" -eq 0 ]
  [[ "$output" == *"a.md: regenerated"* ]]
  [ "$(paste -s -d '|' - < a.md)" = 'before|<!-- :generate:./lines.sh one two:until:"^<!-- end -->$" -->|one|two|<!-- end -->|after' ]
  [ "$(paste -s -d '|' - < b.toml)" = '# :generate:./lines.sh x=1 y=2:while:"^[a-z]+=[0-9]+$"|x=1|y=2|keep this' ]
  run "${kit_checkout}/bin/kit" generate a.md b.toml
  [ "$status" -eq 0 ]
  [[ "$output" == *"settled in 1 pass"* ]]
}

@test "the whole repository is discovered, tracked or not, and --check names a stale file without writing it" {
  printf '<!-- :generate:./lines.sh fresh:until:"end" -->\nstale\n<!-- end -->\n' > a.md
  mkdir sub && printf '# :generate:./lines.sh deep:until:"end"\n# end\n' > sub/b.yml
  printf 'no directive here\n' > c.md
  git add a.md && printf 'ignored/\n' > .gitignore && mkdir ignored && cp a.md ignored/d.md
  run "${kit_checkout}/bin/kit" generate --check
  [ "$status" -eq 1 ]
  [[ "$output" == *"a.md: stale; run: kit generate"* ]]
  [[ "$output" == *"sub/b.yml: stale"* ]]
  [[ "$output" != *"ignored/d.md"* ]]
  [[ "$output" == *"2 of 2 generated file(s) are stale"* ]]
  grep --quiet '^stale$' a.md
  cd sub
  run "${kit_checkout}/bin/kit" generate
  [ "$status" -eq 0 ]
  [[ "$output" == *"settled in 2 pass(es)"* ]]
  [ "$(paste -s -d '|' - < b.yml)" = '# :generate:./lines.sh deep:until:"end"|deep|# end' ]
  run "${kit_checkout}/bin/kit" generate --check b.yml ../c.md
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" generate --check ../c.md
  [ "$status" -eq 0 ]
  [[ "$output" == *"no :generate: directive to expand"* ]]
}

@test "a command is one command and its arguments, and a failing one, an unclosed region, or an empty result leaves the file as it was" {
  printf '<!-- :generate:./lines.sh a | cat:until:"end" -->\n<!-- end -->\n' > pipe.md
  run "${kit_checkout}/bin/kit" generate pipe.md
  [ "$status" -eq 1 ]
  [[ "$output" == *"pipe.md:1: a :generate: command is one command and its arguments, with no shell operator: ./lines.sh a | cat"* ]]
  printf '<!-- :generate:./missing.sh:until:"end" -->\nheld\n<!-- end -->\n' > fail.md
  run "${kit_checkout}/bin/kit" generate fail.md
  [ "$status" -eq 1 ]
  [[ "$output" == *"fail.md:1: the command failed: ./missing.sh"* ]]
  grep --quiet '^held$' fail.md
  printf 'top\n<!-- :generate:./lines.sh a:until:"never" -->\nheld\n' > open.md
  run "${kit_checkout}/bin/kit" generate open.md
  [ "$status" -eq 1 ]
  [[ "$output" == *'open.md:2: the region never closes; no later line matches "never"'* ]]
  grep --quiet '^held$' open.md
  printf '<<<<<<< ours\n<!-- :generate:./lines.sh a:until:"end" -->\n<!-- end -->\n>>>>>>> theirs\n' > conflict.md
  run "${kit_checkout}/bin/kit" generate conflict.md
  [ "$status" -eq 1 ]
  [[ "$output" == *"conflict.md carries an unresolved conflict"* ]]
}

@test "a directive inside a fenced code block is not live, and a region inside a symlinked file lands in its target" {
  printf '```markdown\n<!-- :generate:./lines.sh shown:until:"end" -->\n<!-- end -->\n```\nprose\n' > doc.md
  run "${kit_checkout}/bin/kit" generate doc.md
  [ "$status" -eq 0 ]
  [[ "$output" == *"no :generate: directive"* || "$output" == *"settled"* ]]
  ! grep --quiet '^shown$' doc.md
  mkdir real && printf '# :generate:./lines.sh linked:until:"end"\n# end\n' > real/x.sh && ln -s real/x.sh x.sh
  run "${kit_checkout}/bin/kit" generate x.sh
  [ "$status" -eq 0 ]
  [ -L x.sh ]
  grep --quiet '^linked$' real/x.sh
}

@test "a command runs with the running kit first on PATH, so kit region fills the kit's own regions" {
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 0 ]
  printf '## Ours\n' >> AGENTS.md
  run env PATH="$(dirname "$(command -v bash)"):/usr/bin:/bin" "${kit_checkout}/bin/kit" generate --check
  [ "$status" -eq 0 ]
  [[ "$output" == *"every region as its command makes it"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
