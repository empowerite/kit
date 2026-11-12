# Templates

A template is a file of one kind with holes in it, attached to the rule or the style for that kind. `kit new file`
fills the holes to make a new file, and `kit lint` checks that every existing file still has the shape of one of
its templates. A style package delivers the templates for its kind; without that package, a kind has no style
templates. The kit package delivers five of its own, `.kit/templates/bash/<seam>-provider`, which the bash rule
lists: they hold a seam's providers to its calls in every repository.

```sh
kit new file "${path}" --purpose="${purpose}"
```

## Where a template lives

A style file, `.kit/styles/<kind>.md`, or a rule file, `.kit/rules/<kind>.md`, lists its templates in its
frontmatter, in the order they are tried. Each template is a variant of the file, stored at
`.kit/templates/<kind>/<variant>`. A variant can be limited to one path glob (`path:`) or marked executable
(`executable: true`):

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
| markdown | `document` | | |
| markdown | `notice` | | |
| markdown | `frontmatter` | | |
| yaml | `document` | | |
| workflows | `workflow` | | |
| gitignore | `file` | | |
| gitignore | `region` | `.gitignore` | a file the kit made of its region alone |
| bash (the rule) | `dependencies-provider`, `forge-provider`, `policy-provider`, `toolchain-provider`, `tracker-provider` | `**/<seam>/*.sh` | a seam's provider; see below |

JSON style has no template. A JSON file has no comments and no fixed opening, so there is no shape to check.

To change a template, write a package of your own under `.kit/packages/` that depends on the style package that
delivers it, and place a file at the same path. The later package in dependency order wins. This repository does
that with its `house` package, `.kit/packages/house/`: its bash, gitignore, and just templates add a line of dashes
under the banner and at the end of the file. The templates the kit delivers have none; see
[packages.md](packages.md).

## Holes

A hole is a name between guillemets, `«name»`, with `[a-z_]+` for the name, or a dotted path of such names. No
file kind the rules govern uses guillemets. On a Mac, type `«` with Option-\ and `»` with Shift-Option-\, and `…`
with Option-;. Without them, type `<<name>>`, `<<name...>>`, or `<<name=default>>`, and `kit template fix <path>`
rewrites them.

| Hole | When making a file | When checking a file |
| --- | --- | --- |
| `«name»` within a line | Replaced by a value. | Matches any text on that line. |
| `«name…»` | Replaced by a value. | Matches any text, across lines. |
| `«name»` on a line of its own | Replaced by the value's lines. With no value, one blank line, or nothing when it is the template's last line. | Matches any run of whole lines, including none. |
| `«name=default»`, `«name…=default»` | The value, else the default; `\n` in a default is a newline. | As without the default. |
| `<prefix>§name` on a line of its own, a block | Each of the value's lines after the prefix. With no value, no lines. | Matches the lines that start with the prefix, and the lines the prefix starts with, up to the last that starts with it. Its value is those lines without the prefix. |
| `<prefix>§name=default` | The value's lines, else the default's. | As without the default. |

- A name may be a dotted path in `kit.yml`, such as `«answers.ci_runner»`, which is `ci_runner` under
  `answers:`. Making a file fills it with that setting's value, and `kit new file` refuses when the settings leave it
  unset and it has no default. Checking a file binds it as it binds `path`: the setting's value is literal text.
- `path`, `name`, and `stem` come from the path. For `bin/deploy.sh` they are `bin/deploy.sh`, `deploy.sh`, and
  `deploy`. They are bound before a file is checked, so they are literal text there: a script whose banner names
  another file does not match.
- A name that appears once takes one value. A name that appears more than once takes one value per appearance, in
  template order: the three `«settings»` holes of the just templates are three values. Making a file with one value
  fills every appearance.
- Every other hole within a line, or across lines, without a default takes its value from the flag of the same name,
  such as `--purpose=<…>` or `--title=<…>`. A missing flag is refused, and the error names it.
  `kit tui` offers those flags as fields: `kit syntax new --extend` lists them once the rule and the variant are
  known.
- A region in a template, the lines between a `kit:generate` directive and its `kit:end` line, belongs to its
  command. A file may hold anything there, and `kit new file` keeps the directive and fills the region with
  `kit generate`.
- Two holes with nothing between them could split a file any way at all, so `kit lint` refuses such a template,
  naming its line. A hole may follow a block, which ends where its indentation ends.

### Blocks

A block reads an indented section, such as a script header's argument list, and gives it back without its
indentation. The prefix is the text before the `§`, and a block needs one. On a Mac, type `§` with Option-6.

```text
# Arguments:
#   §arguments
#
# see: «see»
```

