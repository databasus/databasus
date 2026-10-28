# implementation-review Specification

## Purpose
Reviews the finished implementation of an OpenSpec change as a whole, so that bugs, mismatches with the spec and scenarios without tests are found by the hook rather than by the user in a separate review session.

## Requirements

### Requirement: The full implementation review starts after a change is implemented
The hook that runs when the agent finishes a turn SHALL start the full implementation review when all of these hold: every task of an active change is checked, the working tree changed during this turn, and its state differs from the last state the full implementation review accepted for this change. The accepted state SHALL be stored per change, not per session. The full implementation review MUST NOT start while a task is unchecked, in a turn that left the working tree unchanged, or in a turn whose request carries an OpenSpec planning command. Archived changes are not reviewed.

#### Scenario: The agent checks the last task
- **WHEN** the agent checks the last task of a change and finishes the turn
- **THEN** the hook starts the full implementation review of that change

#### Scenario: Tasks remain
- **WHEN** a task of the change is unchecked and the agent finishes the turn
- **THEN** the full implementation review does not start and only the quick checks run

#### Scenario: A turn without changes
- **WHEN** every task of the change is checked but the working tree did not change during the turn
- **THEN** the full implementation review does not start

#### Scenario: New session after acceptance
- **WHEN** the full implementation review accepted a change in one session, and in a new session the agent finishes a turn with the working tree still in the accepted state
- **THEN** the full implementation review does not start again

#### Scenario: Edit after acceptance
- **WHEN** after an accepted review the agent changes files and finishes the turn
- **THEN** the full implementation review starts again

### Requirement: Read-only checks of the touched modules run first
Before starting reviewers the hook SHALL run the checks of every module the working tree changes relative to the last commit: the Go linter for the backend and for the verification agent, the linter for the frontend, the linter and the build for the website, and the hook tests when the hooks changed. After them the hook SHALL run the change verification commands of every change it reviews, each command once even when several changes name it. The checks MUST NOT modify files in the working tree, so formatters do not run. The hook MUST NOT run the full backend suite or the Docker filesystem suite. When a check fails, the hook SHALL block the turn and pass the agent the command and the end of its output. Reviewers MUST NOT start in that case.

Each check SHALL have its own time limit, and all checks of one review SHALL share a total time limit, both within the time budget of the requirement "The full review ends within the hook timeout". When a check exceeds either limit, the hook SHALL stop the check together with every process it started, block the turn and name the limit that was exceeded.

#### Scenario: The backend linter fails
- **WHEN** the change touches the backend and its linter fails
- **THEN** the turn is blocked, the agent gets the end of the linter output, and no reviewer starts

#### Scenario: Only the frontend changed
- **WHEN** the change touches only the frontend and has no change verification commands
- **THEN** the hook runs the frontend linter and no backend, agent or website check

#### Scenario: The website changed
- **WHEN** the change touches the website
- **THEN** the hook runs both the website linter and the website build

#### Scenario: A package test fails
- **WHEN** the change verification commands run the tests of a backend package and one of them fails
- **THEN** the turn is blocked, the agent gets the command and the end of the test output, and no reviewer starts

#### Scenario: A test hangs
- **WHEN** a change verification command runs longer than its time limit
- **THEN** the hook stops the command and every process it started, blocks the turn and names the exceeded limit

#### Scenario: The checks together exceed the total limit
- **WHEN** each check stays under its own time limit, but together they run past the total limit
- **THEN** the hook stops the running check and every process it started, blocks the turn and names the total limit

#### Scenario: The linters pass and the change has no verification commands
- **WHEN** the change has no "Change verification" section and the linters pass
- **THEN** the reviewers start without any test run

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

### Requirement: Full review models
The full review reviewers SHALL run on Opus with `medium` effort. The quick checks SHALL run on Opus with `low` effort. The reviewers SHALL run in Claude Code whichever client the agent works in, so a change implemented in Codex gets the same review as one implemented in Claude Code.

#### Scenario: Full review in Claude Code
- **WHEN** the full implementation review starts from Claude Code
- **THEN** the reviewers run on Opus with `medium` effort

#### Scenario: Quick check in Claude Code
- **WHEN** the diff check starts from Claude Code
- **THEN** the reviewer runs on Opus with `low` effort

#### Scenario: Full review in Codex
- **WHEN** the full implementation review starts from Codex
- **THEN** the reviewers run in Claude Code on Opus with `medium` effort

#### Scenario: Quick check in Codex
- **WHEN** the diff check starts from Codex
- **THEN** the reviewer runs in Claude Code on Opus with `low` effort

