#!/usr/bin/env bats
# claims.bats - what the pages say of the kit is what the kit is: its commands, settings, paths, lists, and counts
# ----------------------------------------------------------------------------------------------------------------------
# A page names commands, settings, and files, and lists and counts what the code holds. Each such claim drifts when
# the code moves and the page does not, and a reader who follows it fails. These tests read every page a reader sees:
# each tracked *.md and *.mdx but those under docs/superpowers/, whose plans, specs, and reviews are dated records of
# what was meant then, and those under tests/ and a testdata/, which are fixtures.
#
# A command is judged as the loader's guard judges a command line, by syntax::match over the verb's syntax, which kit
# syntax prints. A span or a block a page means as an example of a command that fails follows a line that holds
# kit:claims skip, in an HTML comment, <!-- kit:claims skip -->, or in MDX, {/* kit:claims skip */}: nothing is judged
# up to the next blank line, or, where a fence opens next, to its close.
#
# The docs check is a test, not a rule's check that kit lint runs: kit lint is not one of kit ci's four checks, and
# what these tests hold is this repository's pages to this repository's code, which no repository that installs the
# kit has.

bats_require_minimum_version 1.5.0

load helper

setup_file() {
  "${kit_root}/bin/kit" syntax > "${BATS_FILE_TMPDIR}/specs.json"
}

setup() {
  root=$(cd "${kit_root}/.." && pwd)
}

# pages prints every page a reader sees, from the checkout's root, as this file's header says.
pages() {
  git -C "${root}" ls-files '*.md' '*.mdx' \
    | grep --invert-match --extended-regexp '^(docs/superpowers|tests)/|/testdata/'
}

