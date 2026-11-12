# Terminals

kittui is a full-screen program, and the terminal around it can make it slower or worse without anything in kit being
wrong: a trigger that runs on every line kittui repaints, an ⌥ key sent as a plain character, an emoji drawn one cell
wide. This page lists, for each terminal kit knows, the settings that do that and how to change them. `kit welcome`
judges the terminal it runs in against this list, in its setup step, and links each finding to its terminal's section
here.

## What kittui needs

kittui draws on the alternate screen, turns the mouse on in cell-motion mode, asks for the kitty keyboard protocol,
and draws its menus with two-cell emoji. A terminal must give it:

| Need | What it means |
| --- | --- |
| speed | Repaints of the alternate screen that nothing in the terminal stalls. |
| clicks | A plain left click reaches kittui, and so do a drag and a scroll. |
| keys | ⌥ arrives as Meta, ⇧⌫ is not ⌫, esc arrives at once, and, where the terminal can, the kitty keyboard protocol. |
| widths | A two-cell emoji takes two cells, and an arrow or a rule takes one, so every column stays in place. |
| color | 24-bit color, or at least 256 colors, with `COLORTERM` saying which. |
| copy | An app may set the clipboard through the terminal (OSC 52), so ⧉ copies, over ssh too. |

A theme, a font, or anything else that is taste is not a finding.

## How kit judges a terminal

kit names the terminal from the environment: a multiplexer first (`TMUX`, `STY`, `ZELLIJ`), then ssh
(`SSH_CONNECTION`, `SSH_TTY`), then the terminal's own variables, and inside WSL the window Windows runs it in. It
judges the terminal two ways:

- Settings: where it knows the terminal, kit reads the settings the terminal keeps, once, within a second, and never
  writes them. A finding fails only on a value kit read. A setting kit could not read is not a finding, and a finding
  that reads a setting its reader did not give is kit's own bug, which the terminal's heading names.
- Probes: kit asks the terminal what it does, before it hands the terminal to kittui, and passes the answers down in
  `KIT_TERMINAL_PROBES`. The probes run in every terminal, known or not, over ssh and inside a multiplexer.

| Probe | Asks | Need |
| --- | --- | --- |
| `kitty-keyboard` | `CSI ? u`: does it answer the kitty keyboard protocol | keys |
| `emoji-width` | prints 📚, then `CSI 6n`: did the cursor move two columns | widths |
| `ambiguous-width` | prints ↑, then `CSI 6n`: did the cursor move one column | widths |
| `color` | `COLORTERM` and `TERM`, as kittui's library reads them | color |

A setting explains, and a probe measures, so they never both report one cause: a probe is left out where a setting
of its need failed. A probe that fails where no setting of the terminal can fix it holds, and says what kittui does
instead; Terminal and Tabby have no kitty keyboard protocol, so kittui offers > and < for ⌥ with an arrow.

Every fix is the person's to make, so `kit welcome` shows it and never runs it.

## Keeping a finding

A finding you keep on purpose, such as triggers you want in vim, stops failing:

```sh
kit welcome --keep=terminal-iterm-triggers
```

kit writes it under `terminal: kept:` in `~/.kit/settings.yml`, after the profile it was judged in where the terminal
has profiles, so a keep in one profile never silences another; where it cannot read which profile that is, kit
refuses the keep. The finding then holds, and its line starts with `kept:`, as `kit welcome --keep=` says when it ends;
with `--json`, it gives the check as `kit welcome --checks --json` would.
`kit welcome --unkeep=terminal-iterm-triggers` undoes it, and the last one takes `terminal:` out of the file. In
kittui, a failed finding offers later and keep.

## iTerm2

kit knows iTerm2 by `TERM_PROGRAM=iTerm.app`, and judges the profile `ITERM_PROFILE` names, else the default. It reads
the profiles from iTerm2's preferences (`defaults export com.googlecode.iterm2`); a key a profile does not set comes
from a dynamic profile's parent, then from iTerm2's own default. Every path below starts at iTerm2 → Settings →
Profiles → the profile, but `clipboard`'s, which is iTerm2's own setting, under General, and takes effect once iTerm2
starts again.

