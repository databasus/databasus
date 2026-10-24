# proposal-review Specification

## Purpose
Reviews a finished OpenSpec proposal as a whole before implementation, so that gaps in requirements, contradictions between artifacts and needless complexity are found before code is written against them.

## Requirements

### Requirement: The full proposal review starts once the proposal is written
The hook that runs when the agent finishes a turn SHALL start the full proposal review of an active change when all of these hold: the change has every artifact its schema requires, its task list has tasks and none is checked, the change's files changed during this turn, and their state differs from the last state the full proposal review accepted for this change. The turn SHALL be recognized by the change's files, not by the request text: whether the agent started through a command, a skill or a plain request MUST NOT matter. The accepted state SHALL be stored per change, not per session. Archived changes are not reviewed.

#### Scenario: The agent writes every artifact after a plain request
- **WHEN** the user writes "write a spec for this", and the agent creates the proposal, design, delta specs and tasks and finishes the turn
- **THEN** the hook starts the full proposal review of that change

#### Scenario: Artifacts are not finished
- **WHEN** a change has a proposal but no tasks, and the agent finishes the turn
- **THEN** the full proposal review does not start and the quick checks run

#### Scenario: Implementation has started
- **WHEN** at least one task of the change is checked, and the agent edits a delta spec
- **THEN** the full proposal review does not start

#### Scenario: New session without edits
- **WHEN** the full proposal review accepted a proposal in one session, and in a new session the agent finishes a turn without touching the change's files
- **THEN** the full proposal review does not start

#### Scenario: Edit after acceptance
- **WHEN** after an accepted review the agent edits the design of a change whose tasks are not started
- **THEN** the full proposal review starts again

### Requirement: OpenSpec validation runs first
Before starting reviewers the hook SHALL check the format of the change's "Change verification" section by the rules the full implementation review applies to it, without running its commands, and then run strict OpenSpec validation of the change. When the section's format is wrong or validation fails, the hook SHALL block the turn and pass the agent the reason or the end of the validation output. Reviewers MUST NOT start in that case.

#### Scenario: A requirement without a scenario
- **WHEN** a delta spec has a requirement without a scenario and validation fails
- **THEN** the turn is blocked, the agent gets the validation output, and no reviewer starts

#### Scenario: A forbidden command in the verification section
- **WHEN** the change's "Change verification" section contains `mise run test`
- **THEN** the turn is blocked with the reason and the list of allowed forms, the command does not run, and no reviewer starts

### Requirement: What the full proposal review checks
The full proposal review SHALL run several read-only reviewers in parallel, on the same models as the full implementation review: one per delta spec of the change, and one each for necessity, architecture, names, consistency of the proposal and design, and the task list. Each reviewer SHALL check only its own role, so that one reviewer's findings do not crowd out another's.

The delta spec reviewer SHALL check that requirements are complete and testable, do not contradict the main specs, the proposal or the design, and that claims about existing code are true.

The necessity reviewer SHALL look for a noticeably simpler way to reach the same result, following the repository rule that the best code is unwritten: something that already exists in the repository or its dependencies, a small refactoring instead of new code, or a narrower option. It SHALL return a `simplify` finding when the user loses nothing and a `scope` finding when the user loses a feature, screen, scenario or setting.

The architecture reviewer SHALL check the decisions in the design and the structure the proposal implies against `docs/architecture.md` and the architecture rules of the module docs, and SHALL return a `mismatch` finding for each contradiction that the change does not resolve by editing `docs/architecture.md` itself. When the change adds or changes an HTTP operation, the architecture reviewer SHALL also check it against the repository's API design skill and return a `mismatch` finding for each rule it breaks. A design that keeps the existing structure of Go code under `backend/` and `agent/` outside the layout block in "Backend layout" of `docs/architecture.md` SHALL NOT be reported, unless the change moves that code.

