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

#### Scenario: Storage-provider checks run alone

- **WHEN** CI runs the root storage-provider test package on either architecture
- **THEN** all its tests run once in `storage-providers`, without concurrent packages or scenarios on that runner
- **AND** the root package is excluded from every `other` shard while its nested packages remain covered

#### Scenario: Storage-provider verification fails

- **WHEN** the isolated provider check fails while all backend shards pass
- **THEN** its failed result prevents release version selection and publication

### Requirement: A failed group neither cancels nor hides the others

When one backend test group fails, the other groups SHALL run to completion and report their own results. The developer SHALL be able to rerun only the failed group.

#### Scenario: One group fails

- **WHEN** a test in the `other` group fails while the `physical` and `logical` groups still run
- **THEN** the `physical` and `logical` groups keep running and report their own results

#### Scenario: Rerun of the failed group

- **WHEN** the developer reruns the failed jobs of that workflow run
- **THEN** only the failed group runs again, and the groups that passed keep their results

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
### Requirement: Release requires the security scans

CI SHALL select a version, publish an image or create a release for a commit on `main` only after secret scanning, rule-based security scanning and static analysis (CodeQL) have passed for that commit. A scan that fails, is cancelled or times out SHALL stop the release. A running scan of a commit on `main` SHALL NOT be cancelled because a newer commit arrived or a scheduled scan started.

#### Scenario: Secret scan finds a secret on main

- **WHEN** every test and image check passes on a push to `main` but the secret scan reports a finding that is not allowlisted
- **THEN** version selection does not run, and CI publishes no image and creates no release for that commit

#### Scenario: Rule-based scan fails on main

- **WHEN** every other check passes on a push to `main` but the rule-based security scan fails
- **THEN** version selection does not run, and CI publishes no image and creates no release for that commit

#### Scenario: Static analysis fails on main

- **WHEN** every other check passes on a push to `main` but the static analysis fails or times out
- **THEN** version selection does not run, and CI publishes no image and creates no release for that commit

#### Scenario: Static analysis is cancelled on main

- **WHEN** a maintainer cancels the static analysis of a commit on `main`
- **THEN** version selection does not run, and CI publishes no image and creates no release for that commit

#### Scenario: A newer commit arrives during the scans

- **WHEN** a commit is pushed to `main` while the scans of the previous commit run
- **THEN** the scans of the previous commit run to completion

#### Scenario: The scheduled scan starts during a release scan

- **WHEN** the weekly scheduled scan starts while the release scan of a commit on `main` runs
- **THEN** the release scan runs to completion and its result decides the release

#### Scenario: A queued commit is replaced by a newer one

- **WHEN** a run for a commit on `main` is still queued behind an active run and a newer commit arrives
- **THEN** the queued run does not release, and the run of the newer commit passes the same checks before its own release, which includes both commits

### Requirement: CodeQL alerts gate a release by severity

The release analysis is the static analysis that runs as part of the release gate for a commit on `main`. The severity gate SHALL stop the release of that commit when the release analysis reports an open critical CodeQL alert on `main`. Open high CodeQL alerts that the release analysis reports SHALL appear as a warning and SHALL NOT stop the release. A dismissed CodeQL alert SHALL NOT count. A CodeQL alert that the release analysis no longer reports SHALL NOT count, even while a scheduled analysis still shows it as open. If the severity gate cannot read the CodeQL alerts, or finds no release analysis of the released commit, the release SHALL stop. The severity gate SHALL be repeatable for the same commit, so a dismissal takes effect without a new commit.

#### Scenario: An open critical CodeQL alert exists

- **WHEN** the static analysis of a commit on `main` completes and reports an open critical CodeQL alert
- **THEN** version selection does not run and the severity gate output lists the alert

#### Scenario: Only high CodeQL alerts are open

- **WHEN** the static analysis completes and high CodeQL alerts are open but no critical one is
- **THEN** the severity gate passes and reports the high alerts as a warning

#### Scenario: The owner dismisses the only critical CodeQL alert