Against that template, the lines `#   --json   one JSON object per row`, `#`, and `#   --jq     the same JSON` are
the block, and `arguments` is `--json   one JSON object per row`, an empty line, and `--jq     the same JSON`. The
bare `#` between them is a line the prefix starts with, so it stays in the block; the bare `#` after the last is
left for the template's own `#` line. A block may hold no lines, so a script with no arguments matches too. A block
also ends at one of the template's own lines, as below, so `# §description` ends at `# Arguments:` though that line
starts with the block's prefix too.

Making a file puts the prefix back on each line, and an empty line becomes the prefix without its trailing spaces,
`#` here. So a block's line that holds less than the prefix comes back as that, which is how such a line is written.

### Where a hole ends

A template is matched in one pass, with no regular expression. A block ends where its indentation ends. Every other
hole extends to the first place the text after it appears, which leaves the most for what follows. So a file
matches exactly when this pass finds the match, and the time is linear in the file. The same pass gives each hole's
text, and `template:unapply <template> <file>` prints those values as JSON. Rendering a template with the values a
file gives it makes that file again; the suite checks this for every file in this repository that a template
governs.

A hole's text never holds one of the template's own whole lines that has a letter or digit, such as a section's
label. Such a line inside a section would end the section there, and the rest of the file could still match with
the label inside the next section, read wrong without a word. So the file does not match, at the hole's line. Lines
of punctuation alone, a bare `#` or a line of dashes, are left free, and so is a region's text. Inside a block, a
label is indented, so it is the block's text, not the template's line.

## Making a file

```sh
kit new file "${path}" [--rule="${rule}"] [--variant="${variant}"] [--"${hole}"="${value}"]…
```

- The path is relative to the repository root, wherever you type the command. A path outside the repository, or
  one that already exists, is refused. Nothing is overwritten.
- The template comes from the most specific rule or style that governs the path and has templates: the one whose
  matching glob has the most characters other than wildcards. So `.github/workflows/check.yml` gets the workflows
  style, not the YAML style. Use `--rule` when the path alone cannot select one, such as a script with no extension.
- The variant is the first one the style lists that the path allows. Use `--variant` to pick another.
- An executable variant is made executable.
- The command then prints the rules to read before you write the rest, as `kit context` does, and records the call
  in the context log.

A new file is a starting point, not a finished file. For example, a workflow made from its template has empty
`on:`, `permissions:`, and `jobs:` keys, and actionlint rejects it until you fill them in.

## Making the rest

`kit new` makes the other things a repository extends the kit with, each in its place and its house shape, and never
overwrites anything:

| Command | Makes |
| --- | --- |
| `kit new kind "${name}" --path="${glob}"` | a kind of file, by `name:` |
| `kit new rule "${kind}" [--lint="${command}"]` | a rule for a kind, with the frontmatter that names it |
| `kit new style "${kind}" [--lint="${command}"]` | a style for a kind |
| `kit new template "${rule}" "${variant}" [--from="${path}"]` | a template, listed under `templates:` in its package's rule or style |
| `kit new package "${name}" [--depends="${package}"]…` | `.kit/packages/<name>/package.yml`, depending on `kit` |
| `kit new module "${name}" --purpose="${purpose}"` | a bash module in the repository's own directory on the module path |
| `kit new provider "${seam}" "${name}"` | a provider of a seam, with one function for each call the seam makes, each refusing until it is written |
| `kit new verb "${name}" --purpose="${purpose}" --group="${group}"` | a verb of the kit, in the kit's own source alone |
| `kit new module "${name}" --jq --purpose="${purpose}"` | a jq module and its tests, in the kit's own source alone |

- A kind, a rule, a style, and a template go into a package of the repository's own: the one named with
  `--package`, else the repository's one package under `.kit/packages/`, else `local`, made when there is none.
- When that package is active, the new file is placed at once. Otherwise the command names `kit packages add`.
- A module and a provider go into the repository's first directory on the module path, the `bash_path` setting.
  They are made from the bash style's `module` template where the repository has one.

## Checking a file

For each path governed by a rule or a style with templates, `kit lint` checks that the file has the shape of one
variant its path allows. It does this in addition to running the linters of the rule and the style. When a file
matches no variant, `kit lint` reports:

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
/tools.lock linguist-generated
```

A seam's provider is checked against its seam's template too, which the bash rule lists for `**/<seam>/*.sh`. The
shape is the seam's calls first, one function each in the order of their names; then the line
`# The seam's calls end here; what follows is this provider's own.`; then the provider's own functions. A provider
that lacks a call, renames one, or puts one of its own among them leaves that template.

A template fixes the shape a style requires: the lines a file must have, and their order. What varies line by
line, such as the inside of a recipe or a function, is left to the linters and to the text of the rule and the
style.
