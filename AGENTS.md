@.kit/1st.md

Read .kit/1st.md first. It is the kit's entry file, and this repository uses its own kit.

## This repository's own rules

- The delivered tree is `.kit/` and `bin/kit`. Everything in it is written for a consumer: no file names a
  consumer, a person, or another repository as the source of a rule.
- `.kit/version` is the release version. A release is a commit on `main` that sets it, then the tag `v<version>`
  on that commit. The release workflow refuses a tag whose version differs from the file, or whose commit is not
  on `main`.
- `.claude/rules/kit-*.md`, `.github/workflows/kit-ci.yml`, `.claude/settings.json`, and `.kit/manifest` are
  this repository's installed copies of its own kit. Change a delivered file at its source under `.kit/`, then
  run `just refresh` to rewrite the copies, and commit both.
- `just ci` is the gate: shellcheck, actionlint, the bats suite, and `bin/kit check`.
