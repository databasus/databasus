## MODIFIED Requirements

### Requirement: What the full implementation review checks

The full implementation review SHALL run several read-only reviewers in parallel: one per delta spec of the change, one for documentation and the task list, one for tests, and one for architecture. The delta spec reviewer SHALL find the code and the test for every requirement and scenario, check that the test actually proves the scenario, check the design decisions against the code, and look for bugs. The documentation reviewer SHALL check that everything each task states was done, including its verification, and that documentation agrees with itself and with the code. The documentation reviewer SHALL also check edits to agent instructions made by the change against the requirement "What belongs in agent instructions" of the `agent-instructions` capability.

The tests reviewer SHALL check every test the change added or changed against the repository's testing skill and the testing rules of the module docs: the test fails without the behavior it pins, a bug fix has a test that reproduces the bug, the cases the module doc asks for are covered, the test waits for a condition instead of a fixed pause, and it cleans up its data. The tests reviewer SHALL also check that the change verification commands run the tests of every backend package, verification agent package and frontend test file the change added or changed tests in. Tests that no allowed command form runs, such as the hook tests, SHALL name an explicit verification command in their task instead of this section.

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

- **WHEN** the change adds tests to a backend package that none of its change verification commands runs
- **THEN** the tests reviewer returns a `missing_test` finding that names the package

#### Scenario: A change with only hook tests

- **WHEN** the change adds tests only to the hook tests and has no "Change verification" section
- **THEN** the tests reviewer returns no finding about the missing section when the task names an explicit command for the agent to run those tests

#### Scenario: Frontend architecture has an authoritative document

- **WHEN** a frontend plan or implementation changes a slice boundary or screen composition
- **THEN** the architecture reviewer reads the frontend section of the architecture document and the applicable module conventions
- **AND** it reports any contradiction, including a violation authorized only by an overly broad migration exception

### Requirement: Which change verification commands the hook accepts

The change verification commands SHALL be the lines of the first code block fenced as `sh` under the "Change verification" heading of the change's task list. A block fenced as `bash`, `shell` or without a language does not count. Empty lines and comment lines are skipped. The hook SHALL accept only these command forms:

- backend package tests: `mise run backend:test-package` followed by one or more package paths, which the agent runs from the repository root and the task resolves in the backend directory;
- backend CI shards: `mise run backend:test-group` followed by one or more shard names of `physical`, `logical` or `physical-api`, which the agent runs from the repository root;
- the verification agent's tests: `mise run agent:test`, with no further arguments, which the agent runs from the repository root;
- frontend tests: `pnpm --dir frontend test`, optionally followed by test file paths or filters;
- isolated frontend browser UI tests: exactly `pnpm --dir frontend test:browser --project=ui`, with no additional arguments.

The implementing agent SHALL run these commands before review. The hook SHALL parse them only to validate the section and MUST NOT execute them. The backend task passes its package paths on to a shell script, so every argument after the fixed prefix MUST consist only of letters, digits and the characters `.`, `/`, `_`, `-`, `=`, `:` and `@`, and a package path MUST start with `./`. A shard name MUST match `<group>-<index>-of-<count>` with positive integers, and its index MUST NOT exceed its count. Unsharded groups, unknown groups and `other` MUST be rejected. This character set leaves out every shell operator and substitution. A line that is not one of the allowed forms or has an argument outside the allowed characters, a section without a closed `sh` block, and a block without commands SHALL each block the turn with the reason and the list of allowed forms. No reviewer starts in that case.

#### Scenario: A full suite run in the block

- **WHEN** the change verification block contains `mise run test`
- **THEN** the turn is blocked with the reason that the command is not allowed, and nothing runs

#### Scenario: A shell operator

- **WHEN** a line reads `mise run backend:test-package ./internal/features/intervals/... && rm -rf /tmp/x`
- **THEN** the turn is blocked with the reason that the argument `&&` has a forbidden character, and nothing runs

#### Scenario: An argument with a forbidden character

- **WHEN** a line reads `mise run backend:test-package ./internal/a;rm`
- **THEN** the turn is blocked with the reason that an argument has a forbidden character, and nothing runs

#### Scenario: Two changes name the same command

- **WHEN** the hook reviews two changes whose blocks both contain `pnpm --dir frontend test`
- **THEN** the hook validates both sections without running the command

#### Scenario: Backend package tests

- **WHEN** a block contains `mise run backend:test-package ./internal/features/intervals/...`
- **THEN** the hook accepts the command format without running the package tests

#### Scenario: The old command form

- **WHEN** a block contains `just test-package ./internal/features/intervals/...`
- **THEN** the turn is blocked with the reason that the command is not allowed and the list of allowed forms, and nothing runs

#### Scenario: Verification agent tests

- **WHEN** a block contains `mise run agent:test`
- **THEN** the hook accepts the command format without running the verification agent tests

#### Scenario: Verification agent tests with an argument

- **WHEN** a block contains `mise run agent:test ./x`
- **THEN** the turn is blocked with the reason that the line does not match the form, and nothing runs

#### Scenario: Parallel physical API shards

- **WHEN** the block contains `mise run backend:test-group physical-api-1-of-3 physical-api-2-of-3 physical-api-3-of-3`
- **THEN** the hook accepts the command format without running the CI selections

#### Scenario: An unbounded backend group

- **WHEN** the block names an unsharded group, `other`, an unknown group or a shard outside its count
- **THEN** the hook rejects the command before any test or reviewer starts

#### Scenario: The isolated browser UI command

- **WHEN** the verification block contains exactly `pnpm --dir frontend test:browser --project=ui`
- **THEN** the hook accepts the command format without executing the browser runner
- **AND** the implementing agent remains responsible for running the declared tests and providing verification evidence

#### Scenario: An unbounded browser command

- **WHEN** the block omits the browser project, selects another project, adds arguments or uses a different package script
- **THEN** the hook rejects the command before any reviewer starts
