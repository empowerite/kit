@.kit/1st.md

Read .kit/1st.md first. It is the kit's entry file, and this repository uses its own kit.

## This repository's own rules

- The release is `bin/`, `tree/`, and `targets/`. Everything in it is written for a
  consumer: no file names a consumer, a person, or another repository as the source of a rule.
- A release is the tag `v<version>` on a commit on `main`; a pre-release is `v<version>-<suffix>` on a commit that is
  not on `main`. The tag is the version: no file carries it, the workflow stamps it into the packed kit, and a
  published release is immutable, so a tag is never reused. A release is permanent; a pre-release is retired when the branch it was tagged on is deleted.
- `.claude/rules/kit-*.md`, `.github/workflows/kit-ci.yml`, `.claude/settings.json`, and `.kit/manifest` are
  this repository's installed copies of its own kit, written by `just refresh` from `tree/` and `targets/`; see
  [README.md](README.md) § Maintaining. Edit the source, run `just refresh`, and commit both.
- `just ci` is the gate: shellcheck, actionlint, the bats suite, and `bin/kit check`.
