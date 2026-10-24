## Diff review

The candidate is the agent's final response, and the change is the working tree diff.

Use `git status` and the full `git diff HEAD` including staged changes, and read new untracked files directly.

- Check that the diff is focused, technically sound, complete, and matches the latest user request and the accepted plan, if there was one.
- Look for bugs: wrong behavior, missed edge cases, broken error handling, races, resource leaks, security problems.
- Check the material claims of the candidate's response against the repository, including claims that tests and linters ran and passed. A claim that cannot be true for this diff is a problem.
- Check tests against `.agents/skills/testing/SKILL.md` and the testing rules of the module docs: new behavior has a test at the level the module doc prefers, test names follow the module doc, tests wait for conditions instead of fixed pauses, and test data is cleaned up.
- Check that code, comments, log messages, API strings and documentation are in English, except the places "Language in code" in `AGENTS.md` lists.
- Check documentation consistency per the section "Documentation consistency" in `AGENTS.md`: if the change touches a command, rule, path or order of steps that `AGENTS.md`, the module docs, `docs/`, the repository's own skills or a README describe, those places are updated too. Search the repository for mentions of what changed. `openspec/` is outside this check.
- Check edits to agent instructions per the same section. If an edit adds to `AGENTS.md` or an own skill an inventory of what exists, the state of the code, a description of how something works or a retelling of another document, return `ok: false` and name the place to move it to from that section's table. Do not block an edit that adds or changes a rule of agent behavior under this point.
- Apply the humanizer rules to all prose in the diff and in the candidate's response.
- Architecture and names are checked by separate reviewers. Do not duplicate them.
