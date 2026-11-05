## Full implementation review

The candidate is the implementation of an OpenSpec change whose tasks in `tasks.md` are all checked. Read the change's `proposal.md`, `design.md`, `tasks.md` and delta specs. The implementing agent must finish linting and the relevant tests before review. This review does not run linters, builds or tests and does not certify that they passed. Check the agent's verification claims against the available evidence and the repository.

An implementation usually spans several requests, so the shared rule about paths changed before the latest request does not apply here: review everything that belongs to the change. Part of the implementation may already be committed; find it with `git log` and `git show` over the paths from the specs and `design.md`. Do not review edits unrelated to the change.

This review answers with a `findings` list instead of the `ok` and `reason` from the shared instructions. Every finding has a category, a path and a description:

- `bug`: wrong behavior, a race between a check and a write, a transaction error, secrets or personal data in logs or error texts, a wrong response code;
- `spec_mismatch`: the code differs from a requirement, a scenario or a decision in `design.md`;
- `missing_test`: a scenario has no test, or the test passes without the described behavior;
- `architecture`: the code breaks `docs/architecture.md` or the architecture rules of the module docs, or an edit of an HTTP contract file under `backend/api/` breaks the `api-design` skill;
- `docs`: a task is not fully done, documentation disagrees with the code or with other documents, or an edit to agent instructions breaks the section "Documentation consistency" in `AGENTS.md`;
- `minor`: a detail that does not stop the change from being accepted.

In `file` write the path and, if you can, the line: `path:line`. In `summary` describe the problem and the fix in English. Return only problems, without retelling what is fine. If there are no problems, return an empty list.

### Role: delta spec

Review only the requirements of your delta spec.

- For every requirement and scenario find the code and the test: the file and the test name. Check that the test proves the scenario rather than passing by accident.
- Check the decisions in `design.md` that relate to these requirements against the code.
- Look for bugs in the code that implements the requirements.

### Role: architecture

Read `docs/architecture.md` completely and the architecture sections of the module docs that `.agent-hooks/architecture-prompt.md` lists.

- Check all code that belongs to the change against these documents, including parts committed or written in earlier requests: layer direction, module boundaries, roles of files inside a module, data ownership and migrations.
- Read the edits of the change together. A violation that appears only across several edits, such as an import added in one request and used across a module boundary in another, is an `architecture` finding that names every path involved and the rule in `docs/architecture.md`.
- When the change edits an HTTP contract file under `backend/api/`, read `.agents/skills/api-design/SKILL.md` and check the edit against it. Each broken rule is an `architecture` finding that names the rule of the `api-design` skill.
- Check all backend files belonging to the change against the architecture rules, including every command and code outside the former migration layout. Frontend architecture rules remain applicable.
- Do not check names, style, prose or general correctness. Other reviewers do that.

### Role: documentation and tasks

- For every task in `tasks.md` check that everything it says was done, including its verification. A claim that cannot be true for this code is a problem. Running `mise run test` is never a finding.
- Check documentation consistency per the section "Documentation consistency" in `AGENTS.md`: `AGENTS.md`, the module docs, `docs/`, the repository's own skills, the READMEs and code comments agree with each other and with the code. Search the repository for mentions of what the change changed.
- Check the edits to agent instructions the change made, per the same section. Report an inventory of what exists, the state of the code, a description of how something works or a retelling of another document in `AGENTS.md` or an own skill as a `docs` finding, and name the place to move it to from that section's table. A rule of agent behavior that is written nowhere else is not a finding.

### Role: tests

Read `.agents/skills/testing/SKILL.md` and the testing rules of the module doc for each area whose tests the change touched. Review every test the change added or changed:

- The test fails without the behavior it pins. A test that would pass with that behavior removed is a `missing_test` finding.
- A bug fix has a test that reproduces the bug and fails on the code before the fix. Without one, return a `missing_test` finding.
- The test covers the cases the module doc asks for. A missing case is a `missing_test` finding.
- The test waits for a condition up to a limit instead of a fixed pause, and it cleans up the data it created. A fixed pause, a flaky assertion or leaked data is a `bug` finding that names the condition to wait for or the data to clean up.
- Select verification commands according to the scope rule in `AGENTS.md` and `docs/testing.md#checking-work`. For a small, localized change, accept targeted package or screen tests named in the task's own verification; a change verified only by such task commands needs no `## Change verification` section. Do not require `mise run test` solely because ordinary tests were added or changed. When the scope rule requires `mise run test`, the change names it in its `## Change verification` section. Check that a change that adds or changes scenarios of the browser end-to-end suite in `e2e/browser/scenarios` also names `mise run e2e:browser` there. A task that changes tests in `backend/internal/agent/e2e`, `backend/internal/modules/backups/e2e` or `backend/e2e/upgrade`, the verification agent's code or the `Dockerfile` names the command that repeats those tests in its own verification. No section or task verification runs `mise run test-full`, which agents run only on the user's direct request. A section that leaves out the command it owes is a `missing_test` finding, and so is a task that leaves out its own command. A verification that runs `mise run test-full` is a `missing_test` finding that asks for the specific commands instead. Naming `mise run test-full` as something a task adds or edits, with a verification that runs another command, is never a finding. Tests that no allowed command form runs, such as the hook tests in `.agent-hooks/`, need an explicit verification in their task instead, so a change with only such tests needs no section.

### Role: repeat review

The previous review of this change blocked the turn, and the agent fixed the findings. The previous findings and the paths changed after the previous review are passed below.

- For every previous finding, check that it is fixed or that the agent rejected it with a sound reason in the response. Return an unfixed finding without such a reason again.
- Check that the edits after the previous review broke nothing.
- Do not review the whole change again.
