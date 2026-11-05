## MODIFIED Requirements

### Requirement: Root commands lint and format every module

The repository SHALL provide `mise run lint` and `mise run format`, covering the backend, whose module includes the verification agent, and the frontend. `mise run lint` SHALL exit with a non-zero status when any module fails its checks, and SHALL report the failures of every failing module, not only the first. `mise run format` SHALL rewrite files in both modules to their formatter's output. Each module SHALL also have its own lint and format tasks, `mise run backend:lint`, `mise run frontend:lint` and the matching `format` tasks, and every task of a module SHALL run in that module's directory whichever directory of the repository it is started from. The backend's lint and format tasks SHALL check and format the verification agent's code together with the server's, with the same rules; there SHALL be no separate lint or format task for the agent. The Go mock server of the agent's old end-to-end suite, which stays outside both modules until it is rewritten, is not linted or formatted by these commands. The suite's shell scripts remain covered by the shell script check.

#### Scenario: One module fails lint

- **WHEN** the frontend has a lint error and the backend is clean
- **THEN** `mise run lint` exits with a non-zero status and reports the frontend error

#### Scenario: Two modules fail lint

- **WHEN** both the backend and the frontend have a lint error
- **THEN** `mise run lint` exits with a non-zero status and reports the errors of both modules

#### Scenario: The verification agent fails lint

- **WHEN** the verification agent's code breaks a rule of the backend lint and the rest of the backend is clean
- **THEN** `mise run backend:lint` and `mise run lint` exit with a non-zero status and report the agent's error

#### Scenario: Developer formats everything

- **WHEN** a developer runs `mise run format`
- **THEN** a later `mise run lint` reports no formatting difference in either module, the verification agent included

#### Scenario: Module task started from another directory

- **WHEN** a developer runs `mise run backend:lint` from the frontend directory
- **THEN** the backend checks run against the backend module and pass or fail as they would from the backend directory

### Requirement: Commits are checked before they are created

Creating a commit SHALL check that each staged file of each module the commit touches matches that module's formatter, SHALL run that module's lint checks, and for the frontend its type check as well, and SHALL refuse the commit when any of them fails. When a staged file is not formatted, the refusal SHALL name the file and the command that formats it. The check reads each staged file as it is in the working tree, because it leaves the working tree and the index as they are; content staged for the commit that differs from the working tree is not checked separately. A module the commit does not touch SHALL NOT be checked. The verification agent's code belongs to the backend module, so a commit that touches it runs the backend checks. A commit that touches only the agent's old end-to-end suite, which lies outside both modules, runs no check.

#### Scenario: Unformatted file is staged

- **WHEN** a developer commits a staged frontend file that is not formatted
- **THEN** no commit is created, and the output names the file and the command that formats it

#### Scenario: Lint error in a staged module

- **WHEN** a developer commits a backend change that fails the backend lint checks
- **THEN** no commit is created and the lint output is shown

#### Scenario: Lint error in the verification agent

- **WHEN** a developer commits a change to the verification agent that fails the backend lint checks
- **THEN** no commit is created and the backend lint output is shown

#### Scenario: Untouched module

- **WHEN** a commit touches only the backend
- **THEN** no frontend check runs

#### Scenario: Only the old agent end-to-end suite changes

- **WHEN** a commit touches only files of the verification agent's old end-to-end suite, such as its mock server
- **THEN** no backend or frontend check runs

#### Scenario: Staged content differs from the working tree

- **WHEN** a developer stages an unformatted Go file, then formats it in the working tree without staging the result, and commits
- **THEN** the formatting check passes, because it reads the formatted working-tree file
