Read .kit/1st.md at the start of every session. It defines how any agent uses empowerite/kit to keep this
repository trustworthy. This repository is the kit's own source, and uses it.

## This repository's own rules

- The release is `bin/` and `modules/`. Everything in it is written for a
  consumer: no file names a consumer, a person, or another repository as the source of a rule.
- A release is the tag `v<version>` on a commit on `main`; a pre-release is `v<version>-<suffix>` on a commit that is
  not on `main`. The tag is the version: no file carries it, the workflow stamps it into the packed kit, and a
  published release is immutable, so a tag is never reused. A release is permanent; a pre-release is retired when
  the branch it was tagged on is deleted.
- `.kit/`, `.github/workflows/kit-ci.yml`, and `.claude/settings.json` are this repository's installed copies of its
  own kit, written by `just refresh` from `modules/`; see [README.md](README.md) § Maintaining. Edit the source; the
  pre-commit hook refreshes the copies into the same commit, and a desk without the hook runs `just refresh` and
  commits both.
- `just ci` is the gate; the justfile says what it runs.
