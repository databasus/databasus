# implementation-review Specification

## Purpose

Reviews the finished implementation of an OpenSpec change as a whole, so that bugs, mismatches with the spec and scenarios without tests are found by the hook rather than by the user in a separate review session.

## Requirements

### Requirement: The full implementation review starts after a change is implemented

The hook that runs when the agent finishes a turn SHALL start the full implementation review when all of these hold: every task of an active change is checked, the working tree changed during this turn, and its state differs from the last state the same review client and review profile accepted for this change. The accepted state SHALL be stored per review client, review profile and change, and SHALL remain reusable across sessions of the same review client and review profile. The full implementation review MUST NOT start while a task is unchecked, in a turn that left the working tree unchanged, or in a turn whose request carries an OpenSpec planning command. Archived changes are not reviewed.

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

- **WHEN** the full implementation review accepted a change in one session, and another session of the same review client and review profile finishes with a working tree matching that accepted state but differing from its own request snapshot
- **THEN** the full implementation review does not start again because the stored acceptance matches the current state

#### Scenario: Edit after acceptance

- **WHEN** after an accepted review the agent changes files and finishes the turn
- **THEN** the full implementation review starts again

#### Scenario: Another client's acceptance does not suppress review

- **WHEN** one client's eligible implementation review sees a working tree changed since its request snapshot that another client accepted
- **THEN** it starts its own full review because the stored acceptance belongs to a different review client

### Requirement: Verification precedes AI review

The implementing agent SHALL run the relevant formatters, linters and tests before finishing the implementation turn that starts review. The implementation review hook SHALL run only read-only AI reviewers and SHALL NOT run linters, formatters, builds, hook tests or change verification commands. It SHALL validate the format of the change verification section without executing its commands. Reviewer prompts MUST NOT claim that review ran checks or that those checks passed; reviewers SHALL assess verification claims against the available evidence.

#### Scenario: A change names package tests

- **WHEN** implementation review starts for a change whose verification section contains `mise run test`
- **THEN** the hook starts AI reviewers without running those tests or any module linter

#### Scenario: The hooks changed

- **WHEN** implementation review starts for a change that modifies the review hooks
- **THEN** the hook does not run its own tests; the implementing agent runs them before review

#### Scenario: A verification claim

- **WHEN** a reviewer receives an implementation review prompt
- **THEN** the prompt requires checking the agent's verification evidence and does not assert that review ran or passed checks

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

### Requirement: Full review models

The review client SHALL be selected by the review-hook-launch capability. In Claude Code, full implementation and proposal reviewers SHALL run on Opus with `medium` effort, and quick checks SHALL run on Opus with `low` effort. In Codex, full implementation and proposal reviewers SHALL use the developer's configured Codex model with `high` reasoning effort, and quick checks SHALL use that model with `medium` reasoning effort. Full-review repeats SHALL use the same client and review profile for full reviews as their initial review. Quick checks include the plan, necessity, diff, architecture and code-shape gates wherever the client supports that gate. Review tooling MUST NOT require the other client to be installed or authenticated.

#### Scenario: Full review in Claude Code

- **WHEN** the full implementation review starts from Claude Code
- **THEN** its reviewers run in Claude Code on Opus with `medium` effort

#### Scenario: Quick check in Claude Code

- **WHEN** a supported quick check starts from Claude Code
- **THEN** its reviewer runs in Claude Code on Opus with `low` effort

#### Scenario: Full review in Codex

- **WHEN** the full implementation review starts from Codex
- **THEN** its reviewers run in Codex on the configured model with `high` reasoning effort

#### Scenario: Quick check in Codex

- **WHEN** a supported quick check starts from Codex
- **THEN** its reviewer runs in Codex on the configured model with `medium` reasoning effort

#### Scenario: A Codex implementation needs fixes

- **WHEN** a full implementation review in Codex returns a blocking finding and the agent fixes it
- **THEN** the repeat review runs in Codex with `high` reasoning effort and the existing repeat limit and summary behavior apply

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

After a blocked full review the next run SHALL check the previous findings and the files changed after that review, not the whole change again. The repeat SHALL run on the same models. There SHALL be at most 15 repeats per user request, after which the hook SHALL stop the agent and show the remaining findings.

#### Scenario: Findings are fixed

- **WHEN** the agent fixed every finding of the previous review and finishes the turn
- **THEN** the repeat review checks the previous findings and the new edits and accepts the change

#### Scenario: Repeat limit

- **WHEN** the 15th repeat within one user request blocks the turn again
- **THEN** the hook stops the agent and shows the remaining findings, says that the edits stay in the working tree without being accepted, and names `mise run review` for running the review again

### Requirement: The whole review runs by hand

