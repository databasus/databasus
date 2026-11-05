## ADDED Requirements

### Requirement: CI lints each module with its local lint command

The CI lint job of the backend, of the verification agent and of the frontend SHALL run the same lint command a developer runs locally for that module, and the shell script check SHALL run the same command as locally too. These jobs SHALL take every tool the command calls at the version the repository pins. A check added to a module's local lint SHALL run in that module's CI lint job without any change to the CI configuration, as long as the check uses tools that job already installs. A check that needs a tool the job does not install yet adds that tool to the job's list of tools, and only that.

#### Scenario: A check is added to the backend lint

- **WHEN** a developer adds a check that uses an already installed tool to the backend's local lint and pushes a commit that fails it
- **THEN** the backend lint job fails on that check, and the CI configuration was not changed

#### Scenario: Generated code is out of date

- **WHEN** a pull request changes the API contract without regenerating the code
- **THEN** the backend lint job fails and names each stale generated Go file
- **AND** the frontend lint job fails and names the generated frontend API types file

#### Scenario: A pinned tool version changes

- **WHEN** a developer changes the pinned version of the Go linter
- **THEN** the next backend lint job runs the linter at the new version, and the CI configuration was not changed
