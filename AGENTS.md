<!-- :generate:kit region AGENTS.md:until:"kit:end" -->
Read .kit/1st.md at the start of every session. It defines how any agent uses empowerite/kit to keep this
repository trustworthy.
<!-- kit:end -->

This repository is the kit's own source, and uses it.

## This repository's own rules

- The release is `bin/` and `modules/`. Everything in it is written for a consumer: no file names a consumer, a
  person, or another repository as the source of a rule.
- Releases and pre-releases: [README.md](README.md) § Release.
- `.kit/`, `.github/workflows/kit-ci.yml`, `.claude/settings.json`, and the regions in `AGENTS.md`, `.gitignore`, and
  `.claude/CLAUDE.md` are this repository's installed copies of its own kit, written by `just refresh` from `modules/`;
  see [README.md](README.md) § Maintaining. Edit the source; the pre-commit hook refreshes the copies into the same
  commit, and a desk without the hook runs `just refresh` and commits both.
- `just ci` is the gate; the justfile says what it runs.
