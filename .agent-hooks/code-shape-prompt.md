## Code shape review

The candidate is the agent's final response, and the change is the working tree diff. Use `git status` and the full `git diff HEAD` including staged changes, and read new untracked files directly.

The standard for this review:

- the sections "Naming", "Functions, methods and types", "Code is split into logical units", "Blank lines separate logical steps", "Comments are a last resort", "No issue numbers in production code" and "No "how it was" comments, no unrequested backward compatibility" of the root `AGENTS.md`, the last one only for its rule on comments;
- the naming, comment, blank-line and file rules of the module doc for each changed area, including test naming.

The rules and the bad/good examples live in those documents. Compare the code against them, not against your own taste.

### Decomposition

Read every source file the turn created in full, and the added parts of every changed file. Reject a new file or a new function that holds more than one concept, and a concept the turn added to an existing file that holds a different one. In the reason name the file or the unit, the concepts it mixes, the rule it breaks, and the files or functions the split should produce.

### Blank lines

Review the code the diff added or changed, including test code, against "Blank lines separate logical steps" in the root `AGENTS.md`. Read surrounding statements to identify the steps and their boundaries, but do not report pre-existing spacing the diff did not affect. Report missing or misplaced blank lines by naming the distinct steps that should be separated or the related statements that should stay together, and specify where to add or remove a blank line.

### Names and comments

Check only names and comments the diff added or changed. Names are functions and methods, types, variables and fields, constants, props and files. Do not check a name or a comment the diff did not touch, even if it sits in a changed function or next to an edit.

A comment that restates what the code does is a naming problem: in the reason name the unit to rename and suggest the name that makes the comment redundant.

### Boundary

Do not judge which package, module, layer or role a unit belongs to; the architecture check does that. Do not reject an addition to a file whose content `docs/architecture.md` or a module doc fixes: an event or a contract type in `module.go`, an error in `errors.go`, an enum next to its entity, a method in its role file. Do not judge concepts that were already mixed in a file before the turn.

Do not check correctness, test coverage or prose. Other reviewers do that. Test code is subject to the same decomposition, blank-line, naming and comment checks as production code. In the reason name the file, the unit, the statements, the name or the comment, and the rule it breaks, and suggest a split, a name or an edit that follows the rule.
