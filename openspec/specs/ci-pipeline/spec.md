# CI Pipeline Specification

## Purpose

Defines when the backend checks run in CI, how the backend test suite is split across runners, and which checks must pass before an image is published or a release is made.

## Requirements

### Requirement: Backend tests start without waiting for the backend lint

When CI runs the backend lint and the backend tests for a commit, the backend tests SHALL start without waiting for the lint result. A failed lint SHALL NOT cancel or skip the backend tests.

#### Scenario: Lint and tests start together

- **WHEN** CI starts the backend checks for a pull request
- **THEN** the backend lint check and every backend test check start without one waiting for the other

#### Scenario: Lint fails

- **WHEN** the backend lint check fails while the backend tests run
- **THEN** the backend test checks run to completion and report their own results

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

### Requirement: A failed group neither cancels nor hides the others

When one backend test group fails, the other groups SHALL run to completion and report their own results. The developer SHALL be able to rerun only the failed group.

#### Scenario: One group fails

- **WHEN** a test in the `other` group fails while the `physical` and `logical` groups still run
- **THEN** the `physical` and `logical` groups keep running and report their own results

#### Scenario: Rerun of the failed group

- **WHEN** the developer reruns the failed jobs of that workflow run
- **THEN** only the failed group runs again, and the groups that passed keep their results

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

### Requirement: The backend lint job runs the architecture check

The CI backend lint job SHALL run the backend's architecture check next to the linters, so that the backend-wide structure rules and the production-binary rules fail a pull request, not only a local run.

#### Scenario: A production package imports an assertion library

- **WHEN** a pull request adds an assertion library import to a production package reachable from the server's entry point
- **THEN** the backend lint job fails and names the package

#### Scenario: Old backup code writes to a provider

- **WHEN** a pull request makes backup production code reach a storage provider directly across the storage module boundary
- **THEN** the backend lint job fails under the module-boundary rule

#### Scenario: A backend command violates an architecture rule

- **WHEN** a pull request makes a backend command import a dependency forbidden for that command
- **THEN** the backend lint job fails and identifies the command and the rule

### Requirement: CI backend tests run with the race detector

Every CI backend test group SHALL run with the race detector. A data race SHALL fail the group like any other test failure, under the same rules for failed groups as before.

#### Scenario: A race in one shard

- **WHEN** a test in one shard of the physical backup group triggers a data race
- **THEN** that shard fails, and the other groups keep running and report their own results

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

### Requirement: Selected backend CI shards can run together locally

`mise run backend:test-group <shard>...` SHALL run the named CI selections concurrently in separate test processes, with race checking, `-failfast` and a 40-minute timeout per binary, without a helper build tag. It SHALL resolve selections through the existing group package and test selectors. It SHALL validate every shard before starting any test and reject duplicate names, unsharded groups, `other`, unknown groups and indices outside their count. It SHALL wait for every selected shard and return failure when any shard fails. On interruption it SHALL stop only its own test processes and remove its temporary logs.

#### Scenario: Every physical API selection runs

- **WHEN** all three `physical-api` shard names are passed
- **THEN** they run concurrently, and together they execute every top-level physical API test exactly once

#### Scenario: One selected shard fails

- **WHEN** one shard fails while another selected shard still runs
- **THEN** the task waits for the remaining shard, reports the selected shard names with their output and exits unsuccessfully

#### Scenario: A selection is invalid

- **WHEN** any requested shard is invalid or repeated
- **THEN** the task rejects the selection before any test starts

#### Scenario: Ordinary builds select the tests

- **WHEN** a developer runs a valid backend shard selection without a helper build tag in the process environment
- **THEN** its package and test selectors find the selection and its test processes start without adding such a tag

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

### Requirement: Browser UI verification gates frontend release

CI SHALL run the browser UI suite as its own check named `test-frontend-browser-ui`, using the same entry point, pinned toolchain and visual environment as local verification. Dependency installation SHALL preserve the committed lockfile. Release version selection and subsequent release publication SHALL require that check to pass alongside their existing requirements. The check SHALL not require the shared development application or a real-backend test service.

#### Scenario: Browser UI verification fails on main

- **WHEN** ordinary frontend tests pass but browser UI verification fails for a commit on the release branch
- **THEN** release version selection and release publication do not proceed for that commit

#### Scenario: CI starts the browser UI check

- **WHEN** CI prepares browser UI verification
- **THEN** it runs the documented local entry point with the committed dependency pins and the same screenshot environment
- **AND** it requires no manually started application