The names reviewer SHALL check every name the proposal, design and delta specs introduce for code, data or the contract, such as modules, packages, types, tables, columns, HTTP operations, fields, commands and environment variables, against the naming rules of the root `AGENTS.md`, the module docs, the table naming rules in the "Data" section of `docs/architecture.md` and `docs/glossary.md`. It SHALL return a `mismatch` finding that names the rule and suggests a name, and a `gap` finding when the change introduces a domain term that the glossary does not define and the change does not add.

The consistency reviewer SHALL check that the decisions agree with each other and with the proposal, and that the facts the proposal and design state about the code are true.

The task reviewer SHALL check that a task with a test covers every scenario, that every task has a verification, and that the task order is feasible. The task reviewer SHALL also check that a change whose tasks add or change tests in a backend package, a verification agent package or the frontend has a "Change verification" section, and that the section runs the tests of every such package the tasks' verifications name. Tests that no allowed command form runs and that a module check already runs, such as the hook tests, need no section.

#### Scenario: A change with two delta specs
- **WHEN** the full proposal review starts for a change with two delta specs
- **THEN** the hook starts seven reviewers, at most six at a time: one per delta spec, and one each for necessity, architecture, names, proposal and design consistency, and tasks

#### Scenario: A design that bypasses a module boundary
- **WHEN** the design lets one module read another module's table directly and leaves `docs/architecture.md` unchanged
- **THEN** the architecture reviewer returns a `mismatch` finding that names the rule in `docs/architecture.md`

#### Scenario: A design that edits old code in place
- **WHEN** the design changes a service under `backend/internal/features` and keeps its existing structure
- **THEN** the architecture reviewer returns no finding about the layout of that code

#### Scenario: A proposed operation that breaks an API rule
- **WHEN** the design adds an HTTP operation whose error responses do not use the error format the API design skill requires
- **THEN** the architecture reviewer returns a `mismatch` finding that names the rule of the API design skill

#### Scenario: A table name that breaks the naming rules
- **WHEN** the design introduces a table without the owning module's prefix
- **THEN** the names reviewer returns a `mismatch` finding that names the rule and suggests a prefixed name

#### Scenario: A new term missing from the glossary
- **WHEN** the delta specs introduce a domain term that `docs/glossary.md` does not define and the change does not add it
- **THEN** the names reviewer returns a `gap` finding that asks to add the term to the glossary

#### Scenario: A simpler option that already exists
- **WHEN** an existing field in a table covers what the proposal adds a new table for, and the user loses nothing
- **THEN** the necessity reviewer returns a `simplify` finding that points to the existing field

#### Scenario: The task reviewer's role
- **WHEN** the hook starts the task reviewer
- **THEN** its prompt names the role and requires checking that a task with a test covers every scenario, every task has a verification, the task order is feasible, and the "Change verification" section covers the tested packages

#### Scenario: Tests without a verification section
- **WHEN** the tasks add tests to a backend package and the change has no "Change verification" section
- **THEN** the task reviewer returns a `tasks` finding that asks for the section with that package

#### Scenario: A section that leaves out a package
- **WHEN** a task's verification runs the tests of two backend packages and the "Change verification" section runs only one of them
- **THEN** the task reviewer returns a `tasks` finding that names the missing package

#### Scenario: A change with only hook tests
- **WHEN** the tasks add tests only to the hook tests and the change has no "Change verification" section
- **THEN** the task reviewer returns no finding about the missing section

### Requirement: The agent resolves findings itself, except cutting the feature
A reviewer SHALL return findings, each with a category, a path and a description. The categories are `gap` (a requirement is incomplete, a case is missing, or a new domain term is missing from the glossary), `mismatch` (artifacts contradict each other, the main specs, the code, the architecture rules or the naming rules), `tasks` (a scenario without a task with a test, a task without verification, an infeasible order), `simplify` (the same result for the user, noticeably simpler), `scope` (an option that cuts the feature) and `minor`. The hook SHALL block the turn when there is a `gap`, `mismatch`, `tasks` or `simplify` finding, and require the agent to resolve each one without asking the user. `scope` and `minor` findings MUST NOT block the turn. The agent MUST NOT apply a `scope` finding itself: cutting the task is the user's decision.

