## ADDED Requirements

### Requirement: Frontend CI uses the repository-pinned toolchain

Frontend lint, build and test checks SHALL use the same runtime and package-manager versions pinned for local development. Dependency installation SHALL require the committed lockfile without updating it. The addition of architecture checks SHALL remain part of frontend lint; existing release requirements for frontend checks SHALL remain in force.

#### Scenario: A frontend test job starts

- **WHEN** CI prepares frontend tests for a commit
- **THEN** it selects the repository-pinned runtime and package manager rather than a separately hardcoded runtime version
- **AND** it installs dependencies without changing the lockfile

#### Scenario: The pinned frontend runtime changes

- **WHEN** the repository's runtime pin changes
- **THEN** frontend lint, build and test jobs all select that same version without a second runtime-version edit

#### Scenario: Architecture verification fails

- **WHEN** a commit introduces a forbidden frontend dependency and architecture verification fails
- **THEN** the frontend lint check fails and the existing publication requirements prevent release of that commit
