## MODIFIED Requirements

### Requirement: OpenSpec validation runs first
Before starting reviewers the hook SHALL check the format of the change's test plan by the rules the full implementation review applies to it, without running anything it selects, and then run strict OpenSpec validation of the change. When the plan is missing, its format is wrong or validation fails, the hook SHALL block the turn and pass the agent the reason or the end of the validation output. Reviewers MUST NOT start in that case.

#### Scenario: A requirement without a scenario
- **WHEN** a delta spec has a requirement without a scenario and validation fails
- **THEN** the turn is blocked, the agent gets the validation output, and no reviewer starts

#### Scenario: A forbidden command in the verification section
- **WHEN** the selection of the change's test plan contains `mise run backend:test-package ./internal/lib/...`
- **THEN** the turn is blocked with the reason and the list of allowed selector forms, nothing runs, and no reviewer starts

#### Scenario: A forbidden selector in the test plan
- **WHEN** the selection of the change's test plan contains `backend ./...`
- **THEN** the turn is blocked with the reason and the list of allowed selector forms, nothing runs, and no reviewer starts

#### Scenario: A change on the old schema without a test plan
- **WHEN** a change created with `spec-driven` has every artifact of that schema but no `tests.md`, and the agent finishes a turn
- **THEN** the turn is blocked with the reason the test plan check gives, nothing runs, and no reviewer starts

#### Scenario: A change without a test plan
- **WHEN** a change created with the project's schema has a proposal, design, delta specs and tasks but no `tests.md`
- **THEN** OpenSpec reports the planning as incomplete, and the full proposal review does not start

### Requirement: What the full proposal review checks
The full proposal review SHALL run several read-only reviewers in parallel, on the same models as the full implementation review: one per delta spec of the change, and one each for necessity, architecture, names, consistency of the proposal and design, and the task list. Each reviewer SHALL check only its own role, so that one reviewer's findings do not crowd out another's.

The delta spec reviewer SHALL check that requirements are complete and testable, do not contradict the main specs, the proposal or the design, and that claims about existing code are true.

The necessity reviewer SHALL look for a noticeably simpler way to reach the same result, following the repository rule that the best code is unwritten: something that already exists in the repository or its dependencies, a small refactoring instead of new code, or a narrower option. It SHALL return a `simplify` finding when the user loses nothing and a `scope` finding when the user loses a feature, screen, scenario or setting.

The architecture reviewer SHALL check the decisions in the design and the structure the proposal implies against `docs/architecture.md` and the architecture rules of the module docs, and SHALL return a `mismatch` finding for each contradiction that the change does not resolve by editing `docs/architecture.md` itself. When the change adds or changes an HTTP operation, the architecture reviewer SHALL also check it against the repository's API design skill and return a `mismatch` finding for each rule it breaks.

The names reviewer SHALL check every name the proposal, design and delta specs introduce for code, data or the contract, such as modules, packages, types, tables, columns, HTTP operations, fields, commands and environment variables, against the naming rules of the root `AGENTS.md`, the module docs, the table naming rules in the "Data" section of `docs/architecture.md` and `docs/glossary.md`. It SHALL return a `mismatch` finding that names the rule and suggests a name, and a `gap` finding when the change introduces a domain term that the glossary does not define and the change does not add.

The consistency reviewer SHALL check that the decisions agree with each other and with the proposal, and that the facts the proposal and design state about the code are true.

The task reviewer SHALL check that a task with a test covers every scenario, that every task has a verification, and that the task order is feasible. The task reviewer SHALL also check the test plan against the proposal, the design and the tasks: every area the change touches appears in the impact map, every test package, frontend test file and suite the tasks add or change tests in is selected, each exclusion gives a reason that holds, engine versions follow the version rule of the `change-test-plans` capability, and a change to end-to-end code selects the suites that capability names. It SHALL also check that no task verification runs `mise run test` or the Go test command over the whole backend module.

#### Scenario: A change with two delta specs
- **WHEN** the full proposal review starts for a change with two delta specs
- **THEN** the hook starts seven reviewers, at most six at a time: one per delta spec, and one each for necessity, architecture, names, proposal and design consistency, and tasks

