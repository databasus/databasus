## MODIFIED Requirements

### Requirement: The backend suite runs as whole and sharded groups that cover every test once

For each required processor architecture, CI SHALL run the backend suite as separate checks on separate runners without introducing dependencies between the groups. The `physical` group SHALL run physical backup end-to-end tests, and `logical` SHALL run logical backup end-to-end tests. The `physical-api` group SHALL select the physical-backup API test subtree, including newly added packages without a CI configuration change. The `agent-e2e` group SHALL select the verification agent's Docker end-to-end test subtree, including newly added packages without a CI configuration change. The `storage-providers` group SHALL select only the root storage-provider test package. The `other` group SHALL select from the whole backend module and run all remaining packages except that root package, the upgrade suite and its comparison helper. Every package other than the two upgrade packages SHALL belong to exactly one logical group on each architecture, with only the explicitly unavailable version scenarios excluded on arm64. New packages outside the named groups SHALL join `other` automatically, including nested storage-provider packages.

Every top-level test of a sharded group's packages SHALL be assigned to exactly one shard per architecture. New tests SHALL join a shard without a CI configuration change. The same sources and duration weights SHALL produce the same shard assignment regardless of runner locale. Missing or stale recorded durations SHALL affect assignment only, never inclusion. Every check SHALL have the tools its tests invoke at repository-pinned versions. `physical`, `physical-api` and `other` SHALL each have three shards; `logical`, `agent-e2e` and `storage-providers` SHALL run whole on both architectures. The `storage-providers` check SHALL run its complete package alone on its runner with one package and one scenario at a time, retaining race detection and failure propagation.

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
- **THEN** each architecture has separate `storage-providers`, `logical`, `agent-e2e`, three `other` shard, three `physical` shard and three `physical-api` shard results

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

#### Scenario: Storage-provider timing checks run alone

- **WHEN** CI runs the root storage-provider test package on either architecture
- **THEN** all its tests run once in `storage-providers`, without concurrent packages or scenarios on that runner
- **AND** the root package is excluded from every `other` shard while its nested packages remain covered

#### Scenario: Storage-provider verification fails

- **WHEN** the isolated provider check fails while all backend shards pass
- **THEN** its failed result prevents release version selection and publication

### Requirement: Browser UI verification gates frontend release

CI SHALL run the browser UI suite as three `test-frontend-browser-ui` shards on each required architecture, using the same entry point, pinned toolchain and visual environment as local verification. Dependency installation SHALL preserve the committed lockfile. Each UI test SHALL run in exactly one shard per architecture, and newly added tests SHALL be included without changing the workflow. Checks and retained failure artifacts SHALL identify the architecture and shard. Each shard SHALL retain two workers, zero retries and exact screenshot comparison. Release version selection and subsequent release publication SHALL require all six shard results to pass alongside their existing requirements. A failed, cancelled, skipped, missing or timed-out required shard SHALL prevent these operations. The checks SHALL not require the shared development application or a real-backend test service.

The local command without shard selection SHALL continue to run the complete UI suite. Invalid or repeated shard selection SHALL fail before starting the browser environment. A shard selection combined with reference updates SHALL be rejected.

#### Scenario: Browser UI verification fails on main

- **WHEN** ordinary frontend tests pass but any browser UI shard fails for a commit on the release branch
- **THEN** release version selection and release publication do not proceed for that commit

#### Scenario: CI starts the browser UI check

- **WHEN** CI prepares browser UI verification
- **THEN** each shard uses the documented local entry point with its explicit selection, committed dependency pins and the same screenshot environment
- **AND** it requires no manually started application

#### Scenario: UI shards cover the complete suite

- **WHEN** the three shard selections are combined for either architecture
- **THEN** every test from the complete UI suite appears exactly once
- **AND** a newly added test requires no hand-maintained file-list change

#### Scenario: One UI shard is unavailable

- **WHEN** five UI shard results pass and the sixth is cancelled, skipped, missing or times out
- **THEN** release version selection and publication remain blocked

