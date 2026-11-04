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

CI SHALL run the browser UI suite as its own check named `test-frontend-browser-ui`, using the same entry point, pinned toolchain and visual environment as local verification. Dependency installation SHALL preserve the committed lockfile. Release version selection and subsequent release publication SHALL require that check to pass alongside their existing requirements. The check SHALL not require the shared development application or a real-backend test service.

#### Scenario: Browser UI verification fails on main

- **WHEN** ordinary frontend tests pass but browser UI verification fails for a commit on the release branch
- **THEN** release version selection and release publication do not proceed for that commit

#### Scenario: CI starts the browser UI check

- **WHEN** CI prepares browser UI verification
- **THEN** it runs the documented local entry point with the committed dependency pins and the same screenshot environment
- **AND** it requires no manually started application

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

### Requirement: The application image is checked on every published architecture

Each run of the CI and Release workflow, except a run on `develop` or a pull request from `develop`, SHALL build the application image on amd64 once the lint jobs and the Dockerfile scan have passed. Each such run on the `main` branch SHALL also build it on arm64, and no other run SHALL. The same smoke checks and the same image vulnerability scan SHALL run on each built architecture before any image is published. A failure on one architecture SHALL stop the release even if the other passes. A critical vulnerability with an available fix SHALL fail the image check. A high vulnerability SHALL NOT fail it.

#### Scenario: The arm64 image fails its smoke checks

- **WHEN** the amd64 image passes its image check but the arm64 image fails its smoke checks
- **THEN** on `main` CI selects no version and publishes no image

#### Scenario: A pull request is checked on amd64 only

- **WHEN** CI runs for a pull request from a branch other than `develop`
- **THEN** the amd64 image check runs and no arm64 image is built

#### Scenario: A manual run on main checks arm64

- **WHEN** a maintainer starts the workflow by hand on `main`
- **THEN** the arm64 image check runs before version selection

#### Scenario: A bundled client tool is broken on arm64

- **WHEN** the arm64 image starts but reports that a bundled client tool cannot run
- **THEN** the arm64 image check fails

#### Scenario: The arm64 image has a fixable critical vulnerability

- **WHEN** the image vulnerability scan of the arm64 image finds a critical vulnerability with an available fix
- **THEN** the arm64 image check fails and CI publishes no image

#### Scenario: The image has only high vulnerabilities

- **WHEN** the image vulnerability scan finds high vulnerabilities and no critical one with an available fix
- **THEN** the image check passes

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
