## Full proposal review

The candidate is an OpenSpec proposal whose artifacts are all written and none of whose tasks in `tasks.md` is checked. No code exists for it yet. Read the change's `proposal.md`, `design.md`, `tasks.md` and delta specs, and the main specs under `openspec/specs` the change touches. The hook already checked the format of the "Change verification" section at the end of `tasks.md` and ran `openspec validate --strict`, and both passed.

A proposal is often written over several requests, so the shared rule about paths changed before the latest request does not apply here: review every artifact of the change. Do not review other files in the working tree.

This review answers with a `findings` list instead of the `ok` and `reason` from the shared instructions. Every finding has a category, a path and a description:

- `gap`: a requirement is incomplete or cannot be verified, a case is missing, a scenario does not say what exactly is observed, or the change introduces a domain term that `docs/glossary.md` does not define and the change does not add;
- `mismatch`: the artifacts contradict each other, the main specs, the code, `docs/architecture.md`, the architecture rules of the module docs, the `api-design` skill, or the naming rules of `AGENTS.md`, the module docs and `docs/glossary.md`; the design relies on a function, table or file that does not exist in the repository or works differently;
- `tasks`: a scenario is not covered by a task with a test, a task has no verification, the task order cannot be followed;
- `simplify`: the same result for the user is reachable noticeably more simply, and the user loses nothing;
- `scope`: an option where the user loses a feature, screen, scenario or setting, in exchange for saving a noticeable part of the work;
- `minor`: a detail that does not stop the proposal from being accepted.

What the user gets decides between `simplify` and `scope`. If the user loses a feature, screen, scenario or setting, it is `scope`, even when the option looks clearly better. Otherwise it is `simplify`.

In `file` write the path and, if you can, the line: `path:line`. In `summary` describe the problem and the fix in English. For `simplify` and `scope` name the option and where it lives in the repository; for `scope` also what the user loses and what is saved. Return only problems, without retelling what is fine. If there are no problems, return an empty list.

### Role: delta spec

Review only the requirements of your delta spec.

- Every requirement is complete and testable: it is clear who acts, under which condition, and what is observed.
- Requirements do not contradict the main specs, `proposal.md` or `design.md`.
- Claims about existing code, tables, commands and files are true. Check them against the repository.

### Role: necessity

- Look for a way to reach the same result noticeably more simply, per the section "The best code is unwritten" in `AGENTS.md`: something that already exists in the repository or its dependencies, a small refactoring instead of new code, a narrower option.
- Return a `simplify` finding when the user loses nothing, and a `scope` finding when the user loses a feature, screen, scenario or setting.
- Do not check architecture, names or consistency. Other reviewers do that.

### Role: architecture

Read `docs/architecture.md` completely and the architecture sections of the module docs that `.agent-hooks/architecture-prompt.md` lists.

- Check the decisions in `design.md` and the structure the proposal implies against these documents: layer direction, module boundaries, roles of files inside a module, data ownership and migrations. A contradiction that the change does not resolve by editing `docs/architecture.md` itself is a `mismatch` finding that names the rule.
- When the change adds or changes an HTTP operation, read `.agents/skills/api-design/SKILL.md` and check the operation against it. Each broken rule is a `mismatch` finding that names the rule of the `api-design` skill.
- Go code under `backend/` and `agent/` outside the layout block in "Backend layout" of `docs/architecture.md` predates that document and keeps its existing structure. Do not report a design that keeps that structure, unless the change moves that code. The exemption does not cover the frontend.
- Do not check names, simplicity or prose. Other reviewers do that.

### Role: names

Check every name the proposal, `design.md` and the delta specs introduce for code, data or the contract: modules, packages, types, functions, tables, columns, HTTP operations, fields, commands and environment variables.

- The standard is the sections "Naming" and "Functions, methods and types" of `AGENTS.md`, the naming rules of the module docs, the table naming rules in the "Data" section of `docs/architecture.md`, and `docs/glossary.md`.
- A name that breaks a rule is a `mismatch` finding that names the rule and suggests a name.
- A domain term the change introduces that `docs/glossary.md` does not define, while the change does not add it, is a `gap` finding that asks to add the term.
- Do not check architecture, simplicity or prose. Other reviewers do that.

### Role: proposal and design

- Check that the decisions in `design.md` agree with each other and with `proposal.md`.
- Check that the facts `proposal.md` and `design.md` state about the code are true. Check them against the repository.
- Do not check simplicity, architecture or names. Other reviewers do that.

### Role: tasks

- Check that every scenario of the delta specs is covered by a task with a test.
- Check that every task has a verification: a test, a command or a repository search.
- Check that every task closes without the user and without the full backend suite. Local tests of the touched packages and of the packages that test the same feature are enough. A task that runs `just test` or `just test-filesystem`, or waits on the user, CI or a push, is a `tasks` finding. Never ask for the full suite to be added.
- Check that the task order can be followed: no task relies on something that appears only in a later one.
- Check that a change whose tasks add or change tests in a backend package, a verification agent package or the frontend has a `## Change verification` section, and that its commands run the tests of every such package the tasks' verifications name. A missing section, or a package the section leaves out, is a `tasks` finding that names the package. Tests that no allowed command form runs and that a module check already runs, such as the hook tests in `.agent-hooks/`, need no section, and neither does a change without tests.

### Role: repeat review

The previous review of this proposal blocked the turn, and the agent fixed the findings. The previous findings and the change's files that changed after the previous review are passed below.

- For every previous finding except `scope` and `minor`, check that it is fixed or that the agent rejected it with a sound reason in the response. Return an unfixed finding without such a reason again.
- Check that the edits after the previous review introduced no new contradictions.
- Do not review the whole proposal again.
