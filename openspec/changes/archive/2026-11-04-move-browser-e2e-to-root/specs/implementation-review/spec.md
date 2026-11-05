## MODIFIED Requirements

### Requirement: Change verification commands the hook accepts

The change verification commands SHALL be the lines of the first code block fenced as `sh` under the "Change verification" heading of the change's task list. A block fenced as `bash`, `shell` or without a language does not count. Empty lines and comment lines are skipped. The hook SHALL accept only these command forms:

- the routine local tests: exactly `mise run test`, with no further arguments, which the agent runs from the repository root;
- isolated frontend browser tests: exactly `pnpm --dir frontend test:browser --project=ui`, with no additional arguments;
- the browser end-to-end suite: exactly `mise run e2e:browser`, with no further arguments, which the agent runs from the repository root.

The implementing agent SHALL run these commands before review. The hook SHALL parse them only to validate the section and MUST NOT execute them. A line that is not one of the allowed forms, a section without a closed `sh` block, and a block without commands SHALL each block the turn with the reason and the list of allowed forms. No reviewer starts in that case.

#### Scenario: The full run in the block

- **WHEN** the change verification block contains `mise run test`
- **THEN** the hook accepts the command format without running the tests

#### Scenario: A shell operator

- **WHEN** a line reads `mise run test && rm -rf /tmp/x`
- **THEN** the turn is blocked with the reason that the line does not match the form, and nothing runs

#### Scenario: The full run with an argument

- **WHEN** a block contains `mise run test ./x`
- **THEN** the turn is blocked with the reason that the line does not match the form, and nothing runs

#### Scenario: A removed narrow command

- **WHEN** a block contains `mise run agent:test`, `pnpm --dir frontend test` or `mise run backend:test-group physical-api-1-of-3`
- **THEN** the turn is blocked with the reason that the command is not allowed and the list of allowed forms, and nothing runs

#### Scenario: The old command form

- **WHEN** a block contains `mise run backend:test-package ./internal/features/intervals/...`
- **THEN** the turn is blocked with the reason that the command is not allowed and the list of allowed forms, and nothing runs

#### Scenario: The isolated browser UI command

- **WHEN** the verification block contains exactly `pnpm --dir frontend test:browser --project=ui`
- **THEN** the hook accepts the command format without executing the browser runner
- **AND** the implementing agent remains responsible for running the declared tests and providing verification evidence

#### Scenario: The isolated real-API browser command

- **WHEN** the verification block contains exactly `mise run e2e:browser`
- **THEN** the hook accepts the command format without executing the suite
- **AND** the implementing agent remains responsible for running the declared tests and providing verification evidence

#### Scenario: An unbounded browser command

- **WHEN** the block omits the browser project, selects a project other than `ui`, adds arguments or uses a different package script
- **THEN** the hook rejects the command before any reviewer starts

#### Scenario: The removed browser project

- **WHEN** the block contains `pnpm --dir frontend test:browser --project=e2e`
- **THEN** the hook rejects the command before any reviewer starts

#### Scenario: The end-to-end command with an argument

- **WHEN** the block contains `mise run e2e:browser ./x`
- **THEN** the hook rejects the command before any reviewer starts

### Requirement: What the full implementation review checks

The full implementation review SHALL run several read-only reviewers in parallel: one per delta spec of the change, one for documentation and the task list, one for tests, and one for architecture. The delta spec reviewer SHALL find the code and the test for every requirement and scenario, check that the test actually proves the scenario, check the design decisions against the code, and look for bugs. The documentation reviewer SHALL check that everything each task states was done, including its verification, and that documentation agrees with itself and with the code. The documentation reviewer SHALL also check edits to agent instructions made by the change against the requirement "What belongs in agent instructions" of the `agent-instructions` capability.

The tests reviewer SHALL check every test the change added or changed against the repository's testing skill and the testing rules of the module docs: the test fails without the behavior it pins, a bug fix has a test that reproduces the bug, the cases the module doc asks for are covered, the test waits for a condition instead of a fixed pause, and it cleans up its data. The tests reviewer SHALL also check that the change verification commands run the tests of every backend package and verification agent package outside the end-to-end ones, and of every frontend test file, the change added or changed tests in, and that a change adding or changing scenarios of the browser end-to-end suite names `mise run e2e:browser`. It SHALL also check that a task changing tests in `backend/internal/agent/e2e`, `backend/internal/modules/backups/e2e` or `backend/e2e/upgrade`, the verification agent's code or the `Dockerfile` names the command that repeats those tests, and that no section or task verification runs `mise run test-full`. Tests that no allowed command form runs, such as the hook tests, SHALL name an explicit verification command in their task instead of this section.

