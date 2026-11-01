# Completion

`kit completion zsh` prints the zsh completion for kit. It completes:

- the verbs, each with its purpose
- a verb's flags and options
- its literal words, such as `add` after `kit packages`
- the values a verb lists for an argument, such as the packages in reach for `kit packages add` or the rules for
  `kit new file --rule=`
- the files, for a path

A `kit-<verb>` command called directly completes the same way.

## Install it

Put the script on zsh's `fpath` as `_kit`, which is where a package manager puts one:

```bash
kit completion zsh > "${completion_dir}/_kit"
```

Or evaluate it from `.zshrc`, after `compinit`:

```bash
eval "$(kit completion zsh)"
```

## How it works

The script knows no verb. On every completion it runs `kit completion candidates` with the words typed so far, and
offers what comes back. A candidate is one line: the word, a tab, and its description, with `:files` when files fit
as well. The candidates come from the verbs themselves:

- the verbs and their purposes from their headers, as `kit help` reads them
- a verb's flags, options, literal words, and operands from `kit syntax`, the grammar of its usage lines
- an argument's values from the choices the verb's module lists, as `kit syntax <verb> --choices` prints them

So a new verb, flag, or choice completes with no edit to the script. A completion takes as long as `kit syntax` for
one verb, a fraction of a second.

see: [modules/bash/completion.sh](../kit/modules/bash/completion.sh),
[modules/jq/completion.jq](../kit/modules/jq/completion.jq)
