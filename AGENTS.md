# AGENTS.md

<!-- kit:generate AGENTS.md -->
Read .kit/1st.md at the start of every session; if it is missing, run `kit hydrate` first. It defines how any agent
uses empowerite/kit to keep this repository trustworthy.
<!-- kit:end AGENTS.md -->

This repository is the kit's own source, and uses it.

## This repository's own rules

- The release is `kit/`, and only `kit/`: `kit/bin/`, `kit/bash/`, and `kit/packages/`. Everything in it is written
  for a consumer: no file names a consumer, a person, or another repository as the source of a rule. A path in a
  file under `kit/` is written from `kit/`, as it reads in the tarball: `bash/forge.sh`, not `kit/bash/forge.sh`.
- Releases and pre-releases: [README.md](README.md) § Release.
- The kit imposes no taste on a consumer. The `kit` package delivers rules: what the work must do to be trusted.
  Style, how a file looks, ships in the optional `style/<kind>` packages. This repository's own taste beyond those,
  such as the line of dashes under a banner, is the `house` package in `.kit/packages/house/`, which takes the
  `empowerite` bundle and applies here only. When a new item is about how a file looks, it goes in a style package
  or in house, never in a rule.
- `.kit/`, `.github/workflows/kit-ci.yml`, `.claude/settings.json`, and the regions in `AGENTS.md`, `.gitignore`, and
  `.claude/CLAUDE.md` are this repository's installed copies of its own kit, written by `bin/refresh` from
  `kit/packages/` and `.kit/packages/house/`; see [README.md](README.md) § Maintaining. Edit the source; the
  pre-commit hook refreshes the copies into the same commit, and a desk without the hook runs `bin/refresh` and
  commits both.
- `kit gate` runs the gate, `bin/ci`, which says what it runs.