The repository SHALL have a command, `mise run review [<change>] [--client <codex|claude>]`, that runs the review a finished change gets from the hooks without a hook event, as a fallback when hooks did not run or another review is needed. The client SHALL be selected by the review-hook-launch capability. It SHALL run only AI reviewers: the quick checks `diff`, `architecture` and `code-shape` over the working tree, `code-shape` only when a code file changed, and the full implementation review of the named change, or of every active change whose tasks are all checked. Quick and full reviewers SHALL share one parallel pool with at most six reviewers running at a time; full reviewers MUST NOT wait for all quick reviewers to finish. The command MUST NOT run linting, tests or builds. Verification is the implementing agent's preceding step, and manual review is not a routine extra step after the automatic hooks. The full review SHALL run even when the selected client and review profile accepted this state. When an earlier review stored findings for the same client, review profile and change, the full review SHALL be the repeat over those findings, as in the hook; otherwise it SHALL start from scratch. The command SHALL read no hook payload, report the selected client, print every finding and a closing verdict, and exit with code 0 when everything accepted the change, 1 when something did not or a reviewer could not run, and 2 when there is nothing to review or client selection is invalid. It SHALL store the full review's outcome per change and selected review profile like the hook does, so the next turn of that agent repeats review over those findings. The command SHALL run through the launcher of the hooks; when the launcher cannot start it, the command SHALL print the reason and exit with a non-zero code instead of answering with a stop message for the agent.

#### Scenario: Everything accepts

- **WHEN** the user runs manual review with an implemented change and every reviewer accepts
- **THEN** the command reports its client, prints that review accepted the change in its current state and exits with code 0

#### Scenario: Parallel manual review

- **WHEN** manual review includes quick reviewers and full implementation reviewers
- **THEN** both kinds run concurrently in the same pool without a separate quick-review phase

#### Scenario: Verification already ran

- **WHEN** the implementing agent completed linting and tests and starts manual review because a hook did not run
- **THEN** the command starts AI reviewers without repeating linting, tests or builds

#### Scenario: A finding to fix

- **WHEN** a manual reviewer returns a `bug` finding and the agent then fixes it and finishes its turn
- **THEN** the command prints the finding and exits with code 1, and the same client's automatic review repeats over the finding

#### Scenario: A run after the fixes

- **WHEN** manual review stored findings, the agent fixed them and manual review runs again with the same client and review profile
- **THEN** the full review repeats over those findings and the files changed since

#### Scenario: Nothing to review

- **WHEN** no active change has every task checked, or the named change does not exist
- **THEN** the command says so, runs no reviewer and exits with code 2

#### Scenario: A Codex-only manual run

- **WHEN** Codex runs manual review without an explicit client option and Claude Code is not installed
- **THEN** all quick and full reviewers run in Codex with their corresponding review profiles

#### Scenario: A Claude-only manual run

- **WHEN** Claude Code runs manual review without an explicit client option and Codex is not installed
- **THEN** all quick and full reviewers run in Claude Code with their corresponding review profiles

#### Scenario: Selected reviewer cannot run

- **WHEN** a selected reviewer crashes, cannot authenticate or times out during manual review
- **THEN** the command reports the reason, exits with code 1 and does not start the other client

#### Scenario: Manual review has no valid client selection

- **WHEN** manual review runs without a client option or agent declaration, or with an invalid selected client value
- **THEN** it explains the accepted client options, starts no reviewer and exits with code 2

### Requirement: The necessity check does not interfere with fixes

While a change has an unaccepted full implementation review, the necessity check MUST NOT start because of edits to that change's artifacts.

#### Scenario: The agent fixes a delta spec after a finding

- **WHEN** after a `spec_mismatch` finding the agent edits the change's delta spec and finishes the turn
- **THEN** the necessity check does not start, and the full review checks the edit

### Requirement: The full review ends within the hook timeout

The full implementation review SHALL work within a time budget that ends before the hook timeout of the client configuration, with a margin for the hook's own Git, OpenSpec and file work. A reviewer SHALL start only when the budget still holds its whole time limit. A reviewer that cannot start for that reason SHALL count as a reviewer that could not run: the review is not accepted, and the agent gets the reason. The hook MUST NOT be killed by the client timeout without a message.

#### Scenario: Too many reviewers for the budget

- **WHEN** a change has so many delta specs that a third wave of reviewers would not fit in the remaining budget
- **THEN** the reviewers of that wave do not start, the turn is blocked, and the agent gets the reason that the review time budget ran out

#### Scenario: Two waves of reviewers

- **WHEN** the change needs two waves of reviewers and their time limits fit the review budget
- **THEN** both waves run and the hook answers before the client timeout

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
