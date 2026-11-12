# Terminal Checks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** kit welcome's setup step judges the terminal kit runs in, from its settings and from probes, and names the
setting that makes kittui slower or worse, for ten terminals, with a docs page that lists every finding.

**Architecture:** `kit/modules/jq/terminals.json` holds every terminal, finding, and probe as data, and
`kit/modules/jq/terminal.jq` turns a terminal's settings, its probes, and the kept list into checks. `kit/modules/bash/terminal.sh`
detects the terminal, runs its reader (`kit/modules/bash/terminal/<key>.sh`, one per terminal), evaluates each finding's
`bad`, and probes `/dev/tty` before any verb hands it to kittui, exporting `KIT_TERMINAL_PROBES`. welcome.sh asks
terminal.sh for the checks once, as machine checks of setup; kittui folds held terminal checks under their heading and
offers later and keep.

**Tech Stack:** bash, jq, yq, bats with fakes on `PATH`; Go (Bubble Tea v2.1.1) in `projects/tui/`.

**Spec:** [the design](../specs/2026-11-11-terminal-checks-design.md); issue #211.

## Global Constraints

- Commits: `kit commit` only, never `kit submit` or push. Subjects start `#211 kit: …`, `#211 kittui: …`, or
  `#211 docs: …`. `export PATH="$PWD/kit/bin:$PATH"` before `kit commit`.
- Peers share the worktree. Stage named paths only; guard with
  `test "$(git diff --cached --name-only)" = "<your paths, sorted>" && kit commit --message=…`; never leave anything
  staged; `git show --stat HEAD` after.
- Everything under `kit/` is written for a consumer; a path in it is written from `kit/`: `modules/bash/terminal.sh`.
  A link to kit's docs is a full URL: `https://github.com/empowerite/kit/blob/main/docs/terminals.md#iterm2`.
- Every option with a value is `--opt=value`: `--keep=<check>`, `--unkeep=<check>`. Lines are 120 cells at most.
- kit never writes a terminal's settings. Every terminal check has `runs: false` and `interactive: false`.
- A reader or probe takes at most 1 second (`terminal:timed 1 …`); a probe exchange at most 0.5 seconds.
- Check ids: `terminal-<key>` (the heading), `terminal-<key>-<finding>`, `terminal-probe-<probe>`. Keys: `iterm`,
  `apple`, `ghostty`, `kitty`, `wezterm`, `alacritty`, `warp`, `tabby`, `wt`, `console`, `tmux`, `screen`, `zellij`,
  `ssh`, `unknown`. A key never contains `-`.
- Kept ids live in `terminal: kept:` of `${KIT_HOME:-$HOME}/.kit/settings.yml`, as `<id>@<profile>` where the reader
  gave a profile, else `<id>`.
- Run bats as `kit/bin/kit tools run bats tests/<file>.bats`; Go tests as `cd projects/tui && go test ./...` with the
  kit toolchain's go (`export PATH="$(kit/bin/kit tools path | paste -sd: -):$PATH"`). Coverage in `projects/tui/`
  stays at 98.0 or more. `kit gate` before each commit.
- A verb's header is its help page and its usage lines its grammar; a changed usage line refreshes
  `projects/tui/internal/syntax/testdata/welcome.json` with the command `tests/syntax.bats` prints.

## Review Focus

1. A profile name with spaces, parentheses, quotes, or a dot ("Default (copy)", "Pro.Dark"): iTerm2 finds it; Terminal
   says it cannot read a name with a dot and holds. Tasks 5 and 6.
2. A terminal that answers after the half second, as over a slow ssh link: its late answer must not land in the
   person's shell as typed text. Task 3 drains the tty before restoring it.
3. No `~/.kit/settings.yml`, or one without `terminal:`; `--keep=` twice; `--unkeep=` of an id not kept: kept is `[]`,
   no duplicate, exit 0. Task 2.
4. `KIT_TERMINAL_PROBES` that is not a JSON object (set by hand, or empty): ignored as if unset; welcome still runs.
   Task 3.
5. A reader whose command hangs (`ghostty +show-config`, `reg.exe` with interop slow): cut at one second, and the
   heading says kit could not read the settings in time. Task 2's `terminal:timed`, tested there.

---

## Task 1: The findings table and its judge

**Files:**

- Create: `kit/modules/jq/terminals.json`, `kit/modules/jq/terminal.jq`, `tests/jq/terminal.test`

**Interfaces:**

- Produces `terminals.json`:

```json
{
  "docs": "https://github.com/empowerite/kit/blob/main/docs/terminals.md",
  "probes": {
    "kitty-keyboard": {"need": "keys", "held": "…", "failed": "…"},
    "emoji-width": {"need": "widths", "held": "…", "failed": "…"},
    "ambiguous-width": {"need": "widths", "held": "…", "failed": "…"},
    "color": {"need": "color", "held": "…", "failed": "…"}
  },
  "terminals": {
    "iterm": {
      "name": "iTerm2", "anchor": "iterm2",
      "findings": [{"id": "triggers", "need": "speed", "bad": "<jq over the reader's object>",
                    "line": "…{profile}…", "fix": "…{profile}…"}],
      "probes": {"kitty-keyboard": {"fix": "…"}}
    },
    "ssh": {"name": "a terminal over ssh", "anchor": "over-ssh", "unread": "kit runs over ssh, so it cannot read …"}
  }
}
```

  A terminal's `probes.<probe>` is `{"fix": "…"}` (a setting turns it on) or `{"none": "…"}` (no setting can; the line
  that says what kittui does instead). `unread` marks the five keys with no reader, `ssh`, `tmux`, `screen`, `zellij`,
  and `unknown`, and is the line their heading says. This task writes the probes, those five keys, and iTerm2's
  `triggers` row; Tasks 5 to 9 add the rest.

