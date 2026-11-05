## MODIFIED Requirements

### Requirement: The backend suite runs as whole and sharded groups that cover every test once

CI SHALL run the backend test suite as separate checks on separate runners, and the checks SHALL run at the same time. The `physical` group SHALL run the end-to-end physical backup tests, and the `logical` group SHALL run the end-to-end logical backup tests. The `physical-api` group SHALL run the test packages under `backend/internal/modules/backups/physicalapitests/`, the API tests of physical backups, which are slow because many of them start source servers and wait on real time. A package added under that directory SHALL join the group without any change to the CI configuration. The `other` group SHALL run every remaining backend package except the upgrade test. Each backend package other than the upgrade test SHALL belong to exactly one group. The upgrade test SHALL run as its own check, `test-upgrade`, against the image CI builds for the commit, without waiting for the backend groups. A backend package added outside these named groups SHALL run in the `other` group without any change to the CI configuration.

A group MAY run as several shards. Each shard SHALL run the tests of its group's packages that fall into that shard, and every top-level test of those packages SHALL run in exactly one shard. A test added to a sharded group SHALL run in one of its shards without any change to the CI configuration. How tests are assigned to shards MAY use recorded test durations, and a missing or outdated duration SHALL only change which shard runs a test, never whether it runs. `physical` and `physical-api` SHALL run as three shards each, and `logical` and `other` SHALL each run whole.

#### Scenario: The groups cover the suite

- **WHEN** the package lists of the groups are combined
- **THEN** they contain every backend package except the upgrade test exactly once

#### Scenario: The shards cover their group

- **WHEN** the tests of all shards of one group are combined
- **THEN** they contain every top-level test of that group's packages exactly once

#### Scenario: A new package is added

- **WHEN** a developer adds a backend package outside the named groups
- **THEN** CI runs its tests in the `other` group

#### Scenario: A new package of physical API tests is added

- **WHEN** a developer adds a test package under `backend/internal/modules/backups/physicalapitests/`
- **THEN** CI runs its tests in the `physical-api` group and in no other group

#### Scenario: A new test is added to a sharded group

- **WHEN** a developer adds a top-level test to a package of a sharded group and records no duration for it
- **THEN** CI runs that test in exactly one shard of the group

#### Scenario: Checks are reported per group

- **WHEN** CI runs the backend tests
- **THEN** each whole group and each shard reports its own check, named `test-backend (other)`, `test-backend (logical)`, `test-backend (physical-1-of-3)` through `test-backend (physical-3-of-3)` and `test-backend (physical-api-1-of-3)` through `test-backend (physical-api-3-of-3)`

#### Scenario: The upgrade test runs on its own

- **WHEN** CI runs the backend tests for a commit
- **THEN** the upgrade test reports its own check named `test-upgrade` and runs in no backend group

#### Scenario: The upgrade test uses the commit's image without waiting for the groups

- **WHEN** CI runs for a commit
- **THEN** `test-upgrade` starts once the candidate image of that commit is built, uses that image as the current image, and does not wait for any `test-backend` check to finish


## ADDED Requirements

### Requirement: Selected backend CI shards can run together locally

`mise run backend:test-group <shard>...` SHALL run the named CI selections concurrently in separate test processes, with race checking, the `testhelpers` build tag, `-failfast` and a 15-minute timeout per binary. It SHALL resolve selections through the existing group package and test selectors. It SHALL validate every shard before starting any test and reject duplicate names, unsharded groups, `other`, unknown groups and indices outside their count. It SHALL wait for every selected shard and return failure when any shard fails. On interruption it SHALL stop only its own test processes and remove its temporary logs.

#### Scenario: Every physical API selection runs

- **WHEN** all three `physical-api` shard names are passed
- **THEN** they run concurrently, and together they execute every top-level physical API test exactly once

#### Scenario: One selected shard fails

- **WHEN** one shard fails while another selected shard still runs
- **THEN** the task waits for the remaining shard, reports the selected shard names with their output and exits unsuccessfully

#### Scenario: A selection is invalid

- **WHEN** any requested shard is invalid or repeated
- **THEN** the task rejects the selection before any test starts
