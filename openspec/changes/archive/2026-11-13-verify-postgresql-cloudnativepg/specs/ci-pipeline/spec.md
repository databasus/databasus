## MODIFIED Requirements

### Requirement: The backend suite runs as whole and sharded groups that cover every test once

For each required processor architecture, CI SHALL run the backend suite as separate checks on separate runners without introducing dependencies between the groups. The `physical` group SHALL run physical backup end-to-end tests except the Patroni and CloudNativePG suites, and `logical` SHALL run logical backup end-to-end tests. The `patroni` group SHALL select only the Patroni physical backup suite and the packages nested under it, including newly added ones without a CI configuration change. The `cnpg` group SHALL select only the CloudNativePG physical backup suite and the packages nested under it, including newly added ones without a CI configuration change. The `physical-api` group SHALL select the physical-backup API test subtree, including newly added packages without a CI configuration change. The `agent-e2e` group SHALL select the verification agent's Docker end-to-end test subtree, including newly added packages without a CI configuration change. The `storage-providers` group SHALL select only the root storage-provider test package. The `other` group SHALL select from the whole backend module and run all remaining packages except that root package, the upgrade suite and its comparison helper. Every package other than the two upgrade packages SHALL belong to exactly one logical group on each architecture, with only the explicitly unavailable version scenarios excluded on arm64. New packages outside the named groups SHALL join `other` automatically, including nested storage-provider packages. A new package of physical backup end-to-end tests SHALL join `physical` automatically.

Every top-level test of a sharded group's packages SHALL be assigned to exactly one shard per architecture. New tests SHALL join a shard without a CI configuration change. The same sources and duration weights SHALL produce the same shard assignment regardless of runner locale. Missing or stale recorded durations SHALL affect assignment only, never inclusion. Every check SHALL have the tools its tests invoke at repository-pinned versions. `physical`, `physical-api` and `other` SHALL each have three shards, `patroni` SHALL have five and `cnpg` SHALL have five; `logical`, `agent-e2e` and `storage-providers` SHALL run whole on both architectures. The Patroni backup scenarios SHALL be divided into several top-level tests, each with a recorded duration weight, so that no single top-level test holds the whole scenario matrix and every `patroni` shard receives a part of it. The CloudNativePG backup scenarios SHALL be divided the same way, so that every `cnpg` shard receives a part of them. The `cnpg` runner size and scenario parallelism SHALL be chosen from measured duration and peak memory, using runners up to 16 vCPUs and 32 GB RAM without a fixed two-test limit. The `cnpg` check SHALL allow privileged containers and SHALL raise the inotify instance and watch limits before its tests, because its disposable Kubernetes clusters need both. It SHALL fail rather than skip scenarios when either is unavailable; a failure to raise the inotify limits SHALL fail the check before any test runs. The `storage-providers` check SHALL run its complete package alone on its runner with one package and one scenario at a time, retaining race detection and failure propagation.

The upgrade suite and its comparison helper SHALL run only in `test-upgrade`, separately for each architecture against that architecture's candidate. Each upgrade check SHALL start after its candidate is available without depending on backend-group results.

#### Scenario: The groups cover the suite

- **WHEN** logical group selections are combined separately for amd64 and arm64
- **THEN** each contains every backend package except the upgrade suite and its comparison helper exactly once

#### Scenario: The shards cover their group

- **WHEN** the shard selections of a group are combined for either architecture
- **THEN** they contain every top-level test of the group's packages exactly once, identified by both package and test

#### Scenario: A new package is added

- **WHEN** a developer adds a backend package outside the named groups
- **THEN** CI assigns its tests to `other` shards on both architectures without changing the workflow

#### Scenario: A new package of physical backup end-to-end tests is added

- **WHEN** a developer adds a package of physical backup end-to-end tests outside the Patroni and CloudNativePG suites
- **THEN** CI selects it only in `physical` on each architecture