- Produces `terminal.jq` (`import "terminal" as terminal;` with `jq -L "${kit_jq}"`):
  - `terminal::data` gives terminals.json.
  - `terminal::fill($o)`: a string with `{name}` replaced by `$o.name`'s value for every key of `$o`.
  - `terminal::checks($key; $read; $bad; $probes; $kept)` gives an array of `{id, ok, line, fix, more}`:
    `$read` is the reader's object, `{"unreadable": "<why>"}`, or null for a key with `unread`; `$bad` is
    `{"<finding>": true|false}`; `$probes` is the probes object or null; `$kept` is an array of strings.

- [ ] **Step 1: Write the failing tests** in `tests/jq/terminal.test`, each case named by a comment, its table given
  inline (as `tests/jq/glyphs.test` does), plus one against `terminals.json` itself:
  - the heading comes first and holds: `.[0] == {id: "terminal-iterm", ok: true, line: "iTerm2, profile Work", …}`
    when a finding failed; `line` ends `": nothing in it slows or breaks kittui"` when every finding held and every
    judged probe held.
  - a failed finding: `{id: "terminal-iterm-triggers", ok: false, line: <filled>, fix: <filled>, more: "<docs>#iterm2"}`.
  - kept: with `$kept == ["terminal-iterm-triggers@Work"]` and `$read.profile == "Work"`, the finding is
    `ok: true`, `line` starts `"kept: "`, `fix: null`, `more == "kit welcome --unkeep=terminal-iterm-triggers"`; with
    profile `"Default"` the same keep does not apply.
  - a probe whose need has a failed finding is not in the output at all.
  - a probe with `{"none": …}` in the terminal's row that measured badly: `ok: true`, `line` is the `none` text.
  - a probe with `{"fix": …}` that measured badly: `ok: false`, `fix` is that text.
  - a probe measured badly where settings were read and every finding of its need held, and no row for it: `ok: false`,
    `fix: null`, `line` ends `"; no setting kit knows explains it"`.
  - `emoji-width` failed with `grapheme_width: false`: its line adds `", and it does not measure by grapheme"`;
    `grapheme_width` alone (emoji width 2) yields no check.
  - `$probes == {"answered": false}`: every probe check holds, its line `"<name> did not answer kit's question"`.
  - `$probes == null`: no `terminal-probe-*` check.
  - `$read == {"unreadable": "it took more than a second"}`: heading `"iTerm2: kit could not read its settings: it took
    more than a second"`, no finding checks, probes judged as for an unread terminal.
  - against the real table: every terminal's findings have unique ids without `-` in the key, every `need` is one of
    `speed clicks keys widths color`, and every `bad` compiles: `[data.terminals[].findings[]?.bad] | length > 0`
    (compilation is checked in Task 2's bats, since jq cannot eval a string).
- [ ] **Step 2: Run them to see them fail.** Run: `kit/bin/kit tools run bats tests/jq.bats`. Expected: FAIL,
  `modules/jq/terminal.jq` missing.
- [ ] **Step 3: Write `terminals.json` and `terminal.jq`.** The probe judgments are fixed rules in `checks`, in the
  spec's order: kitty-keyboard fails on `.kitty_keyboard == false`; emoji-width on `.emoji_width != 2`;
  ambiguous-width on `.ambiguous_width != 1`; color on `.color == "fewer"`. Precedence for a probe that measured badly:
  skipped when a finding of its need failed; else `none` holds; else `fix` fails with it; else fails unexplained. The
  module header says the table's keys, as `glyphs.jq` does.
- [ ] **Step 4: Run them to see them pass.** Run: `kit/bin/kit tools run bats tests/jq.bats`. Expected: PASS.
- [ ] **Step 5: Commit** the three files: `#211 kit: terminals.json and terminal.jq, the findings and how a terminal's
  settings and probes become checks`.

## Task 2: terminal.sh: detection, readers, bad, kept, and the checks

**Files:**

- Create: `kit/modules/bash/terminal.sh`, `tests/terminal.bats`

**Interfaces:**

- Consumes: Task 1's `terminal::checks`, `terminal::data`.
- Produces:
  - `terminal:detect` prints a key: rule 1 is the caller's (Task 4); then `TMUX` → `tmux`, `STY` → `screen`,
    `ZELLIJ` → `zellij`; `SSH_CONNECTION` or `SSH_TTY` → `ssh`; then the spec's table of variables, in its order;
    `WSL_DISTRO_NAME` → `terminal:windows-host` (Task 9; until then `unknown`); else `unknown`.
  - `terminal:timed <seconds> <command> [<arg>…]` runs the command with stdout captured, kills it after the seconds,
    and returns 124 then.
  - `terminal:settings <key>` prints the reader's object: `use "terminal/<key>"` and runs `terminal/<key>:settings`
    under `terminal:timed 1`; a reader that fails prints its reason on stdout, wrapped as `{"unreadable": "<reason>"}`;
    124 gives `{"unreadable": "it took more than a second"}`. A key with `unread` prints `null`.
  - `terminal:bad <key> <settings-json>` prints `{"<finding>": bool}`, one `jq --argjson s … -n "\$s | (<bad>)"` per
    finding; a `bad` that does not compile, or yields other than a boolean, is a bug: it errors and returns 2.
  - `terminal:kept` prints the kept list as a JSON array (`[]` with no file or no key), read with yq.
  - `terminal:keep <id>` and `terminal:unkeep <id>` write `terminal: kept:` with yq, appending `@<profile>` from a
    fresh `terminal:settings`; no duplicate; unkeep of an absent id changes nothing.
  - `terminal:checks` prints one check object per line, as `terminal::checks` gives them, from `terminal:detect`,
    `terminal:settings`, `terminal:bad`, `terminal:probes` (Task 3; until then `null`), and `terminal:kept`.

- [ ] **Step 1: Write the failing tests** in `tests/terminal.bats` (source `modules/bash/1st.sh` and `use terminal`, as
  `tests/use.bats` does for a module):
  - `TMUX=… TERM_PROGRAM=iTerm.app terminal:detect` prints `tmux`; `SSH_TTY=… TERM_PROGRAM=iTerm.app` prints `ssh`;
    with every variable unset, `unknown`.
  - `terminal:checks` with `SSH_TTY` set prints one line, `{"id":"terminal-ssh","ok":true,…}`.
  - `terminal:timed 1 sleep 5` returns 124 within 2 seconds; `terminal:timed 1 echo hi` prints `hi`.
  - every `bad` in `terminals.json` compiles: for each finding, `jq -n --argjson s '{}' "\$s | (<bad>)"` exits 0 or 5,
    never 3.
  - `KIT_HOME=$BATS_TEST_TMPDIR`: `terminal:kept` prints `[]` with no file; after `terminal:keep terminal-ssh` twice,
    `["terminal-ssh"]`; after `terminal:unkeep terminal-ssh` and again, `[]`, exit 0 both times.
- [ ] **Step 2: Run them to see them fail.** Run: `kit/bin/kit tools run bats tests/terminal.bats`. Expected: FAIL,
  no module `terminal`.
- [ ] **Step 3: Write `terminal.sh`.** Its header lists the functions above and the detection order, citing the spec.
  `terminal:timed` backgrounds the command into a scratch file (`file:scratch`), backgrounds `sleep <s>; kill <pid>`,
  waits for the command, then kills the sleeper.
- [ ] **Step 4: Run them to see them pass.** Run: `kit/bin/kit tools run bats tests/terminal.bats`. Expected: PASS.
- [ ] **Step 5: Commit** `terminal.sh` and `tests/terminal.bats`: `#211 kit: terminal.sh detects the terminal, reads
  it, and judges it with terminals.json, and keeps what the person keeps`.

## Task 3: Probes, and the hand-off that carries them

**Files:**

- Modify: `kit/modules/bash/terminal.sh`, `tests/terminal.bats`; `kit/bin/kit-tui`, `kit/bin/kit-help`,
  `kit/bin/kit-welcome`, `kit/bin/kit-flow`, `kit/bin/kit-workspace` (one line each, before kittui is run)

**Interfaces:**

- Produces:
  - `terminal:ask` writes `ESC[?u ESC[?2027$p CR 📚 ESC[6n CR ↑ ESC[6n CR ESC[2K ESC[c` to `/dev/tty` in one
    `printf`, in raw mode with echo off, reads to the first `c` (`read -d c -t 0.5`), restores the saved `stty`, and
    prints what it read; with no answer, nothing. Before restoring, it drains with `read -t 0.05` until empty.
  - `terminal:parse <reply>` prints `{"answered": true, "kitty_keyboard": bool, "grapheme_width": bool,
    "emoji_width": n, "ambiguous_width": n, "color": "truecolor"|"256"|"fewer"}`: kitty is `ESC[?<n>u` present;
    grapheme is `ESC[?2027;<p>$y` with p in 1, 2, 3; the widths are the first and second `ESC[<r>;<c>R`, c − 1;
    color from `COLORTERM` (`truecolor`, `24bit`) and `TERM` (`*256color`). An empty reply prints `{"answered": false}`.
  - `terminal:probes` prints `KIT_TERMINAL_PROBES` when it is a JSON object, else `terminal:parse "$(terminal:ask)"`
    when stdin and stdout are terminals and `/dev/tty` opens, else `null`.
  - `terminal:export-probes` exports `KIT_TERMINAL_PROBES` from `terminal:probes` when it is unset and the result is
    not `null`.

- [ ] **Step 1: Write the failing tests** in `tests/terminal.bats`:
  - `terminal:parse $'\e[?1u\e[?2027;2$y\e[5;3R\e[5;2R\e[?62;22c'` with `COLORTERM=truecolor` prints
    `{"answered":true,"kitty_keyboard":true,"grapheme_width":true,"emoji_width":2,"ambiguous_width":1,"color":"truecolor"}`.
  - `terminal:parse $'\e[5;2R\e[5;2R\e[?1;2c'` with `COLORTERM` unset and `TERM=xterm-256color`: kitty false,
    grapheme false, emoji_width 1, color `"256"`.
  - `terminal:parse ''` prints `{"answered":false}`.
  - `KIT_TERMINAL_PROBES='{"answered":false}' terminal:probes` prints it; `KIT_TERMINAL_PROBES='nope'` and
    `KIT_TERMINAL_PROBES=''` with no terminal print `null`.
  - with no terminal (bats' stdin is not one), `terminal:export-probes` leaves `KIT_TERMINAL_PROBES` unset.
  - `SSH_TTY=… KIT_TERMINAL_PROBES='<emoji_width 1>' terminal:checks` includes `terminal-probe-emoji-width` with
    `ok: false`.
- [ ] **Step 2: Run them to see them fail.** Run: `kit/bin/kit tools run bats tests/terminal.bats`. Expected: FAIL.
- [ ] **Step 3: Write the four functions,** and in each of the five launchers add `use terminal` and
  `terminal:export-probes` on the line before kittui is run.
- [ ] **Step 4: Run them to see them pass,** then `kit/bin/kit tools run bats tests/tui.bats tests/help.bats
  tests/welcome.bats tests/flow.bats tests/workspace.bats`. Expected: PASS, unchanged.
- [ ] **Step 5: Try it on a real terminal.** In iTerm2, run `bash -c 'source kit/modules/bash/1st.sh; use terminal;
  terminal:parse "$(terminal:ask)"'`. Expected: `answered: true`, `emoji_width: 2`, and no stray characters left on
  the line or typed at the next prompt. Write what it printed in the task's report.
- [ ] **Step 6: Commit** `terminal.sh`, `tests/terminal.bats`, and the five launchers: `#211 kit: kit probes the
  terminal before it hands it to kittui, and passes the answers down in KIT_TERMINAL_PROBES`.

## Task 4: kit welcome judges the terminal, keeps, and folds what holds

**Files:**

- Modify: `kit/modules/bash/welcome.sh` (`welcome:machine`, `welcome:step`, `welcome:plan-machine`, `welcome:judge`,
  `welcome:show`, header), `kit/bin/kit-welcome` (usage lines, header, option handling), `tests/welcome.bats`,
  `projects/tui/internal/syntax/testdata/welcome.json`

**Interfaces:**

- Consumes: `terminal:checks`, `terminal:keep`, `terminal:unkeep`, `terminal:probes`.
- Produces: `kit welcome --keep=<check>` and `kit welcome --unkeep=<check>`: exit 0, or 2 for an id that does not start
  `terminal-` or is a heading (`terminal-<key>`). In `--checks --json`, the terminal checks follow the forge logins,
  with `step: "setup"`, `runs: false`, `interactive: false`, folder fields null.

- [ ] **Step 1: Write the failing tests** in `tests/welcome.bats`:
  - "the terminal's checks are machine checks of setup, after the forge logins, with fixes kit never runs": with
    `SSH_TTY=x` and `KIT_TERMINAL_PROBES='{"answered":true,"kitty_keyboard":true,"grapheme_width":true,"emoji_width":1,"ambiguous_width":1,"color":"truecolor"}'`,
    `checks | jq` shows `terminal-ssh` then `terminal-probe-emoji-width` (`ok: false`, `runs: false`, `step: "setup"`)
    after the last `forge-login-*`.
  - "with no terminal and no probes, no terminal check runs": `env -u KIT_TERMINAL_PROBES` and stdout a pipe gives no
    id starting `terminal-`.
  - "--keep= keeps a terminal check, and --unkeep= undoes it": after `kit welcome --keep=terminal-probe-emoji-width`,
    that check is `ok: true` and its line starts with `kept:`; after `--unkeep=`, `ok: false` again.
  - "--keep= refuses with 2 a check that is not a terminal finding": `--keep=tool-jq` and `--keep=terminal-ssh` exit 2.
  - "the text guide shows a terminal's heading and only the findings that failed or are kept": with the probes all
    good, `kit welcome --text | under` shows `terminal-ssh`'s line and no `terminal-probe-*` line.
- [ ] **Step 2: Run them to see them fail.** Run: `kit/bin/kit tools run bats tests/welcome.bats`. Expected: FAIL.
- [ ] **Step 3: Wire it in.** `welcome:plan-machine` calls `terminal:checks` once, when `KIT_TERMINAL_PROBES` is set or
  stdout is a terminal, keeps each object in `welcome_terminal[<id>]`, and appends the ids; `welcome:judge` handles
  `terminal-*` by printing the kept object through `welcome:record` with `runs` false; `welcome:machine` and
  `welcome:step` take `terminal-*`; `welcome:show` skips a `terminal-*` check that is not a heading, holds, and whose
  line does not start `kept:`. kit-welcome's header gets the two usage lines, a paragraph on the terminal checks with
  `see: https://github.com/empowerite/kit/blob/main/docs/terminals.md`, and `--keep=`/`--unkeep=` in its options list.
- [ ] **Step 4: Run them to see them pass,** then `kit/bin/kit tools run bats tests/syntax.bats` and refresh
  `welcome.json` with the command it prints. Expected: PASS.
- [ ] **Step 5: Commit** the four files: `#211 kit: kit welcome judges the terminal in setup, keeps a finding with
  --keep=, and shows a terminal's heading with only what failed or is kept`.

## Task 5: iTerm2

**Files:**

- Create: `kit/modules/bash/terminal/iterm.sh`, `tests/fixtures/terminal/iterm-bookmarks.json`
- Modify: `kit/modules/jq/terminals.json`, `tests/terminal.bats`

**Interfaces:**

- Produces `terminal/iterm:settings`: runs `defaults export com.googlecode.iterm2 -` to a scratch file, then
  `plutil -extract "New Bookmarks" json -o - <file>` and `plutil -extract "Default Bookmark Guid" raw -o - <file>`;
  picks the profile whose `Name` is `ITERM_PROFILE`, else the default's; a missing key comes from the profile named by
  `Dynamic Profile Parent Name`, then the default below. Prints:

| Key | From | Default |
| --- | --- | --- |
| `profile` | `Name` | |
| `triggers` | count of `Triggers` entries without `disabled: true` | 0 |
| `triggers_interactive` | `Enable Triggers in Interactive Apps` | true |
| `option_left`, `option_right` | `Option Key Sends`, `Right Option Key Sends` | 0, 0 |
| `apps_change_keys` | `Allow modifyOtherKeys` | true |
| `mouse`, `mouse_clicks` | `Mouse Reporting`, `Mouse Reporting allow clicks and drags` | false, true |
| `no_alternate_screen` | `Disable Smcup Rmcup` | false |
| `unicode_version` | `Unicode Version` | 9 |
| `ambiguous_wide` | `Ambiguous Double Width` | false |
| `encoding` | `Character Encoding` | 4 |

- Rows: `triggers` `.triggers > 0 and .triggers_interactive`; `option` `.option_left != 2 and .option_right != 2`;
  `apps-change-keys` `.apps_change_keys | not`; `mouse` `(.mouse and .mouse_clicks) | not`; `alternate-screen`
  `.no_alternate_screen`; `unicode-9` `.unicode_version < 9`; `ambiguous-wide` `.ambiguous_wide`; `encoding`
  `.encoding != 4`. Each row's line and fix name the window's path from the spec's iTerm2 table, with `{profile}`.
  `probes.kitty-keyboard.fix`: the path to "Apps can change how keys are reported".

- [ ] **Step 1: Write the failing tests:** fakes `defaults` (prints nothing) and `plutil` (by its `-extract` key,
  prints the fixture or `"GUID-1"`); the fixture holds profiles "Default" (GUID-1; one enabled trigger, option 0,
  no `Mouse Reporting` key), "Work (copy)" (all good), and a dynamic "Dyn" with parent "Work (copy)" and no keys.
  - `ITERM_PROFILE='Work (copy)'` → every `bad` false.
  - `ITERM_PROFILE` unset → profile `Default`; `triggers`, `option`, `mouse` true; the rest false.
  - `ITERM_PROFILE=Dyn` → as "Work (copy)", profile `Dyn`.
  - `TERM_PROGRAM=iTerm.app terminal:checks` with the "Default" fixture: heading line `iTerm2, profile Default`, and
    `terminal-iterm-triggers` `ok: false`.
- [ ] **Step 2: Run them to see them fail.** Expected: FAIL, no module `terminal/iterm`.
- [ ] **Step 3: Write the reader and the rows.**
- [ ] **Step 4: Run them to see them pass.** Expected: PASS.
- [ ] **Step 5: Run it here.** `TERM_PROGRAM=iTerm.app` is this desk's: `kit/bin/kit welcome --checks --json | jq '.[] |
  select(.id | startswith("terminal-"))'`. Expected: `terminal-iterm-option` fails (this desk's Default sends Normal).
  Put the output in the report.
- [ ] **Step 6: Commit** the reader, fixture, table, and tests: `#211 kit: iTerm2's reader and findings`.

## Task 6: Terminal

**Files:** Create `kit/modules/bash/terminal/apple.sh`; modify `terminals.json`, `tests/terminal.bats`.

**Interfaces:**

- Produces `terminal/apple:settings`: the profile is `plutil -extract "Default Window Settings" raw`; a name with a
  `.` fails with "Terminal's profile {name} has a dot in its name, which kit cannot read". Each key with
  `plutil -extract "Window Settings.<profile>.<key>" raw -o - <file>`; exit 1 is the default. Prints `profile`,
  `option_meta` (`useOptionAsMetaKey`, false), `ambiguous_wide` (`EastAsianAmbiguousWide`, false), `encoding`
  (`CharacterEncoding`, 4).
- Rows: `option` `.option_meta | not`, its line adding "; turning it on breaks layouts such as AZERTY";
  `ambiguous-wide` `.ambiguous_wide`; `encoding` `.encoding != 4`. The heading says it judged the default profile.
  `probes.kitty-keyboard.none`: "Terminal has no kitty keyboard protocol, so kittui offers > and < for ⌥ with an arrow".

- [ ] **Step 1: Verify the keys by hand on this Mac.** For each of the three settings, set it both ways in Terminal →
  Settings → Profiles → (the default), and after each, run `defaults export com.apple.Terminal - | plutil -extract
  "Window Settings.<profile>.<key>" raw -o - -`. Record the values seen in the report. A key that does not change is
  dropped to docs only, and the report says so.
- [ ] **Step 2: Write the failing tests** with fake `defaults`/`plutil`: a profile "Basic" with every key absent
  (option fails), and a default named "Pro.Dark" (the heading holds and says kit cannot read it).
- [ ] **Step 3: Run them to see them fail.** Expected: FAIL.
- [ ] **Step 4: Write the reader and the rows.**
- [ ] **Step 5: Run them to see them pass.** Expected: PASS.
- [ ] **Step 6: Commit:** `#211 kit: Terminal's reader and findings, its keys read back from a profile set both ways`.

## Task 7: Ghostty and kitty

**Files:** Create `kit/modules/bash/terminal/ghostty.sh`, `kit/modules/bash/terminal/kitty.sh`; modify
`terminals.json`, `tests/terminal.bats`.

**Interfaces:**

- `terminal/ghostty:settings`: `"${GHOSTTY_BIN_DIR:-}/ghostty" +show-config` (else `ghostty` on PATH), `key = value`
  lines, the last of a key winning; plus `defaults read com.apple.HIToolbox AppleCurrentKeyboardLayoutInputSourceID`
  on macOS. Prints `option` (`macos-option-as-alt` or null), `layout_us` (the source is `com.apple.keylayout.US` or
  `com.apple.keylayout.USInternational-PC`), `alt_arrows_legacy` (false only when both `alt+left` and `alt+right`
  appear in a `keybind` line), `mouse` (`mouse-reporting`, true), `grapheme` (`grapheme-width-method`, `unicode`).
  Rows: `option` `(.option // (if .layout_us then "true" else "false" end)) == "false"`; `alt-arrows`
  `.alt_arrows_legacy`, fix `keybind = alt+left=unbind` and `keybind = alt+right=unbind`; `mouse` `.mouse | not`;
  `grapheme-width` `.grapheme == "legacy"`.
- `terminal/kitty:settings`: the file `<cache>/effective-config/${KITTY_PID}` (cache as the spec says); fails "kitty
  wrote no effective config for this window" when absent. Prints `option` (last `macos_option_as_alt`, `no` on macOS,
  null elsewhere) and `grabbed_left` (a line `mouse_map left press grabbed …`, the modifier exactly `left`). Rows:
  `option` `.option == "no"`, its line noting kitty must be restarted; `mouse` `.grabbed_left`.