### Requirement: Which findings block the turn
A reviewer SHALL return findings, each with a category (`bug`, `spec_mismatch`, `missing_test`, `architecture`, `docs`, `minor`), a path and a description. The hook SHALL block the turn when at least one finding is not `minor`, and pass the agent every finding with `minor` marked optional. The block text SHALL require the agent to resolve every finding without asking the user: fix the code or, when the implementation is right, the change's delta spec; for an `architecture` finding, move the code to follow `docs/architecture.md`, or change `docs/architecture.md` in the same change when the rule itself is wrong. The block text SHALL also require the agent to finish the turn again with one short line about its fixes that names every rejected finding with the reason, and to leave the summary for the user until the review accepts the change. With no findings or only `minor` ones, the review SHALL count as accepted. When the review accepts the change, the hook SHALL block the turn once and ask the agent to write the user the full summary of the change. The summary SHALL open by saying that the automatic review passed and accepted the change in its current state, because the agent's reply before the review could not know the outcome. It SHALL then give what was done for the change's tasks, which decisions were made, which checks passed, which `minor` findings were left, and the next step, and SHALL end with a short list of what the review made the agent fix. The hook SHALL pass the `minor` findings in that request.

#### Scenario: Accepted at the first review
- **WHEN** every finding is `minor` or there are none, and no earlier review in this request blocked the turn
- **THEN** the state is stored as accepted, and the hook blocks the turn once and asks the agent for a summary that opens with the accepted review and lists the `minor` findings

#### Scenario: Only minor findings
- **WHEN** every finding is `minor`
- **THEN** the summary request passes them marked optional, and the next turn without edits is not blocked

#### Scenario: Accepted after fixes
- **WHEN** a repeat review accepts a change the previous review blocked
- **THEN** the hook blocks the turn once and asks the agent for the full summary of the change that ends with a short list of the review's fixes, and the next turn without edits is not blocked

#### Scenario: A bug is found
- **WHEN** a reviewer returns a `bug` finding
- **THEN** the turn is blocked, and the agent gets every finding with the instruction to resolve them itself and to finish the turn with one short line about the fixes and the rejected findings

#### Scenario: An architecture finding
- **WHEN** a reviewer returns an `architecture` finding
- **THEN** the turn is blocked, and the block text tells the agent to move the code or to change `docs/architecture.md` in the same change

#### Scenario: The reviewer cannot run
- **WHEN** the reviewer process crashes or times out
- **THEN** the review is not accepted and the agent gets the failure reason

### Requirement: Repeat review after fixes
After a blocked full review the next run SHALL check the previous findings and the files changed after that review, not the whole change again. The repeat SHALL run on the same models. There SHALL be at most two repeats per user request, after which the hook SHALL stop the agent and show the remaining findings.

#### Scenario: Findings are fixed
- **WHEN** the agent fixed every finding of the previous review and finishes the turn
- **THEN** the repeat review checks the previous findings and the new edits and accepts the change

#### Scenario: Repeat limit
- **WHEN** the third review in a row within one user request blocks the turn again
- **THEN** the hook stops the agent and shows the remaining findings, says that the edits stay in the working tree without being accepted, and names `mise run review` for running the review again

### Requirement: The whole review runs by hand
The repository SHALL have a command, `mise run review [<change>]`, that runs the review a finished change gets from the hooks without a hook event: the quick checks `diff`, `architecture` and `code-shape` over the working tree, `code-shape` only when a code file changed, then the full implementation review of the named change, or of every active change whose tasks are all checked. The full review SHALL run even when the hook accepted this state. When an earlier review stored findings for the change, the full review SHALL be the repeat over those findings, as in the hook; otherwise it SHALL start from scratch. The command SHALL read no hook payload, print a report with every finding and a closing verdict, and exit with code 0 when everything accepted the change, 1 when something did not, and 2 when there is nothing to review. It SHALL store the full review's outcome per change like the hook does, so the next turn of the agent repeats the review over those findings. The command SHALL run through the launcher of the hooks; when the launcher cannot start it, the command SHALL print the reason and exit with a non-zero code instead of answering with a stop message for the agent.

#### Scenario: Everything accepts
- **WHEN** the user runs `mise run review` with an implemented change and every reviewer accepts
- **THEN** the command prints that the review accepted the change in its current state and exits with code 0

