## MODIFIED Requirements

### Requirement: What the full implementation review checks
The full implementation review SHALL run several read-only reviewers in parallel: one per delta spec of the change, one for documentation and the task list, one for tests, and one for architecture. The delta spec reviewer SHALL find the code and the test for every requirement and scenario, check that the test actually proves the scenario, check the design decisions against the code, and look for bugs. The documentation reviewer SHALL check that everything each task states was done, including its verification, and that documentation agrees with itself and with the code. The documentation reviewer SHALL also check edits to agent instructions made by the change against the requirement "What belongs in agent instructions" of the `agent-instructions` capability.

The tests reviewer SHALL check every test the change added or changed against the repository's testing skill and the testing rules of the module docs: the test fails without the behavior it pins, a bug fix has a test that reproduces the bug, the cases the module doc asks for are covered, the test waits for a condition instead of a fixed pause, and it cleans up its data. The tests reviewer SHALL also check that the change verification commands run the tests of every backend package, verification agent package and frontend test file the change added or changed tests in. Tests that no allowed command form runs and that a module check already runs, such as the hook tests, count as covered.

The architecture reviewer SHALL check all code that belongs to the change, including parts committed or written in earlier requests, against `docs/architecture.md` and the architecture rules of the module docs: layer direction, module boundaries, roles of files inside a module, data ownership and migrations. It SHALL report a violation that appears only when several edits of the change are read together, such as an import added in one request and used across a module boundary in another. When the change edits an HTTP contract file under `backend/api/`, the architecture reviewer SHALL also check the edit against the repository's API design skill and report each broken rule as an `architecture` finding. Code under `backend/` and `agent/` outside the layout block in "Backend layout" of `docs/architecture.md`, which keeps its existing structure, SHALL NOT be reported, unless the change moves that code. The exemption does not cover the frontend.

#### Scenario: A scenario without a test
- **WHEN** a delta spec scenario has no test, or its test passes without the described behavior
- **THEN** the reviewer returns a `missing_test` finding with the path to the test or code

#### Scenario: Documentation lags behind
- **WHEN** the change adds an environment variable and the configuration documentation does not mention it
- **THEN** the documentation reviewer returns a `docs` finding

#### Scenario: The change retells a document in AGENTS.md
- **WHEN** the change adds a list of a module doc's sections to the root `AGENTS.md`
- **THEN** the documentation reviewer returns a `docs` finding that suggests naming the doc's topic instead

#### Scenario: The change adds a rule to AGENTS.md
- **WHEN** the change adds a rule to `AGENTS.md` that changes agent behavior and is written nowhere else
- **THEN** the documentation reviewer returns no finding under the instruction content criterion

#### Scenario: A change with two delta specs
- **WHEN** the full implementation review starts for a change with two delta specs
- **THEN** the hook starts five reviewers in parallel: one per delta spec, one for documentation and tasks, one for tests, and one for architecture

#### Scenario: A boundary violation spread over two requests
- **WHEN** one request of the change adds a dependency on another module's internal package and a later request calls it from a service
- **THEN** the architecture reviewer returns an `architecture` finding that names both paths and the rule in `docs/architecture.md`

#### Scenario: Old code that keeps its structure
- **WHEN** the change edits a file outside the target layout, such as one under `backend/internal/middleware`, and keeps that file's existing structure
- **THEN** the architecture reviewer returns no finding about the layout of that file

#### Scenario: A frontend violation is still reported
- **WHEN** the change makes a frontend slice import another slice of the same layer
- **THEN** the architecture reviewer returns an `architecture` finding

#### Scenario: A contract edit that breaks an API rule
- **WHEN** the change adds an operation to `backend/api/openapi.yaml` whose error responses do not use the error format the API design skill requires
- **THEN** the architecture reviewer returns an `architecture` finding that names the rule of the API design skill

#### Scenario: A test that waits with a fixed pause
- **WHEN** a test the change added sleeps for a fixed time before checking that a backup finished
- **THEN** the tests reviewer returns a finding that asks to wait for the condition instead

#### Scenario: A bug fix without a reproducing test
- **WHEN** a change fixes a bug and no test it added or changed fails on the code before the fix
- **THEN** the tests reviewer returns a `missing_test` finding

#### Scenario: Changed tests outside the verification commands
- **WHEN** the change adds tests to a backend package that none of its change verification commands runs
- **THEN** the tests reviewer returns a `missing_test` finding that names the package

#### Scenario: A change with only hook tests
- **WHEN** the change adds tests only to the hook tests and has no "Change verification" section
- **THEN** the tests reviewer returns no finding about the missing section, because the module check runs the hook tests

### Requirement: Which findings block the turn
A reviewer SHALL return findings, each with a category (`bug`, `spec_mismatch`, `missing_test`, `architecture`, `docs`, `minor`), a path and a description. The hook SHALL block the turn when at least one finding is not `minor`, and pass the agent every finding with `minor` marked optional. When the turn is not blocked, the user SHALL see the `minor` findings. The block text SHALL require the agent to resolve every finding without asking the user: fix the code or, when the implementation is right, the change's delta spec; for an `architecture` finding, move the code to follow `docs/architecture.md`, or change `docs/architecture.md` in the same change when the rule itself is wrong. The block text SHALL also require the agent to finish the turn again with one short line about its fixes that names every rejected finding with the reason, and to leave the summary for the user until the review accepts the change. With no findings or only `minor` ones, the review SHALL count as accepted. When the review accepts the change after at least one block within this user request, the hook SHALL block the turn once and ask the agent to write the user a summary of the implementation as if there had been no review: what was done for the change's tasks, which decisions were made, which checks passed, which `minor` findings were left, and the next step, without review rounds, findings or the fixes made for them. The hook SHALL pass the `minor` findings in that request.

#### Scenario: Only minor findings
- **WHEN** every finding is `minor`, and no earlier review in this request blocked the turn
- **THEN** the state is stored as accepted and the user sees the findings

#### Scenario: Accepted after fixes
- **WHEN** a repeat review accepts a change the previous review blocked
- **THEN** the hook blocks the turn once and asks the agent for a summary of the implementation that leaves out review rounds and the fixes made for them, and the next turn without edits is not blocked

#### Scenario: A bug is found
- **WHEN** a reviewer returns a `bug` finding
- **THEN** the turn is blocked, and the agent gets every finding with the instruction to resolve them itself and to finish the turn with one short line about the fixes and the rejected findings

#### Scenario: An architecture finding
- **WHEN** a reviewer returns an `architecture` finding
- **THEN** the turn is blocked, and the block text tells the agent to move the code or to change `docs/architecture.md` in the same change

#### Scenario: The reviewer cannot run
- **WHEN** the reviewer process crashes or times out
- **THEN** the review is not accepted and the agent gets the failure reason
