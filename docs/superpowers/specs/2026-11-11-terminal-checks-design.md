# Terminal checks: design

Status: designed, 2026-11-11. Issue #211.

## Intent

kittui is a full-screen program, and the terminal around it can make it worse without anything in kit being wrong. A
trigger in iTerm2 runs its regex against every line kittui repaints: one prompt-detection trigger took 399 ms on a
single match, and iTerm2 itself offered to turn triggers off in interactive apps. iTerm2's default profile sends ⌥ as
a plain key, so ⌥ with an arrow never reaches kittui as one. The Windows console draws in a renderer too slow for
full-screen repaints. Each of these is one setting the person can change in a minute, once they know it exists.

Welcome's setup step already tells a newcomer what to fix on their machine. It also tells them when the terminal they
are sitting in will make kittui work worse, names the exact setting to change, and does so before kittui misbehaves in
front of them.

## Terms

- **Terminal**: the program that draws kittui's screen: iTerm2, Terminal, Ghostty, kitty, WezTerm, Alacritty, Warp,
  Tabby, Windows Terminal, or the Windows console. A shell (zsh, bash, PowerShell, Command Prompt) is not a terminal.
- **Multiplexer**: a program between the terminal and kittui that is a terminal to kittui in turn: tmux, GNU screen,
  or zellij.
- **Need**: one thing kittui asks of a terminal, listed below.
- **Finding**: one way one terminal breaks a need, with its fix. A finding is one check.
- **Setting finding**: a finding judged from the terminal's settings, read where the terminal keeps them.
- **Probe finding**: a finding judged from what the terminal does, by asking it: a query it answers, or a glyph
  printed and the cursor's column read back.
- **Kept**: a finding the person has chosen to keep as it is.

## What kittui needs from a terminal

