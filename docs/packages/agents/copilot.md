# The agents/copilot package

Adapts the kit for GitHub Copilot. Copilot reads `AGENTS.md` on its own, in the editor and as the coding agent, so
the neutral tree already reaches it. What the package adds is Copilot's native delivery of the rules: one instruction
file per kind under `.github/instructions/`, applied when a file it names is changed.

Each `.github/instructions/kit-<kind>.instructions.md` has the one frontmatter key Copilot reads, `applyTo`, set to
the `paths` of the kind, `.kit/kinds/<kind>.yml`, comma-separated in one quoted string, which is the documented
form. The body tells the agent to read `.kit/rules/<kind>.md` and follow it, as the `agents/cursor` rules do, so the
rule text is in one place. A rule that governs a file by its content, such as a bash script with no suffix, is
beyond a glob, so the body names `kit context <path>` for those.

On GitHub.com the files apply to the Copilot coding agent and to Copilot code review. VS Code attaches them in chat
and agent mode. Copilot offers no deny list a repository can commit, so the git commands `.kit/1st.md` forbids are
refused by the hooks and the ruleset alone.

`.github/instructions/` otherwise belongs to the repository. The package places only its seven files, `kit check`
holds them to the release, and `kit package remove agents/copilot` takes them out.

Verified on 2026-10-30 against GitHub's [repository custom instructions page][copilot-github] and VS Code's
[custom instructions page][copilot-vscode]; neither shows a version.

[copilot-github]: https://docs.github.com/en/copilot/how-tos/configure-custom-instructions/add-repository-instructions
[copilot-vscode]: https://code.visualstudio.com/docs/copilot/customization/custom-instructions
