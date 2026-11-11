# shellcheck shell=bash
# delivery.sh - kit delivery's stages: the artifacts, environments, and deployments the forge holds
# ----------------------------------------------------------------------------------------------------------------------
# Delivery is the app whose other tabs, the releases and the builds, are kit flow's. The three stages here ask the
# forge through the seam of modules/bash/forge.sh, and read nothing else:
#
#   artifacts     a row per artifact a run kept, the newest 100 at most, its digest, size, run, and expiry
#   environments  a row per deployment environment: the rules that guard it, who approves, and the branches that may
#                 deploy there
#   deployments   a row per deployment, the newest 50: where it went, of which ref and commit, its state, and by whom
#
# A run makes each of them, so each stage's mark holds the forge's mark of its runs; environments' and deployments'
# also hold the forge's marks of their own, as a tool outside the runs may deploy, a status may change, and a person
# may edit an environment's protection.
#
# see: bin/kit-delivery, modules/bash/apps.sh, modules/bash/forge.sh, docs/apps.md
use apps forge

# delivery_stages are kit delivery's stages, in Delivery's tab order after kit flow's releases and builds.
# shellcheck disable=SC2034  # read by modules/bash/apps.sh
delivery_stages=(artifacts environments deployments)

# delivery_elsewhere names the tabs of Delivery that kit flow prints.
# shellcheck disable=SC2034  # read by modules/bash/apps.sh
declare -gA delivery_elsewhere=([releases]='kit flow releases' [builds]='kit flow builds')

# apps/delivery:artifacts prints the artifacts document, through apps::artifact, limited when the forge holds more
# than it read.
apps/delivery:artifacts() { apps:forge artifacts artifacts artifact; }

# apps/delivery:environments prints the environments document, through apps::environment.
apps/delivery:environments() { apps:forge environments environments environment; }

# apps/delivery:deployments prints the deployments document, through apps::deployment, limited when the forge holds
# more than it read.
apps/delivery:deployments() { apps:forge deployments deployments deployment; }

# apps/delivery:mark-artifacts prints the forge's mark of its runs.
apps/delivery:mark-artifacts() { apps:forge-mark runs; }

# apps/delivery:mark-environments prints the forge's marks of its environments and its runs.
apps/delivery:mark-environments() { apps:forge-mark environments runs; }

# apps/delivery:mark-deployments prints the forge's marks of its deployments and its runs.
apps/delivery:mark-deployments() { apps:forge-mark deployments runs; }

# ----------------------------------------------------------------------------------------------------------------------
