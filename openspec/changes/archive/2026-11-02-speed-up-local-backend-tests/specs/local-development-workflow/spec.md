## MODIFIED Requirements

### Requirement: One command runs the full backend suite

The repository SHALL provide `mise run test`. It SHALL assume a Fedora host and, when `libedit` is installed on that host, SHALL make the bundled database clients startable without any further manual step. When `libedit` is missing, it SHALL stop before running tests and name the package to install. It SHALL start the shared test fixtures, pull the database engine and object storage images the backend tests reference, reset the shared database servers as the `backend-test-isolation` capability allows, and then run every test package of the backend module except the real-API browser packages and the upgrade test with its comparison helper, each test that asks for a migrated internal database getting one as that capability describes. A migration test that asks for an empty internal database SHALL get one with no migration applied and SHALL apply the migrations it needs itself, and packages that need no database SHALL open none. A package in a new directory of the module SHALL join this run without any change to the selection. After the backend packages and the frontend tests pass, it SHALL run the upgrade test as the requirement "The full run includes the upgrade test" describes. It SHALL NOT prepare internal databases ahead of the tests. It SHALL NOT run the Docker filesystem suite. Its exit status SHALL be non-zero when any step or any test fails.

#### Scenario: Developer runs the suite on Fedora

- **WHEN** a developer runs `mise run test` on a Fedora host with a `.env` file
- **THEN** every selected backend test package runs, and each test that asks for a migrated internal database gets one with every migration of the checkout applied
- **AND** a migration test that asks for an empty internal database gets one with no migration applied, and packages that need no database open none
- **AND** tests that start a bundled database client do not fail on a missing shared library

#### Scenario: A test fails

- **WHEN** any backend test fails during `mise run test`
- **THEN** the command exits with a non-zero status

#### Scenario: Filesystem suite is not part of the run

- **WHEN** a developer runs `mise run test`
- **THEN** no Docker filesystem case runs

#### Scenario: A migration test fails locally

- **WHEN** a test of the database migrations fails during `mise run test`
- **THEN** the command exits with a non-zero status and does not start the upgrade test

#### Scenario: A package in a new directory

- **WHEN** a developer adds a test package in a new top-level directory of the backend module
- **THEN** `mise run test` runs its tests without any change to the selection

#### Scenario: The upgrade and browser packages stay out of the package step

- **WHEN** `mise run test` selects its backend packages
- **THEN** it selects neither the real-API browser packages nor the upgrade test and its comparison helper

### Requirement: The full run includes the upgrade test

After the backend test packages and the frontend tests pass, `mise run test` SHALL run the upgrade test. When the developer names a Databasus image through `DATABASUS_IMAGE`, the upgrade test SHALL use it. Otherwise `mise run test` SHALL build the image from the checkout first. A failed upgrade test SHALL make `mise run test` exit with a non-zero status. When a backend package fails, `mise run test` SHALL stop without building the image or running the upgrade test. A single test package started directly with the Go test command SHALL NOT build an image or run the upgrade test. When the developer interrupts `mise run test` during the upgrade test, the containers, the network and the volume of the upgrade test SHALL be removed, and the shared test fixtures SHALL keep running.

#### Scenario: Developer runs the full suite without an image

- **WHEN** a developer runs `mise run test` without `DATABASUS_IMAGE`
- **THEN** the image is built from the checkout after the backend packages and the frontend tests pass, and the upgrade test runs against it

#### Scenario: Developer names an image

- **WHEN** a developer runs `mise run test` with `DATABASUS_IMAGE` set to an image they built
- **THEN** no image is built and the upgrade test runs against the named image

#### Scenario: The upgrade test fails

- **WHEN** the upgrade test fails during `mise run test`
- **THEN** the command exits with a non-zero status

#### Scenario: The frontend tests fail

- **WHEN** the backend packages pass and a frontend test fails during `mise run test`
- **THEN** no image is built, the upgrade test does not run, and the command exits with a non-zero status

#### Scenario: A backend package fails

- **WHEN** a backend package fails during `mise run test`
- **THEN** no image is built, the upgrade test does not run, and the command exits with a non-zero status

#### Scenario: A single package run

- **WHEN** an agent runs one backend package directly with the Go test command
- **THEN** no Databasus image is built and no upgrade test container starts

#### Scenario: Interrupt during the upgrade test

