## ADDED Requirements

### Requirement: The verification agent's Podman subset runs as its own check

CI SHALL run the Podman subset of the verification agent's end-to-end suite, as the `local-development-workflow` capability defines it, as its own check named `e2e-verification-agent-podman`, with a rootful Podman engine as the agent's container engine and the source databases on Docker. The check SHALL run the same command a developer runs locally for the subset. It SHALL start without waiting for the backend lint or the backend test groups, so a failed lint does not skip it. CI SHALL publish an image or create a release only when this check has passed too. After the check, whether it passed or failed, no container or network carrying a label of the suite's agents SHALL remain on the Podman engine.

#### Scenario: The Podman subset fails

- **WHEN** the restore of a valid backup fails on Podman while every backend test group passes on a push to `main`
- **THEN** the `e2e-verification-agent-podman` check fails, and CI publishes no image and creates no release for that commit

#### Scenario: The Podman subset passes

- **WHEN** CI runs for a commit whose agent restores on Podman as the subset expects
- **THEN** the `e2e-verification-agent-podman` check passes with the three scenarios of the subset run

#### Scenario: The backend lint fails

- **WHEN** the backend lint fails for a commit
- **THEN** the `e2e-verification-agent-podman` check still runs to completion and reports its own result

#### Scenario: The Podman subset fails midway

- **WHEN** a scenario of the subset fails while its agent's restore container is running on Podman
- **THEN** after the check no container or network with the `databasus.verification.agent_id` label remains on the rootful Podman engine

## MODIFIED Requirements

### Requirement: The backend suite runs as whole and sharded groups that cover every test once

CI SHALL run the backend test suite as separate checks on separate runners, and the checks SHALL run at the same time. The `physical` group SHALL run the end-to-end physical backup tests, and the `logical` group SHALL run the end-to-end logical backup tests. The `physical-api` group SHALL run the test packages under `backend/internal/modules/backups/physicalapitests/`, the API tests of physical backups, which are slow because many of them start source servers and wait on real time. A package added under that directory SHALL join the group without any change to the CI configuration. The `agent-e2e` group SHALL run the verification agent's end-to-end scenarios on Docker, the packages under `backend/internal/agent/e2e/`, and a package added under that directory SHALL join the group without any change to the CI configuration. The `other` group SHALL run every remaining backend package except the upgrade test. Each backend package other than the upgrade test SHALL belong to exactly one group. The upgrade test SHALL run as its own check, `test-upgrade`, against the image CI builds for the commit, without waiting for the backend groups. A backend package added outside these named groups SHALL run in the `other` group without any change to the CI configuration.

A group MAY run as several shards. Each shard SHALL run the tests of its group's packages that fall into that shard, and every top-level test of those packages SHALL run in exactly one shard. A test added to a sharded group SHALL run in one of its shards without any change to the CI configuration. How tests are assigned to shards MAY use recorded test durations, and a missing or outdated duration SHALL only change which shard runs a test, never whether it runs. `physical` and `physical-api` SHALL run as three shards each, and `logical`, `agent-e2e` and `other` SHALL each run whole.

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

#### Scenario: A new package of agent end-to-end scenarios is added

- **WHEN** a developer adds a test package under `backend/internal/agent/e2e/`
- **THEN** CI runs its tests in the `agent-e2e` group and in no other group

#### Scenario: A new test is added to a sharded group

- **WHEN** a developer adds a top-level test to a package of a sharded group and records no duration for it
- **THEN** CI runs that test in exactly one shard of the group

#### Scenario: Checks are reported per group

- **WHEN** CI runs the backend tests
- **THEN** each whole group and each shard reports its own check, named `test-backend (other)`, `test-backend (logical)`, `test-backend (agent-e2e)`, `test-backend (physical-1-of-3)` through `test-backend (physical-3-of-3)` and `test-backend (physical-api-1-of-3)` through `test-backend (physical-api-3-of-3)`

#### Scenario: The upgrade test runs on its own

- **WHEN** CI runs the backend tests for a commit
- **THEN** the upgrade test reports its own check named `test-upgrade` and runs in no backend group

#### Scenario: The upgrade test uses the commit's image without waiting for the groups

- **WHEN** CI runs for a commit
- **THEN** `test-upgrade` starts once the candidate image of that commit is built, uses that image as the current image, and does not wait for any `test-backend` check to finish

### Requirement: CI lints each module with its local lint command

The CI lint job of the backend and of the frontend SHALL run the same lint command a developer runs locally for that module, and the shell script check SHALL run the same command as locally too. The backend lint job SHALL check the verification agent's code, its end-to-end scenarios included, because the backend's local lint does. These jobs SHALL take every tool the command calls at the version the repository pins. A check added to a module's local lint SHALL run in that module's CI lint job without any change to the CI configuration, as long as the check uses tools that job already installs. A check that needs a tool the job does not install yet adds that tool to the job's list of tools, and only that.

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