#### Scenario: A design that bypasses a module boundary
- **WHEN** the design lets one module read another module's table directly and leaves `docs/architecture.md` unchanged
- **THEN** the architecture reviewer returns a `mismatch` finding that names the rule in `docs/architecture.md`

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
- **THEN** its prompt names the role and requires checking that a task with a test covers every scenario, every task has a verification, the task order is feasible, and the test plan covers what the change touches

#### Scenario: Tests without a verification section
- **WHEN** the tasks add tests to a backend package and the test plan selects only `none`
- **THEN** the task reviewer returns a `tasks` finding that asks to select that package

#### Scenario: A section that leaves out a package
- **WHEN** a task adds tests to a backend package and the test plan selects only `suite e2e-browser`
- **THEN** the task reviewer returns a `tasks` finding that names the package

#### Scenario: An area missing from the impact map
- **WHEN** the design changes the notifiers module and the impact map does not mention it
- **THEN** the task reviewer returns a `tasks` finding that names the area

#### Scenario: An exclusion without a valid reason
- **WHEN** the plan excludes PostgreSQL 17 with the reason that the code path is version-independent, while the design adds a branch for PostgreSQL 17
- **THEN** the task reviewer returns a `tasks` finding that names the exclusion

#### Scenario: A change to browser end-to-end scenarios
- **WHEN** a task adds or changes a scenario in `e2e/browser/scenarios` and the test plan does not select `suite e2e-browser`
- **THEN** the task reviewer returns a `tasks` finding that asks for that selector

#### Scenario: A task that runs the full run
- **WHEN** a task's verification runs `mise run test`
- **THEN** the task reviewer returns a `tasks` finding that asks for the specific tests instead

#### Scenario: A change with only hook tests
- **WHEN** the tasks add tests only to the hook tests and the test plan selects `suite review-hooks`
- **THEN** the task reviewer returns no finding about the selection

#### Scenario: A task that changes end-to-end tests without their command
- **WHEN** a task changes tests in `backend/internal/agent/e2e`, `backend/internal/modules/backups/e2e` or `backend/e2e/upgrade`, the verification agent's code or the `Dockerfile`, and the test plan selects nothing that repeats those tests
- **THEN** the task reviewer returns a `tasks` finding that asks for that selector

#### Scenario: A verification that runs the whole test set
- **WHEN** a task's verification runs the Go test command over the whole backend module
- **THEN** the task reviewer returns a `tasks` finding that asks for the specific tests instead

#### Scenario: A task that builds the command
- **WHEN** a task names `mise run test` as something it adds or edits, and its verification runs another command
- **THEN** the task reviewer returns no finding for naming it

### Requirement: Proposal review stays within the selected client

Initial and repeat proposal reviews SHALL use the client selected by the review-hook-launch capability and the review profile for full reviews defined by the implementation-review capability. A complete proposal written through a command, a skill or a plain request SHALL receive the same strict OpenSpec validation, reviewer roles, finding treatment, repeat limits and summary behavior in either client. The selected client's CLI SHALL be sufficient; the other client MUST NOT be required. Proposal acceptance and findings SHALL follow the client and review profile isolation rules of the review-hook-launch capability.

#### Scenario: Codex writes a complete proposal

- **WHEN** Codex finishes a turn that makes an unstarted proposal eligible for full review
- **THEN** the test plan format check and strict OpenSpec validation precede the existing parallel reviewer roles, all running in Codex with `high` reasoning effort and without starting Claude Code

#### Scenario: Claude Code writes a complete proposal

- **WHEN** Claude Code finishes a turn that makes an unstarted proposal eligible for full review
- **THEN** the test plan format check and strict OpenSpec validation precede the existing parallel reviewer roles, all running in Claude Code on Opus with `medium` effort and without starting Codex

#### Scenario: A proposal fails validation

- **WHEN** either client finishes an eligible proposal whose strict OpenSpec validation fails
- **THEN** the turn is blocked with the validation reason and no AI reviewer starts

#### Scenario: Codex fixes proposal findings

- **WHEN** a full proposal review in Codex returns a blocking finding and Codex fixes the artifacts
- **THEN** the repeat runs in Codex with `high` reasoning effort over the previous findings and subsequent edits, and acceptance follows the existing summary behavior

#### Scenario: A proposal was reviewed by the other client

- **WHEN** an eligible proposal review starts in a different client from the one whose stored findings or acceptance exist
- **THEN** the selected client starts its own full review rather than reusing the other client's verdict or restricted repeat
