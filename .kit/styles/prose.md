---
# Added by kit's style/prose package. To remove it, run kit package remove style/prose.
# To change it, do not edit it here: see "Changing what kit adds" in .kit/1st.md.
paths:
  - "**"
---
# Prose style

Applies to every file: the prose of a document and the comments in code and configuration alike. The style for
the file's kind, where the repository takes one, also applies.

1. Write plain declarative sentences, with the thing under discussion as the subject.
2. Describe in the indicative and instruct in the imperative. What a function, command, or file does is
   indicative: "`kit check` compares every delivered file with the pinned release." A step the reader takes is
   imperative: "Run `kit check` before you push." A comment that documents a function does both:
   `# use:find <module> prints every <module>.sh on the path. Call use:load-path first.`
3. Do not end a paragraph on a slogan. Do not restate what the code or config already states.
4. Use the serial comma in every list of three or more.
5. Expand an uncommon acronym the first time it appears.
