## MODIFIED Requirements

### Requirement: Verification precedes AI review

The implementing agent SHALL run the relevant formatters, linters and tests before finishing the implementation turn that starts review. The implementation review hook SHALL run only read-only AI reviewers and SHALL NOT run linters, formatters, builds, hook tests or the tests the change's test plan selects. It SHALL validate the format of the change's test plan without running anything it selects. Reviewer prompts MUST NOT claim that review ran checks or that those checks passed; reviewers SHALL assess verification claims against the available evidence.

#### Scenario: A change names package tests

- **WHEN** implementation review starts for a change whose test plan selects backend packages
- **THEN** the hook starts AI reviewers without running those tests or any module linter

#### Scenario: The hooks changed

- **WHEN** implementation review starts for a change that modifies the review hooks
- **THEN** the hook does not run its own tests; the implementing agent runs them before review

#### Scenario: A verification claim

- **WHEN** a reviewer receives an implementation review prompt
- **THEN** the prompt requires checking the agent's verification evidence and does not assert that review ran or passed checks

### Requirement: What the full implementation review checks

The full implementation review SHALL run several read-only reviewers in parallel: one per delta spec of the change, one for documentation and the task list, one for tests, and one for architecture. The delta spec reviewer SHALL find the code and the test for every requirement and scenario, check that the test actually proves the scenario, check the design decisions against the code, and look for bugs. The documentation reviewer SHALL check that everything each task states was done, including its verification, and that documentation agrees with itself and with the code. The documentation reviewer SHALL also check edits to agent instructions made by the change against the requirement "What belongs in agent instructions" of the `agent-instructions` capability.

The tests reviewer SHALL check every test the change added or changed against the repository's testing skill and the testing rules of the module docs: the test fails without the behavior it pins, a bug fix has a test that reproduces the bug, the cases the module doc asks for are covered, the test waits for a condition instead of a fixed pause, and it cleans up its data. The tests reviewer SHALL also check the test plan against the diff: every area the diff touches appears in the impact map, every backend package, frontend test file and suite the change added or changed tests in is selected, every exclusion still holds for the code as written, engine versions follow the version rule of the `change-test-plans` capability, and a change to end-to-end code selects the suites that capability names. It SHALL also check that no task verification runs `mise run test`.

The architecture reviewer SHALL check all code that belongs to the change, including parts committed or written in earlier requests, against the architecture document and the architecture rules of the module docs: layer direction, module boundaries, roles of files inside a module, data ownership and migrations. It SHALL report a violation that appears only when several edits of the change are read together, such as an import added in one request and used across a module boundary in another. When the change edits the agent API contract, the architecture reviewer SHALL also check the edit against the repository's API design skill and report each broken rule as an `architecture` finding. No reviewer SHALL receive or read generated files or the user API contract; the path lists SHALL leave them out and the prompts SHALL tell reviewers to exclude them from the diff. All backend code belonging to the change SHALL be checked against the architecture rules, without an exemption for code predating the documented layout. For frontend changes, the architecture reviewer SHALL read the frontend architecture section of the architecture document and the applicable module conventions, including the rules for placement, composition, import direction and public APIs. Existing migration findings SHALL be accepted only when individually recorded in the finite baseline; a new or multiplied violation SHALL still be reported.

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

- **WHEN** the change adds an operation to the agent API contract whose error responses do not use the error format the API design skill requires
- **THEN** the architecture reviewer returns an `architecture` finding that names the rule of the API design skill

#### Scenario: Generated files and the user API contract stay out of review

- **WHEN** the change edits a generated file and the user API contract
- **THEN** neither path appears in a reviewer's path list and every reviewer prompt tells the reviewer to exclude them from `git diff`

#### Scenario: A test that waits with a fixed pause

- **WHEN** a test the change added sleeps for a fixed time before checking that a backup finished
- **THEN** the tests reviewer returns a finding that asks to wait for the condition instead

#### Scenario: A bug fix without a reproducing test

- **WHEN** a change fixes a bug and no test it added or changed fails on the code before the fix
- **THEN** the tests reviewer returns a `missing_test` finding

#### Scenario: Changed tests outside the verification commands

- **WHEN** the change adds tests to a backend package that the test plan does not select
- **THEN** the tests reviewer returns a `missing_test` finding that names the package

#### Scenario: Changed code outside the impact map

- **WHEN** the diff changes the storages module and the impact map does not mention it
- **THEN** the tests reviewer returns a `missing_test` finding that names the area