- [ ] **Step 1: Write the failing tests:** a fake `ghostty` printing `mouse-reporting = false` and a fake `defaults`
  printing `com.apple.keylayout.French`: `option`, `alt-arrows`, `mouse` fail. A second fake printing
  `macos-option-as-alt = left`, both unbinds: all hold. A fake `ghostty` that sleeps 5: the heading says it took more
  than a second. kitty: a fixture effective-config with `macos_option_as_alt no` then `macos_option_as_alt left` holds;
  with `mouse_map left press grabbed mouse_selection normal` fails `mouse`; `mouse_map shift+left press grabbed …`
  holds.
- [ ] **Step 2: Run them to see them fail.** Expected: FAIL.
- [ ] **Step 3: Write both readers and their rows.**
- [ ] **Step 4: Run them to see them pass.** Expected: PASS.
- [ ] **Step 5: Commit:** `#211 kit: Ghostty's and kitty's readers and findings`.

## Task 8: WezTerm, Alacritty, and Warp

**Files:** Create `kit/modules/bash/terminal/{wezterm,alacritty,warp}.sh`,
`tests/fixtures/terminal/wezterm-show-keys.txt`; modify `terminals.json`, `tests/terminal.bats`.

**Interfaces:**

- `terminal/wezterm:settings`: `wezterm show-keys`; prints `alt_enter_fullscreen` (a key line for `Enter` with `ALT`
  bound to `ToggleFullScreen`) and `mouse_reporting_bound` (a mouse binding with `mouse_reporting=true`), and `config`
  (`WEZTERM_CONFIG_FILE`). Rows: `alt-enter` `.alt_enter_fullscreen`; `mouse` `.mouse_reporting_bound`.
  `probes.kitty-keyboard.fix`: "add config.enable_kitty_keyboard = true to {config}".
