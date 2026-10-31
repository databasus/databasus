## MODIFIED Requirements

### Requirement: Which change verification commands the hook accepts

The change verification commands SHALL be the lines of the first code block fenced as `sh` under the "Change verification" heading of the change's task list. A block fenced as `bash`, `shell` or without a language does not count. Empty lines and comment lines are skipped. The hook SHALL accept only these command forms:

- backend package tests: `mise run backend:test-package` followed by one or more package paths, which the agent runs from the repository root and the task resolves in the backend directory;
- backend CI shards: `mise run backend:test-group` followed by one or more shard names of `physical`, `logical` or `physical-api`, which the agent runs from the repository root;
- the verification agent's tests: `mise run agent:test`, with no further arguments, which the agent runs from the repository root;
- frontend tests: `pnpm --dir frontend test`, optionally followed by test file paths or filters;
- isolated frontend browser tests: exactly `pnpm --dir frontend test:browser --project=ui` or `pnpm --dir frontend test:browser --project=e2e`, with no additional arguments.

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

#### Scenario: The isolated real-API browser command

- **WHEN** the verification block contains exactly `pnpm --dir frontend test:browser --project=e2e`
- **THEN** the hook accepts the command format without executing the real-API browser runner
- **AND** the implementing agent remains responsible for running the declared tests and providing verification evidence

#### Scenario: An unbounded browser command

- **WHEN** the block omits the browser project, selects a project other than `ui` or `e2e`, adds arguments or uses a different package script
- **THEN** the hook rejects the command before any reviewer starts
