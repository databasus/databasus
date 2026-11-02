## MODIFIED Requirements

### Requirement: The backend suite runs as whole and sharded groups that cover every test once

CI SHALL run the backend test suite as separate checks on separate runners, and the checks SHALL run at the same time. The `physical` group SHALL run the end-to-end physical backup tests, and the `logical` group SHALL run the end-to-end logical backup tests. The `physical-api` group SHALL run the test packages under `backend/internal/modules/backups/physicalapitests/`, the API tests of physical backups, which are slow because many of them start source servers and wait on real time. A package added under that directory SHALL join the group without any change to the CI configuration. The `agent-e2e` group SHALL run the verification agent's end-to-end scenarios on Docker, the packages under `backend/internal/agent/e2e/`, and a package added under that directory SHALL join the group without any change to the CI configuration. The `browser-e2e` group SHALL contain the test packages owning real-API frontend browser scenarios and SHALL run as the separate `test-frontend-browser-e2e` check. The `other` group SHALL select from the whole backend module and SHALL run every remaining backend package except the upgrade test and its comparison helper package, so that a directory of the module stays outside the groups only when it is named as running in another check. Each backend package other than the upgrade test and its comparison helper SHALL belong to exactly one group, including browser ownership packages in `browser-e2e` rather than `other`. The upgrade test, with its comparison helper, SHALL run as its own check, `test-upgrade`, against the image CI builds for the commit, without waiting for the backend groups. A backend package added outside these named groups SHALL run in the `other` group without any change to the CI configuration.

A group MAY run as several shards. Each shard SHALL run the tests of its group's packages that fall into that shard, and every top-level test of those packages SHALL run in exactly one shard. A test added to a sharded group SHALL run in one of its shards without any change to the CI configuration. How tests are assigned to shards MAY use recorded test durations, and a missing or outdated duration SHALL only change which shard runs a test, never whether it runs. Each backend test check SHALL have every tool its tests run at the version the repository pins. `physical` and `physical-api` SHALL run as three shards each, and `logical`, `agent-e2e`, `browser-e2e` and `other` SHALL each run whole.

#### Scenario: The groups cover the suite

- **WHEN** the package lists of the groups, including the browser group, are combined
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

- **WHEN** CI selects packages for real-API frontend browser verification
- **THEN** those packages run once through the dedicated browser check and are excluded from ordinary backend selections

#### Scenario: A new test is added to a sharded group

- **WHEN** a developer adds a top-level test to a package of a sharded group and records no duration for it
- **THEN** CI runs that test in exactly one shard of the group

#### Scenario: Checks are reported per group

- **WHEN** CI runs the backend tests
- **THEN** each whole group and each shard reports its own check, named `test-backend (other)`, `test-backend (logical)`, `test-backend (agent-e2e)`, `test-backend (physical-1-of-3)` through `test-backend (physical-3-of-3)` and `test-backend (physical-api-1-of-3)` through `test-backend (physical-api-3-of-3)`, with `browser-e2e` reported separately as `test-frontend-browser-e2e`

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