#### Scenario: The Patroni suite runs in its own group

- **WHEN** CI selects backend packages for its groups
- **THEN** the Patroni suite and any package nested under it are selected only in `patroni` on each architecture
- **AND** no `physical` shard runs a Patroni scenario

#### Scenario: The Patroni matrix spreads across its shards

- **WHEN** the five `patroni` shard selections are computed from the recorded duration weights
- **THEN** every shard contains at least one top-level test with Patroni backup scenarios
- **AND** each top-level test with Patroni scenarios has a recorded duration weight

#### Scenario: The CloudNativePG suite runs in its own group

- **WHEN** CI selects backend packages for its groups
- **THEN** the CloudNativePG suite and any package nested under it are selected only in `cnpg` on each architecture
- **AND** no `physical` shard runs a CloudNativePG scenario

#### Scenario: The CloudNativePG matrix spreads across its shards

- **WHEN** the five `cnpg` shard selections are computed from the recorded duration weights
- **THEN** every shard contains at least one top-level test with CloudNativePG backup scenarios
- **AND** each top-level test with CloudNativePG scenarios has a recorded duration weight

#### Scenario: The CloudNativePG check runs on a prepared runner

- **WHEN** a `cnpg` shard runs on either architecture after its job raised the inotify limits
- **THEN** it starts its Kubernetes cluster and runs every selected scenario, with none skipped

#### Scenario: The CloudNativePG check cannot raise the inotify limits

- **WHEN** the `cnpg` job's step that raises the inotify instance and watch limits fails on either architecture
- **THEN** the shard fails at that step, before any test runs
- **AND** no CloudNativePG scenario is reported as skipped or passed

#### Scenario: The CloudNativePG check lacks its prerequisites

- **WHEN** a `cnpg` shard cannot start its disposable Kubernetes cluster on either architecture
- **THEN** that shard fails with the missing prerequisite named and reports no skipped scenario as passed

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

#### Scenario: Runners use different locales

- **WHEN** runners select shards from identical sources and duration weights under different locales
- **THEN** they assign each test to the same shard
- **AND** coverage remains complete and disjoint

#### Scenario: Checks are reported per group

- **WHEN** CI reports backend checks
- **THEN** each architecture has separate `storage-providers`, `logical`, `agent-e2e`, three `other` shard, three `physical` shard, five `patroni` shard, five `cnpg` shard and three `physical-api` shard results

#### Scenario: The upgrade test runs on its own

- **WHEN** CI runs the backend checks
- **THEN** the upgrade suite and its comparison helper run only in the dedicated `test-upgrade` checks for amd64 and arm64

#### Scenario: The upgrade test uses the commit's image without waiting for the groups

- **WHEN** the candidates are available while backend groups are still running
- **THEN** the upgrade checks can start against the matching candidates without waiting for those groups

#### Scenario: A migration test fails in CI

- **WHEN** a migration test fails on either architecture
- **THEN** that architecture's `other` shard containing the test fails

#### Scenario: An architecture linter test fails

- **WHEN** a test of the backend architecture linter fails on either architecture
- **THEN** that architecture's `other` shard containing the test fails

#### Scenario: A package falls out of the groups

- **WHEN** a non-upgrade backend package belongs to no logical group or to two logical groups on either architecture
- **THEN** repository script tests fail

#### Scenario: A test runs a pinned tool

- **WHEN** a test invokes the Go linter on either architecture
- **THEN** the repository-pinned linter is available natively

#### Scenario: Storage-provider checks run alone

- **WHEN** CI runs the root storage-provider test package on either architecture
- **THEN** all its tests run once in `storage-providers`, without concurrent packages or scenarios on that runner
- **AND** the root package is excluded from every `other` shard while its nested packages remain covered

#### Scenario: Storage-provider verification fails

- **WHEN** the isolated provider check fails while all backend shards pass
- **THEN** its failed result prevents release version selection and publication
