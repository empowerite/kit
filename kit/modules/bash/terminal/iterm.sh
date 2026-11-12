# shellcheck shell=bash
# iterm.sh - iTerm2's reader: the settings of the profile kit runs in, read from its preferences (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# iTerm2 keeps its profiles in its preferences, under New Bookmarks, and saves a change as it is made in its window.
# The whole export holds dates and data that plutil cannot turn into JSON, so the reader takes the two keys it needs
# from it: the profiles, and Default Bookmark Guid. The profile is the one ITERM_PROFILE names, else the default. A key
# a profile does not set comes from the parent a dynamic profile names, then from iTerm2's own default, which is the
# one in its source, not its factory profile's: an absent Mouse Reporting is off.
#
#   profile                 Name
#   triggers                the Triggers not marked disabled
#   triggers_interactive    Enable Triggers in Interactive Apps, true
#   option_left             Option Key Sends: 0 normal, 1 Meta, 2 Esc+; 0
#   option_right            Right Option Key Sends, 0
#   apps_change_keys        Allow modifyOtherKeys, true
#   mouse                   Mouse Reporting, false
#   mouse_clicks            Mouse Reporting allow clicks and drags, true
#   no_alternate_screen     Disable Smcup Rmcup, false
#   unicode_version         Unicode Version, 9
#   ambiguous_wide          Ambiguous Double Width, false
#   encoding                Character Encoding, 4, UTF-8
#
# see: modules/bash/terminal.sh, https://github.com/gnachman/iTerm2/blob/master/sources/Settings/iTermProfilePreferences.m

use file

# terminal/iterm:settings prints the settings of the profile kit runs in, or why it cannot read them, and fails.
terminal/iterm:settings() {
  local dir bookmarks default
  file:scratch dir || return 1
  if ! defaults export com.googlecode.iterm2 - > "${dir}/preferences" 2>/dev/null
  then
    printf "kit cannot read iTerm2's preferences\n"
    return 1
  fi
  if ! bookmarks=$(plutil -extract "New Bookmarks" json -o - "${dir}/preferences" 2>/dev/null) \
    || ! jq --exit-status 'type == "array" and length > 0' <<<"${bookmarks}" >/dev/null 2>&1
  then
    printf "iTerm2's preferences hold no profile\n"
    return 1
  fi
  default=$(plutil -extract "Default Bookmark Guid" raw -o - "${dir}/preferences" 2>/dev/null) || default=''
  jq --compact-output --arg name "${ITERM_PROFILE:-}" --arg default "${default}" '
    . as $all
    | (first(.[] | select($name != "" and .Name == $name))
       // first(.[] | select(.Guid == $default)) // .[0]) as $p
    | (first($all[] | select(($p["Dynamic Profile Parent Name"] // null) as $parent | $parent != null
                             and .Name == $parent))
       // first($all[] | select(($p["Dynamic Profile Parent GUID"] // null) as $parent | $parent != null
                                and .Guid == $parent))
       // {}) as $parent
    | def get($key; $default): if $p | has($key) then $p[$key] elif $parent | has($key) then $parent[$key]
                                else $default end;
    {profile: $p.Name,
     triggers: (get("Triggers"; []) | map(select(.disabled != true)) | length),
     triggers_interactive: get("Enable Triggers in Interactive Apps"; true),
     option_left: get("Option Key Sends"; 0),
     option_right: get("Right Option Key Sends"; 0),
     apps_change_keys: get("Allow modifyOtherKeys"; true),
     mouse: get("Mouse Reporting"; false),
     mouse_clicks: get("Mouse Reporting allow clicks and drags"; true),
     no_alternate_screen: get("Disable Smcup Rmcup"; false),
     unicode_version: get("Unicode Version"; 9),
     ambiguous_wide: get("Ambiguous Double Width"; false),
     encoding: get("Character Encoding"; 4)}' <<<"${bookmarks}"
}

# ----------------------------------------------------------------------------------------------------------------------
