## MODIFIED Requirements

### Requirement: Interrupting the local app stops what it started

When a developer interrupts `mise run dev` with Ctrl+C, the backend and the frontend development server SHALL exit, and the development database, the WAL backup source and its load generator SHALL stop. These containers SHALL be stopped rather than removed, so the database's data survives into the next run. The shared test fixtures SHALL keep running, because development sessions in other worktrees use them.

#### Scenario: Developer presses Ctrl+C

- **WHEN** a developer presses Ctrl+C in the `mise run dev` window
- **THEN** no backend or frontend process from that run remains
- **AND** the development database, the WAL backup source and its load generator are stopped

#### Scenario: Data survives a restart

- **WHEN** a developer interrupts `mise run dev` and runs it again
- **THEN** the development database still holds the data written during the first run

#### Scenario: Test fixtures are running

- **WHEN** a developer interrupts `mise run dev`
- **THEN** the shared test fixtures it started are still running afterwards

### Requirement: One command runs the full backend suite

The repository SHALL provide `mise run test`. It SHALL assume a Fedora host and, when `libedit` is installed on that host, SHALL make the bundled database clients startable without any further manual step. When `libedit` is missing, it SHALL stop before running tests and name the package to install. It SHALL start the shared test PostgreSQL server and no other development service, pull the database engine and object storage images the backend tests reference, and then run every test package of the backend module except the real-API browser packages and the upgrade test with its comparison helper, each test that asks for a migrated internal database getting one as that capability describes. A migration test that asks for an empty internal database SHALL get one with no migration applied and SHALL apply the migrations it needs itself, and packages that need no database SHALL open none. A package in a new directory of the module SHALL join this run without any change to the selection. After the backend packages and the frontend tests pass, it SHALL run the upgrade test as the requirement "The full run includes the upgrade test" describes. It SHALL NOT prepare internal databases ahead of the tests. It SHALL NOT run the Docker filesystem suite. Its exit status SHALL be non-zero when any step or any test fails.

#### Scenario: Developer runs the suite on Fedora

- **WHEN** a developer runs `mise run test` on a Fedora host with a `.env` file
- **THEN** every selected backend test package runs, and each test that asks for a migrated internal database gets one with every migration of the checkout applied
- **AND** a migration test that asks for an empty internal database gets one with no migration applied, and packages that need no database open none
- **AND** tests that start a bundled database client do not fail on a missing shared library

#### Scenario: Only the test PostgreSQL server starts

- **WHEN** a developer runs `mise run test`
- **THEN** the shared test PostgreSQL server is started
- **AND** the command starts no other service of the development compose file

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

After the backend test packages and the frontend tests pass, `mise run test` SHALL run the upgrade test. When the developer names a Databasus image through `DATABASUS_IMAGE`, the upgrade test SHALL use it. Otherwise `mise run test` SHALL build the image from the checkout first. A failed upgrade test SHALL make `mise run test` exit with a non-zero status. When a backend package fails, `mise run test` SHALL stop without building the image or running the upgrade test. A single test package started directly with the Go test command SHALL NOT build an image or run the upgrade test. When the developer interrupts `mise run test` during the upgrade test, the containers, the network and the volume of the upgrade test SHALL be removed, and the shared test PostgreSQL server SHALL keep running.

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
- **AND** the shared test PostgreSQL server is still running

### Requirement: Interrupting the backend suite leaves no test processes or containers

When a developer interrupts `mise run test` with Ctrl+C, every test process of that run SHALL exit, and every test container and test network that run started SHALL be removed. Test containers and networks started by another run or by a single package run, in this worktree or another one, SHALL NOT be removed. The shared test PostgreSQL server SHALL keep running, so that the next run and other worktrees can use it.

#### Scenario: Developer presses Ctrl+C mid-suite

- **WHEN** a developer presses Ctrl+C while `mise run test` runs tests
- **THEN** no test process from that run remains
- **AND** no test container that run started remains

#### Scenario: Another worktree's tests are running

- **WHEN** a developer interrupts `mise run test` while a test package of another worktree runs with its own test container
- **THEN** that container is still running afterwards and that package's tests continue

#### Scenario: Shared fixtures survive the interrupt

