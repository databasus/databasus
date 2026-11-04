## RENAMED Requirements

- FROM: `### Requirement: Real-API browser verification gates frontend release`
- TO: `### Requirement: Browser end-to-end verification gates release`

## MODIFIED Requirements

### Requirement: The backend suite runs as whole and sharded groups that cover every test once

CI SHALL run the backend test suite as separate checks on separate runners, and the checks SHALL run at the same time. The `physical` group SHALL run the end-to-end physical backup tests, and the `logical` group SHALL run the end-to-end logical backup tests. The `physical-api` group SHALL run the test packages under `backend/internal/modules/backups/physicalapitests/`, the API tests of physical backups, which are slow because many of them start source servers and wait on real time. A package added under that directory SHALL join the group without any change to the CI configuration. The `agent-e2e` group SHALL run the verification agent's end-to-end scenarios on Docker, the packages under `backend/internal/agent/e2e/`, and a package added under that directory SHALL join the group without any change to the CI configuration. The `other` group SHALL select from the whole backend module and SHALL run every remaining backend package except the upgrade test and its comparison helper package, so that a directory of the module stays outside the groups only when it is named as running in another check. Each backend package other than the upgrade test and its comparison helper SHALL belong to exactly one group. The upgrade test, with its comparison helper, SHALL run as its own check, `test-upgrade`, against the image CI builds for the commit, without waiting for the backend groups. A backend package added outside these named groups SHALL run in the `other` group without any change to the CI configuration.

A group MAY run as several shards. Each shard SHALL run the tests of its group's packages that fall into that shard, and every top-level test of those packages SHALL run in exactly one shard. A test added to a sharded group SHALL run in one of its shards without any change to the CI configuration. How tests are assigned to shards MAY use recorded test durations, and a missing or outdated duration SHALL only change which shard runs a test, never whether it runs. Each backend test check SHALL have every tool its tests run at the version the repository pins. `physical` and `physical-api` SHALL run as three shards each, and `logical`, `agent-e2e` and `other` SHALL each run whole.

#### Scenario: The groups cover the suite

- **WHEN** the package lists of the groups are combined
- **THEN** they contain every backend package except the upgrade test and its comparison helper exactly once

#### Scenario: The shards cover their group

- **WHEN** the tests of all shards of one group are combined
- **THEN** they contain every top-level test of that group's packages exactly once

#### Scenario: A new package is added

- **WHEN** a developer adds a backend package outside the named groups, in a new top-level directory of the module or in an existing one
- **THEN** CI runs its tests in the `other` group

#### Scenario: A new package of physical API tests is added

- **WHEN** a developer adds a test package under `backend/internal/modules/backups/physicalapitests/`
- **THEN** CI runs its tests in the `physical-api` group and in no other group

#### Scenario: A new package of agent end-to-end scenarios is added

- **WHEN** a developer adds a test package under `backend/internal/agent/e2e/`
- **THEN** CI runs its tests in the `agent-e2e` group and in no other group

#### Scenario: A browser ownership package is selected

- **WHEN** CI selects backend packages for its groups
- **THEN** no backend package runs browser scenarios, and the browser end-to-end suite runs only in its own check

#### Scenario: A new test is added to a sharded group

- **WHEN** a developer adds a top-level test to a package of a sharded group and records no duration for it
- **THEN** CI runs that test in exactly one shard of the group

#### Scenario: Checks are reported per group

- **WHEN** CI runs the backend tests
- **THEN** each whole group and each shard reports its own check, named `test-backend (other)`, `test-backend (logical)`, `test-backend (agent-e2e)`, `test-backend (physical-1-of-3)` through `test-backend (physical-3-of-3)` and `test-backend (physical-api-1-of-3)` through `test-backend (physical-api-3-of-3)`

#### Scenario: The upgrade test runs on its own

- **WHEN** CI runs the backend tests for a commit
- **THEN** the upgrade test and its comparison helper report their results in the check named `test-upgrade` and run in no backend group

#### Scenario: The upgrade test uses the commit's image without waiting for the groups

- **WHEN** CI runs for a commit
- **THEN** `test-upgrade` starts once the candidate image of that commit is built, uses that image as the current image, and does not wait for any `test-backend` check to finish

#### Scenario: A migration test fails in CI

- **WHEN** a pull request breaks a test of the database migrations
- **THEN** the `test-backend (other)` check fails

#### Scenario: An architecture linter test fails

- **WHEN** a pull request breaks a test of the architecture linter itself
- **THEN** the `test-backend (other)` check fails

#### Scenario: A package falls out of the groups

- **WHEN** a developer changes the group selection so that a backend package other than the upgrade test and its comparison helper belongs to no group or to two groups
- **THEN** the repository's script tests fail

#### Scenario: A test runs a pinned tool

- **WHEN** an architecture linter test runs the Go linter in the `test-backend (other)` check
- **THEN** the linter is available at the version the repository pins

### Requirement: Browser end-to-end verification gates release

CI SHALL run the browser end-to-end suite as its own check named `test-e2e-browser` on every run except a run on `develop` or a pull request from `develop`, using the documented local entry point, repository-pinned tools and committed dependency lockfiles. It SHALL run against the candidate image CI builds for the commit, and it SHALL NOT build another image. It SHALL start once that image is built and SHALL run independently of backend lint and backend test-group results. It SHALL also run `mise run e2e:unit`, and the suite's lifecycle probes against the same image, and a failure there SHALL fail the check. Release version selection and publication SHALL require its success alongside existing checks. Browser failure SHALL not cancel unrelated test groups, and a failing run's printed output SHALL be its diagnostic, with secrets masked.

#### Scenario: Browser end-to-end verification fails on the release branch

- **WHEN** ordinary frontend tests and fixture-based browser checks pass but the browser end-to-end suite fails
- **THEN** release version selection and publication do not proceed for that commit

#### Scenario: Backend lint fails while browser verification runs

- **WHEN** backend lint fails for a commit
- **THEN** the browser end-to-end suite still runs to completion and reports its own status

#### Scenario: CI prepares the real-API browser check

- **WHEN** CI starts the `test-e2e-browser` check
- **THEN** it runs the same documented entry point as local verification on the commit's candidate image, with pinned tools and isolated test state

#### Scenario: The check loses its image

- **WHEN** the suite runs in CI without an image named for it
- **THEN** the check fails before any scenario and does not build an image of its own

#### Scenario: A lifecycle probe fails in CI

- **WHEN** a lifecycle probe fails against the candidate image
- **THEN** `test-e2e-browser` fails, and release version selection does not proceed

### Requirement: CI backend tests run with the race detector

Every CI backend test group SHALL run with the race detector. So SHALL every other CI check that runs backend tests: the verification agent's unit tests, its Podman subset and the upgrade test. The browser end-to-end suite runs no backend tests and tests an image built without the race detector. This SHALL hold for any backend test command a CI step calls, including one that CI does not call today, although the same command runs without the race detector on a developer's machine. A data race SHALL fail the group or check like any other test failure, under the same rules for failed groups as before.

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
