## ADDED Requirements

### Requirement: Release CI tests both supported architectures natively

Every automatic release-workflow run on a push to `main` and every manual release-workflow run on `develop` SHALL execute the same required test suites on native amd64 and arm64 runners. These suites SHALL include every backend group and shard, verification agent unit tests and Docker and Podman end-to-end tests, frontend unit and screen tests, browser UI tests, real-backend browser end-to-end tests and their unit and lifecycle checks, repository script tests, all filesystem test groups and the upgrade suite. Architecture-independent lint and security scans need not be duplicated. The review-hook suite and standalone browser lifecycle-interruption suite remain outside this requirement.

Each test check SHALL identify its architecture and, where applicable, its group and shard. A failure on one architecture SHALL NOT cancel the other architecture or unrelated test checks. A supported suite SHALL NOT be skipped, allowed to fail or run through processor emulation to obtain an architecture's successful result. Manual runs SHALL NOT select a release version or publish release artifacts.

#### Scenario: An automatic main run starts

- **WHEN** a push to `main` starts release CI
- **THEN** every required test suite runs on native amd64 and arm64 runners
- **AND** its reported result identifies the architecture

#### Scenario: A maintainer verifies develop

- **WHEN** a maintainer starts release CI manually on `develop`
- **THEN** the same required test suites run on both architectures, including real-backend browser end-to-end tests
- **AND** the run selects no release version and publishes no release artifacts

#### Scenario: One architecture fails

- **WHEN** an arm64 test check fails while amd64 checks and other arm64 checks are running
- **THEN** those checks continue and report their own outcomes

#### Scenario: An architecture's check is missing

- **WHEN** the release workflow loses one required suite or shard for either architecture
- **THEN** the repository's automated workflow checks fail without needing a hosted run

### Requirement: Unavailable database images have explicit architecture exclusions

On arm64, server-container scenarios for MySQL 5.7 and MariaDB 5.5 SHALL be explicitly excluded because their official images have no native variant. They SHALL continue running on amd64. The exclusion SHALL apply consistently to image preparation and scenario execution, SHALL identify the engine version and architecture in the test output, and SHALL NOT omit other scenarios in the same package. Checks that establish unsupported-version behavior without those server containers SHALL continue running on ARM.

Image download, authentication, startup or manifest failures for any image outside those exclusions SHALL fail the check. The workflow SHALL NOT infer permission to skip a scenario from an image-preparation failure.

#### Scenario: ARM selects the logical backup matrix

- **WHEN** arm64 CI prepares and runs the database-version scenarios
- **THEN** it neither pulls nor starts the MySQL 5.7 or MariaDB 5.5 server image
- **AND** it reports those version-specific exclusions and runs the remaining supported versions

#### Scenario: The x64 matrix includes old versions

- **WHEN** amd64 CI prepares and runs the database-version scenarios
- **THEN** the MySQL 5.7 and MariaDB 5.5 scenarios remain included

#### Scenario: A supported image cannot be fetched

- **WHEN** preparation cannot obtain an image outside the explicit architecture exclusions
- **THEN** the check fails instead of skipping its scenarios

#### Scenario: ARM rejects an unsupported client version

- **WHEN** a container-free test checks rejection of a version with no client on ARM
- **THEN** that test runs and verifies the rejection

### Requirement: Image-consuming tests use the commit's matching native candidate

Each architecture's browser end-to-end, filesystem and upgrade suites SHALL use a candidate application image built for that architecture from the checked-out commit. Image transfers and retained failure evidence SHALL be distinct by architecture, so parallel checks cannot overwrite or consume another architecture's artifacts. Before a suite runs, a missing image or an image for the wrong processor architecture SHALL fail the check. Auxiliary database, browser, container-engine and filesystem fixtures SHALL run natively on the suite's architecture. Native network-filesystem coverage SHALL retain both CIFS and NFS root-squash scenarios.

#### Scenario: Both candidates are available

- **WHEN** amd64 and arm64 image-consuming checks start in the same workflow run
- **THEN** each consumes only the candidate built for its architecture and commit
- **AND** each retains failure evidence separately from the other architecture

#### Scenario: ARM receives an x64 candidate

- **WHEN** an arm64 suite is given an amd64 application candidate
- **THEN** the check fails before its scenarios instead of running the candidate under emulation

#### Scenario: ARM runs network-filesystem tests

- **WHEN** the arm64 network-filesystem check runs
- **THEN** its native fixtures exercise both CIFS and NFS root-squash behavior without dropping either scenario

### Requirement: Native backup checks prove recovery through restored data