kittui draws on the alternate screen, turns the mouse on in cell-motion mode, asks for the kitty keyboard protocol and
uses it where the terminal answers, and draws its menus with two-cell emoji (`projects/tui/internal/shell/view.go`,
`tea.KeyboardEnhancementsMsg` in `internal/app/app.go`, and the README's notes on ⌥ and the right button). A terminal
must give it:

1. Speed: repaints of the alternate screen that nothing in the terminal stalls.
2. Clicks: a plain left click reaches kittui, and a drag or a scroll does too.
3. Keys: ⌥ (Alt) arrives as Meta; ⇧⌫ is not ⌫; esc arrives at once; and, where the terminal can, the kitty keyboard
   protocol, so ⌥ with an arrow is one key.
4. Widths: a two-cell emoji takes two cells, and box drawing and arrows take one, so every column stays in place.
5. Color: 24-bit color, or at least 256 colors, and `COLORTERM` saying which.

Anything a terminal does that breaks one of these is in scope. Anything that does not, such as a theme or a font, is
not.

## Goals

- In welcome's setup step, a machine check for each finding of the terminal kit is running in, with the fix the person
  makes: the menu path, or the line in its config file.
- A finding fails only on what kit read or measured, never on a guess.
- One page in the docs, `docs/terminals.md`, that lists every known finding for every terminal, readable on its own
  by someone choosing or setting up a terminal, and the target of each check's `more`.
- Adding a terminal or a finding is a row in a data file, and a reader for a new terminal's settings, not a change to
  welcome.
- A finding the person keeps on purpose stops failing, and stays visible as kept.

## Non-goals

- Changing a terminal's settings. Every terminal check's fix is shown and never run (`runs: false`), as brew-path's
  is. Terminals rewrite their own settings (iTerm2 saves its preferences as they change in its window, and may write a
  custom folder on quit), and a terminal kit broke is worse than one that is slow.
- Judging a terminal kit does not run in. The checks describe the terminal around this run of kit welcome; another
  terminal on the same machine is judged when welcome runs there.
- Reading a terminal's settings across ssh. They are on the other machine; probes still run.
- PowerShell and Command Prompt as places to run kit. kit runs on Windows inside WSL (README); the window around WSL
  is Windows Terminal, Tabby, or the Windows console, and that window is what kit judges.
- Fonts, themes, and anything else that is taste rather than a need.

## Anti-goals

- A check that fails on a guess. A setting finding fails only on a value kit read; where kit cannot read a terminal's
  settings, one passing check says so and points at the docs. A finding the research marks as inferred is in the docs,
  and is not a check until it is verified.
- Nagging. A kept finding never fails again, and a terminal with nothing wrong adds one passing line.
- Breaking welcome on a terminal kit does not know. An unknown terminal gets the probes and nothing else.
- A slow welcome. Each terminal's settings are one read, of one file or one command, with a timeout of a second. The
  Windows process tree, which takes about a second to walk, is never walked.
- Asking the person for anything to judge their terminal. Terminal's current profile can be read only through
  AppleScript, which raises an Automation prompt; kit judges Terminal's default profile instead, and says so.

## Constraints

- kit is bash, and reads JSON with jq and YAML with yq. On macOS, `defaults export <domain> -` gives the live
  preferences, and `plutil -extract <keypath> json -o - -` takes one key of it as JSON. Converting a whole export to
  JSON fails on its date and data values, so a reader extracts the keys it needs.
- kittui owns the terminal while it is up, and runs `kit welcome --checks --json` as a child. A child that wrote a
  probe to the terminal and read its answer would race kittui's own reader. So kit probes before it hands the
  terminal to kittui, and the answers reach every child through the environment (below).
- A terminal check is a machine check, judged once, in `welcome:plan-machine`; `welcome:machine` says so for
  `terminal-*`, and `welcome:step` puts it in setup.
- `kit welcome --checks --json` keeps its shape: a terminal check is one more object with the same fields.
- Without a terminal (CI, or a pipe), no terminal check runs.

## The design

### Detecting the terminal

kit names the terminal from the environment, and stops at the first rule that answers:

1. No terminal: stdout is not a terminal and kittui passed no probes. No terminal check runs.
2. A multiplexer: `TMUX`, `STY`, or `ZELLIJ`. Inside one, the outer terminal's variables are those of whichever
   terminal started the multiplexer's server, and may be stale, and tmux sets `TERM_PROGRAM=tmux`. kit judges the
   multiplexer (its findings are left for later, below), runs the probes, which pass through it to whatever answers,
   and does not read an outer terminal's settings.
3. Over ssh: `SSH_CONNECTION` or `SSH_TTY`. One passing check says kit cannot read the terminal's settings from here;
   the probes still run.
4. The terminal's own variables, in the table below.
5. Inside WSL (`WSL_DISTRO_NAME`) with none of those: Windows hands a console session to its default terminal
   without setting `WT_SESSION` (microsoft/terminal#13006), and Tabby forwards none of its variables into WSL. kit
   reads the default terminal from `reg.exe query 'HKCU\Console\%%Startup'` (`DelegationTerminal`): the Windows
   console's GUID names the console; Windows Terminal's names Windows Terminal, without a profile; absent or zero
   names Windows Terminal on Windows 11 and the console on Windows 10. Where interop is off and `reg.exe` cannot run,
   the terminal is unknown.
6. Otherwise, unknown: the probes alone.

| Terminal | Detected by | Profile |
| --- | --- | --- |
| iTerm2 | `TERM_PROGRAM=iTerm.app` | `ITERM_PROFILE`, a profile's `Name`; else `Default Bookmark Guid` |
| Terminal | `TERM_PROGRAM=Apple_Terminal` | `Default Window Settings` (no variable names the tab's) |
| Ghostty | `TERM_PROGRAM=ghostty` | none |
| kitty | `KITTY_PID`, or `TERM=xterm-kitty` | none |
| WezTerm | `TERM_PROGRAM=WezTerm` | none |
| Alacritty | `ALACRITTY_WINDOW_ID`, or `TERM=alacritty` | none |
| Warp | `TERM_PROGRAM=WarpTerminal` | none |
| Tabby | `TERM_PROGRAM=Tabby` (macOS, Linux) | none |
| Windows Terminal | `WT_SESSION`, or rule 5 | `WT_PROFILE_ID`, a profile's `guid`, case-insensitive; else `defaultProfile` |
| Windows console | rule 5 | none |

### Reading the settings

Each terminal kit can read has one reader, `terminal:<name>:settings`, which prints the settings its findings need as
one flat JSON object, defaults filled in, or fails when it cannot read them. A key a terminal does not write means its
default, and the default comes from the terminal's source, not its factory profile: iTerm2 counts an absent
`Mouse Reporting` as off, though its factory profile has it on.

| Terminal | Where | How |
| --- | --- | --- |
| iTerm2 | `defaults export com.googlecode.iterm2 -` | `plutil -extract "New Bookmarks" json`, then jq; dynamic profiles are in the array, and a missing key comes from `Dynamic Profile Parent Name` |
| Terminal | `defaults export com.apple.Terminal -` | `plutil -extract 'Window Settings.<name>.<key>' raw` a key at a time; exit 1 is the default; a profile name with a `.` cannot be read |
| Ghostty | `"${GHOSTTY_BIN_DIR}/ghostty" +show-config` | prints only what differs from the defaults; `key = value` lines |
| kitty | `<cache>/effective-config/${KITTY_PID}` | the effective config the running kitty wrote, includes resolved, later lines winning; the cache is `KITTY_CACHE_DIRECTORY`, else `~/Library/Caches/kitty` on macOS, else `${XDG_CACHE_HOME:-~/.cache}/kitty` |
| WezTerm | `wezterm show-keys` for bindings; `WEZTERM_CONFIG_FILE` for the rest | the config is Lua, and nothing prints its effective values; kit reads only the bindings, and the settings below are probe findings or docs |
| Alacritty | the first `alacritty.toml` on its search path | a narrow read of `[window] option_as_alt`; nothing prints the effective config, and `alacritty msg config` changes it, so kit never runs it |
| Warp | `~/.warp/settings.toml` (macOS) | a narrow read of two keys; Warp is closed source, and its runtime toggles are not in the file |
| Tabby | `${TABBY_CONFIG_DIRECTORY:-<userData>/tabby}/config.yaml` | yq; only non-defaults are written; `encrypted: true` means it cannot be read |
| Windows Terminal | `settings.json` under `%LOCALAPPDATA%`: Store, then Preview, then unpackaged | the path from `cmd.exe /c echo %LOCALAPPDATA%` run from `/mnt/c`, then `wslpath -u`; JSONC, comments and trailing commas stripped before jq; a profile's setting falls back to `profiles.defaults`, then to the default kit holds |
| Windows console | `reg.exe query 'HKCU\Console'` | the root key alone; a shortcut's and an executable's own properties override it and cannot be read, so the line says so |

### Probing the terminal

A probe asks the terminal what it does, and needs no settings file, so it runs in every terminal, known or not, over
ssh and inside a multiplexer.

| Probe | Asks | Need |
| --- | --- | --- |
| `kitty-keyboard` | `CSI ? u`: does it answer the kitty keyboard protocol | keys |
| `grapheme-width` | DECRQM 2027: does it measure by grapheme cluster | widths |
| `emoji-width` | prints 📚, then `CSI 6n`: is the cursor two columns on | widths |
| `ambiguous-width` | prints ↑, then `CSI 6n`: is the cursor one column on | widths |
| `color` | `COLORTERM` and `TERM`, as kittui's library reads them | color |

Every verb that hands the terminal to kittui first runs the probes, in bash, on `/dev/tty`, and exports the answers
in `KIT_TERMINAL_PROBES`, a JSON object, so kittui and each `kit welcome --checks` it runs as a child read them from the
environment and never touch the terminal. When kit welcome runs as text with stdout a terminal, it probes the same way.
The probes go out in one write, with a request for the terminal's primary attributes (`CSI c`) last: every terminal
answers that one, so its answer ends the exchange, and half a second without it means the terminal did not answer. A
terminal that never answers fails no probe, and its probe checks say it did not answer. With neither the variable nor
a terminal, no probe check runs.

A probe fails on these answers, and on no others (my call):

- `kitty-keyboard`: no answer to `CSI ? u`.
- `emoji-width`: the cursor is not two columns on. `grapheme-width` never fails alone: many terminals answer no to
  mode 2027 and still draw 📚 in two cells, so its answer only explains an `emoji-width` that failed.
- `ambiguous-width`: the cursor is not one column on.
- `color`: fewer than 256 colors. kit reads the variables kittui's library reads: `COLORTERM` of `truecolor` or
  `24bit` is 24-bit, a `TERM` ending in `256color` is 256, and anything else is fewer.

Settings explain, and probes measure, so they never both report one cause (my call):

- A probe is judged only for a need where the terminal's settings were not read, or where every setting finding for
  that need held. A setting finding that fails (iTerm2's `unicode-9`) says why the width probe would fail, and the
  probe adds nothing.
- A probe that fails where every setting finding for its need held says what kittui measured and that no setting kit
  knows explains it, and points at the docs.
- A probe that fails where no setting of the terminal can fix it, as for Terminal and Tabby, which have no kitty
  keyboard protocol, holds, and its line says what kittui does instead: "Terminal has no kitty keyboard protocol, so
  kittui offers > and < for ⌥ with an arrow". It is a fact about the terminal, not a setting to change, and failing
  on it would fail every run.
- A probe whose fix is a setting kit did not read, as WezTerm's `enable_kitty_keyboard`, fails, with that setting as
  its fix.

### Check ids

- `terminal-<terminal>`: always present when a terminal is detected; holds; its line names the terminal and its
  profile ("iTerm2, profile Default"), or says kit cannot read its settings and why: ssh, a multiplexer, Tabby's
  encrypted config, interop off. `<terminal>` is the table's key: `iterm`, `apple`, `ghostty`, `kitty`, `wezterm`,
  `alacritty`, `warp`, `tabby`, `wt`, `console`, `tmux`, `screen`, `zellij`, `ssh`, or `unknown`.
- `terminal-<terminal>-<finding>`: a setting finding, such as `terminal-iterm-triggers`.
- `terminal-probe-<probe>`: a probe finding, such as `terminal-probe-emoji-width`, the same id in every terminal.

### The findings table

`kit/modules/jq/terminals.json` holds every terminal and every finding as data:

```json
{
  "iterm": {
    "name": "iTerm2",
    "findings": [
      {
        "id": "triggers",
        "need": "speed",
        "from": "settings",
        "bad": "(.triggers | any(.disabled | not)) and .triggers_in_interactive_apps",
        "line": "iTerm2 runs the triggers of profile {profile} on every line kittui repaints",
        "fix": "iTerm2 → Settings → Profiles → {profile} → Advanced → Triggers: untick Enabled in interactive apps"
      }
    ]
  }
}
```

`from` is `settings` or `probes`. `bad` is a jq expression over the reader's object, or over the probes' object; true
fails the check. `{profile}` and other names in braces come from the reader's object. Each check's `more` is the
terminal's section of the docs page, by its full URL, as welcome links kit's other docs:
`https://github.com/empowerite/kit/blob/main/docs/terminals.md#iterm2`.

### Keeping a finding

`kit welcome --keep=<check>` adds the check's id to `terminal: kept:` in `${KIT_HOME:-$HOME}/.kit/settings.yml`, the
person's own settings file, where `kit toolchains` records the system toolchain. Where the terminal has profiles, the
keep is for the profile it was judged in, written `terminal-iterm-triggers@Work`, so keeping triggers in one profile
never silences another (my call). A kept finding is judged as before;
where it would fail, it holds instead, its line opens with "kept:", and its `more` names
`kit welcome --unkeep=<check>`. In kittui, a failed terminal check offers later and keep; it offers no now, since kit
never runs its fix, which stands under it. `--keep=` refuses a check that is not a terminal check, with exit 2.

### What welcome shows

The terminal checks are machine checks of the setup step, after the forge logins. `--checks --json` lists every
one, held or failed, as it lists every other check: `terminal-<terminal>` first, then each setting finding, then each
probe judged. kittui and the text guide draw `terminal-<terminal>` as the heading, and under it only the findings that
failed or are kept; where none did, the heading alone, "iTerm2, profile Default: nothing in it slows or breaks
kittui" (my call).

### Two keys for ⌥

Where a terminal sets the left and right ⌥ apart (iTerm2, Ghostty, Alacritty, Warp), the `option` finding fails only
when neither sends Meta or Esc+. Many people on a layout other than U.S. keep one ⌥ for typing characters, on purpose
(my call). Ghostty's default depends on the keyboard layout, so its reader also reads the current input source,
`defaults read com.apple.HIToolbox AppleCurrentKeyboardLayoutInputSourceID`; that is the one second read the design
allows.

## The terminals and their findings

The tables list what the research of 2026-11-11 found, from each terminal's source where it is open and its docs where
it is not (cited in `docs/terminals.md`, which the builder writes from these tables). Every row in a table is a check,
whether verified against source or against the terminal's own docs; what is under "Docs only" goes in the docs and is
not a check. "Default" is the terminal's own default.

The first cut builds a reader for all ten terminals, and the probes (the person's choice): Warp, WezTerm, and
Alacritty, whose rows rest on their docs, get readers as the others do.

### iTerm2

Verified against iTerm2's source at c3560d2f (3.7.7).

| Finding | Setting, as the window names it | Key: bad value | Default | Need |
| --- | --- | --- | --- | --- |
| `triggers` | Profiles → Advanced → Triggers → Enabled in interactive apps | `Enable Triggers in Interactive Apps`: on, with an enabled trigger | on | speed |
| `option` | Profiles → Keys → General → Left Option key, Right Option key | `Option Key Sends`, `Right Option Key Sends`: 0 (Normal) or 1 (Meta); Esc+ is 2 | 0 | keys |
| `apps-change-keys` | Profiles → Keys → General → Apps can change how keys are reported | `Allow modifyOtherKeys`: off | on | keys |
| `mouse` | Profiles → Terminal → Enable mouse reporting, and Report mouse clicks & drags | `Mouse Reporting`: off or absent; `Mouse Reporting allow clicks and drags`: off | off; on | clicks |
| `alternate-screen` | Profiles → Terminal → Disable save/restore alternate screen | `Disable Smcup Rmcup`: on | off | speed |
| `unicode-9` | Profiles → Text → Use Unicode version 9+ widths | `Unicode Version`: 8 | 9 | widths |
| `ambiguous-wide` | Profiles → Text → Ambiguous characters are double-width | `Ambiguous Double Width`: on | off | widths |
| `encoding` | Profiles → Terminal → Character encoding | `Character Encoding`: not 4 (UTF-8) | 4 | widths |

Docs only: ligatures and transparency turn off the GPU renderer, which is likely slower; an instant trigger
(`partial`) costs more than a plain one; a session's own edits (Session → Edit Session) override the profile and are
not in the preferences; iTerm2 keeps ⌥, ⌃, and the right button with a click for itself.

### Terminal

Terminal is closed source. Its keys come from the strings in the app (2.15, build 488.9), so these rows are verified
only once the builder has read each key from a profile set both ways.

| Finding | Setting | Key: bad value | Default | Need |
| --- | --- | --- | --- | --- |
| `option` | Profiles → Keyboard → Use Option as Meta key | `useOptionAsMetaKey`: off | off | keys |
| `ambiguous-wide` | Profiles → Advanced → Unicode East Asian Ambiguous characters are wide | `EastAsianAmbiguousWide`: on | off | widths |
| `encoding` | Profiles → Advanced → Text encoding | `CharacterEncoding`: not UTF-8 | UTF-8 | widths |

Docs only: Use Option as Meta key breaks keyboard layouts such as AZERTY, so its line says so; View → Allow Mouse
Reporting is per window and not in the preferences; Terminal has no kitty keyboard protocol (the probe says so); 24-bit
color depends on the macOS release (the color probe says so).

### Ghostty

Verified against Ghostty's `src/config/Config.zig`.

| Finding | Setting | Bad value | Default | Need |
| --- | --- | --- | --- | --- |
| `option` | `macos-option-as-alt` | `false`, or unset on a keyboard layout other than U.S. | `true` on U.S. layouts, else `false` | keys |
| `alt-arrows` | `keybind = alt+left=…`, `alt+right=…` | the default bindings to `esc:b` and `esc:f`, which turn ⌥← and ⌥→ into ⌥b and ⌥f | bound | keys |
| `mouse` | `mouse-reporting` | `false` | `true` | clicks |
| `grapheme-width` | `grapheme-width-method` | `legacy` | `unicode` | widths |

Docs only: `shell-integration-features` without `ssh-terminfo` and `sudo` leaves `TERM=xterm-ghostty` on hosts
without its terminfo; the `toggle_mouse_reporting` action turns reporting off at run time where no setting shows it;
`custom-shader` and `background-blur` cost frames.

### kitty

Verified against kitty's `kitty/options/definitions.py`.

| Finding | Setting | Bad value | Default | Need |
| --- | --- | --- | --- | --- |
| `option` | `macos_option_as_alt` | `no` | `no` | keys |
| `mouse` | `mouse_map left press grabbed …` | any plain-left mapping in `grabbed` mode | none | clicks |

Docs only: `repaint_delay` (10 ms) and `input_delay` (3 ms) well above their defaults slow repaints; plain `ssh`
leaves `TERM=xterm-kitty` on hosts without its terminfo, which `kitten ssh` fixes; `macos_option_as_alt` needs kitty
restarted, not reloaded.

### WezTerm

Verified against WezTerm's docs; its config is Lua, so only bindings are read.

| Finding | From | Bad | Default | Need |
| --- | --- | --- | --- | --- |
| `kitty-keyboard` | probe; fix `config.enable_kitty_keyboard = true` | no answer to `CSI ? u` | off | keys |
| `mouse` | `wezterm show-keys`: a mouse binding with `mouse_reporting = true` | any | none | clicks |
| `alt-enter` | `wezterm show-keys`: ⌥⏎ bound to ToggleFullScreen | bound | bound | keys |

Docs only: `unicode_version` below 14 draws ✔️ and the like in one cell; `send_composed_key_when_left_alt_is_pressed`
set to `true`; `max_fps` below 60 and `front_end = "Software"`; `term = "wezterm"` needs the terminfo on every host.

### Alacritty

Verified against Alacritty's docs.

| Finding | Setting | Bad value | Default | Need |
| --- | --- | --- | --- | --- |
| `option` | `[window] option_as_alt` | `"None"` | `"None"` | keys |

Docs only: a leftover `alacritty.yml` is not read since 0.13; `TERM=alacritty` needs the terminfo on remote hosts.

### Warp

Closed source; from Warp's settings docs.

| Finding | Setting | Bad value | Default | Need |
| --- | --- | --- | --- | --- |
| `option` | `[terminal.input] extra_meta_keys` | `left_alt = false` | `false` | keys |
| `mouse` | `[terminal] mouse_reporting_enabled` | `false` | `true` | clicks |

Docs only: View → Toggle Mouse Reporting changes it at run time where the file does not show it; Warp's docs suggest
turning off custom padding in the alternate screen where full-screen apps draw badly; shift with a click is Warp's.

### Tabby

From Tabby's source; on Windows inside WSL it cannot be told from the Windows console, so it is judged there only by
the probes.

| Finding | Setting | Bad value | Default | Need |
| --- | --- | --- | --- | --- |
| `option` | `terminal.altIsMeta` (macOS) | `false` | `false` | keys |
| `renderer` | `terminal.frontend`, `hacks.disableGPU` | `xterm`; `true` | `xterm-webgl`; `false` | speed |
| `conpty` | `terminal.useConPTY` (Windows) | `false` | `true` | clicks |

With `encrypted: true`, kit cannot read the rest, and `terminal-tabby` says so and holds.

Docs only: Tabby's hotkeys turn ⌥← and ⌥→ into `ESC b` and `ESC f` on macOS; it has no kitty keyboard protocol, so
⇧⌫ arrives as ⌫ (the probe says so).

### Windows Terminal

Verified against microsoft/terminal's source and settings schema.

| Finding | Setting, as the window names it | Key: bad value | Default | Need |
| --- | --- | --- | --- | --- |
| `kitty-keyboard` | profile → Advanced → kitty keyboard mode | `compatibility.kittyKeyboardMode`: `false` | `true` (1.25.2733 and later) | keys |
| `force-vt` | profile → Advanced → force VT input | `compatibility.input.forceVT`: `true` | `false` | keys |
| `text-measurement` | Settings → Compatibility → Text measurement mode | `compatibility.textMeasurement`: `"console"` | `"graphemes"` | widths |
| `software-rendering` | Settings → Rendering → Use software rendering | `rendering.software` (or `experimental.rendering.software`): `true` | `false` | speed |
| `full-repaint` | Settings → Rendering → Redraw entire screen | `experimental.rendering.forceFullRepaint`: `true` | `false` | speed |

kit reads no version of Windows Terminal: none of its readers gives one cheaply, and the terminal's answer to a version
query is a constant. So `kitty-keyboard` fails only on a `false` kit read; an absent key, on a release older than
1.25.2733, is the `kitty-keyboard` probe's to find, with this setting and the release as its fix (my call).

Docs only: releases older than 1.22, before the rewrite of ConPTY, repaint slower and less correctly; ⌥ with an arrow
moves focus between panes, and is swallowed only in a split tab; ⌥⏎, ⌥Space, and others are
always Windows Terminal's; `rendering.graphicsAPI = "direct2d"` is mildly slower; a session Windows handed to it has
no `WT_SESSION`, so its color comes from `TERM` alone.

### Windows console

From microsoft/terminal's source; the console that ships with Windows varies with the Windows build.

| Finding | Setting | Key: bad value | Default | Need |
| --- | --- | --- | --- | --- |
| `console` | none | the Windows console is the terminal: slow to repaint, no color emoji, no kitty keyboard protocol; the fix is Windows Terminal | none | speed |
| `text-measurement` | `HKCU\Console` `TextMeasurement` | 2 (console) | 0 | widths |

Docs only: Use legacy console (`ForceV2=0`) stops WSL from starting at all, so welcome never sees it.

### Multiplexers

Not yet researched. tmux's `escape-time` (esc held back), `extended-keys` (the kitty protocol), and truecolor through
`terminal-features` are the likely findings. Until they are researched, a multiplexer gets the probes and one passing
line naming it, with a pointer to the docs.

## Left to the builder

- The order of the readers' work, within the one-second timeout.
- Reading JSONC and TOML: a narrow read of the keys a finding needs, or a reader the system toolchain has, said in the
  module's header.
- Verifying Terminal's keys, and Windows Terminal's setting names as its window shows them, before they become checks.
- The wording of each line and fix, within kit's style.
- Researching the multiplexers, as its own issue.
