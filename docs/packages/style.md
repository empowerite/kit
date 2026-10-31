# The style packages

Eight optional packages: one per kind of file, each delivering the style for its kind, and `style/prose`, for the
prose in every text file. Style is how a file looks, as opposed to what the work must do, which is the rule's. A
repository takes the ones it agrees with.

| Package | Delivers |
| --- | --- |
| `style/bash` | `.kit/styles/bash.md`; the `script` and `module` templates under `.kit/templates/bash/` |
| `style/markdown` | `.kit/styles/markdown.md`; four templates under `.kit/templates/markdown/`; `.kit/markdownlint.jsonc` |
| `style/just` | `.kit/styles/just.md`; the `root` and `module` templates under `.kit/templates/just/` |
| `style/workflows` | `.kit/styles/workflows.md`; the `workflow` template under `.kit/templates/workflows/` |
| `style/yaml` | `.kit/styles/yaml.md`; the `document` template under `.kit/templates/yaml/` |
| `style/json` | `.kit/styles/json.md` |
| `style/gitignore` | `.kit/styles/gitignore.md`; the `file` and `region` templates under `.kit/templates/gitignore/` |
| `style/prose` | `.kit/styles/prose.md`, which governs every file of kind `text`: plain sentences, the indicative to describe, the imperative to instruct |

Each depends only on `kit`. The `empowerite` bundle includes all eight.

- `kit context` prints a style file after the rule for the same kind, so an agent reads both.
- `kit lint` runs a style's linter, and holds a file to the shape of one of its style's templates.
- `kit new file` makes a file from a style's template. With no style package for a kind, there is no template for
  it, and the file is written by hand.

`style/markdown` is the one that brings a linter: `markdownlint-cli2`, configured by `.kit/markdownlint.jsonc`. The
repository pins `markdownlint-cli2` like any other tool its gate runs.

A repository changes a style without editing the delivered file: a package of its own under `.kit/packages/`
depends on the style package and places its own file at the same path. The later package in dependency order wins.
See [templates.md](../templates.md) for an example.
