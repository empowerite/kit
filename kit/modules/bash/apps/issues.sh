# shellcheck shell=bash
# issues.sh - kit issues' stages: the forge's milestones and iterations (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# Issues is the app whose other tabs, the backlog and what is underway, are kit flow's. The two stages here ask the forge
# through the seam of modules/bash/forge.sh, and read nothing else:
#
#   milestones    a row per milestone, open and closed, by number: its title, state, due date, and how many of its issues
#                 are open and closed, GitLab's and GitHub's; Azure DevOps has none
#   iterations    a row per iteration, by start: GitHub's, from the projects linked to the repository, GitLab's, with the
#                 groups', and Azure DevOps' team's, where the iteration that holds now is current
#
# milestones' mark is the forge's own mark of its milestones, as the forge's marks read it; iterations has none, which
# kittui reads again only on Refresh and Watch.
#
# see: bin/kit-issues, modules/bash/apps.sh, modules/bash/forge.sh, docs/apps.md
use apps forge

# issues_stages are kit issues' stages, in Issues' tab order, the two the forge answers.
# shellcheck disable=SC2034  # read by modules/bash/apps.sh
issues_stages=(milestones iterations)

# issues_elsewhere names the tabs of Issues that kit flow prints.
# shellcheck disable=SC2034  # read by modules/bash/apps.sh
declare -gA issues_elsewhere=([backlog]='kit flow backlog' [underway]='kit flow underway')

# apps/issues:milestones prints the milestones document: each milestone the forge's milestones call prints, through
# apps::milestone. A primary with no forge, or a forge without milestones, gives none.
apps/issues:milestones() { apps:forge milestones milestones milestone; }

# apps/issues:iterations prints the iterations document, through apps::iteration, as the forge's iterations call gives
# them. A forge that has none for this repository is the error {feature: iterations, off: true}.
apps/issues:iterations() { apps:forge iterations iterations iteration; }

# apps/issues:mark-milestones prints the forge's mark of its milestones.
apps/issues:mark-milestones() { apps:forge-mark milestones; }

# ----------------------------------------------------------------------------------------------------------------------
