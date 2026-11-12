# The policy/conventional package

Holds commit subjects and pull request titles to [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).
It delivers no file. Its `package.yml` sets three rules of the policy, which `kit policy`, `kit commit`, and
`kit work` read; see [policy.md](../../policy.md#packages-that-set-rules).

A commit has a **subject**, which `policy.commit_subject` judges at commit and in CI. A pull request has a **title**,
which `policy.change_title` judges in CI. The package sets both, and either can be turned off on its own.

| Rule | What the package sets |
| --- | --- |
| `commit_subject` | `^(build\|chore\|ci\|docs\|feat\|fix\|perf\|refactor\|revert\|style\|test)(\([a-z0-9-]+\))?!?: [^ ]` |
| `change_title` | the same pattern |
| `commit_reference` | `trailer` |

The pattern is the specification's common types, an optional scope in parentheses, an optional `!`, then a colon, a
space, and a description, as `feat(cli)!: drop the old flag`. It is in the subset that bash and RE2 both read. Each
rule's remedy names the form and the types.

## Where the issue reference goes

Conventional Commits keeps references in a footer, so `#7 feat: x` is not a conventional subject. With the package
active, the reference is a `Refs:` trailer in the message's last paragraph:

```text
feat(cli): read the flag from the environment

Refs: #7
```

- `kit commit` adds `Refs: <reference>` from the branch, as `7-fix-the-thing` gives `#7`, to a message that has no
  `Refs:` trailer, and puts no reference in front of the subject.
- `kit work` makes the start commit `chore: start <title>`, with the trailer.
- `kit policy` refuses a commit past the fork with no `Refs:` trailer naming an issue: `#7`, `AB#4521`, or several,
  apart by commas.
- A `Refs:` trailer that names another issue attaches it to the branch's pull request, as a subject's reference does.
- `kit linked` still requires the pull request to close an issue.

## Turning either half off

`kit.yml`'s `policy:` wins over the package. A repository that squash-merges cares about the title, which becomes
the commit on the default branch, and turns the subjects off:

```yaml
policy:
  commit_subject: ''
```

Each commit still carries its `Refs:` trailer. A repository that merges or rebases, and wants only the commits held,
sets the title back to the kit's default:

```yaml
policy:
  change_title: '^[^[:cntrl:]]{1,120}$'
```

A custom type list is a pattern of the repository's own in `kit.yml`. The package is opt-in: no bundle includes it.
`kit packages add policy/conventional` takes it.
