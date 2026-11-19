# Decisions

What the owner decided, one line each, the newest last. A question this page has decided is opened again only by the
owner, and only the owner changes this page.

| Date | Decision | Where it is written |
| --- | --- | --- |
| 2026-11-18 | What kit is for, and the principles it holds. | [the charter](charter.md) |
| 2026-11-18 | How work is done here. | [CONTRIBUTING.md](../CONTRIBUTING.md) |
| 2026-11-18 | What all of kit stands on, with the points under "What the owner decided". | [the foundations design](design/foundations.md) |
| 2026-11-18 | The order of the work: foundations, the toolchain, rules where agents look, kit for manual work, the bot. | [the foundations design](design/foundations.md#the-order-of-the-work) |
| 2026-11-18 | A build does not know its release: a release is a tag put afterwards on a commit whose build exists, and one executable can answer to two. | [the foundations design](design/foundations.md#constraints) |
| 2026-11-18 | A pull request that is ready asks for the owner's review; one he does not accept is a draft again, with a comment; and his words stay on the pull request. | [CONTRIBUTING.md](../CONTRIBUTING.md#the-owners-approval) |
| 2026-11-18 | The gate runs on pull requests, and not on `main`: a pull request is up to date with `main` before it merges, so the tree that lands is the tree the gate judged. | [CONTRIBUTING.md](../CONTRIBUTING.md#the-gate) |
| 2026-11-18 | Scripts are bash, with `[[ … ]]` and `(( … ))`, never `[ … ]`. | [CONTRIBUTING.md](../CONTRIBUTING.md#the-gate) |
| 2026-11-18 | A pull request is in its issue's milestone from when it is opened. | [CONTRIBUTING.md](../CONTRIBUTING.md#a-pull-request) |
| 2026-11-18 | The name is kit, of empowerite, and it stands for nothing. | [README.md](../README.md) |
| 2026-11-18 | The README names the pages and says the state in a line. It does not list what the repository holds, so adding a file does not change it. | [CONTRIBUTING.md](../CONTRIBUTING.md#a-pull-request) |
