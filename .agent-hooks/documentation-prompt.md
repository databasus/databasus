## Documentation review

Review documentation changed by this request against "Documentation consistency" in `AGENTS.md` and the communication skill. Read the changed passages in context. Read code only to establish whether it already explains the detail or whether a procedure is necessary. `openspec/` and generated documents are outside this check.

Review only the documentation listed under "Currently changed paths". Other files are context, including pre-existing edits unless this request changed them again.

- Keep each piece of information with its authoritative owner. Shared `docs/` explain lasting concepts, boundaries, reasons and procedures people need. Implementation structure, helper catalogs and algorithms belong with their owning code only when the code needs that explanation. Dependency selections belong in their manifests and bundle records. Agent instructions hold lasting rules of agent behavior.
- Remove information readers can get directly from the code when it adds no necessary explanation. Do not move redundant prose wholesale into a local README. Prefer a link to the authoritative owner over a repeated description or table.
- Preserve actionable commands, configuration examples, architectural reasons, access and security constraints, recovery limitations and the boundaries of verified support. Technical detail is appropriate when a reader needs it to perform a task or understand a non-obvious constraint.
- Make the reader's task clear before explaining the mechanism. Introduce unfamiliar concepts, explain causes and consequences, group related steps and write sentences a junior developer can understand on the first read. Require the minimum sufficient explanation, not a fixed word or line limit.
- Check removals too: shortening must not hide a necessary prerequisite, break a procedure or remove a material limitation. Documentation must still agree with the code and its linked sources.

Block only concrete violations of the rules above, not stylistic preferences. For each problem, name the file and passage, the rule it violates and the action: delete it because the code already explains it, shorten it, link to its owner, or move the necessary explanation to a specific owner. Explain what the reader needs to understand or do. Return the existing `ok` and `reason` response, with the reason in English. Do not rewrite whole documents or request unrelated cleanup.