- `terminal/alacritty:settings`: the first of the spec's search path; an `awk` read of `option_as_alt` under
  `[window]`, or `window.option_as_alt` at the top. Prints `option` (`"None"` on macOS, null elsewhere). Row: `option`
  `.option == "None"`.
- `terminal/warp:settings`: `~/.warp/settings.toml` on macOS, `${XDG_CONFIG_HOME:-~/.config}/warp-terminal/settings.toml`
  elsewhere; an `awk` read of `extra_meta_keys` under `[terminal.input]` and `mouse_reporting_enabled` under
  `[terminal]`. Prints `left_meta`, `right_meta` (false), `mouse` (true). Rows: `option`
  `(.left_meta or .right_meta) | not`; `mouse` `.mouse | not`, its line noting View → Toggle Mouse Reporting.

- [ ] **Step 1: Get a real `wezterm show-keys` output** for the fixture, from WezTerm's own docs or test data, and say
  in the fixture's first line where it came from; add `ALT Enter → ToggleFullScreen` as WezTerm ships it.
- [ ] **Step 2: Write the failing tests:** wezterm from the fixture (alt-enter fails; no mouse finding); alacritty
  fixtures `[window]\noption_as_alt = "OnlyLeft"` (holds), none (fails on `OSTYPE=darwin*`), and the dotted form;
  warp fixtures with `extra_meta_keys = { left_alt = true, right_alt = false }` (holds) and
  `mouse_reporting_enabled = false` (fails).