#### Scenario: An exclusion the code contradicts

- **WHEN** the plan excludes MySQL 5.7 as unaffected and the diff adds a branch for MySQL 5.7
- **THEN** the tests reviewer returns a `missing_test` finding that names the exclusion

#### Scenario: Changed browser scenarios outside the verification commands

- **WHEN** the change adds or changes a browser end-to-end scenario and its test plan does not select `suite e2e-browser`
- **THEN** the tests reviewer returns a `missing_test` finding that names the scenario

#### Scenario: Changed end-to-end tests without their command

- **WHEN** the change edits tests of the verification agent's end-to-end package and the test plan selects nothing that runs them
- **THEN** the tests reviewer returns a `missing_test` finding that names the package

#### Scenario: A verification that runs the whole test set

- **WHEN** a task's verification runs `mise run test`
- **THEN** the tests reviewer returns a `missing_test` finding that asks for the specific tests instead

#### Scenario: A change with only hook tests

- **WHEN** the change adds tests only to the hook tests and its test plan selects `suite review-hooks`
- **THEN** the tests reviewer returns no finding about the selection

#### Scenario: A task that builds the command

- **WHEN** a task names `mise run test` as something it adds or edits, and its verification runs another command
- **THEN** the tests reviewer returns no finding for naming it

#### Scenario: Frontend architecture has an authoritative document

- **WHEN** a frontend plan or implementation changes a slice boundary or screen composition
- **THEN** the architecture reviewer reads the frontend section of the architecture document and the applicable module conventions
- **AND** it reports any contradiction, including a violation authorized only by an overly broad migration exception

## ADDED Requirements

### Requirement: Test plan selections the hook accepts

The hook SHALL accept only the selector forms of the `change-test-plans` capability in a change's test plan, and parse them without running them. A missing plan, an empty selection, `none` beside another selector, an unknown form, a path outside its module or a selector that capability forbids SHALL block the turn with the reason and the allowed forms before any reviewer starts.

#### Scenario: Accepted selectors

- **WHEN** the selection holds `backend ./internal/modules/notifiers/...`, `frontend src/features/edit-notifier -t 'shows the error'` and `suite e2e-browser`
- **THEN** the hook accepts the format and runs none of them

#### Scenario: A plan that selects nothing

- **WHEN** the selection holds only `none`
- **THEN** the hook accepts the format

#### Scenario: A shell operator

- **WHEN** a selector reads `backend ./internal/lib/... && rm -rf /tmp/x`
- **THEN** the turn is blocked with the reason that the selector does not match its form, and nothing runs

#### Scenario: A path outside the module

- **WHEN** a selector reads `backend ../frontend` or `frontend ../backend`
- **THEN** the turn is blocked with the reason that the path leaves its module, and nothing runs

#### Scenario: The whole module or tree

- **WHEN** a selector reads `backend ./...`, `backend ./internal/...`, `frontend src` or `frontend src/features`
- **THEN** the turn is blocked with the reason that a plan never selects the whole test set

#### Scenario: The upgrade test as a package

- **WHEN** a selector reads `backend ./e2e/upgrade` or `backend ./e2e/...`
- **THEN** the turn is blocked with the reason that the upgrade test runs as `suite upgrade`

#### Scenario: A change without a test plan

- **WHEN** a change whose tasks are all checked has no `tests.md`
- **THEN** the turn is blocked with the reason that the test plan is missing, and no reviewer starts

#### Scenario: An empty selection

- **WHEN** the `tests` block under "Selection" holds no selector
- **THEN** the turn is blocked with the reason and the allowed forms

#### Scenario: Nothing beside something

- **WHEN** the selection holds `none` and `suite scripts`
- **THEN** the turn is blocked with the reason that `none` must be the only line

#### Scenario: An unknown suite or a command

- **WHEN** a selector reads `suite test` or `mise run test`
- **THEN** the turn is blocked with the reason and the list of allowed forms and suite names

#### Scenario: A block with another fence

- **WHEN** the selection is fenced as `sh` instead of `tests`
- **THEN** the turn is blocked with the reason that the plan has no selection

## REMOVED Requirements

### Requirement: Change verification commands the hook accepts

**Reason**: The "Change verification" section of the task list is replaced by the change's test plan, whose selectors the runner turns into commands.

**Migration**: The hook validates the plan's selection as "Test plan selections the hook accepts" describes. Changes archived before this change keep their sections, and the hook does not read archived changes.
