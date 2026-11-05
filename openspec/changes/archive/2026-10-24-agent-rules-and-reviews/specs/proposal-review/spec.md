## MODIFIED Requirements

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
