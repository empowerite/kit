# shellcheck shell=bash
# people.sh - kit people's stages: who has access to the repository, its teams, and the bots that acted in it
# ----------------------------------------------------------------------------------------------------------------------
# People is the app whose tabs are all its own. The three stages ask the forge through the seam of
# modules/bash/forge.sh, and read nothing else:
#
#   people  a row per person or bot with access to the repository: the name, the role, and whether it is a bot
#   teams   a row per team with access, 20 at most: its role and its members, 100 of each team's at most
#   agents  a row per bot that acted: among the newest commits, changes, and reviews, or the forge's own bot users
#
# No stage has a mark. The forge's marks call reads every kind at once, for every app: a mark of who has access would
# cost each of them a request more, which a caller who may not push is refused, and a team's members change no mark of
# the teams' list. Each stage is read again on Refresh only.
#
# see: bin/kit-people, modules/bash/apps.sh, modules/bash/forge.sh, docs/apps.md
use apps forge

# people_stages are kit people's stages, in People's tab order.
# shellcheck disable=SC2034  # read by modules/bash/apps.sh
people_stages=(people teams agents)

# apps/people:people prints the document of who has access, through apps::person.
apps/people:people() { apps:forge collaborators collaborators person; }

# apps/people:teams prints the teams document, through apps::team, limited when the forge holds more than it read.
apps/people:teams() { apps:forge teams teams team; }

# apps/people:agents prints the document of the bots that acted, through apps::agent.
apps/people:agents() { apps:forge bots bots agent; }

# ----------------------------------------------------------------------------------------------------------------------
