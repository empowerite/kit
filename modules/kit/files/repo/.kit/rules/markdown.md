---
# Delivered by kit, not edited by hand. A change is a new release, then kit update.
paths:
  - "**/*.md"
lint: markdownlint-cli2
---
# Markdown

Applies to `*.md`.

## Prose

1. Write plain declarative sentences with the thing under discussion as the subject.
2. Describe the current state. Leave the history to git: no "previously", "used to", "replaced", or "ported from".
3. Use a list where the content is a list, and for a procedure or a loop, so it scans. Use a table where the
   content has columns.
4. Do not bold words for emphasis. Do not open a list item with a bolded term and a colon.
5. Do not close a paragraph on a slogan, and do not restate what the code or config already says.
6. Use the serial comma in every list of three or more.
7. Expand an uncommon acronym the first time it appears.
8. A pointer is `see: <ref>`, where `<ref>` is a URL, a repository-relative path, or `issue #N`. Put it inline
   when it keeps the sentence on one line, otherwise on its own line at the end of the paragraph.

## Snippets

1. A value the reader supplies is a shell variable the prose names, `"${signing_key}"`, never an angle-bracket
   placeholder, so the block pastes after one export.
2. Where a reader may lack a prerequisite, link the page that creates it beside the command that uses it.

## Layout

1. Wrap prose at 118 columns, 120 being the ceiling. Inside a list item, pull the right margin in by 4 columns
   per nesting level, to a limit of 12.
2. Balance the wrap: no line, in particular the last, reads as a stub.
3. Do not wrap fenced code, tables, HTML comments, or a line whose only content is a link.
4. Keep a list tight or loose as it was written. Reflow never flips one to the other.
5. Headings are ATX (`#`). One `#` heading per file, at the top; `AGENTS.md` is the exception, opening with the
   sentence a harness reads first.
6. Fenced code blocks declare a language and have a blank line on each side.
7. Table rows use ` | ` with spaces around each pipe, including the delimiter row.

## Links

1. Use inline links with descriptive text, never a bare URL in angle brackets.
2. Use the same anchor text every time the same URL appears in a file.
3. When a label and the link text would say the same thing, make the label the link.
4. A relative link resolves from the file that carries it, and it resolves.

## Files

1. Name an index file `1st.md`. It lists its siblings and says what each is for.
2. A file says what its subject is in its first paragraph, before how to use it.
3. A README has no Status section and no paragraph on what the repository will do or does not yet do. It says
   what the repository is.
4. A delivered file names no repository as the source of a rule.
