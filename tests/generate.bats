#!/usr/bin/env bats
# generate.bats - a kit:generate directive opens a named region its command fills, nested and balanced, held by --check
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
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

# joined <file> prints the file's lines joined by |, so a whole file is one assertion.
joined() { paste -s -d '|' - < "$1"; }

@test "a region is named, fills up to its end marker, and is read whatever comment syntax frames it" {
  printf 'before\n<!-- kit:generate a `./lines.sh one two` -->\nstale\n<!-- kit:end a -->\nafter\n' > a.md
  printf '# kit:generate b `./lines.sh x=1`\nold=0\n# kit:end b\nkeep this\n' > b.toml
  printf '; kit:generate c `./lines.sh z`\n; kit:end c\n' > c.ini
  printf "' kit:generate d \`./lines.sh v\`\n' kit:end d\n" > d.vb
  run "${kit_checkout}/bin/kit" generate a.md b.toml c.ini d.vb
  [ "$status" -eq 0 ]
  [[ "$output" == *"a.md: regenerated"* ]]
  [ "$(joined a.md)" = 'before|<!-- kit:generate a `./lines.sh one two` -->|one|two|<!-- kit:end a -->|after' ]
  [ "$(joined b.toml)" = '# kit:generate b `./lines.sh x=1`|x=1|# kit:end b|keep this' ]
  [ "$(joined c.ini)" = '; kit:generate c `./lines.sh z`|z|; kit:end c' ]
  [ "$(joined d.vb)" = "' kit:generate d \`./lines.sh v\`|v|' kit:end d" ]
  run "${kit_checkout}/bin/kit" generate a.md b.toml c.ini d.vb
  [ "$status" -eq 0 ]
  [[ "$output" == *"settled in 1 pass"* ]]
}

@test "a command runs in bash: a pipe works, a failure holds the file, and its environment names the file and region" {
  printf '# kit:generate sorted `./lines.sh c b a | sort`\n# kit:end sorted\n' > sorted.txt
  printf '# kit:generate env `printf "%%s %%s\\n" "$KIT_GENERATE_FILE" "$KIT_GENERATE_REGION"`\n# kit:end env\n' \
    > env.txt
  run "${kit_checkout}/bin/kit" generate sorted.txt env.txt
  [ "$status" -eq 0 ]
  [ "$(joined sorted.txt)" = '# kit:generate sorted `./lines.sh c b a | sort`|a|b|c|# kit:end sorted' ]
  [ "$(sed -n '2p' env.txt)" = 'env.txt env' ]
  printf '# kit:generate f `./lines.sh a | grep --quiet zzz`\nheld\n# kit:end f\n' > fail.txt
  run "${kit_checkout}/bin/kit" generate fail.txt
  [ "$status" -eq 1 ]
  [[ "$output" == *"fail.txt:1: the command failed: ./lines.sh a | grep --quiet zzz"* ]]
  grep --quiet '^held$' fail.txt
  printf '# kit:generate m `./missing.sh`\nheld\n# kit:end m\n' > missing.txt
  run "${kit_checkout}/bin/kit" generate missing.txt
  [ "$status" -eq 1 ]
  [[ "$output" == *"missing.txt:1: the command failed: ./missing.sh"* ]]
  grep --quiet '^held$' missing.txt
}