- [ ] **Step 3: Run them to see them fail.** Expected: FAIL.
- [ ] **Step 4: Write the three readers and their rows.**
- [ ] **Step 5: Run them to see them pass.** Expected: PASS.
- [ ] **Step 6: Commit:** `#211 kit: WezTerm's, Alacritty's, and Warp's readers and findings`.

## Task 9: Windows Terminal, the Windows console, and Tabby

**Files:** Create `kit/modules/bash/terminal/{wt,console,tabby}.sh`, `tests/fixtures/terminal/wt-settings.jsonc`;
modify `kit/modules/bash/terminal.sh` (`terminal:windows-host`), `terminals.json`, `tests/terminal.bats`.

**Interfaces:**

- `terminal:windows-host` prints `wt` or `console` from `reg.exe query 'HKCU\Console\%%Startup' /v DelegationTerminal`:
  `{B23D10C0-E52E-411E-9D5B-C09FDF709C7D}` → console; `{2EACA947-…}` or `{E12CFF52-…}` → wt; absent or all zeros →
  wt on Windows 11 (build 22000 or later, from `cmd.exe /c ver`), else console; `reg.exe` failing → `unknown`.
  `terminal:detect` uses it for `WSL_DISTRO_NAME` without `WT_SESSION`.
- `terminal/wt:jsonc` strips `//` and `/* */` comments outside strings and trailing commas before `}` or `]`, in
  `awk`, aware of `"…"` with `\"` escapes.
