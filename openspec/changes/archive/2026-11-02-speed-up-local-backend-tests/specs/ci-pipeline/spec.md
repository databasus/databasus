## MODIFIED Requirements

### Requirement: CI backend tests run with the race detector

Every CI backend test group SHALL run with the race detector. So SHALL every other CI check that runs backend tests: the verification agent's unit tests, its Podman subset, the upgrade test and the real-API browser test. This SHALL hold for any backend test command a CI step calls, including one that CI does not call today, although the same command runs without the race detector on a developer's machine. A data race SHALL fail the group or check like any other test failure, under the same rules for failed groups as before.

#### Scenario: A race in one shard

- **WHEN** a test in one shard of the physical backup group triggers a data race
- **THEN** that shard fails, and the other groups keep running and report their own results

#### Scenario: A check that shares a command with local runs

- **WHEN** CI runs the verification agent's unit tests through the command a developer also runs locally
- **THEN** the tests run with the race detector in CI
- **AND** the same command runs them without the race detector on a developer's machine

#### Scenario: A CI step calls a command CI does not use today

- **WHEN** a CI step calls `mise run test` or `mise run agent:e2e`
- **THEN** the backend tests run with the race detector

## REMOVED Requirements

### Requirement: Selected backend CI shards can run together locally

**Reason**: `mise run test` is the one routine local test command, and it schedules every backend package itself. CI's own group and shard selection does not change.

**Migration**: Use `mise run test`. To repeat one failing test, use the Go test command directly.
