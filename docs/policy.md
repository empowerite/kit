# Policy

A policy is the set of patterns that a branch name, a commit subject, and a change's title must match. A change is
a pull request on GitHub or a merge request on GitLab. Run `kit policy` to check the current branch; it prints each
name that breaks the policy and fails.

```sh
kit policy
```

- `kit policy` gathers the facts once, then the policy engine reports every refusal.
- `kit ci` runs it beside the gate (the repository's own CI command), so a name that breaks the policy fails the
  delivered check like a failing test.
- On a runner, the refusals also appear in the step summary.

## The facts

`kit policy --input` prints the JSON document every engine reads:

```json
{
  "branch": "7-fix-the-thing",
  "default_branch": "main",
  "commits": [{"sha": "56d4f4b…", "subject": "#7 cli: fix the thing"}],
  "change": {"number": 3, "title": "Fix the thing", "body": "Closes #7"},
  "policy": {"branch": "…", "commit_subject": "…", "change_title": "…"}
}
```

| Field | Contents |
| --- | --- |
| `branch` | On a runner, the change's head branch. In the root checkout (the desk), the current branch. On the default branch or a detached checkout it is `null`, and nothing is checked. |
| `default_branch` | The repository's default branch. |
| `commits` | Each commit past the fork from the default branch at origin, merges excluded. The delivered workflow fetches full history for this. When the fork is not found, as in a shallow clone, no commit is checked and `kit policy` warns. |
| `change` | The change's number, title, and body, read through the forge (the service that hosts the repository). `null` when there is no forge or no change. |
| `policy` | The patterns, with the kit's defaults filled in. |

## The rules

`policy:` in `.kit/settings.yml` holds one extended regular expression per subject.

- A subject that is absent takes the kit's default: the workflow's own naming from `.kit/1st.md`.
- An empty pattern turns that rule off.
- A subject the kit does not know is refused.

| Subject | Checks | Default |
| --- | --- | --- |
| `branch` | the branch's name | `^[a-z0-9]+-[a-z0-9][a-z0-9-]*$`, the form `kit issue` creates |
| `commit_subject` | every commit's subject past the fork | `^([A-Z]+)?#[0-9]+ [^ ]`, starting with its issue's reference |
| `change_title` | the open change's title | `^[^[:cntrl:]]{1,120}$`, one line of at most 120 characters |

```yaml
policy:
  branch: '^(feature|fix)/[a-z0-9-]+$'
  commit_subject: ''
```

A single-quoted pattern is taken literally. Write patterns in the subset that bash's extended regular expressions
and OPA's RE2 share, so both engines read them the same way:

- character classes
- anchors
- alternation
- counted repetition

## The engines

| Engine | How it checks | Chosen by |
| --- | --- | --- |
| `patterns` | Matches each rule in bash. Needs no other tool. | Default. |
| `opa` | Open Policy Agent evaluates every `.rego` file under `.kit/policy/`, package `kit.policy`. The set `deny` of strings is the refusals. | `policy_engine: opa`, or any `.rego` file in `.kit/policy/`. |

The `opa` engine needs `opa`. Pin it in the repository's toolchain.

The `policy/opa` package delivers `.kit/policy/kit.rego`: the patterns engine's rules written in Rego, with the same
refusals. Because it is a `.rego` file, adding the package selects the `opa` engine, and the results stay the same.
Write a rule that a pattern cannot express in a separate `.rego` file beside it:

```rego
package kit.policy

import rego.v1

deny contains msg if {
    count(input.commits) > 20
    msg := sprintf("%d commits; split the change", [count(input.commits)])
}
```

`tests/policy.bats` checks that both engines give the same refusals for the same facts.