- `terminal/wt:settings`: `%LOCALAPPDATA%` from `cmd.exe /c echo %LOCALAPPDATA%` run in `/mnt/c`, `tr -d '\r'`,
  `wslpath -u`; the first of Store, Preview, unpackaged; the profile `WT_PROFILE_ID` (case-insensitive) else
  `defaultProfile`; each setting from the profile, then `profiles.defaults`, then the default. Prints `profile`
  (its `name`), `kitty_keyboard` (`compatibility.kittyKeyboardMode`, null), `force_vt`
  (`compatibility.input.forceVT`, false), `text_measurement` (global `compatibility.textMeasurement`, `graphemes`),
  `software` (`rendering.software` or `experimental.rendering.software`, false), `full_repaint`
  (`experimental.rendering.forceFullRepaint`, false). Rows: `kitty-keyboard` `.kitty_keyboard == false`; `force-vt`
  `.force_vt`; `text-measurement` `.text_measurement == "console"`; `software-rendering` `.software`; `full-repaint`
  `.full_repaint`. `probes.kitty-keyboard.fix`: the setting, "and Windows Terminal 1.25.2733 or later".
- `terminal/console:settings`: `reg.exe query 'HKCU\Console' /v TextMeasurement`; prints `text_measurement` (0). Rows:
  `console` `true`, line "the Windows console is slow to repaint and has no color emoji or kitty keyboard protocol",
  fix "use Windows Terminal: Settings → Startup → Default terminal application"; `text-measurement`
  `.text_measurement == 2`. The heading notes a shortcut's own properties override the registry.
