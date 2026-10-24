## MODIFIED Requirements

### Requirement: Which change verification commands the hook runs
The change verification commands SHALL be the lines of the first code block fenced as `sh` under the "Change verification" heading of the change's task list. A block fenced as `bash`, `shell` or without a language does not count. Empty lines and comment lines are skipped. The hook SHALL accept only these command forms:

- backend package tests: `mise run backend:test-package` followed by one or more package paths, which the hook runs from the repository root and the task resolves in the backend directory;
- the verification agent's tests: `mise run agent:test`, with no further arguments, which the hook runs from the repository root;
- frontend tests: `pnpm --dir frontend test`, optionally followed by test file paths or filters.

The hook SHALL run each command directly, without a shell. The backend task passes its package paths on to a shell script, so every argument after the fixed prefix MUST consist only of letters, digits and the characters `.`, `/`, `_`, `-`, `=`, `:` and `@`, and a package path MUST start with `./`. This character set leaves out every shell operator and substitution. A line that is not one of the allowed forms or has an argument outside the allowed characters, a section without a closed `sh` block, and a block without commands SHALL each block the turn with the reason and the list of allowed forms. No check or reviewer runs in that case.

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