#### Scenario: A simplification without loss
- **WHEN** a reviewer finds that a field in an existing table replaces a new table, and the user loses nothing
- **THEN** the turn is blocked, and the agent gets the finding with the instruction to resolve it without asking the user

#### Scenario: A proposal to cut the feature
- **WHEN** a reviewer returns only a `scope` finding saying the settings screen can be dropped
- **THEN** the hook does not require a fix and once asks the agent for a summary that shows this option to the user

#### Scenario: A name that breaks the naming rules
- **WHEN** the names reviewer returns a `mismatch` finding for an HTTP operation name
- **THEN** the turn is blocked, and the agent renames the operation in the change's artifacts without asking the user

### Requirement: Repeat and summary of the full proposal review
After a blocked review the next run SHALL check the previous findings and the change's files that changed after that review, not the whole proposal again. The block text SHALL require the agent to finish the turn again with one short line about its fixes that names every rejected finding with the reason, and to leave the summary for the user until the review accepts the proposal. There SHALL be at most two repeats per user request, after which the hook SHALL stop the agent and show the remaining findings. When the review accepts the proposal after at least one block in this request, or returns `scope` findings, the hook SHALL block the turn once and ask the agent to write the user a summary of the proposal as if there had been no review: what is proposed and which decisions were made, without review rounds, findings or the fixes made for them, and separately each option to cut the feature with what the user loses and saves. The next turn without edits to the change's files MUST NOT be blocked.

#### Scenario: A gap blocks the turn
- **WHEN** the proposal review blocks the turn with a `gap` finding
- **THEN** the block text asks the agent to resolve the finding without asking the user and to finish the turn with one short line about the fixes that names each rejected finding with its reason, and not to write the user summary yet

#### Scenario: Accepted after fixes
- **WHEN** a repeat review accepts a proposal the previous review blocked
- **THEN** the hook once asks the agent for a summary of the proposal that leaves out review rounds and the fixes made for them, and the next turn without edits is not blocked

#### Scenario: Accepted at once without cuts
- **WHEN** the first review returns only `minor` findings
- **THEN** the turn is not blocked and the user sees the findings

#### Scenario: Accepted after fixes with an option to cut the feature
- **WHEN** a repeat review accepts a proposal the previous review blocked, and returns a `scope` finding
- **THEN** the summary request asks to list that option with what the user loses and saves, and to leave out the review rounds and the fixes made for them

#### Scenario: Repeat limit
- **WHEN** the third proposal review in a row within one user request blocks the turn again
- **THEN** the hook stops the agent and shows the remaining findings

### Requirement: Quick checks yield to the full proposal review
When only files of changes that have every artifact and no checked task changed during the turn, the `necessity`, `architecture`, `diff` and `naming` checks SHALL skip the turn. This way the quick checks duplicate neither the review turn nor the summary turn after acceptance. When other files changed too, the quick checks SHALL run as usual.

#### Scenario: A turn with only artifacts
- **WHEN** the agent finished every artifact of a change and changed nothing else
- **THEN** the quick checks skip the turn and only the full proposal review runs

#### Scenario: An unfinished proposal
- **WHEN** the agent wrote only the proposal and finishes the turn with a question to the user
- **THEN** the full proposal review does not start, and the quick necessity check reviews the proposal

### Requirement: The full proposal review ends within the hook timeout
The full proposal review SHALL work within the same kind of time budget as the full implementation review: the budget ends before the hook timeout with a margin, validation runs inside it, and a reviewer starts only when the budget still holds its whole time limit. A reviewer that cannot start for that reason SHALL count as a reviewer that could not run. The hook MUST NOT be killed by the client timeout without a message.

#### Scenario: Too many reviewers for the budget
- **WHEN** a proposal has so many delta specs that a third wave of reviewers would not fit in the remaining budget
- **THEN** the reviewers of that wave do not start, the turn is blocked, and the agent gets the reason that the review time budget ran out
