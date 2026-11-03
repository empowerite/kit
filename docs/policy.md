# Policy

A policy is the set of rules that a branch name, a commit subject, a change's title, and the tree of files must
meet. A change is a pull request on GitHub or a merge request on GitLab. Run `kit policy` to check the current
branch: its name, its commits, its open change, and the tree at HEAD. It prints each refusal and fails, and it
prints each warning too.

```sh
kit policy
```

- `kit policy` gathers the facts once, then the policy engine reports every refusal and every warning.
- `kit ci` runs it beside the gate (the repository's own CI command), so a name that breaks the policy fails the
  delivered check like a failing test. That check blocks a merge only when a branch ruleset requires it.
- On a runner, the refusals and the warnings also appear in the step summary.

## The facts

`kit facts` prints the JSON document every engine reads. `kit policy --input` prints the same document.

`kit facts --moment=<moment>` chooses the point the facts are gathered for:

- `ci`, the default: the commits past the fork, the open change, and the tree at HEAD.
- `commit`: the tree in the index, `commits` as `[]`, and `change` as `null`. `kit commit` fills in the commit it makes.

```json
{
  "moment": "ci",
  "branch": "7-fix-the-thing",
  "default_branch": "main",
  "commits": [{"sha": "56d4f4b…", "subject": "#7 cli: fix the thing"}],
  "change": {"number": 3, "title": "Fix the thing", "body": "Closes #7", "author": "ada"},
  "tree": {
    "kit/bin/kit": {"type": "file", "executable": true, "size": 2048, "blob": "9f2c1a…", "kinds": ["bash"]}
  },
  "policy": {"branch": "…", "commit_subject": "…", "change_title": "…"}
}
```

| Field | Contents |
| --- | --- |
| `moment` | `commit` or `ci`: the point the facts are judged at. |
| `branch` | On a runner, the change's head branch. In the root checkout (the desk), the current branch. On the default branch or a detached checkout it is `null`, and nothing is checked. |
| `default_branch` | The repository's default branch. |
| `commits` | Each commit past the fork from the default branch at origin, merges excluded. The delivered workflow fetches full history for this. When the fork is not found, as in a shallow clone, `commits` is `null`, the commits cannot be checked, and `kit policy` fails. |
| `change` | The change's number, title, body, and author's login, read through the code forge (the service that hosts the repository). `null` when there is no code forge, no change, or the code forge could not be read, and `null` at commit. A `null` change leaves `change_title` unchecked. |
| `tree` | The files git holds, keyed by path, read from git: the index at commit, and HEAD at ci. At either moment, a kind matched by its first line, such as a script's shebang, reads that line from the disk. |
| `policy` | The patterns, with the kit's defaults filled in. |

Each entry in `tree` has these fields:

| Field | Contents |
| --- | --- |
| `type` | `file`, `link`, or `submodule`. |
| `executable` | `true` for a file of mode 100755, and `false` for everything else. |
| `size` | The blob's bytes, and 0 for a submodule. |
| `blob` | The blob's object name, or the commit a submodule is at. |
| `kinds` | The kinds of file the path is, such as `bash`, or `[]` for none. |
| `text` | The blob's content. Only a file matching a `facts.read` glob has it. |

### What kit commit judges

`kit commit` runs the policy before the commit exists. A plain `git commit` does not: until CI, only a harness's
deny list stops one.

- It judges the tree its own commit will make: HEAD plus the named paths, or the index when no paths are named.
- Its moment is `commit`, and `commits` holds the one commit it makes.
- When the engine errors, because `opa` is missing or a `.rego` file is broken, it refuses the commit. It fails closed.
- The engine and the patterns are read from the tree being judged, so an unstaged edit to `kit.yml` or
  `.kit/policy/` does not change the verdict.
- Warnings print, and the commit still goes through.

### Reading a file's text

A rule that judges content needs `text`. Name the files in `kit.yml` by glob:

```yaml
facts:
  read:
    - README.md
```

- A file's `text` is capped at 65536 bytes.
- A binary file or an oversized file has no `text`, and kit warns, naming it.
- A rule should treat a missing `text` as unknown, not as empty.

### Testing a rule

`kit policy --facts=<file>` judges a saved facts document, and does not gather any facts.

```sh
kit facts > fixture.json
kit policy --facts=fixture.json
```

Edit the fixture to test a rule against facts that the repository does not have.

- A relative `<file>` starts from the directory the command was typed in.
- A file that is not a facts document, a JSON object with a `policy` key, is refused.

## The rules

`policy:` in `kit.yml` holds one extended regular expression per subject.

- A subject that is absent takes the kit's default: the workflow's own naming from `.kit/1st.md`.
- An empty pattern turns that rule off.
- A subject the kit does not know is refused.
- A change whose author is a login under `linked_exempt` in `kit.yml` passes with no rule checked, the tree rules
  included, and `kit policy` says so by name. Its branch, its commits, and its title are a bot's, such as
  `dependabot[bot]`'s. The same setting passes the change through `kit linked`; see
  [packages.md](packages.md#settings).

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
| `opa` | Open Policy Agent evaluates every `.rego` file under `.kit/policy/`, package `kit.policy`. The set `deny` of strings is the refusals. | `policy_engine: opa`, or any `.rego` file in `.kit/policy/` when `policy_engine` is not set. |

The `opa` engine needs `opa`. Pin it in the repository's toolchain, then run `kit tools install`.

The `policy/opa` package delivers `.kit/policy/kit.rego`: the patterns engine's rules written in Rego, with the same
refusals. Because it is a `.rego` file, adding the package selects the `opa` engine, and the results stay the same.

Rules over `tree` need `opa`. The `patterns` engine ignores `tree`.

### Refusals and warnings

The engine reads two sets of strings from package `kit.policy`:

- `deny` holds refusals. Any one fails `kit policy`. Rego that does not define `deny` at all, as in a misspelled
  package, is refused too, since nothing was judged.
- `warn` holds warnings. They are printed and never fail anything.

### A rule of your own, through a local package

Rego that belongs to one repository ships through a local package, as templates do. Put the file in the package's
source, and kit delivers it as a link:

```text
.kit/packages/house/files/repo/.kit/policy/house.rego   the source
.kit/policy/house.rego                                  delivered by kit, as a link
```

- The package must list `policy/opa` under `depends:`.
- Without it, `kit.rego` is missing. Any `.rego` file selects `opa`, so the naming rules would silently stop.
- `kit packages add` brings in what a package depends on. `kit hydrate` does not: it reads only `packages:` in
  `kit.yml`.
- So a package that is already active, and gains the line, needs one more step. Run `kit packages add policy/opa`,
  or list `policy/opa` under `packages:` in `kit.yml`.

```yaml
description: "This repository's own tree rules"
depends:
  - policy/opa
```

This repository's `house` package holds a rule over `tree`. It refuses a file in `kit/bin/` that is not executable:

```rego
package kit.policy

import rego.v1

bin := "kit/bin/"

# The name after kit/bin/, for a file directly in it.
bin_name(path) := name if {
    startswith(path, bin)
    name := substring(path, count(bin), -1)
    not contains(name, "/")
}

deny contains sprintf("%s is not executable", [path]) if {
    some path, f in input.tree
    bin_name(path)
    not f.executable
}
```

A rule can also judge the commits:

```rego
package kit.policy

import rego.v1

deny contains msg if {
    count(input.commits) > 20
    msg := sprintf("%d commits; split the change", [count(input.commits)])
}
```

`tests/policy.bats` checks that both engines give the same refusals for the same facts.
