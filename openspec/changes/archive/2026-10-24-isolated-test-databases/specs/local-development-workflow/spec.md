## MODIFIED Requirements

### Requirement: One command runs the full backend suite

The repository SHALL provide `just test` at its root. It SHALL assume a Fedora host and, when `libedit` is installed on that host, SHALL make the bundled database clients startable without any further manual step. When `libedit` is missing, it SHALL stop before running tests and name the package to install. It SHALL start the shared test fixtures, pull the database engine and object storage images the backend tests reference, reset the shared database servers as the `backend-test-isolation` capability allows, and then run every backend test package, each test getting its internal database as that capability describes. It SHALL NOT prepare internal databases ahead of the tests. It SHALL NOT run the Docker filesystem suite. Its exit status SHALL be non-zero when any step or any test fails.

#### Scenario: Developer runs the suite on Fedora

- **WHEN** a developer runs `just test` on a Fedora host with a `.env` file
- **THEN** every backend test package runs, each against internal databases with every migration of the checkout applied
- **AND** tests that start a bundled database client do not fail on a missing shared library

#### Scenario: A test fails

- **WHEN** any backend test fails during `just test`
- **THEN** the command exits with a non-zero status

#### Scenario: Filesystem suite is not part of the run

- **WHEN** a developer runs `just test`
- **THEN** no Docker filesystem case runs and no candidate image is built

### Requirement: Interrupting the backend suite leaves no test processes or containers

When a developer interrupts `just test` with Ctrl+C, every test process of that run SHALL exit, and every test container and test network that run started SHALL be removed. Test containers and networks started by another run or by a single package run, in this worktree or another one, SHALL NOT be removed. The shared test fixtures SHALL keep running, so that the next run and other worktrees can use them.

#### Scenario: Developer presses Ctrl+C mid-suite

- **WHEN** a developer presses Ctrl+C while `just test` runs tests
- **THEN** no test process from that run remains
- **AND** no test container that run started remains

#### Scenario: Another worktree's tests are running

- **WHEN** a developer interrupts `just test` while a test package of another worktree runs with its own test container
- **THEN** that container is still running afterwards and that package's tests continue

#### Scenario: Shared fixtures survive the interrupt

- **WHEN** a developer interrupts `just test`
- **THEN** the shared test fixtures are still running afterwards

## ADDED Requirements

### Requirement: Full backend runs take turns

Only one `just test` SHALL run tests on a machine at a time. A `just test` started while another one runs, in any worktree, SHALL say that it waits for the other run and SHALL start its tests after the other run ends. A single test package run SHALL NOT wait for a full run, apart from the wait for the reset of the shared test PostgreSQL servers. A full run SHALL NOT remove containers or networks of a package run, SHALL NOT reset or delete an internal database a package run uses, and SHALL reset the shared test PostgreSQL servers only after the package run's process has ended. Tests of old code that change shared objects on those servers can still affect each other across runs, as they can today; this requirement does not cover that.

#### Scenario: Two worktrees start a full run

- **WHEN** an agent in one worktree starts `just test` while a full run of another worktree is running
- **THEN** the second run prints that it waits for another full run
- **AND** its tests start after the first run ends

#### Scenario: A package run during a full run

- **WHEN** an agent runs one backend test package while a full run of another worktree is running its tests
- **THEN** the package run starts its tests without waiting for the full run to end
- **AND** its containers, networks and internal database stay in place until the package run ends