- **WHEN** a developer interrupts `mise run test`
- **THEN** the shared test PostgreSQL server is still running afterwards

### Requirement: The verification agent's end-to-end suite runs against the real backend

The repository SHALL provide `mise run agent:e2e`, which runs the verification agent's end-to-end scenarios with Docker as the agent's container engine, and `mise run agent:e2e-podman`, which runs the Podman subset with a rootful Podman engine as the agent's container engine. In every scenario the agent binary built from the checkout SHALL run as a separate process against the backend built from the same checkout. A scenario that restores SHALL restore a backup that this backend took of a real source database and served to the agent; no scenario SHALL restore a dump prepared outside the test. The source databases SHALL stay on Docker in both commands.

The Docker run SHALL cover these scenarios:

- a valid backup restores on every supported PostgreSQL major version, and the completed verification reports a restore exit code of 0, a restored size above zero and the restored tables;
- a TimescaleDB backup with a hypertable restores together with its chunks;
- a backup that includes the `public` schema restores;
- a broken backup ends as a failed verification after one attempt, with a non-empty failure message;
- a restore that exceeds its disk budget ends as a failed verification after one attempt, with a failure message about the disk budget;
- a retry of a verification completes over a network an earlier attempt of the same verification left behind;
- agent startup removes the containers and networks left with the agent's label and keeps those of another agent;
- the restore container runs as a non-root user with an empty capability bounding set;
- `start` detaches the agent, `status` reports it running, a second agent in the same directory is refused, and `stop` ends it;
- the agent heartbeats with its capacity, updates itself to the server's version in the background, and heartbeats again after the update.

The Podman subset SHALL be the valid backup on the newest supported PostgreSQL version, the broken backup and the privileges of the restore container.

A scenario SHALL remove every container, network, process and file it started, whether it passes or fails. It SHALL give each of them a name or label no other run uses, so runs from different worktrees on one machine do not disturb each other. Neither command SHALL need a separate build step, docker compose or prepared dump files. `mise run agent:test` SHALL NOT run the end-to-end scenarios, so it needs neither a container engine nor the test database.

#### Scenario: Developer runs the suite on Docker

- **WHEN** a developer runs `mise run agent:e2e` on a host with Docker and the shared test PostgreSQL server
- **THEN** every scenario of the Docker run passes
- **AND** no container or network carrying a label of the suite's agents remains afterwards

#### Scenario: Developer runs the Podman subset

- **WHEN** a developer runs `mise run agent:e2e-podman` with a rootful Podman socket their user can reach
- **THEN** the agent creates its restore containers on Podman, the source databases run on Docker, and the three scenarios of the subset pass

#### Scenario: Podman socket is unreachable

- **WHEN** a developer runs `mise run agent:e2e-podman` and the Podman socket does not exist or their user cannot open it
- **THEN** the command exits with a non-zero status before any test runs and names the socket path and how to make it reachable

#### Scenario: The restored backup comes from the backend

- **WHEN** a restore scenario runs
- **THEN** the backup the agent restores was taken by the backend under test from a source database the scenario filled, and the verification the backend stores reports the tables of that database

#### Scenario: A rejected restore is not retried

- **WHEN** the backup a scenario verifies is broken, or its restore exceeds the disk budget
- **THEN** the verification the backend stores is failed after its first attempt, and its failure message is the agent's report of that failure, not the message of a failure before the restore started

#### Scenario: Two worktrees run the suite at once

- **WHEN** two worktrees run `mise run agent:e2e` at the same time
- **THEN** both runs pass, and neither removes or reuses a container, network, agent directory or port of the other

#### Scenario: A scenario fails midway

- **WHEN** a scenario fails while its agent and restore container are running
- **THEN** the agent process, its containers and its networks are removed before the next scenario starts

#### Scenario: Agent unit tests run without a container engine

- **WHEN** a developer runs `mise run agent:test` on a host where Docker is stopped and the test database is not running
- **THEN** no end-to-end scenario runs and the command's result depends only on the agent's unit tests

#### Scenario: The full run includes the Docker scenarios

- **WHEN** a developer runs `mise run test`
- **THEN** the Docker scenarios of the agent's end-to-end suite run with the other backend packages, and the Podman subset does not run
