# shellcheck shell=bash
# wt.sh - Windows Terminal's reader: the settings of the profile kit runs in, from its settings.json (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# Windows Terminal keeps settings.json under LOCALAPPDATA: the Store's, then the Preview's, then an unpackaged
# install's. Inside WSL, cmd.exe says where LOCALAPPDATA is, run from /mnt/c so it does not warn of a path it cannot
# use, and wslpath maps it. The file is JSONC: comments and trailing commas, which terminal/wt:jsonc takes out before jq
# reads it. A profile's setting comes from the profile, then profiles.defaults, then Windows Terminal's own default.
# The profile is the one WT_PROFILE_ID names, its GUID in any case, else defaultProfile; a session Windows handed to
# Windows Terminal has no WT_PROFILE_ID.
#
#   profile             the profile's name
#   kitty_keyboard      compatibility.kittyKeyboardMode; null where it is not set, as its default depends on the release
#   force_vt            compatibility.input.forceVT, false
#   text_measurement    compatibility.textMeasurement, a global: graphemes, wcswidth, or console; graphemes
#   software            rendering.software, or experimental.rendering.software, a global; false
#   full_repaint        rendering.forceFullRepaint, or experimental.rendering.forceFullRepaint, a global; false
#
# see: modules/bash/terminal.sh, https://github.com/microsoft/terminal/blob/main/doc/cascadia/profiles.schema.json

# terminal/wt:settings prints the settings of the profile kit runs in, or why it cannot read them, and fails.
terminal/wt:settings() {
  local local_dir file='' candidate
  local_dir=$(terminal/wt:local-app-data 2>/dev/null | tr -d '\r')
  if [[ -z "${local_dir}" || "${local_dir}" == *%* ]] || ! local_dir=$(wslpath -u "${local_dir}" 2>/dev/null)
  then
    printf "kit cannot reach Windows' LOCALAPPDATA from here\n"
    return 1
  fi
  for candidate in "${local_dir}/Packages/Microsoft.WindowsTerminal_8wekyb3d8bbwe/LocalState/settings.json" \
    "${local_dir}/Packages/Microsoft.WindowsTerminalPreview_8wekyb3d8bbwe/LocalState/settings.json" \
    "${local_dir}/Microsoft/Windows Terminal/settings.json"
  do
    if [[ -r "${candidate}" ]]
    then
      file=${candidate}
      break
    fi
  done
  if [[ -z "${file}" ]]
  then
    printf "Windows Terminal's settings.json is in none of the places it keeps it\n"
    return 1
  fi
  if ! terminal/wt:jsonc < "${file}" | jq --compact-output --arg id "${WT_PROFILE_ID:-}" '
      (.profiles | if type == "array" then . else .list // [] end) as $list
      | (.profiles | if type == "object" then .defaults // {} else {} end) as $defaults
      | (first($list[] | select($id != "" and ((.guid // "") | ascii_downcase) == ($id | ascii_downcase)))
         // first($list[] | select(((.guid // "") | ascii_downcase) == ((.defaultProfile // "") | ascii_downcase)))
         // $list[0] // {}) as $p
      | def get($key; $default): if $p | has($key) then $p[$key] elif $defaults | has($key) then $defaults[$key]
                                  else $default end;
      . as $global
      | def global($keys; $default): first($keys[] as $key | $global | select(has($key)) | .[$key]) // $default;
      {profile: ($p.name // "the default"),
       kitty_keyboard: get("compatibility.kittyKeyboardMode"; null),
       force_vt: get("compatibility.input.forceVT"; false),
       text_measurement: global(["compatibility.textMeasurement"]; "graphemes"),
       software: global(["rendering.software", "experimental.rendering.software"]; false),
       full_repaint: global(["rendering.forceFullRepaint", "experimental.rendering.forceFullRepaint"]; false)}' \
      2>/dev/null
  then
    printf "kit cannot read Windows Terminal's settings.json as JSON\n"
    return 1
  fi
}

# terminal/wt:local-app-data prints Windows' LOCALAPPDATA, as cmd.exe says it, run from /mnt/c so it does not warn of
# a path it cannot use.
terminal/wt:local-app-data() {
  cd /mnt/c 2>/dev/null || true  # outside WSL's /mnt/c, cmd.exe runs where it is, and only warns
  cmd.exe /c 'echo %LOCALAPPDATA%'
}

# terminal/wt:jsonc reads JSONC on stdin and prints it as JSON: // and /* */ comments outside strings taken out, and
# a comma before a closing brace or bracket.
terminal/wt:jsonc() {
  tr -d '\r' | awk '
    {
      n = length($0)
      for (i = 1; i <= n; i++)
      {
        c = substr($0, i, 1)
        if (block)
        {
          if (substr($0, i, 2) == "*/") { block = 0; i++ }
          continue
        }
        if (string)
        {
          out = out c
          if (c == "\\") { out = out substr($0, i + 1, 1); i++ }
          else if (c == "\"") string = 0
          continue
        }
        if (c == "\"") { string = 1; out = out c; continue }
        if (substr($0, i, 2) == "//") break
        if (substr($0, i, 2) == "/*") { block = 1; i++; continue }
        if (c == "}" || c == "]") sub(/,[ \t\n]*$/, "", out)
        out = out c
      }
      out = out "\n"
    }
    END { printf "%s", out }'
}

# ----------------------------------------------------------------------------------------------------------------------