| Finding | Setting | Bad | Default | Need |
| --- | --- | --- | --- | --- |
| `triggers` | Advanced → Triggers → Enabled in interactive apps | on, with a trigger enabled | on | speed |
| `option` | Keys → General → Left Option key, Right Option key | neither is Esc+ | Normal | keys |
| `apps-change-keys` | Keys → General → Apps can change how keys are reported | off | on | keys |
| `mouse` | Terminal → Enable mouse reporting, and Report mouse clicks & drags | either off | off, on | clicks |
| `alternate-screen` | Terminal → Disable save/restore alternate screen | on | off | speed |
| `unicode-9` | Text → Use Unicode version 9+ widths | off | on | widths |
| `ambiguous-wide` | Text → Ambiguous characters are double-width | on | off | widths |
| `encoding` | Terminal → Character encoding | not UTF-8 | UTF-8 | widths |
| `clipboard` | General → Selection → Applications in terminal may access clipboard | off | off | copy |

iTerm2 runs a trigger's regex on every line a full-screen program repaints; one prompt-detection trigger took 399 ms
on a single match. iTerm2 keeps ⌥, ⌃, and the right button with a click for its own selection and menu. Ligatures and
transparency turn off its GPU renderer, which is likely slower. A session's own edits, under Session → Edit Session,
override the profile and are not in the preferences.