- **WHEN** the owner dismisses the only open critical CodeQL alert and reruns the failed severity gate for the same commit
- **THEN** the severity gate passes and the release continues

#### Scenario: A commit fixes a critical CodeQL alert

- **WHEN** a commit on `main` fixes a critical CodeQL alert that the last scheduled analysis still shows as open
- **THEN** the severity gate passes without waiting for the next scheduled analysis

#### Scenario: The release analysis of the commit is not found

- **WHEN** the severity gate finds no release analysis of the released commit
- **THEN** the severity gate fails and the release stops, whatever the CodeQL alerts show

#### Scenario: CodeQL alerts cannot be read

- **WHEN** the severity gate cannot read the CodeQL alerts or the service returns an error
- **THEN** the severity gate fails and the release stops

### Requirement: The secret scan that gates a release reads the released history

The secret scan that gates a release SHALL read only commits reachable from the released commit. The scheduled secret scan SHALL read every branch and SHALL NOT affect a release.

#### Scenario: A finding exists only on another branch

- **WHEN** a branch not merged into `main` holds a secret scan finding and a commit on `main` is otherwise clean
- **THEN** the secret scan that gates the release of that commit passes

#### Scenario: The scheduled scan reads every branch

- **WHEN** the scheduled secret scan runs and a branch not merged into `main` holds a finding
- **THEN** the scheduled scan fails and reports the finding

### Requirement: Version selection waits for the pre-release image checks

CI SHALL select a version only after the application image checks and the verification image build have passed. A failure in either SHALL leave every image without a tag for that version. This requirement SHALL NOT cover a failure in the image rebuild after version selection, which can still leave some images tagged.

#### Scenario: The application image check fails on main

- **WHEN** every test and scan passes on a push to `main` and the verification image builds, but the application image check fails
- **THEN** version selection does not run and no image gets a tag for a new version

#### Scenario: The verification image build fails on main

- **WHEN** every test, scan and application image check passes on a push to `main` but the verification image build fails
- **THEN** version selection does not run and no image gets a tag for a new version

### Requirement: Build inputs are pinned

Every base image of the published application image SHALL be referenced by an immutable digest of its multi-architecture manifest. The image vulnerability scanner and the version selection tool SHALL be installed at exact versions, and the scanner download SHALL be verified against its published checksum. CI and a developer's machine SHALL take the scanner version from one pin in the repository.

#### Scenario: A base image tag moves

- **WHEN** a publisher moves a tag of a base image to new content
- **THEN** a build of the same commit uses the same base image content as before

#### Scenario: The scanner publishes a new release

- **WHEN** a new scanner version appears, including a compromised one
- **THEN** CI keeps using the pinned version until a commit changes it

#### Scenario: The scanner download does not match its checksum

- **WHEN** the downloaded scanner does not match its published checksum
- **THEN** the job that installs it fails and no scan runs with that download

#### Scenario: CI and a developer use the same scanner

- **WHEN** CI and a developer's machine install the repository's tools
- **THEN** both resolve the same scanner version from the same pin

#### Scenario: The version selection tool publishes a new release

- **WHEN** a new version of the version selection tool appears
- **THEN** version selection keeps using the exact version the workflow names

### Requirement: Workflow write permissions are granted per job

A workflow SHALL default to read-only repository contents at its top level. A write permission SHALL be granted only to the job that needs it.

#### Scenario: The dependency review comments on a pull request

- **WHEN** the dependency review finds a problem in a pull request from a branch of this repository
- **THEN** its job can post the comment, and the top-level permissions of the workflow stay read-only

### Requirement: Workflow files are linted

The repository's lint command SHALL lint the workflow files. A CI check SHALL do the same on each run of the CI and Release workflow except a pull request from `develop`, so pushes to `develop` are covered. A failure of the check on `main` SHALL stop version selection.

#### Scenario: A workflow names a job that does not exist

- **WHEN** a workflow file has a `needs` reference to a job that does not exist
- **THEN** the repository's lint command and the CI check fail

#### Scenario: A commit is pushed to develop

- **WHEN** a commit is pushed to `develop`
- **THEN** the CI check lints the workflow files of that commit