The architecture reviewer SHALL check all code that belongs to the change, including parts committed or written in earlier requests, against the architecture document and the architecture rules of the module docs: layer direction, module boundaries, roles of files inside a module, data ownership and migrations. It SHALL report a violation that appears only when several edits of the change are read together, such as an import added in one request and used across a module boundary in another. When the change edits an HTTP contract file in the backend contracts, the architecture reviewer SHALL also check the edit against the repository's API design skill and report each broken rule as an `architecture` finding. All backend code belonging to the change SHALL be checked against the architecture rules, without an exemption for code predating the documented layout. For frontend changes, the architecture reviewer SHALL read the frontend architecture section of the architecture document and the applicable module conventions, including the rules for placement, composition, import direction and public APIs. Existing migration findings SHALL be accepted only when individually recorded in the finite baseline; a new or multiplied violation SHALL still be reported.

#### Scenario: A scenario without a test

- **WHEN** a delta spec scenario has no test, or its test passes without the described behavior
- **THEN** the reviewer returns a `missing_test` finding with the path to the test or code

#### Scenario: Documentation lags behind

- **WHEN** the change adds an environment variable and the configuration documentation does not mention it
- **THEN** the documentation reviewer returns a `docs` finding

#### Scenario: The change retells a document in AGENTS.md

- **WHEN** the change adds a list of a module doc's sections to the root agent instructions
- **THEN** the documentation reviewer returns a `docs` finding that suggests naming the doc's topic instead

#### Scenario: The change adds a rule to AGENTS.md

- **WHEN** the change adds a rule to the root agent instructions that changes agent behavior and is written nowhere else
- **THEN** the documentation reviewer returns no finding under the instruction content criterion

#### Scenario: A change with two delta specs

- **WHEN** the full implementation review starts for a change with two delta specs
- **THEN** the hook starts five reviewers in parallel: one per delta spec, one for documentation and tasks, one for tests, and one for architecture

#### Scenario: A boundary violation spread over two requests

- **WHEN** one request of the change adds a dependency on another module's internal package and a later request calls it from a service
- **THEN** the architecture reviewer returns an `architecture` finding that names both paths and the rule in the architecture document

#### Scenario: Old code that keeps its structure

- **WHEN** a changed backend production file violates the documented layout, including in a directory that previously had a migration exemption
- **THEN** the architecture reviewer returns an `architecture` finding naming the file and the rule

#### Scenario: A frontend violation is still reported

- **WHEN** the change makes a frontend slice import another slice of the same layer
- **THEN** the architecture reviewer returns an `architecture` finding

#### Scenario: A contract edit that breaks an API rule

- **WHEN** the change adds an operation to the user API contract whose error responses do not use the error format the API design skill requires
- **THEN** the architecture reviewer returns an `architecture` finding that names the rule of the API design skill

#### Scenario: A test that waits with a fixed pause

- **WHEN** a test the change added sleeps for a fixed time before checking that a backup finished
- **THEN** the tests reviewer returns a finding that asks to wait for the condition instead

#### Scenario: A bug fix without a reproducing test

- **WHEN** a change fixes a bug and no test it added or changed fails on the code before the fix
- **THEN** the tests reviewer returns a `missing_test` finding

#### Scenario: Changed tests outside the verification commands

- **WHEN** the change adds tests to a backend package outside the end-to-end ones that none of its change verification commands runs
- **THEN** the tests reviewer returns a `missing_test` finding that names the package

#### Scenario: Changed browser scenarios outside the verification commands

- **WHEN** the change adds or changes a browser end-to-end scenario and its change verification commands leave out `mise run e2e:browser`
- **THEN** the tests reviewer returns a `missing_test` finding that names the scenario

#### Scenario: Changed end-to-end tests without their command

- **WHEN** a task changes tests in `backend/internal/agent/e2e`, `backend/internal/modules/backups/e2e` or `backend/e2e/upgrade`, the verification agent's code or the `Dockerfile`, and its verification names no command that repeats those tests
- **THEN** the tests reviewer returns a `missing_test` finding that names the task

#### Scenario: A verification that runs the whole test set

- **WHEN** the "Change verification" section or a task's verification runs `mise run test-full`
- **THEN** the tests reviewer returns a `missing_test` finding that asks for the specific commands instead

#### Scenario: A task that builds the command

- **WHEN** a task names `mise run test-full` as something it adds or edits, and its verification runs another command
- **THEN** the tests reviewer returns no finding for naming it

#### Scenario: A change with only hook tests

- **WHEN** the change adds tests only to the hook tests and has no "Change verification" section
- **THEN** the tests reviewer returns no finding about the missing section when the task names an explicit command for the agent to run those tests

#### Scenario: Frontend architecture has an authoritative document

- **WHEN** a frontend plan or implementation changes a slice boundary or screen composition
- **THEN** the architecture reviewer reads the frontend section of the architecture document and the applicable module conventions
- **AND** it reports any contradiction, including a violation authorized only by an overly broad migration exception