- `terminal/tabby:settings`: `${TABBY_CONFIG_DIRECTORY:-<userData>/tabby}/config.yaml` with yq; `encrypted: true`
  fails "Tabby's config is encrypted". Prints `alt_meta` (`terminal.altIsMeta`, false on macOS, null elsewhere),
  `frontend` (`terminal.frontend`, `xterm-webgl`), `disable_gpu` (`hacks.disableGPU`, false). Rows: `option`
  `.alt_meta == false`; `renderer` `.frontend == "xterm" or .disable_gpu`. `probes.kitty-keyboard.none`: "Tabby has no
  kitty keyboard protocol, so kittui offers > and < for ⌥ with an arrow". The spec's `conpty` row is docs only:
  Tabby is detected only by `TERM_PROGRAM`, which it never sets inside WSL, so no reader could see it.

- [ ] **Step 1: Write the failing tests** with fake `reg.exe`, `cmd.exe`, and `wslpath`: delegation to conhost →
  `console`; zeros with `ver` 10.0.22631 → `wt`; with 10.0.19045 → `console`; `reg.exe` exiting 1 → `unknown`. The
  JSONC fixture has a `"https://…"` value, a `/* … */` block, a trailing comma, a profile `{2C4DE342-…}` with
  `kittyKeyboardMode: false`, and `profiles.defaults.compatibility.input.forceVT: true`: `WT_PROFILE_ID` in lower case
  finds it; `kitty-keyboard` and `force-vt` fail; `text-measurement` holds. console with `TextMeasurement 0x2` fails
  both rows. tabby with `encrypted: true` holds with the reason; with `terminal: {frontend: xterm}` fails `renderer`.