#### Scenario: The check fails on main

- **WHEN** every other check passes on a push to `main` but the workflow lint fails
- **THEN** version selection does not run, and CI publishes no image and creates no release for that commit

### Requirement: CI checks Python with the local quality command

Release and dev-image workflows SHALL run a dedicated Python quality check using the same entry point, configuration and repository-pinned tools as local Python lint. The check SHALL cover all repository-owned Python, including scripts, agent hooks, tests and browser scenarios. It SHALL install its Python dependencies from the committed dependency lockfile without updating it, and SHALL require no application image, running application, Docker service or browser installation. A formatting violation, lint violation, missing annotation, type error or setup failure SHALL fail the check. The Python check SHALL run independently of application builds and test results; its failure SHALL NOT cancel unrelated running checks. Existing workflow events and branch selection SHALL remain unchanged.

#### Scenario: A hook fails type checking in release CI

- **WHEN** an agent hook has a type error in a release workflow run
- **THEN** the Python quality check fails even when application lint and tests pass

#### Scenario: The dev workflow checks a browser fixture

- **WHEN** a browser test fixture is missing an annotation in a dev-image workflow run
- **THEN** the Python quality check fails without building an image or starting the fixture

#### Scenario: A quality tool pin changes

- **WHEN** a repository pin changes for the Python linter or type checker
- **THEN** both workflows use the updated version through the local quality command

#### Scenario: Python lint fails while application tests run

- **WHEN** the Python quality check fails while other test jobs are running
- **THEN** those jobs continue and report their own results

### Requirement: Python quality gates release and dev-image publication

Release version selection and all subsequent publication SHALL require a successful Python quality check for the same commit, in addition to their existing requirements. Every dev-image publishing job SHALL require that commit's Python quality check to succeed before publishing any architecture image or manifest. A failed, cancelled, skipped or timed-out Python check SHALL prevent these dependent operations. These requirements SHALL be verified locally through automated workflow-dependency checks without needing a push or a hosted pipeline run.

#### Scenario: Python quality fails on the release branch

- **WHEN** all other release prerequisites pass but the Python quality check fails
- **THEN** release version selection does not run and no release artifact is published

#### Scenario: Python quality fails on the development branch

- **WHEN** the Python quality check fails in the dev-image workflow
- **THEN** no architecture image or combined manifest for that commit is published

#### Scenario: Python quality is cancelled or skipped

- **WHEN** the Python quality check is cancelled, skipped or times out
- **THEN** neither release publication nor dev-image publication proceeds in that workflow run

#### Scenario: Python quality succeeds

- **WHEN** Python quality and every existing prerequisite succeed
- **THEN** the workflow remains eligible to select a release version or publish the dev image according to its existing branch and release rules

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

### Requirement: Bounded dependency checks gate release and development-image publication

CI SHALL run the same bounded local dependency check available to maintainers against the candidate commit. Release version selection, release publication and both supported architectures' development-image publication routes SHALL require its success for that commit. Failed, cancelled, skipped, missing or timed-out required results SHALL block those operations. The check SHALL inspect the recorded inputs for both supported architectures without downloading or executing their artifacts. Existing native frozen-installation, integrity, test and security checks SHALL remain required on their existing publication routes. CI SHALL NOT require an independent publication-age proof service or trusted observation history.

#### Scenario: A known dependency pin is invalid

- **WHEN** a candidate has an unpinned third-party reference, conflicting known versions or a changed recorded client file
- **THEN** the bounded dependency check fails and publication remains blocked

#### Scenario: The required check does not succeed

- **WHEN** other checks pass but the required dependency result is missing, skipped, cancelled or timed out
- **THEN** release version selection and publication remain blocked

#### Scenario: One architecture's required record is missing

- **WHEN** a known artifact supports both architectures but its records cover only one
- **THEN** the bounded check fails even if it runs on the covered architecture

#### Scenario: Offline pin checks pass but a native installation fails

- **WHEN** local pins are valid but a native frozen installation finds a stale graph or corrupted package
- **THEN** the native job fails and its existing publication gate remains closed
