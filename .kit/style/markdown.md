---
# Added by kit's style/markdown package. To remove it, run kit package remove style/markdown.
# To change it, do not edit it here: see "Changing what kit adds" in .kit/1st.md.
paths:
  - "**/*.md"
lint: markdownlint-cli2 --config .kit/markdownlint.jsonc
templates:
  - variant: agents
    path: AGENTS.md
  - variant: document
  - variant: notice
  - variant: frontmatter
---
# Markdown style

Applies to `*.md`. The rule, `.kit/rules/markdown.md`, also applies.

## Prose

1. Write plain declarative sentences, with the thing under discussion as the subject.
2. Describe in the indicative and instruct in the imperative. A description of what a function, command, or
   file does is indicative: "`kit check` compares every delivered file with the pinned release." A step the reader
   takes, such as how to call it, is imperative: "Run `kit check` before you push."
3. Use a list for a list, a procedure, or a workflow, so it scans. Use a table where the content has columns.
4. Do not bold words for emphasis.
5. Do not end a paragraph on a slogan. Do not restate what the code or config already states.
6. Use the serial comma in every list of three or more.
7. Expand an uncommon acronym the first time it appears.
8. A pointer is `see: <ref>`, where `<ref>` is a URL, a repository-relative path, or `issue #N`. Put it inline
   when it keeps the sentence on one line. Otherwise put it on its own line at the end of the paragraph.

## Snippets

1. Write a value the reader supplies as a shell variable that the prose names, such as `"${signing_key}"`, never
   as an angle-bracket placeholder. The block then works when pasted after one `export`.
2. Where a reader may lack a prerequisite, link the page that creates it beside the command that uses it.

## Layout

1. Wrap prose at 118 columns, with 120 as the ceiling. Inside a list item, pull the right margin in by 4
   columns per nesting level, to a limit of 12.
2. Balance the wrap: no line, especially the last, is a short stub.
3. Do not wrap fenced code, tables, HTML comments, or a line that contains only a link.
4. Keep a list tight or loose as it was written. Reflowing never changes one into the other.
5. Table rows use ` | ` with spaces around each pipe, including the delimiter row.

## Links

1. Use inline links with descriptive text, never a bare URL in angle brackets.
2. Use the same link text every time the same URL appears in a file.
3. When a label and the link text would be the same, make the label the link.

## Files

1. Name an index file `1st.md`. It lists its siblings and states what each is for.
2. The first paragraph of a file states what its subject is, before explaining how to use it.
3. A README states what the repository is. It has no Status section and no paragraph on what the repository
   will do or does not yet do.
