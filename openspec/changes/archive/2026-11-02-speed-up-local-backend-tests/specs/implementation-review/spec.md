## REMOVED Requirements

### Requirement: Which change verification commands the hook accepts
**Reason**: It lists the narrow backend, verification agent and frontend commands and has scenarios for each. A delta cannot drop one scenario of a requirement, and this change removes those forms.
**Migration**: "Change verification commands the hook accepts" holds the same rule for the remaining forms: `mise run test` and the two browser commands.

## ADDED Requirements

### Requirement: Change verification commands the hook accepts

The change verification commands SHALL be the lines of the first code block fenced as `sh` under the "Change verification" heading of the change's task list. A block fenced as `bash`, `shell` or without a language does not count. Empty lines and comment lines are skipped. The hook SHALL accept only these command forms:

- the routine local tests: exactly `mise run test`, with no further arguments, which the agent runs from the repository root;
- isolated frontend browser tests: exactly `pnpm --dir frontend test:browser --project=ui` or `pnpm --dir frontend test:browser --project=e2e`, with no additional arguments.

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

- **WHEN** the verification block contains exactly `pnpm --dir frontend test:browser --project=e2e`
- **THEN** the hook accepts the command format without executing the real-API browser runner
- **AND** the implementing agent remains responsible for running the declared tests and providing verification evidence

#### Scenario: An unbounded browser command

- **WHEN** the block omits the browser project, selects a project other than `ui` or `e2e`, adds arguments or uses a different package script
- **THEN** the hook rejects the command before any reviewer starts

## MODIFIED Requirements

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