- **WHEN** a developer presses Ctrl+C while the upgrade test runs during `mise run test`
- **THEN** none of the upgrade test's containers, its network or its volume remains
- **AND** the shared test fixtures are still running

## ADDED Requirements

### Requirement: The full run is the one routine test command

`mise run test` SHALL be the one routine local test command. A developer or an agent SHALL use it to check a change that adds or changes tests or the code those tests cover, and the OpenSpec change verification of such a change SHALL name it. The repository SHALL NOT provide a command that takes backend packages or shard names as arguments and runs their tests. The selectors CI uses and the fixed suites `mise run agent:test`, `mise run agent:e2e`, `mise run agent:e2e-podman`, `mise run backend:test-upgrade` and `mise run test-filesystem` SHALL remain. A developer or an agent SHALL still be able to repeat one failing test directly with the Go test command or the frontend test runner while fixing it.

#### Scenario: Checking a finished change

- **WHEN** an agent finishes implementing a change that adds or changes tests or the code they cover
- **THEN** it runs `mise run test` and reports the command and its result

#### Scenario: A change without tests or covered code

- **WHEN** a change edits only documentation
- **THEN** it owes no `mise run test`

#### Scenario: Repeating one failing test

- **WHEN** a developer fixes a test that `mise run test` reported
- **THEN** the developer can repeat that test directly with the Go test command, without a full run

#### Scenario: A removed selection command

- **WHEN** a developer runs `mise run backend:test-package ./internal/lib/...`
- **THEN** mise reports that the task does not exist

### Requirement: The full run includes the frontend tests

`mise run test` SHALL install the frontend dependencies from the lockfile, as `mise run dev` does, and SHALL run the frontend unit and screen tests at the same time as the backend packages. It SHALL wait for every test process, and a failure in one SHALL NOT stop the others. It SHALL exit with a non-zero status when any fails, and its output SHALL name each failed process and test. The upgrade test SHALL start only after all have passed. The browser suites SHALL stay out of the run.

#### Scenario: A fresh worktree

- **WHEN** a developer runs `mise run test` in a worktree without `frontend/node_modules`
- **THEN** the dependencies are installed from the lockfile and the frontend tests run

#### Scenario: A frontend test fails

- **WHEN** a frontend test fails while backend packages still run
- **THEN** the backend packages finish, the upgrade test does not start, the output names the failed frontend test, and the command exits with a non-zero status

#### Scenario: A backend package fails

- **WHEN** a package fails in one backend test process while the other backend process and the frontend tests still run
- **THEN** they finish, the upgrade test does not start, the output names the failed process and test, and the command exits with a non-zero status

#### Scenario: Browser suites stay out

- **WHEN** a developer runs `mise run test`
- **THEN** no browser test runs

#### Scenario: Interrupting every process

- **WHEN** a developer presses Ctrl+C while both backend test processes and the frontend tests run
- **THEN** all of them exit, and no test process or container of this run remains

### Requirement: Slow backend packages do not hold back the others

`mise run test` SHALL run the backend packages named in a repository list of slow packages separately from all other selected backend packages, in concurrent test processes, so that fast packages start without waiting for slow ones. The list SHALL be ordered by measured time, longest first, and the slow process SHALL receive the packages in that order, with a fixed package limit lower than that of the other process. Both limits SHALL be chosen by measurement. The run SHALL take slow packages only from those it selects anyway, so an entry that no longer exists or that the run excludes SHALL neither fail the run nor add a package, and a script test SHALL fail and name such an entry. A selected package missing from the list SHALL run with the other packages. Every test of every selected package SHALL run exactly once, including when a package runs as several local shards.

#### Scenario: The order and the limits

- **WHEN** `mise run test` starts its backend packages
- **THEN** the slow process receives the list's packages in the list's order with the lower package limit
- **AND** the other process starts at the same time with the remaining packages

#### Scenario: A new package

- **WHEN** a developer adds a test package without adding it to the list
- **THEN** `mise run test` runs it with the other packages

#### Scenario: A stale list entry

- **WHEN** the list names a package that no longer exists, or the browser or upgrade packages the run excludes
- **THEN** `mise run test` ignores the entry and the script test fails and names it

#### Scenario: A package runs as local shards

- **WHEN** a package runs as several local shards and one of its tests has no measured weight
- **THEN** that test still runs, in exactly one shard, and the package is not also run whole
