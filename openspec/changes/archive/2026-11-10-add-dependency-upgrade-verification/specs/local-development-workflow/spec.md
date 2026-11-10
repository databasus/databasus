## MODIFIED Requirements

### Requirement: Everyday commands are separate from internal tasks

The repository SHALL expose exactly nine everyday mise tasks: `run`, `test`, `test-full`, `lint`, `format`, `doctor`, `review`, `generate-api` and `verify-deps`. In `mise.toml`, everyday commands SHALL appear first, internal CI/CD commands second, and tool dependencies and environment settings last. Internal tasks SHALL appear in the CI/CD section or in `mise-tasks/`, SHALL be hidden from the default listing, and SHALL remain callable by name from the repository root, backend and frontend. The CI/CD section SHALL explain that its tasks also serve hooks. The repository SHALL NOT retain aliases for `dev`, `full` or the removed root `pull-testcontainers` wrapper.

#### Scenario: Listing commands from a module

- **WHEN** a developer lists mise tasks from the repository root, backend or frontend
- **THEN** the default list shows exactly the nine everyday tasks
- **AND** CI and hooks can still invoke the hidden tasks by name