On each supported architecture, logical backup end-to-end tests SHALL restore the backed-up data for every included engine-version scenario and retain their encryption, exclusions, limited-user, TLS and SSH assertions where supported. Physical PostgreSQL tests SHALL exercise full backups, incremental backups and archived WAL recovery on that architecture. The real-backend browser suite SHALL execute the recovery commands shown to the user with both Host tools and Docker and verify the recorded full, incremental and later WAL-stage rows in the recovered clusters. An image build, backup status or completed download alone SHALL NOT substitute for those assertions.

#### Scenario: ARM completes logical recovery

- **WHEN** a logical backup and restore scenario runs on arm64
- **THEN** it checks the restored data and the scenario's existing assertions against native database servers

#### Scenario: ARM completes physical recovery through the UI

- **WHEN** the browser end-to-end suite runs on arm64
- **THEN** it executes both displayed recovery methods and verifies full, incremental and WAL-stage data in the recovered clusters

### Requirement: Release application images are checked natively on both architectures

Every CI and Release workflow run SHALL build and check the application image natively on amd64 and arm64, including manual runs on `develop`. Image checks SHALL not wait for unrelated lint results. Each built architecture SHALL receive the same smoke checks and image vulnerability scan before publication. A failure on either SHALL prevent release version selection and publication. A critical image vulnerability with an available fix SHALL fail the check; high vulnerabilities alone SHALL NOT fail it. Manual runs SHALL remain verification-only.

#### Scenario: The ARM image fails its smoke checks

- **WHEN** amd64 image checks pass but arm64 smoke checks fail
- **THEN** CI selects no release version and publishes no image

#### Scenario: A manual develop run checks both images

- **WHEN** a maintainer starts the workflow manually on `develop`
- **THEN** both native image checks run without selecting a release version or publishing images

#### Scenario: A manual main run checks both images

- **WHEN** a maintainer starts the workflow manually on `main`
- **THEN** both native image checks run and the run remains verification-only

#### Scenario: A bundled client is broken on ARM

- **WHEN** the arm64 image starts but a bundled client cannot run
- **THEN** the arm64 image check fails

#### Scenario: ARM has a fixable critical image vulnerability

- **WHEN** the arm64 image scan finds a critical vulnerability with an available fix
- **THEN** the check fails and no release image is published

#### Scenario: The image has only high vulnerabilities

- **WHEN** an image scan finds high vulnerabilities without a fixable critical vulnerability
- **THEN** those findings do not fail the image check

## MODIFIED Requirements

### Requirement: The backend suite runs as whole and sharded groups that cover every test once

For each required processor architecture, CI SHALL run the backend suite as separate checks on separate runners without introducing dependencies between the groups. The `physical` group SHALL run physical backup end-to-end tests, and `logical` SHALL run logical backup end-to-end tests. The `physical-api` group SHALL select the physical-backup API test subtree, including newly added packages without a CI configuration change. The `agent-e2e` group SHALL select the verification agent's Docker end-to-end test subtree, including newly added packages without a CI configuration change. The `other` group SHALL select from the whole backend module and run all remaining packages except the upgrade suite and its comparison helper. Every package other than those two SHALL belong to exactly one group on each architecture, with only the explicitly unavailable version scenarios excluded on arm64. New packages outside the named groups SHALL join `other` automatically.

A group MAY run as several shards. Every top-level test of its packages SHALL be assigned to exactly one shard per architecture. New tests SHALL join a shard without a CI configuration change. Missing or stale recorded durations SHALL affect assignment only, never inclusion. Every check SHALL have the tools its tests invoke at repository-pinned versions. `physical` and `physical-api` SHALL each have three shards; `logical`, `agent-e2e` and `other` SHALL run whole on both architectures.

The upgrade suite and its comparison helper SHALL run only in `test-upgrade`, separately for each architecture against that architecture's candidate. Each upgrade check SHALL start after its candidate is available without depending on backend-group results.

#### Scenario: The groups cover the suite

- **WHEN** the group selections are combined separately for amd64 and arm64
- **THEN** each contains every backend package except the upgrade suite and its comparison helper exactly once

#### Scenario: The shards cover their group

- **WHEN** the shard selections of a group are combined for either architecture
- **THEN** they contain every top-level test of the group's packages exactly once

#### Scenario: A new package is added

- **WHEN** a developer adds a backend package outside the named groups
- **THEN** CI runs it in `other` on both architectures without changing the workflow

#### Scenario: A new package of physical API tests is added

- **WHEN** a developer adds a package to the physical-backup API test subtree
- **THEN** CI selects it only in `physical-api` on each architecture

#### Scenario: A new package of agent end-to-end scenarios is added

- **WHEN** a developer adds a package to the verification agent's Docker end-to-end subtree
- **THEN** CI selects it only in `agent-e2e` on each architecture

#### Scenario: A browser ownership package is selected

- **WHEN** CI selects backend packages for its groups
- **THEN** no backend package runs browser scenarios, which run only in their dedicated check

