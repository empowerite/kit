# Policy

What a branch, a commit subject, or a change's title may be is a policy, and a policy is checked, not described.
`kit policy` gathers the facts once and an engine names every refusal; `kit ci` runs it beside the gate, so a name
off the policy fails the delivered check like a failing test.

## The facts

`kit policy --input` prints the document every engine reads:

```json
{
  "branch": "7-fix-the-thing",
  "default_branch": "main",
  "commits": [{"sha": "56d4f4b…", "subject": "#7 cli: fix the thing"}],
  "change": {"number": 3, "title": "Fix the thing", "body": "Closes #7"},
  "policy": {"branch": "…", "commit_subject": "…", "change_title": "…"}
}
```

The branch is the change's head on a runner and the current branch on a desk; on the default branch, or detached,
nothing is under check. The commits are those past the fork from the default branch at origin, merges aside, which
is why the delivered workflow fetches full history. The change is read through the forge, and is `null` where no
forge or no change answers.

## The rules

`policy:` in `.kit/settings.yml` holds one extended regular expression per subject. A subject absent takes the
kit's default, the loop's own names from `.kit/1st.md`; an empty pattern turns that rule off.

| Subject | Checks | Default |
| --- | --- | --- |
| `branch` | the branch's name | `^[a-z0-9]+-[a-z0-9][a-z0-9-]*$`, as `kit issue` makes it |
| `commit_subject` | every commit's subject past the fork | `^([A-Z]+)?#[0-9]+ [^ ]`, opening with its issue's reference |
| `change_title` | the open change's title | `^[^[:cntrl:]]{1,120}$`, one line of at most 120 |

```yaml
policy:
  branch: '^(feature|fix)/[a-z0-9-]+$'
  commit_subject: ''
```

A single-quoted pattern is taken literally. Write in the subset bash's extended regular expressions and OPA's RE2
share, character classes, anchors, alternation, and counted repetition, so either engine reads it the same way.

## The engines

- `patterns`, the default, matches each rule in bash and needs no tool.
- `opa`, Open Policy Agent, evaluates every `.rego` file under `.kit/policy/`, package `kit.policy`, the set
  `deny` of strings being the refusals. It is chosen by `policy_engine: opa`, or by a `.rego` file being there. The
  `policy-opa` module delivers `.kit/policy/kit.rego`, the patterns engine's rules in Rego, refusal for refusal, so
  taking it changes nothing until the repository adds a rule a pattern cannot say in a file of its own beside it.
  The repository pins `opa` in its toolchain.

```rego
package kit.policy

import rego.v1

deny contains msg if {
    count(input.commits) > 20
    msg := sprintf("%d commits; split the change", [count(input.commits)])
}
```

`tests/policy.bats` holds the two engines to the same refusals on the same facts.