@test "a region nests when its command's output opens one, and an output must be balanced on its own" {
  cat > nest.sh <<'SH'
#!/usr/bin/env bash
printf '%s\n' head '# kit:generate inner `./lines.sh deep`' '# kit:end inner' tail
SH
  chmod +x nest.sh
  printf '# kit:generate outer `./nest.sh`\n# kit:end outer\n' > n.txt
  run "${kit_checkout}/bin/kit" generate n.txt
  [ "$status" -eq 0 ]
  nested='# kit:generate outer `./nest.sh`|head|# kit:generate inner `./lines.sh deep`|deep|# kit:end inner|tail'
  [ "$(joined n.txt)" = "${nested}|# kit:end outer" ]
  run "${kit_checkout}/bin/kit" generate --check
  [ "$status" -eq 0 ]
  [[ "$output" == *"every region as its command makes it"* ]]
  printf '# kit:generate outer `./lines.sh "# kit:end outer"`\nheld\n# kit:end outer\n' > close.txt
  run "${kit_checkout}/bin/kit" generate close.txt
  [ "$status" -eq 1 ]
  [[ "$output" == *"close.txt:1 (the output of ./lines.sh \"# kit:end outer\"):1: closes 'outer', which is not open"* ]]
  grep --quiet '^held$' close.txt
  printf '# kit:generate outer `./lines.sh "# kit:generate loose \\`./lines.sh x\\`"`\nheld\n# kit:end outer\n' \
    > open.txt
  run "${kit_checkout}/bin/kit" generate open.txt
  [ "$status" -eq 1 ]
  [[ "$output" == *"open.txt:1 (the output of"*"):1: the region 'loose' never closes"* ]]
  grep --quiet '^held$' open.txt
}

@test "an unbalanced, malformed, or repeated marker holds the file and names the line, as a conflict marker does" {
  printf 'top\n# kit:end z\n' > stray.txt
  run "${kit_checkout}/bin/kit" generate stray.txt
  [ "$status" -eq 1 ]
  [[ "$output" == *"stray.txt:2: closes 'z', which is not open"* ]]
  printf '# kit:generate b `./lines.sh x`\n# kit:end a\n# kit:end b\n' > cross.txt
  run "${kit_checkout}/bin/kit" generate cross.txt
  [ "$status" -eq 1 ]
  [[ "$output" == *"cross.txt:2: closes 'a' while 'b' is open"* ]]
  printf 'top\n# kit:generate a `./lines.sh x`\nheld\n' > never.txt
  run "${kit_checkout}/bin/kit" generate never.txt
  [ "$status" -eq 1 ]
  [[ "$output" == *"never.txt:2: the region 'a' never closes"* ]]
  grep --quiet '^held$' never.txt
  printf '# kit:generate\n# kit:end\n' > noname.txt
  run "${kit_checkout}/bin/kit" generate noname.txt
  [ "$status" -eq 1 ]
  [[ "$output" == *"noname.txt:1: not a directive: # kit:generate"* ]]
  printf '# kit:generate a `./lines.sh x`\n# kit:end a\n# kit:generate a `./lines.sh y`\n# kit:end a\n' > twice.txt
  run "${kit_checkout}/bin/kit" generate twice.txt
  [ "$status" -eq 1 ]
  [[ "$output" == *"twice.txt: the region 'a' is opened twice, the second time at line 4 of the expanded file"* ]]
  printf '<<<<<<< ours\n# kit:generate a `./lines.sh x`\n# kit:end a\n>>>>>>> theirs\n' > conflict.txt
  run "${kit_checkout}/bin/kit" generate conflict.txt
  [ "$status" -eq 1 ]
  [[ "$output" == *"conflict.txt carries an unresolved conflict"* ]]
  printf 'x\n' > plain.txt
  run "${kit_checkout}/bin/kit" generate plain.txt
  [ "$status" -eq 0 ]
  [[ "$output" == *"no kit:generate directive to expand"* ]]
}

@test "a line is a directive by what frames it: prose is not, a bare marker line is an error, and a markdown fence is not live" {
  printf 'A kit:generate x line opens a region, and kit:end x closes it.\n' > prose.md
  run "${kit_checkout}/bin/kit" generate --check prose.md
  [ "$status" -eq 0 ]
  [[ "$output" == *"no kit:generate directive"* ]]
  printf -- '- kit:generate x is how a region opens\n' > bullet.md
  run "${kit_checkout}/bin/kit" generate bullet.md
  [ "$status" -eq 1 ]
  [[ "$output" == *"bullet.md:1: not a directive: - kit:generate x is how a region opens"* ]]
  printf '```markdown\n<!-- kit:generate shown `./lines.sh shown` -->\n<!-- kit:end shown -->\n```\nprose\n' > doc.md
  run "${kit_checkout}/bin/kit" generate doc.md
  [ "$status" -eq 0 ]
  run grep --quiet '^shown$' doc.md
  [ "$status" -eq 1 ]
  printf 'cat <<EOF\n```\nEOF\n# kit:generate s `./lines.sh live`\n# kit:end s\n' > fenced.sh
  run "${kit_checkout}/bin/kit" generate fenced.sh
  [ "$status" -eq 0 ]
  grep --quiet '^live$' fenced.sh
}

