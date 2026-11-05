## MODIFIED Requirements

### Requirement: The backend suite runs as whole and sharded groups that cover every test once

CI SHALL run the backend test suite as separate checks on separate runners, and the checks SHALL run at the same time. The `physical` group SHALL run the end-to-end physical backup tests, and the `logical` group SHALL run the end-to-end logical backup tests. The `wal-usecases` group SHALL run the package of the physical backup and WAL streaming use cases. The `other` group SHALL run every remaining backend package except the upgrade test. Each backend package other than the upgrade test SHALL belong to exactly one group. The upgrade test SHALL run as its own check, `test-upgrade`, against the image CI builds for the commit, without waiting for the backend groups. A backend package added outside these named groups SHALL run in the `other` group without any change to the CI configuration.

A group MAY run as several shards. Each shard SHALL run the tests of its group's packages that fall into that shard, and every top-level test of those packages SHALL run in exactly one shard. A test added to a sharded group SHALL run in one of its shards without any change to the CI configuration. How tests are assigned to shards MAY use recorded test durations, and a missing or outdated duration SHALL only change which shard runs a test, never whether it runs. `physical` and `wal-usecases` SHALL run as three shards each, and `logical` and `other` SHALL each run whole.

#### Scenario: The groups cover the suite

- **WHEN** the package lists of the groups are combined
- **THEN** they contain every backend package except the upgrade test exactly once

#### Scenario: The shards cover their group

- **WHEN** the tests of all shards of one group are combined
- **THEN** they contain every top-level test of that group's packages exactly once

#### Scenario: A new package is added

- **WHEN** a developer adds a backend package outside the named groups
- **THEN** CI runs its tests in the `other` group

#### Scenario: A new test is added to a sharded group

- **WHEN** a developer adds a top-level test to a package of a sharded group and records no duration for it
- **THEN** CI runs that test in exactly one shard of the group

#### Scenario: Checks are reported per group

- **WHEN** CI runs the backend tests
- **THEN** each whole group and each shard reports its own check, named `test-backend (other)`, `test-backend (logical)`, `test-backend (physical-1-of-3)` through `test-backend (physical-3-of-3)` and `test-backend (wal-usecases-1-of-3)` through `test-backend (wal-usecases-3-of-3)`

#### Scenario: The upgrade test runs on its own

- **WHEN** CI runs the backend tests for a commit
- **THEN** the upgrade test reports its own check named `test-upgrade` and runs in no backend group

#### Scenario: The upgrade test uses the commit's image without waiting for the groups

- **WHEN** CI runs for a commit
- **THEN** `test-upgrade` starts once the candidate image of that commit is built, uses that image as the current image, and does not wait for any `test-backend` check to finish

### Requirement: Publishing and release require the lint and every test group

CI SHALL publish an image or create a release only when the backend lint, every backend test check, each whole group and each shard, and the upgrade test have passed for that commit. Running the lint and the tests in parallel SHALL NOT let a commit with a failed lint, a failed group, a failed shard or a failed upgrade test reach publishing or release.

#### Scenario: Lint fails on main

- **WHEN** the backend lint fails on a push to `main` while every backend test check and the upgrade test pass
- **THEN** CI publishes no image and creates no release for that commit

#### Scenario: One group fails on main

- **WHEN** the lint, every other backend test check and the upgrade test pass on a push to `main` but one shard fails
- **THEN** CI publishes no image and creates no release for that commit

#### Scenario: The upgrade test fails on main

- **WHEN** the lint and every backend test group pass on a push to `main` but the upgrade test fails
- **THEN** CI publishes no image and creates no release for that commit