- [ ] **Step 2: Run them to see them fail.** Expected: FAIL.
- [ ] **Step 3: Write the readers, `terminal:windows-host`, and the rows.**
- [ ] **Step 4: Run them to see them pass.** Expected: PASS.
- [ ] **Step 5: Commit:** `#211 kit: Windows Terminal's, the Windows console's, and Tabby's readers and findings`.

## Task 10: kittui folds what holds and offers keep

**Files:**

- Modify: `projects/tui/internal/kit/kit.go`, `projects/tui/internal/welcome/welcome.go`,
  `projects/tui/internal/welcome/view.go`, `projects/tui/internal/welcome/setup.go`, their tests, and goldens

**Interfaces:**

- Consumes: Task 4's `kit welcome --keep=<check>`.
- Produces:
  - `func (r Runner) WelcomeKeep(ctx context.Context, c Check) error` runs `kit welcome --keep=<c.ID>` over the
    workspace (`r.over`), as the runner's other short calls run.
  - `func terminalFinding(c kit.Check) bool`: `c.ID` starts `terminal-` and has a second `-`.
  - `func folded(c kit.Check) bool`: `terminalFinding(c) && c.OK && !strings.HasPrefix(c.Line, "kept:")`; a folded
    check is drawn on no row and the cursor never lands on it.
  - A choice `keep` beside `later`: a failed `terminalFinding` row draws `later` and `keep` buttons (no `now`); `k`
    or a plain click on keep runs `WelcomeKeep`, then reads the checks again, as a fix does; `l` and later as today.

- [ ] **Step 1: Write the failing tests** in `internal/welcome/welcome_test.go` and `internal/kit/kit_test.go`:
  - `TestAHeldTerminalFindingIsFoldedUnderItsHeading`: checks `terminal-iterm` (ok), `terminal-iterm-triggers` (ok),
    `terminal-iterm-option` (failed): the view has the heading and the option row, not the triggers row.
  - `TestAKeptTerminalFindingStays`: `terminal-iterm-triggers` ok with line `kept: …` is drawn.
  - `TestAFailedTerminalFindingOffersLaterAndKeep`: its row shows `later` and `keep`, not `now`; `n` does nothing.
  - `TestKeepRunsKitWelcomeKeepAndReadsTheChecksAgain`: with the test runner's fake kit, `k` records
    `welcome --keep=terminal-iterm-option` and a second `welcome --checks --json`.
  - `TestAClickOnKeepKeeps`: a plain click on the keep button does what `k` does.
  - `TestWelcomeKeepRunsKitWelcomeKeep` in `kit_test.go`: the args, and an error on exit 2.
- [ ] **Step 2: Run them to see them fail.** Run: `cd projects/tui && go test ./internal/welcome/ ./internal/kit/`.
  Expected: FAIL.
- [ ] **Step 3: Write it.** The package comment at the top of `welcome.go` says the keep choice and the folding.
- [ ] **Step 4: Run every Go test and the coverage.** Run: `cd projects/tui && go test ./... -cover`. Expected: PASS,
  coverage 98.0 or more; refresh a golden only when no peer has work under `projects/tui/`.
- [ ] **Step 5: Commit** the Go files: `#211 kittui: welcome folds a terminal's findings that hold under its heading,
  and offers keep beside later`.

## Task 11: docs/terminals.md

**Files:**

- Create: `docs/terminals.md`
- Modify: `README.md` (one line where it lists the docs), `tests/terminal.bats`

**Interfaces:**

- Consumes: every `anchor` and finding in `terminals.json`.
- Produces: one `##` section, `<Name>`, per terminal key, whose GitHub slug is its `anchor` (`## iTerm2` → `iterm2`,
  `## Over ssh` → `over-ssh`), each with how kit detects it, where kit reads it, a table of its findings (setting,
  bad value, default, need, fix) from the spec, and its "Docs only" notes; an opening section on kittui's five needs;
  and a section on keeping a finding.

- [ ] **Step 1: Write the failing test** in `tests/terminal.bats`: "every terminal in terminals.json has its section in
  docs/terminals.md": each `anchor` equals the slug (lower case, spaces to `-`, other punctuation dropped) of a
  `##` heading, and each finding id appears in that section.
- [ ] **Step 2: Run it to see it fail.** Expected: FAIL, no file.
- [ ] **Step 3: Write the page** from the spec's tables, citing each terminal's source as the spec's research did.
- [ ] **Step 4: Run it and `kit lint docs/terminals.md README.md`.** Expected: PASS.
- [ ] **Step 5: Commit:** `#211 docs: terminals.md, every terminal's findings, the page each check's more opens`.