@test "the whole repository is discovered, tracked or not, --check names a stale file without writing it, and a symlink lands in its target" {
  printf '<!-- kit:generate a `./lines.sh fresh` -->\nstale\n<!-- kit:end a -->\n' > a.md
  mkdir sub && printf '# kit:generate b `./lines.sh deep`\n# kit:end b\n' > sub/b.yml
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
  [ "$(joined b.yml)" = '# kit:generate b `./lines.sh deep`|deep|# kit:end b' ]
  run "${kit_checkout}/bin/kit" generate --check b.yml ../c.md
  [ "$status" -eq 0 ]
  cd ..
  mkdir real && printf '# kit:generate l `./lines.sh linked`\n# kit:end l\n' > real/x.sh && ln -s real/x.sh x.sh
  run "${kit_checkout}/bin/kit" generate x.sh
  [ "$status" -eq 0 ]
  [ -L x.sh ]
  grep --quiet '^linked$' real/x.sh
}

@test "a bare directive means kit region, which reads the name, so the kit's regions carry no command and need only kit on PATH" {
  run "${kit_checkout}/bin/kit" install agents/generic
  [ "$status" -eq 0 ]
  [ "$(sed -n '1p' AGENTS.md)" = '<!-- kit:generate AGENTS.md -->' ]
  printf '## Ours\n' >> AGENTS.md
  run env PATH="$(dirname "$(command -v bash)"):/usr/bin:/bin" "${kit_checkout}/bin/kit" generate --check
  [ "$status" -eq 0 ]
  [[ "$output" == *"every region as its command makes it"* ]]
  printf '# kit:generate nothing\n# kit:end nothing\n' > bare.txt
  run "${kit_checkout}/bin/kit" generate bare.txt
  [ "$status" -eq 1 ]
  [[ "$output" == *"no active package delivers a region at nothing"* ]]
  [[ "$output" == *"bare.txt:1: the command failed: kit region"* ]]
}

@test "a nested command's failure names the output's line, and an output that leaves a markdown fence open is refused" {
  cat > nf.sh <<'SH'
#!/usr/bin/env bash
printf '%s\n' head '# kit:generate inner `false`' '# kit:end inner'
SH
  chmod +x nf.sh
  printf '# kit:generate outer `./nf.sh`\nheld\n# kit:end outer\n' > nf.txt
  run "${kit_checkout}/bin/kit" generate nf.txt
  [ "$status" -eq 1 ]
  [[ "$output" == *"nf.txt:1 (the output of ./nf.sh):2: the command failed: false"* ]]
  grep --quiet '^held$' nf.txt
  printf '#!/usr/bin/env bash\nprintf "%%s\\n" text %s more\n' "'\`\`\`'" > fence.sh && chmod +x fence.sh
  printf '<!-- kit:generate odd `./fence.sh` -->\nheld\n<!-- kit:end odd -->\n' > odd.md
  run "${kit_checkout}/bin/kit" generate odd.md
  [ "$status" -eq 1 ]
  [[ "$output" == *"odd.md:1 (the output of ./fence.sh): a fenced code block opens at line 2 and never closes"* ]]
  grep --quiet '^held$' odd.md
  printf '# Title\n\n```\n<!-- kit:generate a `./lines.sh x` -->\n<!-- kit:end a -->\n' > open.md
  run "${kit_checkout}/bin/kit" generate open.md
  [ "$status" -eq 1 ]
  [[ "$output" == *"open.md: a fenced code block opens at line 3 and never closes"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
