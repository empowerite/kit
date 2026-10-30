# Templates

A template is a file of one kind with holes in it, attached to the style for that kind. `kit new file` fills the
holes to make a new file, and `kit lint` checks that every existing file still has the shape of one of its
templates. A style package delivers the templates for its kind, so a repository that takes no style package for a
kind has no templates for it.

```sh
kit new file "${path}" --purpose="${purpose}"
```

## Where a template lives

A style file, `.kit/styles/<kind>.md`, lists its templates in its frontmatter, in the order they are tried. Each
template is a variant of the file, stored at `.kit/templates/<kind>/<variant>`. A variant can be limited to one
path glob (`path:`) or marked executable (`executable: true`):

```yaml
templates:
  - variant: root
    path: justfile
  - variant: module
```

| Style | Variant | Limited to | Notes |
| --- | --- | --- | --- |
| bash | `script` | | executable |
| bash | `module` | | a sourced file |
| just | `root` | `justfile` | |
| just | `module` | | |
| markdown | `agents` | `AGENTS.md` | |
| markdown | `document` | | |
| markdown | `notice` | | |
| markdown | `frontmatter` | | |
| yaml | `document` | | |
| workflows | `workflow` | | |
| gitignore | `file` | | |
| gitignore | `region` | `.gitignore` | a file the kit made of its region alone |

JSON style has no template. A JSON file has no comments and no fixed opening, so there is no shape to check.

To change a template, write a package of your own under `.kit/packages/` that depends on the style package that
delivers it, and place a file at the same path. The later package in dependency order wins. This repository does
that with its `house` package, `.kit/packages/house/`, whose bash, gitignore, and just templates add a line of
dashes under the banner and at the end of the file. The templates the kit delivers have none.
see: [packages.md](packages.md)

## Holes

A hole is a name between guillemets, `«name»`. No file kind the rules govern uses guillemets.

| Hole | When making a file | When checking a file |
| --- | --- | --- |
| Within a line | Replaced by a value. | Matches any text on that line. A hole used twice must match the same text each time. |
| A line of its own | Replaced by the value's lines. With no value, one blank line, or nothing when it is the template's last line. | Matches any run of lines, including none. |

For example, the just module template's `«stem»_justfile` appears twice, because it is the variable its help
recipe reads.

`path`, `name`, and `stem` come from the path. For `bin/deploy.sh` they are `bin/deploy.sh`, `deploy.sh`, and
`deploy`. Every other hole within a line takes its value from the flag of the same name, such as `--purpose=<…>`
or `--title=<…>`. A missing flag is refused, and the error names it.

## Making a file

```sh
kit new file "${path}" [--rule="${rule}"] [--variant="${variant}"] [--"${hole}"="${value}"]…
```

- The path is relative to the repository root, wherever you type the command. A path outside the repository, or
  one that already exists, is refused. Nothing is overwritten.
- The template comes from the most specific style that governs the path and has templates: the one whose matching
  glob has the most characters other than wildcards. So `.github/workflows/check.yml` gets the workflows style,
  not the YAML style. Use `--rule` when the path alone cannot select one, such as a script with no extension.
- The variant is the first one the style lists that the path allows. Use `--variant` to pick another.
- An executable variant is made executable.
- The command then prints the rules to read before you write the rest, as `kit context` does, and records the call
  in the context log.

A new file is a starting point, not a finished file. For example, a workflow made from its template has empty
`on:`, `permissions:`, and `jobs:` keys, and actionlint rejects it until you fill them in.

## Checking a file

For each path governed by a style with templates, `kit lint` checks that the file has the shape of one variant its
path allows. It does this in addition to running the linters of the rule and the style. When a file matches no
variant, `kit lint` reports:

- the variants it was checked against
- the variant it follows furthest, and the line of that variant where the file stops matching

```text
❌ bin/release.sh does not have the shape of the bash template (script, module); it leaves .kit/templates/bash/module line 2: # «name» - «purpose»
```

Three kinds of path are not checked against any template:

- a template itself
- a package's region, which is a fragment of a file
- a path that `.gitattributes` marks `linguist-generated`, because a tool writes it

```text
/.mise/locks/** linguist-generated
```

A template fixes the shape a style requires: the lines a file must have, and their order. What varies line by
line, such as the inside of a recipe or a function, is left to the linters and to the text of the rule and the
style.