#### Scenario: A new test is added to a sharded group

- **WHEN** a developer adds a top-level test without a recorded duration
- **THEN** that test is assigned to exactly one shard per architecture

#### Scenario: Checks are reported per group

- **WHEN** CI reports backend checks
- **THEN** each architecture has separate `other`, `logical`, `agent-e2e`, three `physical` shard and three `physical-api` shard results

#### Scenario: The upgrade test runs on its own

- **WHEN** CI runs the backend checks
- **THEN** the upgrade suite and its comparison helper run only in the dedicated `test-upgrade` checks for amd64 and arm64

#### Scenario: The upgrade test uses the commit's image without waiting for the groups

- **WHEN** the candidates are available while backend groups are still running
- **THEN** the upgrade checks can start against the matching candidates without waiting for those groups

#### Scenario: A migration test fails in CI

- **WHEN** a migration test fails on either architecture
- **THEN** that architecture's `other` check fails

#### Scenario: An architecture linter test fails

- **WHEN** a test of the backend architecture linter fails on either architecture
- **THEN** that architecture's `other` check fails

#### Scenario: A package falls out of the groups

- **WHEN** a non-upgrade backend package belongs to no group or to two groups on either architecture
- **THEN** repository script tests fail

#### Scenario: A test runs a pinned tool

- **WHEN** a test invokes the Go linter on either architecture
- **THEN** the repository-pinned linter is available natively

### Requirement: Publishing and release require the lint and every test group

CI SHALL select a release version, publish an image or create a release only after the existing lint, security and image checks and every required test suite have passed for the same commit. Every required architecture, whole group and shard SHALL be included. Failed, cancelled, skipped, missing or timed-out required checks SHALL prevent these operations; an explicitly unavailable version scenario does not excuse a missing suite. Parallel execution SHALL NOT permit a result from one architecture to stand in for the other.

#### Scenario: Lint fails on main

- **WHEN** backend lint fails on a push to `main` while every required test passes
- **THEN** CI selects no release version, publishes no image and creates no release

#### Scenario: One group fails on main

- **WHEN** all other prerequisites pass but one arm64 physical shard fails
- **THEN** CI selects no release version, publishes no image and creates no release

#### Scenario: The upgrade test fails on main

- **WHEN** all other prerequisites pass but the upgrade check fails on either architecture
- **THEN** CI selects no release version, publishes no image and creates no release

#### Scenario: A required check is cancelled or skipped

- **WHEN** any required architecture's test check is cancelled, skipped, missing or times out
- **THEN** its absence of success prevents version selection and every publication operation

#### Scenario: Only x64 passes

- **WHEN** every amd64 check passes but the corresponding arm64 checks have not all passed
- **THEN** the commit is not eligible for release

### Requirement: Browser end-to-end verification gates release

CI SHALL run the browser end-to-end suite in its own `test-e2e-browser` checks on both required architectures for automatic `main` runs and manual `develop` runs, with no develop exemption. It SHALL use the documented local entry point, repository-pinned tools and committed lockfiles against the matching candidate image of the commit, without rebuilding the application image. It SHALL start after candidates are available and independently of backend lint and group results. Each architecture SHALL also run the suite's unit checks and lifecycle probes against its matching image; a failure SHALL fail that architecture's check. Release version selection and publication SHALL require both checks in addition to existing prerequisites. A browser failure SHALL NOT cancel unrelated checks, and diagnostics SHALL mask secrets.

#### Scenario: Browser end-to-end verification fails on the release branch

- **WHEN** ordinary frontend and browser UI tests pass but real-backend browser verification fails on either architecture
- **THEN** version selection and publication do not proceed

#### Scenario: Backend lint fails while browser verification runs

- **WHEN** backend lint fails
- **THEN** both browser end-to-end checks continue and report their own results

#### Scenario: CI prepares the real-API browser check

- **WHEN** CI starts either architecture's browser end-to-end check
- **THEN** it uses the documented entry point, matching candidate, pinned tools and isolated state

#### Scenario: The check loses its image

- **WHEN** a browser check has no matching candidate image
- **THEN** it fails before any scenario and does not build an application image of its own

#### Scenario: A lifecycle probe fails in CI

- **WHEN** a lifecycle probe fails on either architecture
- **THEN** that architecture's check fails and release version selection cannot proceed

## REMOVED Requirements

### Requirement: The application image is checked on every published architecture

**Reason:** Its branch-dependent architecture policy and amd64-only pull-request scenario conflict with native verification on every release-workflow run, including manual develop runs.

**Migration:** Use "Release application images are checked natively on both architectures", which retains smoke checks and vulnerability thresholds and requires both native architectures without a develop exemption. Workflow triggers remain unchanged.
