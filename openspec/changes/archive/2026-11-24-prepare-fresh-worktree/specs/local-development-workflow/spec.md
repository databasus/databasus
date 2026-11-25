## MODIFIED Requirements

### Requirement: One command starts the local app

The repository SHALL provide `mise run run`. It SHALL start the development database, the shared backend test fixtures, the live WAL backup source with its load generator, the backend, the frontend development server and the website development server, and SHALL show the database, the load generator, the backend, the frontend, the website and the seed in one terminal window that lists every process separately, with each process's output available on its own. It SHALL work in a fresh worktree that has no `.env` file, has never generated the backend's API documentation and has never installed frontend or website dependencies, whichever tool created the worktree. Before it starts any service, it SHALL install every pinned tool the checkout names that is missing on the host, at the version recorded in the repository's lock file, and SHALL stop with a non-zero status when that installation fails. When a backend from an earlier run still holds the backend port, the command SHALL stop that process before starting the new backend, so that requests reach the code just started. It SHALL also stop a backend file watcher left over from an earlier run in the same worktree, so that no old watcher rebuilds and restarts a backend after the new run starts. A saved change to a frontend source file SHALL reach the open page without restarting any process.

#### Scenario: Developer starts the app

- **WHEN** a developer runs `mise run run` on a Fedora host
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

## ADDED Requirements

### Requirement: Local settings fall back to the example file

Every repository task, and every shell with the version manager activated inside the repository, SHALL receive the settings of the committed example file. Each key set in the local `.env` file SHALL override the same key from the example. A missing `.env` SHALL NOT be an error. A change to either file SHALL reach the next task without a manual export.

#### Scenario: No local settings file

- **WHEN** a developer runs `mise run run` or `mise run test` in a worktree without `.env`
- **THEN** the development database, the backend and the tests use the connection settings of the example file

#### Scenario: Local file overrides one key

- **WHEN** `.env` sets only one key to a value that differs from the example
- **THEN** a task sees that value for that key and the example values for every other key

#### Scenario: A full copy of the example still works

- **WHEN** `.env` is a full copy of the example file with some values changed
- **THEN** a task sees the changed values and runs as before

#### Scenario: The example gains a new key

- **WHEN** a commit adds a key to the example file and a developer's existing `.env` does not set it
- **THEN** the next task sees the new key with its example value

#### Scenario: Migrations from a shell

- **WHEN** a developer with the version manager activated runs the migration tool from a shell inside the repository without exporting anything
- **THEN** the migration tool connects to the development database with the loaded settings
