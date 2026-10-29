## MODIFIED Requirements

### Requirement: The lint jobs check Go dependencies
The CI backend lint job SHALL verify that the downloaded Go modules match the recorded checksums and SHALL scan the module for known vulnerabilities in code it actually calls. The backend module includes the verification agent, so the job covers the server and the agent together, and there SHALL be no separate lint job for the agent. A checksum mismatch or a reachable known vulnerability SHALL fail the job. The vulnerability scanner SHALL be pinned to a fixed version, so a run does not change its behavior when the scanner releases.

#### Scenario: A reachable vulnerability
- **WHEN** the backend calls a function of a dependency version with a published vulnerability
- **THEN** the backend lint job fails and names the vulnerability and the call path

#### Scenario: A vulnerability in code that is never called
- **WHEN** a dependency has a published vulnerability only in a function the backend never calls
- **THEN** the backend lint job passes

#### Scenario: A vulnerability in the verification agent
- **WHEN** the verification agent calls a function of a dependency version with a published vulnerability
- **THEN** the backend lint job fails and names the vulnerability and the call path

#### Scenario: A tampered module
- **WHEN** a downloaded module's content does not match its recorded checksum
- **THEN** the lint job fails before the linters run

### Requirement: CI lints each module with its local lint command

The CI lint job of the backend and of the frontend SHALL run the same lint command a developer runs locally for that module, and the shell script check SHALL run the same command as locally too. The backend lint job SHALL check the verification agent's code, because the backend's local lint does. The Go code of the agent's old end-to-end suite, outside both modules until it is rewritten, is linted by no job. Its shell scripts remain covered by the shell script check. These jobs SHALL take every tool the command calls at the version the repository pins. A check added to a module's local lint SHALL run in that module's CI lint job without any change to the CI configuration, as long as the check uses tools that job already installs. A check that needs a tool the job does not install yet adds that tool to the job's list of tools, and only that.

#### Scenario: A check is added to the backend lint

- **WHEN** a developer adds a check that uses an already installed tool to the backend's local lint and pushes a commit that fails it
- **THEN** the backend lint job fails on that check, and the CI configuration was not changed

#### Scenario: The verification agent breaks a lint rule

- **WHEN** a pull request adds code to the verification agent that breaks a rule of the backend lint
- **THEN** the backend lint job fails and names the agent's file

#### Scenario: Generated code is out of date

- **WHEN** a pull request changes the API contract without regenerating the code
- **THEN** the backend lint job fails and names each stale generated Go file
- **AND** the frontend lint job fails and names the generated frontend API types file

#### Scenario: A pinned tool version changes

- **WHEN** a developer changes the pinned version of the Go linter
- **THEN** the next backend lint job runs the linter at the new version, and the CI configuration was not changed
