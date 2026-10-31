# version.jq - semantic versions: their precedence, which is higher, and the highest of several (jq module)
# ----------------------------------------------------------------------------------------------------------------------
#
# import "version" as version; with jq -L "${kit_jq}". A version is <major>.<minor>.<patch>, with -<pre-release> for
# a pre-release, as kit release takes it and a release tag names it after release/.
#
# see: modules/bash/releases.sh, https://semver.org/#spec-item-11

# key gives a version's sort key, by semantic-version precedence: the three numbers as numbers; then a release above
# any pre-release of it; then the pre-release's identifiers, numeric ones as numbers below alphanumeric ones, which
# compare as text, and a longer set above its own prefix, as jq orders arrays.
def key:
  (split("-") | .[0]) as $core
  | (.[($core | length) + 1:]) as $pre
  | ($core | split(".") | map(tonumber))
    + [if $pre == "" then 1 else 0 end]
    + [if $pre == "" then [] else $pre | split(".") | map(if test("^[0-9]+$") then [0, tonumber] else [1, .] end) end];

# higher($versions) says whether the version is above every one of $versions, true when there are none.
def higher($versions): key as $k | all($versions[]; key < $k);

# highest gives the highest of an array of versions, or null for none.
def highest: if length == 0 then null else max_by(key) end;

# ----------------------------------------------------------------------------------------------------------------------