# spans <page>… prints each inline code span of the pages, and each line of a fenced block of a shell's, as
# <page> <tab> <line> <tab> <where> <tab> <text>. where is inline for a code span, whose line is the one it opens on,
# as a paragraph's lines are read joined by a space; fence for a line of a block whose language is sh, bash, shell,
# zsh, or console, less a leading $ and a space; and fault, its text that the fence is never closed. A fence closes on
# a line of its own marker, at least as long, with nothing after it.
spans() {
  awk '
    function flush(   text, off, k, at, span) {
      text = para
      off = 0
      while (n > 0 && match(text, /`[^`]+`/)) {
        span = substr(text, RSTART + 1, RLENGTH - 2)
        at = off + RSTART
        for (k = n; k > 1 && starts[k] > at; k--) ;
        if (!skip) print file "\t" lines[k] "\tinline\t" span
        off += RSTART + RLENGTH - 1
        text = substr(text, RSTART + RLENGTH)
      }
      para = ""
      n = 0
    }
    function finish() {
      flush()
      if (fence != "") print file "\t" opened "\tfault\tthe fence is never closed"
      fence = ""
      skip = 0
    }
    FNR == 1 { if (file != "") finish(); file = FILENAME }
    fence != "" {
      line = $0
      sub(/^[[:space:]]*/, "", line)
      if (index(line, substr(fence, 1, 1)) == 1 && line ~ /^(`+|~+)[[:space:]]*$/ && \
          length(line) >= length(fence) && substr(line, 1, 1) == substr(fence, 1, 1)) {
        sub(/[[:space:]]*$/, "", line)
        if (line ~ /^`+$/ || line ~ /^~+$/) { fence = ""; skip = 0; next }
      }
      if (shell && !skip) {
        sub(/^\$ /, "", line)
        print file "\t" FNR "\tfence\t" line
      }
      next
    }
    /^[[:space:]]*(```|~~~)/ {
      flush()
      line = $0
      sub(/^[[:space:]]*/, "", line)
      match(line, /^(`+|~+)/)
      fence = substr(line, 1, RLENGTH)
      lang = substr(line, RLENGTH + 1)
      sub(/^[[:space:]]*/, "", lang)
      sub(/[[:space:]].*$/, "", lang)
      shell = (lang ~ /^(sh|bash|shell|zsh|console)$/)
      opened = FNR
      if (skip == 1) skip = 2
      next
    }
    /^[[:space:]]*$/ { flush(); if (skip == 1) skip = 0; next }
    /kit:claims skip/ { flush(); skip = 1; next }
    {
      n++
      lines[n] = FNR
      starts[n] = length(para) + (n > 1 ? 2 : 1)
      para = (n > 1 ? para " " : "") $0
    }
    END { if (file != "") finish() }
  ' "$@"
}

# commands <page>… judges each span and fence line that starts with kit and a space, and prints
# <page>:<line>: `<text>`: <why> for each it refuses and <page>:<line>: <text> for each fault; it fails when it
# printed. The words stop at a shell's operator or comment. Over the words after kit and its own options, in order:
# a verb spelled as a placeholder, a word of prose, or with --help or -h passes, and one that is no verb is refused;
# a --name, up to its =, that is no flag or option of a verb without a wildcard is refused; a line that holds a form,
# not a value, as <…>, ${, …, a word in brackets, or an option that ends in =, passes; else syntax::match judges it:
# status 0 passes, 2 is its refusal, but inline where the last word is an option without its value, as prose names
# one, and 1 passes inline, as prose names a beginning, and in a fence is refused, naming what it stops before.
commands() {
  local found
  found=$(spans "$@" | jq --raw-input --raw-output -L "${kit_root}/modules/jq" \
    --slurpfile specs "${BATS_FILE_TMPDIR}/specs.json" '
    import "syntax" as syntax;
    def words:
      [scan("(?:[^\\s\"'\'']+|\"[^\"]*\"?|'\''[^'\'']*'\''?)+") | gsub("[\"'\'']"; "")]
      | (map(test("^(\\||\\|\\||&&|;|[0-9]*>.*|<|<<.*|#.*)$")) | index(true)) as $stop
      | if $stop == null then . else .[:$stop] end;
    def form: test("<|\\$\\{|…|\\.\\.\\.") or test("^[\\[(]") or test("[\\])]$") or test("=$");
    def judge($where):
      (.[1:] | map(test("^(-h|--help|-v|--version|(-v|--version|--workspace)=.*)$") | not) | index(true)) as $i
      | if $i == null then empty else .[1 + $i:] end
      | if .[0] == "--" then .[1:] else . end
      | .[0] as $verb | .[1:] as $args
      | if $verb == null or ($verb | test("^[a-z][a-z-]*$") | not) or ($args | index("--help") or index("-h"))
        then empty
        elif ([$specs[0][] | select(.verb == $verb)] | length) == 0 then "kit \($verb) is not a verb"
        else ($specs[0][] | select(.verb == $verb)) as $spec
        | ($spec.syntax | [.. | objects | select(.kind == "flag" or .kind == "option")]) as $named
        | [$args[:($args | index("--") // length)][] | select(test("^--[a-z]")) | sub("^--"; "") | sub("=.*$"; "")
            | select(. as $name | $named | map(.name) | index($name) | not)] as $unknown
        | if ($unknown | length) > 0 and ($named | any(.wildcard) | not) then "kit \($verb) takes no --\($unknown[0])"
          elif $args | any(form) then empty
          else ($spec | syntax::match($args)) as $m
          | if $m.status == 0 then empty
            elif $m.status == 2 then
              if $where == "inline" and ($args[-1] | test("^--[a-z][a-z-]*$"))
                and ($m.refusal | test("needs a value|takes its value after =")) then empty
              else $m.refusal end
            elif $where == "inline" then empty
            else "it stops before \($m.missing)" end
          end
        end;
    split("\t") as [$page, $line, $where, $text]
    | if $where == "fault" then "\($page):\($line): \($text)"
      elif $text | test("^kit ") | not then empty
      else ($text | words | judge($where)) as $why | "\($page):\($line): `\($text)`: \($why)" end')
  [[ -z "${found}" ]] || { printf '%s\n' "${found}"; return 1; }
}

# paragraphs <page>… prints each paragraph of the pages outside a fence, its lines joined by a space, as <page> <tab>
# <line> <tab> <text>, the line it opens on; a list's items are lines of one paragraph, as a table's rows are, and a
# paragraph that ends in a colon goes on into the one after it, the list it leads into.
paragraphs() {
  awk '
    function flush() { if (text != "" && !skip) print file "\t" at "\t" text; text = "" }
    FNR == 1 { flush(); file = FILENAME; fence = 0; skip = 0 }
    /^[[:space:]]*(```|~~~)/ { flush(); fence = !fence; if (!fence) skip = 0; next }
    fence { next }
    /^[[:space:]]*$/ { if (text !~ /:$/) { flush(); skip = 0 }; next }
    /kit:claims skip/ { flush(); skip = 1; next }
    { line = $0; sub(/^[[:space:]]+/, "", line); if (text == "") { at = FNR; text = line } else text = text " " line }
    END { flush() }
  ' "$@"
}

# facts prints, as one JSON object, the lists the code holds that the pages give: the stages kit flow draws, the
# checks kit ci runs, the seams and each one's providers, the tools kit serve offers and which read only or run what
# the repository's files name, the steps of kit welcome, the tracker providers, the jq modules, and the keys of
# kit.yml the schema describes.
facts() {
  local seam seams=()
  for seam in "${kit_root}"/modules/bash/*.sh
  do
    grep --quiet '^# [a-z]*\.sh - the seam over ' "${seam}" && seams+=("$(basename "${seam}" .sh)")
  done
  jq --null-input \
    --arg stages "$(cd "${BATS_TEST_TMPDIR}" && "${kit_root}/bin/kit" flow --stages)" \
    --arg checks "$(sed -n 's/^for verb in \([a-z ]*\)$/\1/p' "${kit_root}/bin/kit-ci")" \
    --arg seams "${seams[*]}" \
    --arg providers "$(cd "${kit_root}/modules/bash" && for seam in "${seams[@]}"; do ls "${seam}"/*.sh; done)" \
    --slurpfile tools "${root}/tests/snapshots/tools.json" \
    --arg welcome "$(cd "${kit_root}/packages/kit/files/repo/.kit/welcome" && ls [0-9]*.md)" \
    --arg trackers "$(cd "${kit_root}/modules/bash/tracker" && ls ./*.sh)" \
    --arg jq "$(cd "${kit_root}/modules/jq" && ls ./*.jq | sed 's#^\./##')" \
    --slurpfile schema "${kit_root}/packages/kit/files/repo/.kit/kit.schema.json" '
    def lines: split("\n") | map(select(. != ""));
    {stages: ($stages | lines), checks: ($checks | split(" ") | map(select(. != ""))),
     seams: ($seams | split(" ")), providers: ($providers | lines),
     tools: [$tools[0][].name], readonly: [$tools[0][] | select(.annotations.readOnlyHint) | .name],
     local: [$tools[0][] | select((.annotations.readOnlyHint | not) and (.annotations.openWorldHint | not)) | .name],
     welcome: ($welcome | lines), trackers: ($trackers | lines), jq: ($jq | lines),
     keys: ($schema[0].properties | keys)}'
}

# keys-named <page>… prints <page>:<line>: kit.yml has no key <key> for each key of kit.yml a paragraph names that the
# schema does not describe: as the `<key>` setting, or as `<key>` or `<key>:` in, of, or under `kit.yml`, a dotted
# key by its first part. It reads facts.json, as facts writes it, from the test's directory.
keys-named() {
  paragraphs "$@" | jq --raw-input --raw-output --slurpfile facts "${BATS_TEST_TMPDIR}/facts.json" '
    split("\t") as [$page, $line, $text]
    | [$text | scan("the `([a-z_][a-z_.]*)` settings?\\b"), scan("`([a-z_][a-z_.]*):?` (?:in|of|under) `kit\\.yml`")]
    | .[][0] | split(".")[0] | select(. as $key | $facts[0].keys | index($key) | not)
    | "\($page):\($line): kit.yml has no key \(.)"'
}

# counted <page>… prints <page>:<line>: <what is wrong> for each count a paragraph gives of a list the code holds that
# is not its length, and each list of kit flow'\''s stages it gives that is not the stages, in their order: after "N
# stages:" or "the stages `kit flow` draws, in order:", its items each a `name`:, or its words up to the period. It
# reads facts.json, as facts writes it, from the test's directory.
counted() {
  paragraphs "$@" | jq --raw-input --raw-output --slurpfile facts "${BATS_TEST_TMPDIR}/facts.json" '
    $facts[0] as $f
    | ["zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven", "twelve"]
      as $words
    | def number: ascii_downcase | if test("^[0-9]+$") then tonumber else . as $w | $words | index($w) end;
      def word: "([0-9]+|[Oo]ne|[Tt]wo|[Tt]hree|[Ff]our|[Ff]ive|[Ss]ix|[Ss]even|[Ee]ight|[Nn]ine|[Tt]en|[Ee]leven"
        + "|[Tt]welve)";
    split("\t") as [$page, $line, $text]
    | [ [word + " stages\\b|[Dd]efault: all " + word + ", in that order", $f.stages, "the stages kit flow draws"],
        ["`kit ci` runs " + word + " checks|\\b[Tt]he " + word + " checks are fixed|\\bruns " + word
          + " checks, each a kit command", $f.checks, "the checks kit ci runs"],
        [word + " files each define a seam|\\b[Tt]he " + word + " seams\\b", $f.seams, "the seams"],
        [word + " tools (?:that )?read only", $f.readonly, "the tools kit serve offers that read only"],
        [word + " tools run what the repository.s files name", $f.local,
          "the tools kit serve offers that run what the files name"],
        ["a guide in " + word + " steps", $f.welcome, "the steps of kit welcome"],
        ["its " + word + " providers are described in", $f.trackers, "the tracker providers"] ][]
      as [$pattern, $list, $what]
    | ($text | [match($pattern; "g") | .captures | map(select(.string != null))[0].string]) as $said
    | ($said[] | select(number != ($list | length))
        | "\($page):\($line): says \(.) of \($what), which number \($list | length)"),
      (if $what == "the stages kit flow draws" then
         ($text | capture("(?:" + word + " stages|[Tt]he stages `kit flow` draws, in order): (?<rest>.*)")?.rest)
           as $rest
         | ($rest | if test("^- `") then [scan("(?:^|\\s)- `([a-z]+)`:")[0]]
             else split(".")[0] | split(", ") | map(sub("^and "; "") | gsub("`"; "") | sub("\\s.*$"; "")) end)
         | select(. != $f.stages)
         | "\($page):\($line): lists the stages as \(join(", ")); kit flow --stages prints \($f.stages | join(", "))"
       else empty end)'
}

# paths <page>… prints <page>:<line>: no <path> for each inline span that is a path from the checkout'\''s root, or
# from kit/, as a page writes modules/ and packages/, where nothing is; run from the root.
paths() {
  spans "$@" \
    | awk -F '\t' '$3 == "inline" && $4 ~ /^(bin|docs|kit|projects|tests|modules|packages)\/[A-Za-z0-9_.\/-]*$/' \
    | while IFS=$'\t' read -r page line _ path
      do
        [[ -e "${root}/${path}" || -e "${kit_root}/${path}" ]] || printf '%s:%s: no %s\n' "${page}" "${line}" "${path}"
      done
}

@test "a command is judged by its verb, its flags, and its syntax, --option=value; a form, a beginning, and a skip pass" {
  cat > page.md <<'EOF'
`kit commit`, `kit packages add`, `kit context <path> --json`, and `kit close --duplicate-of` pass. A span the wrap
splits, `kit commit
--bogus`, is one, and `kit nosuch x`, `kit flow --nosuch`, and `kit version now` are each refused.

<!-- kit:claims skip -->
`kit nosuch` is an example that is meant to fail.

```sh
kit commit --message="a b" x.txt | cat
kit facts > fixture.json   # a comment
$ kit commit
kit work --title hi 7
```

```sh
kit version
EOF
  run commands page.md
  [ "$status" -eq 1 ]
  [ "${lines[0]}" = 'page.md:2: `kit commit --bogus`: kit commit takes no --bogus' ]
  [ "${lines[1]}" = 'page.md:3: `kit nosuch x`: kit nosuch is not a verb' ]
  [ "${lines[2]}" = 'page.md:3: `kit flow --nosuch`: kit flow takes no --nosuch' ]
  [ "${lines[3]}" = 'page.md:3: `kit version now`: kit version takes no arguments; run: kit help version' ]
  [ "${lines[4]}" = 'page.md:11: `kit commit`: it stops before --message or --file' ]
  [[ "${lines[5]}" == 'page.md:12: `kit work --title hi 7`: --title takes its value after =: --title=<text>; run: '* ]]
  [ "${lines[6]}" = 'page.md:15: the fence is never closed' ]
  [ "${#lines[@]}" -eq 7 ]
}

@test "every command the pages name is one its verb takes" {
  cd "${root}"
  # shellcheck disable=SC2046  # a page's path has no space
  run commands $(pages)
  cd "${BATS_TEST_TMPDIR}"
  [ "$status" -eq 0 ] || { printf '%s\n' "${output}"; return 1; }
}

@test "a key, a count, a list, and a path the code does not hold are each named; what the code holds passes" {
  local stages
  facts > "${BATS_TEST_TMPDIR}/facts.json"
  cat > page.md <<'EOF'
The `forge` setting, the `nosuch` setting, and `answers.ci_runner` in `kit.yml` are named, and `gone:` of
`kit.yml` too. `kit flow` draws seven stages: issues, desk, and releases.

`kit flow` draws, in order, five stages:

- `issues`: every open issue
- `incoming`: what fetching brought
- `changes`, `builds`, and `releases`

`kit/bin/kit-flow`, `modules/bash/flow.sh`, and `docs/nosuch.md` are paths.
EOF
  run keys-named page.md
  [ "${lines[0]}" = 'page.md:1: kit.yml has no key nosuch' ]
  [ "${lines[1]}" = 'page.md:1: kit.yml has no key gone' ]
  [ "${#lines[@]}" -eq 2 ]
  run counted page.md
  [ "${lines[0]}" = 'page.md:1: says seven of the stages kit flow draws, which number 5' ]
  stages='kit flow --stages prints issues, changes, builds, incoming, releases'
  [ "${lines[1]}" = "page.md:1: lists the stages as issues, desk, releases; ${stages}" ]
  [ "${lines[2]}" = "page.md:4: lists the stages as issues, incoming; ${stages}" ]
  [ "${#lines[@]}" -eq 3 ]
  run paths page.md
  [ "$output" = 'page.md:10: no docs/nosuch.md' ]
}

@test "every key the schema describes is one settings:validate knows, and the pages name no other, and all of them" {
  local key known
  facts > "${BATS_TEST_TMPDIR}/facts.json"
  fresh-repo >/dev/null
  for key in $(jq --raw-output '.keys[]' "${BATS_TEST_TMPDIR}/facts.json") nosuch
  do
    printf '%s:\n' "${key}" > kit.yml
    kit_bash_modules=settings run kit-bash settings:validate
    if [[ "${output}" == *"a key the kit does not know"* ]]
    then
      [ "${key}" = nosuch ] || { echo "settings:validate does not know ${key}"; return 1; }
    else
      [ "${key}" != nosuch ] || { echo "settings:validate knows nosuch"; return 1; }
    fi
  done
  cd "${root}"
  # shellcheck disable=SC2046  # a page's path has no space
  known=$(keys-named $(pages))
  cd "${BATS_TEST_TMPDIR}"
  [ -z "${known}" ] || { printf '%s\n' "${known}"; return 1; }
  # docs/packages.md's table of the keys names every one, from version's row to the next blank line
  known=$(sed -n '/^| `version` |/,/^$/s/^| `\([a-z_]*\)` |.*/\1/p' "${root}/docs/packages.md" | LC_ALL=C sort)
  [ "${known}" = "$(jq --raw-output '.keys[]' "${BATS_TEST_TMPDIR}/facts.json" | LC_ALL=C sort)" ] || {
    echo "docs/packages.md's table of the keys is not the schema's:"
    jq --raw-output '.keys[]' "${BATS_TEST_TMPDIR}/facts.json" | LC_ALL=C sort | diff - <(printf '%s\n' "${known}")
    return 1
  }
}

@test "a path a page names from the root, or from kit/, is there" {
  local found
  cd "${root}"
  # shellcheck disable=SC2046  # a page's path has no space
  found=$(paths $(pages))
  cd "${BATS_TEST_TMPDIR}"
  [ -z "${found}" ] || { printf '%s\n' "${found}"; return 1; }
}

@test "a count a page gives of what the code holds is the count, and a list it gives is the list, in its order" {
  local found
  facts > "${BATS_TEST_TMPDIR}/facts.json"
  cd "${root}"
  # shellcheck disable=SC2046  # a page's path has no space
  found=$(counted $(pages))
  cd "${BATS_TEST_TMPDIR}"
  [ -z "${found}" ] || { printf '%s\n' "${found}"; return 1; }
}

@test "the tables of seams, tools, and jq modules name every one the code holds, and no other" {
  local seam row provider found=() modules
  facts > "${BATS_TEST_TMPDIR}/facts.json"
  # docs/architecture.md: a row a seam, each naming every provider of its seam
  for seam in $(jq --raw-output '.seams[]' "${BATS_TEST_TMPDIR}/facts.json")
  do
    row=$(grep --fixed-strings "| \`kit/modules/bash/${seam}.sh\` |" "${root}/docs/architecture.md" || true)
    [[ -n "${row}" ]] || found+=("docs/architecture.md: no row for the seam ${seam}")
    for provider in $(jq --raw-output --arg seam "${seam}" '.providers[] | select(startswith($seam + "/"))' \
      "${BATS_TEST_TMPDIR}/facts.json")
    do
      [[ -z "${row}" || "${row}" == *"\`${provider}\`"* ]] \
        || found+=("docs/architecture.md: the row of ${seam}.sh does not name ${provider}")
    done
  done
  # docs/serve.md: the tools its tables name are the tools kit serve offers
  [ "$(sed -n 's/^| `\([a-z]*\)` | `kit \1` |.*/\1/p' "${root}/docs/serve.md" | LC_ALL=C sort -u)" \
    = "$(jq --raw-output '.tools[]' "${BATS_TEST_TMPDIR}/facts.json" | LC_ALL=C sort)" ] \
    || found+=("docs/serve.md: its tables do not name the tools kit serve offers")
  # docs/modules.md: a row a jq module
  modules=$(sed -n 's/^| `\([a-z-]*\.jq\)` |.*/\1/p' "${root}/docs/modules.md" | LC_ALL=C sort)
  [ "${modules}" = "$(jq --raw-output '.jq[]' "${BATS_TEST_TMPDIR}/facts.json" | LC_ALL=C sort)" ] \
    || found+=("docs/modules.md: its table of jq modules is not kit/modules/jq/'s")
  (( ${#found[@]} == 0 )) || { printf '%s\n' "${found[@]}"; return 1; }
}