#### Scenario: A finding to fix
- **WHEN** a reviewer of the manual run returns a `bug` finding, and the agent then fixes it and finishes its turn
- **THEN** the command prints the finding and exits with code 1, and the hook of that turn runs the repeat review over the finding

#### Scenario: A run after the fixes
- **WHEN** a manual run stored findings, the agent fixed them, and the user runs `mise run review` again
- **THEN** the full review is the repeat over those findings and the files changed since

#### Scenario: Nothing to review
- **WHEN** no active change has every task checked, or the named change does not exist
- **THEN** the command says so, runs no reviewer and exits with code 2

### Requirement: The necessity check does not interfere with fixes
While a change has an unaccepted full implementation review, the necessity check MUST NOT start because of edits to that change's artifacts.

#### Scenario: The agent fixes a delta spec after a finding
- **WHEN** after a `spec_mismatch` finding the agent edits the change's delta spec and finishes the turn
- **THEN** the necessity check does not start, and the full review checks the edit

### Requirement: Which change verification commands the hook runs
The change verification commands SHALL be the lines of the first code block fenced as `sh` under the "Change verification" heading of the change's task list. A block fenced as `bash`, `shell` or without a language does not count. Empty lines and comment lines are skipped. The hook SHALL accept only these command forms:

- backend package tests: `mise run backend:test-package` followed by one or more package paths, which the hook runs from the repository root and the task resolves in the backend directory;
- backend CI shards: `mise run backend:test-group` followed by one or more shard names of `physical`, `logical` or `physical-api`, which the hook runs from the repository root;
- the verification agent's tests: `mise run agent:test`, with no further arguments, which the hook runs from the repository root;
- frontend tests: `pnpm --dir frontend test`, optionally followed by test file paths or filters.

The hook SHALL run each command directly, without a shell. The backend task passes its package paths on to a shell script, so every argument after the fixed prefix MUST consist only of letters, digits and the characters `.`, `/`, `_`, `-`, `=`, `:` and `@`, and a package path MUST start with `./`. A shard name MUST match `<group>-<index>-of-<count>` with positive integers, and its index MUST NOT exceed its count. Unsharded groups, unknown groups and `other` MUST be rejected. This character set leaves out every shell operator and substitution. A line that is not one of the allowed forms or has an argument outside the allowed characters, a section without a closed `sh` block, and a block without commands SHALL each block the turn with the reason and the list of allowed forms. No check or reviewer runs in that case.

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
- **THEN** the command runs once

#### Scenario: Backend package tests
- **WHEN** a block contains `mise run backend:test-package ./internal/features/intervals/...`
- **THEN** the hook runs that command, and the tests of that backend package run

#### Scenario: The old command form
- **WHEN** a block contains `just test-package ./internal/features/intervals/...`
- **THEN** the turn is blocked with the reason that the command is not allowed and the list of allowed forms, and nothing runs

#### Scenario: Verification agent tests
- **WHEN** a block contains `mise run agent:test`
- **THEN** the hook runs that command from the repository root, and the verification agent's tests run

#### Scenario: Verification agent tests with an argument
- **WHEN** a block contains `mise run agent:test ./x`
- **THEN** the turn is blocked with the reason that the line does not match the form, and nothing runs

#### Scenario: Parallel physical API shards
- **WHEN** the block contains `mise run backend:test-group physical-api-1-of-3 physical-api-2-of-3 physical-api-3-of-3`
- **THEN** the hook runs that command, which runs the three CI selections concurrently in separate processes and fails if any selection fails

#### Scenario: An unbounded backend group
- **WHEN** the block names an unsharded group, `other`, an unknown group or a shard outside its count
- **THEN** the hook rejects the command before any test or reviewer starts

### Requirement: The full review ends within the hook timeout
The full implementation review SHALL work within a time budget that ends before the hook timeout of the client configuration, with a margin for the hook's own Git, OpenSpec and file work. The checks SHALL run inside that budget. A reviewer SHALL start only when the budget still holds its whole time limit. A reviewer that cannot start for that reason SHALL count as a reviewer that could not run: the review is not accepted, and the agent gets the reason. The hook MUST NOT be killed by the client timeout without a message.

#### Scenario: Too many reviewers for the budget
- **WHEN** a change has so many delta specs that a third wave of reviewers would not fit in the remaining budget
- **THEN** the reviewers of that wave do not start, the turn is blocked, and the agent gets the reason that the review time budget ran out

#### Scenario: Two waves of reviewers after slow checks
- **WHEN** the checks use their whole total limit and the change needs two waves of reviewers
- **THEN** both waves run and the hook answers before the client timeout
