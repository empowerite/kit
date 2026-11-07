# kit:generated -- the naming policy in Rego, which kit policy runs on Open Policy Agent, from the policy/opa package
# see: kit packages remove policy/opa, or "Changing what kit adds" in .kit/1st.md
# Put the repository's own rules in a local package that delivers a .rego file, in package kit.policy, adding to deny.
# `kit facts` prints the input; see docs/policy.md.

package kit.policy

import rego.v1

deny contains msg if {
	input.branch != null
	pattern := input.policy.branch
	pattern != ""
	not regex.match(pattern, input.branch)
	msg := sprintf("branch '%s' does not match policy.branch: %s", [input.branch, pattern])
}

deny contains msg if {
	pattern := input.policy.commit_subject
	pattern != ""
	some commit in input.commits
	not regex.match(pattern, commit.subject)
	msg := sprintf("commit %s '%s' does not match policy.commit_subject: %s",
		[substring(commit.sha, 0, 7), commit.subject, pattern])
}

deny contains msg if {
	input.change != null
	pattern := input.policy.change_title
	pattern != ""
	not regex.match(pattern, input.change.title)
	msg := sprintf("change title '%s' does not match policy.change_title: %s", [input.change.title, pattern])
}