#### Scenario: A developer runs the whole UI suite

- **WHEN** the local UI command is invoked without shard selection
- **THEN** it runs the complete suite with the same assertions and reference images

#### Scenario: A developer supplies an invalid shard

- **WHEN** shard selection has an invalid index or count, or is supplied more than once
- **THEN** the command fails before starting the browser environment and identifies the invalid selection

#### Scenario: A developer tries to update references through a shard

- **WHEN** reference updates and shard selection are requested together
- **THEN** the command fails without modifying the stored reference images

### Requirement: Browser end-to-end verification gates release

CI SHALL run the main browser end-to-end suite in its own `test-e2e-browser` checks on both required architectures for automatic `main` runs and manual `develop` runs, with no develop exemption. These checks SHALL run the suite's unit checks and complete real-application scenario suite. Three separate `test-e2e-browser-lifecycle` groups, `failures`, `interruption` and `isolation`, SHALL cover the applicable lifecycle probes on each architecture. Every lifecycle probe SHALL belong to exactly one group under the same execution environment. New tests and test files SHALL join a group automatically; existing image-dependent exclusions SHALL remain the same as for the complete local lifecycle command.

All checks SHALL use documented local entry points, repository-pinned tools and committed lockfiles against the matching candidate image of the commit, without rebuilding the application image. Each main-scenario and lifecycle check SHALL start after candidates are available, independently of backend lint, backend-group results and the other browser checks. Release version selection and publication SHALL require both main-scenario checks and all six lifecycle results in addition to existing prerequisites. A failed, cancelled, skipped, missing or timed-out required result SHALL prevent these operations. A browser failure SHALL NOT cancel unrelated checks, and diagnostics SHALL mask secrets.

The complete local lifecycle command SHALL continue to run every applicable probe when no group is selected. Explicit group selection SHALL preserve failure reporting, signal handling, cleanup and isolation guarantees. Invalid groups or a selected group with no collected cases SHALL fail rather than report a successful empty run.

#### Scenario: Browser end-to-end verification fails on the release branch

- **WHEN** ordinary frontend and browser UI tests pass but real-backend browser verification fails on either architecture
- **THEN** version selection and publication do not proceed

#### Scenario: Backend lint fails while browser verification runs

- **WHEN** backend lint fails
- **THEN** the main browser checks and every lifecycle group continue and report their own results

#### Scenario: CI prepares the real-API browser check

- **WHEN** CI starts either architecture's browser end-to-end check
- **THEN** it uses the documented entry point, matching candidate, pinned tools and isolated state

#### Scenario: The check loses its image

- **WHEN** any main-scenario or lifecycle check has no matching candidate image
- **THEN** it fails before any scenario and does not build an application image of its own

#### Scenario: A lifecycle probe fails in CI

- **WHEN** a lifecycle probe fails on either architecture
- **THEN** that architecture and group's lifecycle check fails and release version selection cannot proceed

#### Scenario: Lifecycle groups cover the complete applicable suite

- **WHEN** the three lifecycle group selections are combined under the same candidate-image environment
- **THEN** their collected cases match the complete local lifecycle suite without omissions or duplicates
- **AND** a newly added test or test file joins exactly one group automatically

#### Scenario: Main scenarios finish before lifecycle verification

- **WHEN** both main-scenario checks pass while one lifecycle group is still running
- **THEN** publication remains blocked until every required lifecycle result passes

#### Scenario: A required lifecycle result is unavailable

- **WHEN** the main scenarios and other lifecycle groups pass but one required group is cancelled, skipped, missing or times out
- **THEN** version selection and publication remain blocked

#### Scenario: A developer runs the complete lifecycle suite

- **WHEN** the local lifecycle command is invoked without a group
- **THEN** it executes the complete applicable probe suite with its existing cleanup and isolation guarantees

#### Scenario: A lifecycle selection is invalid or empty

- **WHEN** a developer selects an unknown group or a group collects no cases
- **THEN** the command fails with a diagnostic instead of reporting successful verification
