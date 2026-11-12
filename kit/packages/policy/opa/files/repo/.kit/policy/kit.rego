# kit:generated -- the naming policy in Rego, which kit policy runs on Open Policy Agent, from the policy/opa package
# see: kit packages remove policy/opa, or "Changing what kit adds" in .kit/1st.md
# Put the repository's own rules in a local package that delivers a .rego file, in package kit.policy, adding to deny.
# `kit facts` prints the input; see docs/policy.md.

package kit.policy

import rego.v1

# remedy(rule) is the rule's remedy in parentheses, after a space, where input.policy.remedy has one, and "" where not.
remedy(rule) := sprintf(" (%s)", [text]) if {
	text := object.get(object.get(input.policy, "remedy", {}), rule, "")
	is_string(text)
	text != ""
} else := ""

deny contains msg if {
	input.branch != null
	pattern := object.get(input.policy, "branch", "")
	pattern != ""
	not regex.match(pattern, input.branch)
	msg := sprintf("branch '%s' does not match policy.branch: %s%s", [input.branch, pattern, remedy("branch")])
}

deny contains msg if {
	object.get(input, "worktree", null) != null
	pattern := object.get(input.policy, "worktree", "")
	pattern != ""
	not regex.match(pattern, input.worktree)
	msg := sprintf("worktree '%s' does not match policy.worktree: %s%s", [input.worktree, pattern, remedy("worktree")])
}

deny contains msg if {
	pattern := object.get(input.policy, "commit_subject", "")
	pattern != ""
	some commit in input.commits
	not regex.match(pattern, commit.subject)
	msg := sprintf(
		"commit %s '%s' does not match policy.commit_subject: %s%s",
		[substring(commit.sha, 0, 7), commit.subject, pattern, remedy("commit_subject")],
	)
}

deny contains msg if {
	input.change != null
	pattern := object.get(input.policy, "change_title", "")
	pattern != ""
	not regex.match(pattern, input.change.title)
	msg := sprintf(
		"change title '%s' does not match policy.change_title: %s%s",
		[input.change.title, pattern, remedy("change_title")],
	)
}
