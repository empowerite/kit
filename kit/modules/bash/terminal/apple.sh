# shellcheck shell=bash
# apple.sh - Terminal's reader: the settings of its default profile, read from its preferences (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# Terminal names the profile of a window in no variable, and only AppleScript, which asks the person's leave, can tell
# which one a tab has. So the reader judges the default profile, Default Window Settings, and its heading says so. The
# export holds raw escapes and data that no JSON reader takes whole, so each key is taken alone, under Window
# Settings.<profile>, and a key the profile does not set is Terminal's default. A profile name with a dot cannot be
# read: plutil takes the dot for a step into the key path.
#
#   profile           Default Window Settings, else Basic
#   option_meta       useOptionAsMetaKey, false
#   ambiguous_wide    EastAsianAmbiguousWide, false
#   encoding          CharacterEncoding, an NSStringEncoding: 4, UTF-8
#
# Terminal's source is closed: the key names are the strings its binary holds.
#
# see: modules/bash/terminal.sh, https://support.apple.com/guide/terminal/trmladvn/mac

use file

# terminal/apple:settings prints the settings of Terminal's default profile, or why it cannot read them, and fails.
terminal/apple:settings() {
  local dir profile option ambiguous encoding
  file:scratch dir || return 1
  if ! defaults export com.apple.Terminal - > "${dir}/preferences" 2>/dev/null
  then
    printf "kit cannot read Terminal's preferences\n"
    return 1
  fi
  profile=$(plutil -extract "Default Window Settings" raw -o - "${dir}/preferences" 2>/dev/null) || profile=Basic
  if [[ "${profile}" == *.* ]]
  then
    printf 'its default profile %s has a dot in its name, which kit cannot read\n' "${profile}"
    return 1
  fi
  option=$(terminal/apple:key "${dir}/preferences" "${profile}" useOptionAsMetaKey false)
  ambiguous=$(terminal/apple:key "${dir}/preferences" "${profile}" EastAsianAmbiguousWide false)
  encoding=$(terminal/apple:key "${dir}/preferences" "${profile}" CharacterEncoding 4)
  jq --null-input --compact-output --arg profile "${profile}" --argjson option "${option}" \
    --argjson ambiguous "${ambiguous}" --argjson encoding "${encoding}" \
    '{profile: $profile, option_meta: $option, ambiguous_wide: $ambiguous, encoding: $encoding}'
}

# terminal/apple:key <preferences> <profile> <key> <default> prints the profile's value of <key>, or <default> where
# the profile does not set it.
terminal/apple:key() {
  plutil -extract "Window Settings.$2.$3" raw -o - "$1" 2>/dev/null || printf '%s\n' "$4"
}

# ----------------------------------------------------------------------------------------------------------------------
