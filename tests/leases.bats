#!/usr/bin/env bats
# leases.bats - kit leases: claims in refs every worktree sees, judged one after another, and a branch held to its own
# ----------------------------------------------------------------------------------------------------------------------
# The world is commit.bats's: origin is a bare repository beside the desk, the desk has the kit installed and its start
# commit on main, and the work happens in worktrees/7-fix-the-thing, the branch kit work would make for issue 7. No
# forge is involved: an issue's key is its number, as the forge's own tracker names it.

bats_require_minimum_version 1.5.0

load helper

# desk-world makes what every test starts from, once for the file, as commit.bats's does.
desk-world() {
  git init --quiet --bare --initial-branch=main origin.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk || return 1
  "${kit_checkout}/bin/kit" install
  printf 'notes\n' > notes.txt
  git add --all && git commit --quiet --message '#1 start: the first' && git push --quiet origin main
  git remote set-head origin main
}

setup_file() {
  world-make desk desk-world
}

setup() {
  work=$(mktemp -d)
  world-copy desk "${work}"
  cd "${work}/desk"
  git config user.name ada
  git worktree add --quiet -b 7-fix-the-thing worktrees/7-fix-the-thing main
  git worktree add --quiet -b 8-other worktrees/8-other main
  cd worktrees/7-fix-the-thing
}

kit() { "${kit_checkout}/bin/kit" "$@"; }

# lease <key> <json> writes a lease blob by hand, as another session or another kit could have left it.
lease() {
  git update-ref "refs/kit/leases/$1" "$(printf '%s' "$2" | git hash-object -w --stdin)"
}

@test "kit leases lists none, then each lease with its holder, branch, expiry, and paths; --json is {now, leases}" {
  run kit leases
  [ "$status" -eq 0 ]
  [[ "$output" == *"No lease is held in this clone; take one: kit leases take <reference> [<path>…]"* ]]
  run kit leases take 7 'docs/**' kit/bin/kit-leases --holder=session-a
  [ "$status" -eq 0 ]
  [[ "$output" == *"#7 holds docs/** kit/bin/kit-leases, until "*", in 1h"* ]]
  run --separate-stderr kit leases
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "#7  session-a on "*", 7-fix-the-thing, until "*", in "* ]]
  [ "${lines[1]}" = '    docs/**  kit/bin/kit-leases' ]
  run --separate-stderr kit leases --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '.leases[] | {key, issue, paths, holder, branch, live, readable}' <<<"$output")" \
    = '{"key":"7","issue":"#7","paths":["docs/**","kit/bin/kit-leases"],"holder":"session-a","branch":"7-fix-the-thing","live":true,"readable":true}' ]
  [ "$(jq '.leases[0] | .expires - .renewed' <<<"$output")" = 3600 ]
  [ "$(jq -r '.leases[0].worktree' <<<"$output")" = "$(pwd -P)" ]
  # the holder is git's user.name without --holder, and every worktree of the clone sees the same refs
  cd ../8-other
  run kit leases take 8 tests/x.bats
  [ "$status" -eq 0 ]
  [ "$(kit leases --json | jq -r '.leases[] | "\(.key) \(.holder) \(.branch)"')" \
    = $'7 session-a 7-fix-the-thing\n8 ada 8-other' ]
}

@test "a claim that may overlap another issue's live lease is refused, naming who holds what, and nothing is written" {
  kit leases take 7 docs --holder=session-a >/dev/null 2>&1
  cd ../8-other
  before=$(git for-each-ref refs/kit)
  run kit leases take 8 docs/policy.md
  [ "$status" -eq 1 ]
  [[ "$output" == *"refusing to lease docs/policy.md to #8: #7 holds docs, taken by session-a on 7-fix-the-thing, until "* ]]
  [[ "$output" == *"ask who holds it, or claim other paths: kit leases take #8 <path>…"* ]]
  [ "$(git for-each-ref refs/kit)" = "${before}" ]
  # without paths a claim is **, which overlaps every other
  run kit leases take 8
  [ "$status" -eq 1 ]
  [[ "$output" == *"refusing to lease ** to #8"* ]]
  # paths kit can prove apart are taken
  run kit leases take 8 'docs.md' 'kit/*.sh'
  [ "$status" -eq 0 ]
  [[ "$output" == *"#8 holds docs.md kit/*.sh"* ]]
}

