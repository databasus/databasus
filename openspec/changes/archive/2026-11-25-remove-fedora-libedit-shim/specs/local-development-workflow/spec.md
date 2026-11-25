# Spec Delta

## MODIFIED Requirements

### Requirement: One command starts the local app

The repository SHALL provide `mise run run`. It SHALL start the development database, the shared backend test fixtures, the live WAL backup source with its load generator, the backend, the frontend development server and the website development server, and SHALL show the database, the load generator, the backend, the frontend, the website and the seed in one terminal window that lists every process separately, with each process's output available on its own. It SHALL work in a fresh worktree that has no `.env` file, has never generated the backend's API documentation and has never installed frontend or website dependencies, whichever tool created the worktree. Before it starts any service, it SHALL install every pinned tool the checkout names that is missing on the host, at the version recorded in the repository's lock file, and SHALL stop with a non-zero status when that installation fails. When a backend from an earlier run still holds the backend port, the command SHALL stop that process before starting the new backend, so that requests reach the code just started. It SHALL also stop a backend file watcher left over from an earlier run in the same worktree, so that no old watcher rebuilds and restarts a backend after the new run starts. A saved change to a frontend source file SHALL reach the open page without restarting any process.

#### Scenario: Developer starts the app

- **WHEN** a developer runs `mise run run` in the Ubuntu development VM
- **THEN** the development database, the shared test fixtures, the WAL backup source, its load generator, the backend, the frontend and the website start
- **AND** the window lists the database, the load generator, the backend, the frontend, the website and the seed as separate processes with their own output

#### Scenario: Fresh worktree

- **WHEN** a developer runs `mise run run` in a worktree that has no `.env` file and nothing else prepared
- **THEN** the backend compiles and starts without a separate generation command
- **AND** the frontend development server and the website start without a separate dependency installation command
- **AND** the seed creates the dev account and its databases using the settings of the example file

#### Scenario: Worktree pins a tool version the host lacks

- **WHEN** a developer runs `mise run run` in a worktree whose commit pins a tool version that is not installed on the host
- **THEN** that version is installed from the lock file before any service starts
- **AND** the processes of the run use that version

#### Scenario: Tool installation fails

- **WHEN** a missing pinned tool cannot be installed during `mise run run`
- **THEN** no service starts and the command exits with a non-zero status

#### Scenario: Stale backend holds the port

- **WHEN** a backend from an earlier run still listens on the backend port
- **THEN** `mise run run` stops that process and the new backend serves requests on that port

#### Scenario: Watcher from a killed run survives

- **WHEN** an earlier `mise run run` in the same worktree was killed without cleanup and its backend file watcher is still running
- **THEN** `mise run run` stops that watcher before starting its own
- **AND** a later saved Go file restarts only the backend of the new run

#### Scenario: Frontend file is saved

- **WHEN** a developer saves a change to a frontend source file while `mise run run` is running
- **THEN** the open page shows the change without any process being restarted

### Requirement: One command runs the routine backend suite

The repository SHALL provide `mise run test`. It SHALL run in the Ubuntu development VM and SHALL start the bundled database clients with the shared libraries that the VM provides under their own names, without creating aliases for missing ones. It SHALL start the shared test PostgreSQL server and no other development service, pull the database engine and object storage images the backend tests reference, and then run every test package of the backend module except the upgrade test with its comparison helper and the end-to-end packages under `internal/agent/e2e` and `internal/modules/backups/e2e`, each test that asks for a migrated internal database getting one as that capability describes. A migration test that asks for an empty internal database SHALL get one with no migration applied and SHALL apply the migrations it needs itself, and packages that need no database SHALL open none. A package in a new directory of the module SHALL join this run without any change to the selection. It SHALL NOT prepare internal databases ahead of the tests. It SHALL NOT run the Docker filesystem suite. Its exit status SHALL be non-zero when any step or any test fails.

#### Scenario: Developer runs the suite in the VM

- **WHEN** a developer runs `mise run test` in the Ubuntu development VM with a `.env` file
- **THEN** every selected backend test package runs, and each test that asks for a migrated internal database gets one with every migration of the checkout applied
- **AND** a migration test that asks for an empty internal database gets one with no migration applied, and packages that need no database open none
- **AND** tests that start a bundled database client do not fail on a missing shared library

#### Scenario: Developer runs the suite on Fedora

- **WHEN** a developer runs `mise run test` directly on a Fedora host, where `libedit` exists only as `libedit.so.0`
- **THEN** `mise run test` creates no library alias directory and passes the caller's `LD_LIBRARY_PATH` to the tests unchanged
- **AND** tests that start a bundled database client can fail on the missing `libedit.so.2`, as a consequence of the host's libraries rather than of the command

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

#### Scenario: End-to-end packages stay out of the routine run

- **WHEN** `mise run test` selects its backend packages
- **THEN** it selects no package under `internal/agent/e2e`, `internal/modules/backups/e2e` or `e2e/upgrade`
- **AND** no browser end-to-end scenario runs

### Requirement: Tool versions are pinned in the repository

The repository SHALL pin the exact versions of a defined tool set, so that two hosts set up from the same commit run the same versions of those tools. The pinned set SHALL cover the backend and frontend toolchains, the commit hook runner, the process window, the migration tool, the Go linter, the API contract linter, the backend file watcher, the browser automation server agents use, the package manager of the browser end-to-end suite, and the Python runtime, linter, formatter and type checker. The version manager that installs these tools also runs the repository's tasks, and the repository SHALL name the minimum version of it that the tasks need. An older version manager SHALL refuse to run a task and name the minimum version. Docker, the shell, the Chrome browser and system libraries such as `libedit` come from the host and are not pinned.

#### Scenario: New host setup

- **WHEN** a developer installs the pinned tools in a new Ubuntu development VM and runs `mise run lint`
- **THEN** every tool from the pinned set that the task calls resolves to its pinned version

#### Scenario: The version manager is too old

- **WHEN** a developer whose version manager is older than the minimum the repository names runs `mise run lint`
- **THEN** no check runs, and the output names the minimum version

#### Scenario: Browser server starts at its pinned version

- **WHEN** an agent client starts the browser automation server registered in the repository
- **THEN** the server runs at the version pinned in the repository, without downloading a newer one

#### Scenario: The suite's package manager

- **WHEN** a developer runs `mise run e2e:browser` after installing the pinned tools
- **THEN** the suite's dependencies install through the pinned package manager from the suite's lockfile

#### Scenario: A Python quality tool is updated

- **WHEN** a developer changes the repository pin of a Python quality tool
- **THEN** local lint, the commit hook and CI use the new version without independent version changes