See: [iTerm2's profile defaults](https://github.com/gnachman/iTerm2/blob/master/sources/Settings/iTermProfilePreferences.m)

## Terminal

kit knows Terminal by `TERM_PROGRAM=Apple_Terminal`. No variable names a tab's profile, so kit judges the default
profile, and its heading says so. It reads each key from Terminal's preferences (`defaults export com.apple.Terminal`);
a profile name with a dot cannot be read. Every path below starts at Terminal → Settings → Profiles → the profile.

| Finding | Setting | Bad | Default | Need |
| --- | --- | --- | --- | --- |
| `option` | Keyboard → Use Option as Meta key | off | off | keys |
| `ambiguous-wide` | Advanced → Unicode East Asian Ambiguous characters are wide | on | off | widths |
| `encoding` | Advanced → Text encoding | not Unicode (UTF-8) | Unicode (UTF-8) | widths |

Use Option as Meta key breaks keyboard layouts such as AZERTY, where ⌥ types characters. View → Allow Mouse Reporting
is set per window and kept in no preference. Terminal has no kitty keyboard protocol. Its 24-bit color depends on the
macOS release; the color probe says what it gives.

See: [Terminal's Advanced settings](https://support.apple.com/guide/terminal/trmladvn/mac)

## Ghostty

kit knows Ghostty by `TERM_PROGRAM=ghostty`, and reads what `ghostty +show-config` prints, every config file and
include read. On macOS, Ghostty's default for ⌥ depends on the keyboard layout, so kit reads the layout too.

| Finding | Setting | Bad | Default | Need |
| --- | --- | --- | --- | --- |
| `option` | `macos-option-as-alt` | `false`, or unset on a layout other than U.S. | `true` on U.S. layouts | keys |
| `alt-arrows` | `keybind = alt+left=…`, `keybind = alt+right=…` | not both rebound | ⌥← and ⌥→ sent as ⌥b and ⌥f | keys |
| `mouse` | `mouse-reporting` | `false` | `true` | clicks |
| `grapheme-width` | `grapheme-width-method` | `legacy` | `unicode` | widths |

`shell-integration-features` without `ssh-terminfo` and `sudo` leaves `TERM=xterm-ghostty` on hosts that lack its
terminfo. The `toggle_mouse_reporting` action turns reporting off for a session, which no setting shows.
`custom-shader` and `background-blur` cost frames.

See: [Ghostty's options](https://github.com/ghostty-org/ghostty/blob/main/src/config/Config.zig)

## kitty

kit knows kitty by `KITTY_PID` or `TERM=xterm-kitty`, and reads the effective config the running kitty wrote for its
window, `effective-config/$KITTY_PID` in kitty's cache.

| Finding | Setting | Bad | Default | Need |
| --- | --- | --- | --- | --- |
| `option` | `macos_option_as_alt` | `no` | `no` | keys |
| `mouse` | `mouse_map left press grabbed …` | any plain left press grabbed | none | clicks |

`macos_option_as_alt` takes effect when kitty restarts, not when it reloads. `repaint_delay` (10 ms) and `input_delay`
(3 ms) well above their defaults slow repaints. Plain `ssh` leaves `TERM=xterm-kitty` on hosts that lack its terminfo;
`kitten ssh` brings it.

See: [kitty's options](https://github.com/kovidgoyal/kitty/blob/master/kitty/options/definitions.py)

## WezTerm

kit knows WezTerm by `TERM_PROGRAM=WezTerm`. Its config is Lua, and nothing prints the values it settles on, so kit
reads only the bindings `wezterm show-keys` prints; whether WezTerm speaks the kitty keyboard protocol is the probe's
to find, and its fix names the file in `WEZTERM_CONFIG_FILE`.

| Finding | Setting | Bad | Default | Need |
| --- | --- | --- | --- | --- |
| `alt-enter` | ⌥⏎ bound to `ToggleFullScreen` | bound | bound | keys |

`enable_kitty_keyboard` is off by default; set `config.enable_kitty_keyboard = true` so ⇧⌫ and ⌥ with an arrow reach
kittui as themselves. A mouse binding with `mouse_reporting = true` keeps that click from kittui. `unicode_version`
below 14 draws ✔️ and the like in one cell. `send_composed_key_when_left_alt_is_pressed = true` sends ⌥ as a plain
key. `max_fps` below 60 and `front_end = "Software"` slow repaints. `term = "wezterm"` needs its terminfo on every
host.

See: [WezTerm's show-keys](https://wezterm.org/cli/show-keys.html)

## Alacritty

kit knows Alacritty by `ALACRITTY_WINDOW_ID` or `TERM=alacritty`, and reads `option_as_alt` from the first
`alacritty.toml` on Alacritty's search path. Alacritty speaks the kitty keyboard protocol with no setting.

| Finding | Setting | Bad | Default | Need |
| --- | --- | --- | --- | --- |
| `option` | `[window] option_as_alt` | `"None"` | `"None"` | keys |

Alacritty reads no `alacritty.yml` since 0.13; `alacritty migrate` converts one. `TERM=alacritty` needs its terminfo on
remote hosts.

See: [Alacritty's config](https://alacritty.org/config-alacritty.html)

## Warp

kit knows Warp by `TERM_PROGRAM=WarpTerminal`, and reads `~/.warp/settings.toml` on macOS. Warp is closed source, and
these rows come from its settings docs.

| Finding | Setting | Bad | Default | Need |
| --- | --- | --- | --- | --- |
| `option` | `[terminal.input] extra_meta_keys` | neither `left_alt` nor `right_alt` | both off | keys |
| `mouse` | `[terminal] mouse_reporting_enabled` | `false` | `true` | clicks |

View → Toggle Mouse Reporting turns reporting off for a session without the file. Warp's docs suggest turning off
custom padding on the alternate screen where full-screen programs draw badly. Shift with a click is Warp's.

See: [Warp's settings](https://docs.warp.dev/terminal/settings/all-settings/)

## Tabby

kit knows Tabby by `TERM_PROGRAM=Tabby`, on macOS and Linux, and reads its `config.yaml`. Tabby forwards none of its
variables into WSL, so on Windows kit cannot tell it from the Windows console, and judges it by the probes alone. With
its vault on, the config is encrypted, and kit says it cannot read it.

| Finding | Setting | Bad | Default | Need |
| --- | --- | --- | --- | --- |
| `option` | `terminal.altIsMeta` (macOS) | `false` | `false` | keys |
| `renderer` | `terminal.frontend`, `hacks.disableGPU` | `xterm`, `true` | `xterm-webgl`, `false` | speed |

Tabby has no kitty keyboard protocol, so ⇧⌫ arrives as ⌫. On macOS its hotkeys turn ⌥← and ⌥→ into `ESC b` and `ESC f`.
On Windows, `terminal.useConPTY: false` falls back to a console that swallows mouse reporting.

See: [Tabby's terminal settings](https://github.com/Eugeny/tabby/blob/master/tabby-terminal/src/config.ts)

## Windows Terminal

kit knows Windows Terminal by `WT_SESSION`, or inside WSL by Windows' default terminal, which hands sessions to it
without setting `WT_SESSION`. It reads `settings.json` under `LOCALAPPDATA` (the Store's, the Preview's, then an
unpackaged install's), comments and trailing commas allowed, and judges the profile `WT_PROFILE_ID` names, else the
default; a profile's setting falls back to `profiles.defaults`.

| Finding | Setting | Bad | Default | Need |
| --- | --- | --- | --- | --- |
| `kitty-keyboard` | `compatibility.kittyKeyboardMode` | `false` | `true` (1.25.2733 and later) | keys |
| `force-vt` | `compatibility.input.forceVT` | `true` | `false` | keys |
| `text-measurement` | `compatibility.textMeasurement` | `"console"` | `"graphemes"` | widths |
| `software-rendering` | `rendering.software` | `true` | `false` | speed |
| `full-repaint` | `rendering.forceFullRepaint` | `true` | `false` | speed |

kit reads no version of Windows Terminal: releases older than 1.22 repaint slower and less correctly, and before
1.25.2733 there is no kitty keyboard protocol, which the probe finds. ⌥ with an arrow moves focus between panes, and
is swallowed only in a split tab; ⌥⏎, ⌥Space, and others are always Windows Terminal's.

See: [Windows Terminal's settings schema](https://github.com/microsoft/terminal/blob/main/doc/cascadia/profiles.schema.json)

## Windows console

Inside WSL, with no `WT_SESSION`, kit reads Windows' default terminal from `HKCU\Console\%%Startup`; the Windows
console's GUID, or none on Windows 10, names the console. kit reads `HKCU\Console` alone: a shortcut's own properties
override it where kit cannot see.

| Finding | Setting | Bad | Default | Need |
| --- | --- | --- | --- | --- |
| `console` | the Windows console draws kittui | always | | speed |
| `text-measurement` | `HKCU\Console` `TextMeasurement` | `2` (console) | `0` (graphemes) | widths |

The Windows console is slow to repaint, has no color emoji, and has no kitty keyboard protocol: use Windows Terminal,
set under Windows Settings → System → For developers → Terminal. Use legacy console (`ForceV2=0`) stops WSL from
starting at all.

See: [the console's registry settings](https://github.com/microsoft/terminal/blob/main/src/propslib/RegistrySerialization.cpp)

## Over ssh

Over ssh (`SSH_CONNECTION` or `SSH_TTY`), the terminal's settings are on the other machine, so kit reads none; the
probes still run, as the terminal answers them across the connection.

## tmux

Inside tmux, the outer terminal's variables are those of whichever terminal started tmux's server, and may be stale,
so kit judges what tmux answers and reads no terminal's settings. tmux's `escape-time` holds esc back, `extended-keys`
off withholds the kitty keyboard protocol, and `terminal-features` decides whether 24-bit color passes through; kit
does not read them yet.

## GNU screen

Inside GNU screen, kit judges what screen answers, as for tmux, and reads no terminal's settings.

## zellij

Inside zellij, kit judges what zellij answers, as for tmux, and reads no terminal's settings.

## Unknown terminals

A terminal kit does not know gets the probes alone: each probe that fails says what kit measured, that no setting kit
knows explains it, and points here.
