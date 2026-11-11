# shellcheck shell=bash
# changes.sh - kit changes' stages: each open change's reviews and checks, and the changes merged last
# ----------------------------------------------------------------------------------------------------------------------
# Changes is the app whose other tab, the open changes, is kit flow's. The three stages here ask the forge through the
# seam of modules/bash/forge.sh, and read nothing else:
#
#   reviews   a row per open change and reviewer: asked, or the reviewer's latest review, approved, changes, or
#             commented, and when, with the change's unresolved threads
#   checks    a row per open change and check: the latest run of each name on its head commit, so a rerun supersedes
#             the run before it
#   merged    a row per change merged last, 50 at most, who merged it and when
#
# reviews and checks read the open changes as kit flow changes does, from the snapshot the forge's open-prs call
# keeps, so they cost no request more. Each stage's mark is the forge's marks of its issues, which move with its
# changes, and of its runs, as kit flow changes' is.
#
# see: bin/kit-changes, modules/bash/apps.sh, modules/bash/forge.sh, docs/apps.md
use apps forge

# changes_stages are kit changes' stages, in Changes' tab order after kit flow's changes, the three the forge answers.
# shellcheck disable=SC2034  # read by modules/bash/apps.sh
changes_stages=(reviews checks merged)

# changes_elsewhere names the tab of Changes that kit flow prints.
# shellcheck disable=SC2034  # read by modules/bash/apps.sh
declare -gA changes_elsewhere=([changes]='kit flow changes')

# apps/changes:reviews prints the reviews document: each open change's reviewers, as the forge's open-prs call prints
# them, through apps::review. A primary with no forge, or a forge without reviewers, gives none.
apps/changes:reviews() { apps:forge reviewers open-prs review; }

# apps/changes:checks prints the checks document: each open change's runs, the latest of each name, through
# apps::check.
apps/changes:checks() { apps:forge checks open-prs check; }

# apps/changes:merged prints the merged document, through apps::merged, as the forge's merged-prs call gives them,
# limited when the forge holds more than it read.
apps/changes:merged() { apps:forge merged merged-prs merged; }

# apps/changes:mark-reviews prints the forge's marks of its issues and its runs.
apps/changes:mark-reviews() { apps:forge-mark issues runs; }

# apps/changes:mark-checks prints the forge's marks of its issues and its runs.
apps/changes:mark-checks() { apps:forge-mark issues runs; }

# apps/changes:mark-merged prints the forge's marks of its issues and its runs.
apps/changes:mark-merged() { apps:forge-mark issues runs; }

# ----------------------------------------------------------------------------------------------------------------------