@test "taking a lease again keeps its live paths and adds the new ones; an expired one is replaced" {
  kit leases take 7 docs >/dev/null 2>&1
  run kit leases take 7 tests/leases.bats --for=2h
  [ "$status" -eq 0 ]
  [ "$(kit leases --json | jq -c '.leases[0] | [.paths, .expires - .renewed]')" = '[["docs","tests/leases.bats"],7200]' ]
  lease 7 '{"lease": 1, "issue": "#7", "paths": ["docs"], "holder": "ada", "branch": "7-fix-the-thing", "taken": 100, "renewed": 100, "expires": 200}'
  run kit leases take 7 kit
  [ "$status" -eq 0 ]
  [ "$(kit leases --json | jq -c '.leases[0] | [.paths, .live]')" = '[["kit"],true]' ]
}

@test "two sessions that claim overlapping paths at once: the one that read the leases before the other landed is refused" {
  # Claimant A stops right after it reads the leases, and claimant B, another process, takes an overlapping lease in
  # that gap. Two fifos order them, never the clock: A's write to read.fifo waits for B to open it, and B's write to
  # done.fifo, after its take, lets A go on. A read the epoch before the leases, so B's take moved it, A's swap fails,
  # and A, judged again, is refused, naming B. Were the epoch read after the leases, A would read B's epoch, its swap
  # would land, and both would hold docs.
  mkfifo "${work}/read.fifo" "${work}/done.fifo"
  (
    cd ../8-other
    read -r _ < "${work}/read.fifo"
    if kit leases take 8 docs/x.md > "${work}/b.out" 2>&1; then echo 0; else echo 1; fi > "${work}/b.st"
    printf 'done\n' > "${work}/done.fifo"
  ) &
  b=$!
  run kit-bash "use leases
    eval \"\$(declare -f list:read | sed '1s/list:read/leases:list-read/')\"
    list:read() {
      local status=0
      leases:list-read \"\$@\" || status=\$?
      if [[ \"\$2 \$3\" == 'git for-each-ref' && ! -e '${work}/paused' ]]
      then
        : > '${work}/paused'
        printf 'read\\n' > '${work}/read.fifo'
        read -r _ < '${work}/done.fifo'
      fi
      return \"\${status}\"
    }
    leases:take 7 '' ada docs"
  # A that never reached the gap leaves B waiting on read.fifo; it is let go, and the test fails below
  [[ -e "${work}/paused" ]] || { kill "${b}" 2>/dev/null || true; }  # B waits on a fifo A never writes
  wait "${b}" || true  # B's own status is in b.st
  [ -e "${work}/paused" ]
  [ "$(cat "${work}/b.st")" = 0 ]
  [ "$status" -eq 1 ]
  [[ "$output" == *"refusing to lease docs to #7: #8 holds docs/x.md, taken by ada on 8-other"* ]]
  [ "$(git for-each-ref --format='%(refname)' 'refs/kit/leases/*')" = refs/kit/leases/8 ]
}

@test "a claim that loses the race is judged again: the epoch moved under it, so the new lease refuses it" {
  run kit-bash "use leases
    eval \"\$(declare -f leases:swap | sed '1s/leases:swap/leases:landed/')\"
    leases:swap() {
      if [[ ! -e '${work}/raced' ]]
      then
        : > '${work}/raced'
        (cd ../8-other && '${kit_checkout}/bin/kit' leases take 8 docs >/dev/null 2>&1)
      fi
      leases:landed \"\$@\"
    }
    leases:take 7 '' ada docs/policy.md"
  [ "$status" -eq 1 ]
  [ -e "${work}/raced" ]
  [[ "$output" == *"refusing to lease docs/policy.md to #7: #8 holds docs, taken by ada on 8-other"* ]]
  [ "$(git for-each-ref --format='%(refname)' 'refs/kit/leases/*')" = refs/kit/leases/8 ]
}

@test "an expired lease refuses nothing and lists as expired; renewing it judges it again" {
  cd ../8-other
  lease 8 '{"lease": 1, "issue": "#8", "paths": ["docs"], "holder": "bob", "branch": "8-other", "taken": 100, "renewed": 100, "expires": 160}'
  cd ../7-fix-the-thing
  run kit leases take 7 docs/policy.md
  [ "$status" -eq 0 ]
  run --separate-stderr kit leases
  [[ "${lines[0]}" == '#7  ada on '* ]]
  [[ "${lines[2]}" == '#8  bob, 8-other, expired '*' ago' ]]
  [ "$(kit leases --json | jq -c '[.leases[] | .live]')" = '[true,false]' ]
  run kit leases renew 8
  [ "$status" -eq 1 ]
  [[ "$output" == *"refusing to renew #8's lease, which expired: #7 holds docs/policy.md"* ]]
  kit leases release 7 >/dev/null 2>&1
  run kit leases renew 8
  [ "$status" -eq 0 ]
  [ "$(kit leases --json | jq -c '.leases[0] | [.live, .expires - .renewed, .taken]')" = '[true,60,100]' ]
}

@test "renew moves a lease's expiry by its term or by --for, and refuses an issue that holds none" {
  kit leases take 7 docs --for=30m >/dev/null 2>&1
  run kit leases renew 7
  [ "$status" -eq 0 ]
  [ "$(kit leases --json | jq '.leases[0] | .expires - .renewed')" = 1800 ]
  run kit leases renew '#7' --for=1d
  [ "$status" -eq 0 ]
  [ "$(kit leases --json | jq '.leases[0] | .expires - .renewed')" = 86400 ]
  run kit leases renew 9
  [ "$status" -eq 1 ]
  [[ "$output" == *"#9 holds no lease to renew; take one: kit leases take #9 [<path>…]"* ]]
  run kit leases take 7 --for=soon
  [ "$status" -eq 2 ]
  [[ "$output" == *"'soon' is not a duration; --for takes a number and a unit, s, m, h, or d: --for=45m"* ]]
}

@test "a path from /, or with a .. segment, is refused before anything is written" {
  run kit leases take 7 /etc
  [ "$status" -eq 2 ]
  [[ "$output" == *"'/etc' starts with /; a path is relative to the repository's root"* ]]
  run kit leases take 7 docs/../kit
  [ "$status" -eq 2 ]
  [[ "$output" == *"'docs/../kit' holds a .. segment"* ]]
  [ -z "$(git for-each-ref refs/kit)" ]
}

@test "a lease that does not read overlaps every claim, lists with the command that releases it, and is released" {
  lease junk 'not a lease'
  run kit leases take 7 docs
  [ "$status" -eq 1 ]
  [[ "$output" == *"refs/kit/leases/junk does not read as a lease, so it holds every path; release it: kit leases release junk"* ]]
  run --separate-stderr kit leases
  [ "$output" = 'refs/kit/leases/junk does not read as a lease, so it holds every path until it is released: kit leases release junk' ]
  run kit leases renew junk
  [ "$status" -eq 1 ]
  [[ "$output" == *"refs/kit/leases/junk does not read as a lease, so it cannot be renewed"* ]]
  run kit leases release junk
  [ "$status" -eq 0 ]
  [[ "$output" == *"junk's lease is released"* ]]
  run kit leases take 7 docs
  [ "$status" -eq 0 ]
  # releasing what no lease holds says so, and is no failure
  run kit leases release 9
  [ "$status" -eq 0 ]
  [[ "$output" == *"#9 holds no lease here"* ]]
}

@test "leases on a branch nest in the branch's own lease, which another branch's claim still competes with" {
  kit leases take 7 >/dev/null 2>&1
  run kit leases take 9 docs/policy.md
  [ "$status" -eq 0 ]
  run kit leases take 10 docs
  [ "$status" -eq 1 ]
  [[ "$output" == *"refusing to lease docs to #10: #9 holds docs/policy.md, taken by ada on 7-fix-the-thing"* ]]
  cd ../8-other
  run kit leases take 8 kit
  [ "$status" -eq 1 ]
  [[ "$output" == *"#7 holds **"* ]]
}

@test "kit commit refuses a path another issue's lease on the branch holds, and renews the lease of the subject's issue" {
  kit leases take 7 >/dev/null 2>&1
  kit leases take 9 notes.txt --holder=session-b --for=10m >/dev/null 2>&1
  printf 'more\n' >> notes.txt && printf 'x\n' > x.txt && git add notes.txt x.txt
  run kit commit --message='#7 notes: more'
  [ "$status" -eq 1 ]
  [[ "$output" == *"refusing to commit notes.txt: #9 holds notes.txt, taken by session-b on 7-fix-the-thing, until "*"; name #9 in the commit, or ask its holder: kit leases"* ]]
  [[ "$output" != *"refusing to commit x.txt"* ]]
  [ "$(git log --oneline main..HEAD | wc -l | tr -d ' ')" = 0 ]
  renewed=$(kit leases --json | jq '.leases[] | select(.key == "9") | .renewed')
  sleep 1
  run kit commit --message='#7 notes: more' --message='Refs: #9'
  [ "$status" -eq 0 ]
  [ "$(kit leases --json | jq --argjson was "${renewed}" '.leases[] | select(.key == "9") | .renewed > $was and .expires - .renewed == 600')" = true ]
  # a lease on another branch holds nothing here, and one that does not read is warned of
  lease 8 "{\"lease\": 1, \"issue\": \"#8\", \"paths\": [\"y.txt\"], \"branch\": \"8-other\", \"expires\": $(( $(date +%s) + 600 ))}"
  lease junk 'not a lease'
  printf 'y\n' > y.txt && git add y.txt
  run kit commit --message='#7 y: a file'
  [ "$status" -eq 0 ]
  [[ "$output" == *"refs/kit/leases/junk does not read as a lease"* ]]
}

@test "kit leases --check and kit policy hold the paths a branch changed to its own lease" {
  run kit leases --check
  [ "$status" -eq 0 ]
  [[ "$output" == *"#7 holds no lease, so 7-fix-the-thing's paths are held to none"* ]]
  mkdir -p docs kit && printf 'a\n' > docs/a.md && printf 'b\n' > kit/b.sh
  git add docs kit && git commit --quiet --message '#7 docs: a and b'
  git mv notes.txt docs/notes.txt && git commit --quiet --message '#7 docs: notes move'
  kit leases take 7 docs >/dev/null 2>&1
  run kit leases --check
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit/b.sh is outside #7's lease, docs; claim it: kit leases take #7 kit/b.sh, then kit submit"* ]]
  [[ "$output" == *"notes.txt is outside #7's lease"* ]]
  [[ "$output" != *"docs/a.md is outside"* ]]
  run kit policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit/b.sh is outside #7's lease"* ]]
  kit leases take 7 kit/b.sh notes.txt >/dev/null 2>&1
  run kit leases --check
  [ "$status" -eq 0 ]
  [[ "$output" == *"the 4 path(s) 7-fix-the-thing changed are inside #7's lease"* ]]
  run kit policy
  [ "$status" -eq 0 ]
  # a lease that does not read cannot judge the branch
  lease 7 'garbled'
  run kit leases --check
  [ "$status" -eq 1 ]
  [[ "$output" == *"refs/kit/leases/7 does not read as a lease, so 7-fix-the-thing's paths cannot be judged"* ]]
  # a branch that names no issue, and the default branch, are held to none
  git switch --quiet --create holding
  run kit leases --check
  [ "$status" -eq 0 ]
  [[ "$output" == *"holding names no issue, so no lease holds it to its paths"* ]]
}

@test "kit submit pushes the branch's own lease, a runner's clone reads it from the primary, and release takes it away" {
  mkdir -p kit && printf 'b\n' > kit/b.sh && git add kit && git commit --quiet --message '#7 kit: b'
  kit leases take 7 docs >/dev/null 2>&1
  kit leases take 9 kit --holder=other >/dev/null 2>&1
  run kit submit
  [ "$status" -eq 0 ]
  [[ "$output" == *"#7's lease is on origin, for CI"* ]]
  [ "$(git ls-remote "${work}/origin.git" 'refs/kit/*' | cut -f 2)" = refs/kit/leases/7 ]
  # the runner: a fresh clone of the change, its base in the event, holding no lease of its own
  git clone --quiet "${work}/origin.git" "${work}/runner" 2>/dev/null
  cd "${work}/runner"
  git switch --quiet 7-fix-the-thing
  printf '{"pull_request": {"base": {"sha": "%s"}}}\n' "$(git rev-parse origin/main)" > "${work}/event.json"
  export GITHUB_HEAD_REF=7-fix-the-thing GITHUB_EVENT_PATH=${work}/event.json
  run kit leases --check
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit/b.sh is outside #7's lease, docs"* ]]
  [ -z "$(git for-each-ref refs/kit)" ]
  # at a desk, a clone with no lease of its own is held to none, whatever the primary holds
  unset GITHUB_HEAD_REF GITHUB_EVENT_PATH
  run kit leases --check
  [ "$status" -eq 0 ]
  [[ "$output" == *"#7 holds no lease"* ]]
  cd "${work}/desk/worktrees/7-fix-the-thing"
  run kit leases release 7
  [ "$status" -eq 0 ]
  [ -z "$(git ls-remote "${work}/origin.git" 'refs/kit/*')" ]
}

@test "kit leases refuses outside a git repository, and a word it does not know" {
  cd "${work}"
  run kit leases
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit leases runs in a git repository, where the leases are refs"* ]]
  cd desk
  run kit leases grab 7
  [ "$status" -eq 2 ]
}

# ----------------------------------------------------------------------------------------------------------------------
