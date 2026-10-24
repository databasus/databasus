## Names and comments review

The candidate is the agent's final response, and the change is the working tree diff. Use `git status` and the full `git diff HEAD` including staged changes, and read new untracked files directly.

The standard for this review:

- the sections "Naming", "Functions, methods and types", "Comments are a last resort", "No issue numbers in production code" and "No "how it was" comments, no unrequested backward compatibility" of the root `AGENTS.md`, the last one only for its rule on comments;
- the naming and comment rules of the module doc for each changed area, including test naming.

The rules and the bad/good examples live in those documents. Compare names and comments against them, not against your own taste.

Check only names and comments the diff added or changed. Names are functions and methods, types, variables and fields, constants, props and files. Do not check a name or a comment the diff did not touch, even if it sits in a changed function or next to an edit.

A comment that restates what the code does is a naming problem: in the reason name the unit to rename and suggest the name that makes the comment redundant.

Do not check architecture, correctness, tests or prose. Other reviewers do that. In the reason name the file, the name or the comment, and the rule it breaks, and suggest a name or an edit that follows the rule.
