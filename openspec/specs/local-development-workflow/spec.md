# local-development-workflow Specification

## Purpose
Defines the commands a developer or an agent uses on a local Fedora host to run Databasus, run the backend suite and check a commit, and what each command stops or leaves running when it is interrupted.

## Requirements

### Requirement: Everyday commands are separate from internal tasks

The repository SHALL expose exactly eight everyday mise tasks: `run`, `test`, `test-full`, `lint`, `format`, `doctor`, `review` and `generate-api`. In `mise.toml`, everyday commands SHALL appear first, internal CI/CD commands second, and tool dependencies and environment settings last. Internal tasks SHALL appear in the CI/CD section or in `mise-tasks/`, SHALL be hidden from the default listing, and SHALL remain callable by name from the repository root, backend and frontend. The CI/CD section SHALL explain that its tasks also serve hooks. The repository SHALL NOT retain aliases for `dev`, `full` or the removed root `pull-testcontainers` wrapper.

#### Scenario: Listing commands from a module

- **WHEN** a developer lists mise tasks from the repository root, backend or frontend
- **THEN** the default list shows exactly the eight everyday tasks
- **AND** CI and hooks can still invoke the hidden tasks by name

### Requirement: The full command runs the Docker test suites

`mise run test-full` SHALL run the routine backend and frontend suites, adding all backup and verification agent end-to-end packages. After those pass, it SHALL run script tests, review hook tests, browser e2e unit tests, frontend browser UI, the upgrade test, browser end-to-end scenarios, lifecycle checks and lifecycle interruption checks, in that order. A failure SHALL return a non-zero status and prevent later stages from starting. It SHALL take the same machine-wide lock as `mise run test`, and SHALL NOT update browser visual references. CI SHALL run its individual suites rather than `mise run test-full`. Podman and filesystem tests SHALL remain separate because they require additional host configuration.

#### Scenario: All ordinary Docker checks pass

- **WHEN** every stage passes during `mise run test-full`
- **THEN** every included suite runs once and the command succeeds
- **AND** neither the Podman subset nor the filesystem suite runs

#### Scenario: An additional suite fails

- **WHEN** an additional suite fails during `mise run test-full`
- **THEN** the command fails and no later suite starts

### Requirement: Frontend lint is one complete check

`pnpm lint` in the frontend SHALL check formatting, source rules, architecture, styles, TypeScript and generated API types without changing tracked files. The frontend mise task SHALL call that command. The frontend SHALL expose only `dev`, `build`, `lint`, `format`, `typecheck`, `test`, `test:browser` and `generate:api` scripts. Watch, preview and visual-reference updates SHALL use the existing tools and arguments rather than separate scripts. The commit hook SHALL run each affected module's lint once, including its formatting check.

#### Scenario: A frontend check fails

- **WHEN** any frontend lint check fails
- **THEN** `pnpm lint`, `mise run frontend:lint` and `mise run lint` fail

### Requirement: One command starts the local app

The repository SHALL provide `mise run run`. It SHALL start the development database, the shared backend test fixtures, the live WAL backup source with its load generator, the backend and the frontend development server, and SHALL show the database, the load generator, the backend, the frontend and the seed in one terminal window that lists every process separately, with each process's output available on its own. It SHALL work in a fresh worktree that has a `.env` file but has never generated the backend's API documentation or installed frontend dependencies. When a backend from an earlier run still holds the backend port, the command SHALL stop that process before starting the new backend, so that requests reach the code just started. It SHALL also stop a backend file watcher left over from an earlier run in the same worktree, so that no old watcher rebuilds and restarts a backend after the new run starts. A saved change to a frontend source file SHALL reach the open page without restarting any process.

#### Scenario: Developer starts the app

- **WHEN** a developer runs `mise run run` on a Fedora host with a `.env` file
- **THEN** the development database, the shared test fixtures, the WAL backup source, its load generator, the backend and the frontend start
- **AND** the window lists the database, the load generator, the backend, the frontend and the seed as separate processes with their own output

#### Scenario: Fresh worktree

- **WHEN** a developer runs `mise run run` in a worktree that has a `.env` file and nothing else prepared
- **THEN** the backend compiles and starts without a separate generation command
- **AND** the frontend development server starts without a separate dependency installation command

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

### Requirement: Interrupting the local app stops what it started

When a developer interrupts `mise run run` with Ctrl+C, the backend and the frontend development server SHALL exit, and the development database, the WAL backup source and its load generator SHALL stop. These containers SHALL be stopped rather than removed, so the database's data survives into the next run. The shared test fixtures SHALL keep running, because development sessions in other worktrees use them.

#### Scenario: Developer presses Ctrl+C

- **WHEN** a developer presses Ctrl+C in the `mise run run` window
- **THEN** no backend or frontend process from that run remains
- **AND** the development database, the WAL backup source and its load generator are stopped

#### Scenario: Data survives a restart

- **WHEN** a developer interrupts `mise run run` and runs it again
- **THEN** the development database still holds the data written during the first run

#### Scenario: Test fixtures are running

- **WHEN** a developer interrupts `mise run run`
- **THEN** the shared test fixtures it started are still running afterwards

### Requirement: One command runs the routine backend suite

The repository SHALL provide `mise run test`. It SHALL assume a Fedora host and, when `libedit` is installed on that host, SHALL make the bundled database clients startable without any further manual step. When `libedit` is missing, it SHALL stop before running tests and name the package to install. It SHALL start the shared test PostgreSQL server and no other development service, pull the database engine and object storage images the backend tests reference, and then run every test package of the backend module except the upgrade test with its comparison helper and the end-to-end packages under `internal/agent/e2e` and `internal/modules/backups/e2e`, each test that asks for a migrated internal database getting one as that capability describes. A migration test that asks for an empty internal database SHALL get one with no migration applied and SHALL apply the migrations it needs itself, and packages that need no database SHALL open none. A package in a new directory of the module SHALL join this run without any change to the selection. It SHALL NOT prepare internal databases ahead of the tests. It SHALL NOT run the Docker filesystem suite. Its exit status SHALL be non-zero when any step or any test fails.

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

#### Scenario: End-to-end packages stay out of the routine run

- **WHEN** `mise run test` selects its backend packages
- **THEN** it selects no package under `internal/agent/e2e`, `internal/modules/backups/e2e` or `e2e/upgrade`
- **AND** no browser end-to-end scenario runs

### Requirement: The full run includes the upgrade test

After the backend test packages and the frontend tests pass, `mise run test-full` SHALL run the upgrade test. When the developer names a Databasus image through `DATABASUS_IMAGE`, the upgrade test SHALL use it. Otherwise `mise run test-full` SHALL build the image from the checkout first. A failed upgrade test SHALL make `mise run test-full` exit with a non-zero status. When a backend package fails, `mise run test-full` SHALL stop without building the image or running the upgrade test. A single test package started directly with the Go test command SHALL NOT build an image or run the upgrade test. When the developer interrupts `mise run test-full` during the upgrade test, the containers, the network and the volume of the upgrade test SHALL be removed, and the shared test PostgreSQL server SHALL keep running.

#### Scenario: Developer runs the full suite without an image

- **WHEN** a developer runs `mise run test-full` without `DATABASUS_IMAGE`
- **THEN** the image is built from the checkout after the backend packages and the frontend tests pass, and the upgrade test runs against it

#### Scenario: Developer names an image

- **WHEN** a developer runs `mise run test-full` with `DATABASUS_IMAGE` set to an image they built
- **THEN** the upgrade test builds no application image and runs against the named image

#### Scenario: The upgrade test fails

- **WHEN** the upgrade test fails during `mise run test-full`
- **THEN** the command exits with a non-zero status

#### Scenario: The frontend tests fail

- **WHEN** the backend packages pass and a frontend test fails during `mise run test-full`
- **THEN** no image is built, the upgrade test does not run, and the command exits with a non-zero status

#### Scenario: A backend package fails

- **WHEN** a backend package fails during `mise run test-full`
- **THEN** no image is built, the upgrade test does not run, and the command exits with a non-zero status

#### Scenario: A single package run

- **WHEN** an agent runs one backend package directly with the Go test command
- **THEN** no Databasus image is built and no upgrade test container starts

#### Scenario: Interrupt during the upgrade test

- **WHEN** a developer presses Ctrl+C while the upgrade test runs during `mise run test-full`
- **THEN** none of the upgrade test's containers, its network or its volume remains
- **AND** the shared test PostgreSQL server is still running

### Requirement: Interrupting the backend suite leaves no test processes or containers

When a developer interrupts `mise run test` or `mise run test-full` with Ctrl+C, every test process of that run SHALL exit, and every test container and test network that run started SHALL be removed. Test containers and networks started by another run or by a single package run, in this worktree or another one, SHALL NOT be removed. The shared test PostgreSQL server SHALL keep running, so that the next run and other worktrees can use it.

#### Scenario: Developer presses Ctrl+C mid-suite

- **WHEN** a developer presses Ctrl+C while `mise run test` or `mise run test-full` runs tests
- **THEN** no test process from that run remains
- **AND** no test container that run started remains

#### Scenario: Another worktree's tests are running

- **WHEN** a developer interrupts `mise run test` while a test package of another worktree runs with its own test container
- **THEN** that container is still running afterwards and that package's tests continue

#### Scenario: Shared fixtures survive the interrupt

- **WHEN** a developer interrupts `mise run test`
- **THEN** the shared test PostgreSQL server is still running afterwards

### Requirement: Test runs take turns

Only one `mise run test` or `mise run test-full` SHALL run tests on a machine at a time. Such a command started while another one runs, in any worktree, SHALL print `waiting for another test run to finish` and SHALL start its tests after the other run ends. A single test package run SHALL NOT wait for either command, apart from the wait for the reset of the shared test PostgreSQL servers. Neither command SHALL remove containers or networks of a package run, reset or delete an internal database a package run uses, or reset the shared test PostgreSQL servers before the package run's process has ended. Tests of old code that change shared objects on those servers can still affect each other across runs, as they can today; this requirement does not cover that.

#### Scenario: Two worktrees start a full run

- **WHEN** an agent in one worktree starts `mise run test-full` while `mise run test` or `mise run test-full` of another worktree is running
- **THEN** the second run prints `waiting for another test run to finish`
- **AND** its tests start after the first run ends

#### Scenario: Two worktrees start the routine run

- **WHEN** an agent in one worktree starts `mise run test` while `mise run test` or `mise run test-full` of another worktree is running
- **THEN** the second run prints `waiting for another test run to finish`
- **AND** its tests start after the first run ends

#### Scenario: A package run during a full run

- **WHEN** an agent runs one backend test package while `mise run test-full` of another worktree is running its tests
- **THEN** the package run starts its tests without waiting for that run to end
- **AND** its containers, networks and internal database stay in place until the package run ends

### Requirement: Test images can be pulled ahead of a run

The repository SHALL provide `mise run backend:pull-testcontainers`. It SHALL pull the database engine images (PostgreSQL, MySQL, MariaDB and MongoDB) and the object storage image that the backend tests reference, and the images the verification agent restores into during its end-to-end scenarios. It SHALL retry each failed pull a bounded number of times, and SHALL exit with a non-zero status when an image still cannot be pulled. Other test images, such as the FTP, SFTP, SMB, mail and Azure storage emulators, are pulled by the tests on first use.

#### Scenario: Pulling succeeds

- **WHEN** a developer runs `mise run backend:pull-testcontainers` with registry access
- **THEN** every database engine and object storage image the backend tests reference is present locally afterwards
- **AND** every image the verification agent restores into during its end-to-end scenarios on Docker is present locally afterwards

#### Scenario: An image cannot be pulled

- **WHEN** an image still fails to pull after its retries
- **THEN** the command names that image and exits with a non-zero status

### Requirement: Root commands lint and format every module

The repository SHALL provide `mise run lint` and `mise run format`, covering the backend, whose module includes the verification agent, and the frontend. `mise run lint` SHALL exit with a non-zero status when any module fails its checks, and SHALL report the failures of every failing module, not only the first. `mise run format` SHALL rewrite files in both modules to their formatter's output. Each module SHALL also have its own lint and format tasks, `mise run backend:lint`, `mise run frontend:lint` and the matching `format` tasks, and every task of a module SHALL run in that module's directory whichever directory of the repository it is started from. The backend's lint and format tasks SHALL check and format the verification agent's code together with the server's, with the same rules, its end-to-end scenarios included; there SHALL be no separate lint or format task for the agent.

#### Scenario: One module fails lint

- **WHEN** the frontend has a lint error and the backend is clean
- **THEN** `mise run lint` exits with a non-zero status and reports the frontend error

#### Scenario: Two modules fail lint

- **WHEN** both the backend and the frontend have a lint error
- **THEN** `mise run lint` exits with a non-zero status and reports the errors of both modules

#### Scenario: The verification agent fails lint

- **WHEN** the verification agent's code breaks a rule of the backend lint and the rest of the backend is clean
- **THEN** `mise run backend:lint` and `mise run lint` exit with a non-zero status and report the agent's error

#### Scenario: Developer formats everything

- **WHEN** a developer runs `mise run format`
- **THEN** a later `mise run lint` reports no formatting difference in either module, the verification agent included

#### Scenario: Module task started from another directory

- **WHEN** a developer runs `mise run backend:lint` from the frontend directory
- **THEN** the backend checks run against the backend module and pass or fail as they would from the backend directory

### Requirement: Tool versions are pinned in the repository

The repository SHALL pin the exact versions of a defined tool set, so that two hosts set up from the same commit run the same versions of those tools. The pinned set SHALL cover the backend and frontend toolchains, the commit hook runner, the process window, the migration tool, the Go linter, the API contract linter, the backend file watcher, the browser automation server agents use and the package manager of the browser end-to-end suite. The version manager that installs these tools also runs the repository's tasks, and the repository SHALL name the minimum version of it that the tasks need. An older version manager SHALL refuse to run a task and name the minimum version. Docker, the shell, the Chrome browser and system libraries such as `libedit` come from the host and are not pinned.

#### Scenario: New host setup

- **WHEN** a developer installs the pinned tools on a new Fedora host and runs `mise run lint`
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

### Requirement: A commit is checked before it is created

Creating a commit SHALL check that each staged file of each module the commit touches matches that module's formatter, SHALL run that module's lint checks, and for the frontend its type check as well, and SHALL refuse the commit when any of them fails. When a staged file is not formatted, the refusal SHALL name the file and the command that formats it. The check reads each staged file as it is in the working tree, because it leaves the working tree and the index as they are; content staged for the commit that differs from the working tree is not checked separately. A module the commit does not touch SHALL NOT be checked. The verification agent's code, its end-to-end scenarios included, belongs to the backend module, so a commit that touches it runs the backend checks.

#### Scenario: Unformatted file is staged

- **WHEN** a developer commits a staged frontend file that is not formatted
- **THEN** no commit is created, and the output names the file and the command that formats it

#### Scenario: Lint error in a staged module

- **WHEN** a developer commits a backend change that fails the backend lint checks
- **THEN** no commit is created and the lint output is shown

#### Scenario: Lint error in the verification agent

- **WHEN** a developer commits a change to the verification agent that fails the backend lint checks
- **THEN** no commit is created and the backend lint output is shown

#### Scenario: Untouched module

- **WHEN** a commit touches only the backend
- **THEN** no frontend check runs

#### Scenario: Staged content differs from the working tree

- **WHEN** a developer stages an unformatted Go file, then formats it in the working tree without staging the result, and commits
- **THEN** the formatting check passes, because it reads the formatted working-tree file

### Requirement: The backend restarts when its source changes

While `mise run run` is running, saving a change to a backend Go source file SHALL rebuild the backend and replace the running backend with the new build, without the developer touching the process window. A change to a test file alone SHALL NOT restart the backend. When the new code fails to compile, the backend that was already running SHALL keep serving requests, and the compile error SHALL appear in the backend's output in the process window. Interrupting `mise run run` SHALL leave no backend process from that run, including one started by a restart.

#### Scenario: Developer saves a backend change

- **WHEN** a developer changes the response of an endpoint and saves the Go file while `mise run run` is running
- **THEN** a request to that endpoint returns the new response once the rebuild finishes, without any manual restart

#### Scenario: The change does not compile

- **WHEN** a developer saves a Go file that does not compile
- **THEN** the backend output shows the compile error
- **AND** the previous backend still answers requests on the backend port

#### Scenario: Only a test file changes

- **WHEN** a developer saves a change to a Go test file and nothing else
- **THEN** the backend is not restarted

#### Scenario: Interrupt after restarts

- **WHEN** the backend has restarted several times and the developer presses Ctrl+C in the `mise run run` window
- **THEN** no backend process from that run remains

### Requirement: The local app is seeded with a dev account and databases

When the backend started by `mise run run` begins answering requests, the seed SHALL make sure the local app contains:

- the dev account `test@email.com` with password `Q1234567`;
- a workspace owned by that account;
- a local storage in that workspace;
- a PostgreSQL database with logical backups of the shared logical test fixture;
- a PostgreSQL database with WAL-streaming physical backups of the live WAL backup source.

The seed SHALL go through the same public API and validation as the web interface. It SHALL find each item by its name, SHALL create only the items that are missing and SHALL NOT change an item that already exists. When the seed creates a database, it SHALL switch its backups on with the local storage and SHALL start the first backup. The logical database SHALL keep its 3 newest backups, and the WAL-streaming database SHALL keep its 2 newest backup chains.

The seed command, `go run ./cmd/seed_dev` in the backend directory, SHALL also run on its own against the app already answering on the backend port, with the same behavior. The repository SHALL NOT provide a separate task for it. When nothing answers on the backend port, the seed command SHALL exit with a non-zero status and say that the app has to be started with `mise run run`. When the backend answers its health check with an error status, for example because disk usage is too high, the seed SHALL exit with a non-zero status and show the reason the backend gave, and SHALL create nothing.

When the seed cannot sign in as the dev account and cannot create it, for example because the account exists with a different password or registration is closed, it SHALL exit with a non-zero status and a message that names the account and the reason the backend gave, and SHALL create nothing. When the instance requires two-factor sign-in, the seed SHALL exit with a non-zero status and a message that names the account and says that two-factor sign-in is required, and SHALL create nothing. When the backend refuses to create a seeded item after sign-in, for example because the dev account is a member that may not create workspaces, the seed SHALL exit with a non-zero status, show the name of that item and the reason the backend gave, and SHALL NOT change any item that already exists. A seed that fails SHALL NOT stop the other processes of `mise run run`.

#### Scenario: First run on an empty app

- **WHEN** a developer runs `mise run run` against a development database that has no users
- **THEN** after the backend starts, signing in as `test@email.com` with `Q1234567` succeeds
- **AND** the account's workspace lists the local storage and both databases, each with backups switched on and a first backup started
- **AND** the logical database's backup settings keep 3 backups, and the WAL-streaming database's keep 2 chains

#### Scenario: Second run

- **WHEN** a developer runs `mise run run` again with the seeded items already present
- **THEN** no item is created a second time
- **AND** no existing item changes, including one the developer edited after the first run

#### Scenario: One seeded database was deleted

- **WHEN** a developer deleted the WAL-streaming database and runs the seed command on its own
- **THEN** the seed creates that database again with backups switched on
- **AND** the other items stay unchanged

#### Scenario: Dev account has another password

- **WHEN** `test@email.com` exists with a password other than `Q1234567` and the seed runs
- **THEN** the seed exits with a non-zero status and a message naming the account and the failed sign-in
- **AND** no workspace, storage or database was created
- **AND** the account's password and role are unchanged
- **AND** the backend and the frontend keep running

#### Scenario: Backend refuses to create the workspace

- **WHEN** the dev account is a member, members may not create workspaces, and the seed runs
- **THEN** the seed exits with a non-zero status and a message naming the workspace and the backend's reason
- **AND** no existing item changed
- **AND** the backend and the frontend keep running

#### Scenario: Two-factor sign-in is required

- **WHEN** the instance requires two-factor sign-in and the seed runs
- **THEN** the seed exits with a non-zero status and a message naming the account and saying that two-factor sign-in is required
- **AND** no workspace, storage or database was created

#### Scenario: Backend reports itself unhealthy

- **WHEN** the backend answers its health check with an error status and the reason "disk usage is above 95%", and the seed command runs on its own
- **THEN** the seed exits with a non-zero status and shows that reason
- **AND** nothing is created

#### Scenario: App is not running

- **WHEN** a developer runs the seed command on its own while no backend answers on the backend port
- **THEN** the command exits with a non-zero status and tells the developer to start the app with `mise run run`

### Requirement: Seed credentials exist only in local development

The dev account, its password and the seeded databases SHALL be created only by `mise run run` and by the seed command run on its own in a development checkout. The production image and the production backend SHALL neither contain the seed nor run it, whatever the environment variables, and SHALL create no account with known credentials. The seed SHALL NOT change the password, the role or the settings of any existing account, including the dev account.

#### Scenario: Production image starts

- **WHEN** the production image starts with an empty data directory
- **THEN** no account exists until someone signs up, and `test@email.com` cannot sign in

#### Scenario: Another account exists

- **WHEN** the development database holds an account other than the dev account and the seed runs
- **THEN** that account's password, role and settings are unchanged afterwards

### Requirement: The commit check leaves the working tree as it is

The commit check SHALL NOT rewrite any file, SHALL NOT add any file or change to the commit, and SHALL NOT stash, restore or otherwise change unstaged changes in the working tree, whether it passes or refuses the commit. Several agents may edit one working tree at once, and a change one of them has not staged belongs to it alone.

#### Scenario: Another agent has unstaged edits

- **WHEN** a developer commits a staged backend file while another backend file has unstaged edits
- **THEN** after the check, passed or refused, the other file holds exactly the unstaged edits it held before
- **AND** the commit contains only what was staged

#### Scenario: A staged file has further unstaged edits

- **WHEN** a staged file has further unstaged edits and the commit check runs
- **THEN** the file in the working tree keeps both its staged and its unstaged content, and nothing is added to the index

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

- **WHEN** a developer runs `mise run test-full`
- **THEN** the Docker scenarios of the agent's end-to-end suite run with the other backend packages, and the Podman subset does not run
- **AND** `mise run test` runs neither

### Requirement: One routine test command checks a change

`mise run test` SHALL be the one routine local test command. A developer or an agent SHALL use it to check a change that adds or changes tests or the code those tests cover, and the OpenSpec change verification of such a change SHALL name it. An agent SHALL NOT run `mise run test-full` unless the user directly asks for that run. At the end of implementing a change it SHALL run `mise run test`, and by hand the specific long or end-to-end tests the change touches. A change that adds or changes scenarios of the browser end-to-end suite SHALL run and name `mise run e2e:browser`, and a task changing tests in `backend/internal/agent/e2e`, `backend/internal/modules/backups/e2e` or `backend/e2e/upgrade`, the verification agent's code or the `Dockerfile` SHALL name and run the command that repeats those tests. The repository SHALL NOT provide a command that takes backend packages or shard names as arguments and runs their tests. The selectors CI uses and the fixed suites `mise run agent:test`, `mise run agent:e2e`, `mise run agent:e2e-podman`, `mise run backend:test-upgrade` and `mise run test-filesystem` SHALL remain. `mise run e2e` SHALL NOT exist. The browser end-to-end suite SHALL have three hidden commands of its own, none of which `mise run test` runs: the hidden `mise run e2e:browser` runs its scenarios, `mise run e2e:unit` runs its parts that need no Docker daemon and `mise run e2e:lifecycle` runs its lifecycle probes. A change to the suite's other code SHALL run `mise run e2e:unit`, and SHALL also run `mise run e2e:lifecycle` when it changes how a run starts or cleans up. A developer or an agent SHALL still be able to repeat one failing test directly with the Go test command or the frontend test runner while fixing it.

#### Scenario: Checking a finished change

- **WHEN** an agent finishes implementing a change that adds or changes tests or the code they cover
- **THEN** it runs `mise run test` and reports the command and its result

#### Scenario: Checking a change to a browser scenario

- **WHEN** an agent finishes a change that adds or changes a browser end-to-end scenario
- **THEN** it runs `mise run test` and `mise run e2e:browser`, names them in the change verification and reports its result

#### Scenario: Checking a change to the suite's own code

- **WHEN** an agent finishes a change to the cleanup of a browser end-to-end run
- **THEN** it runs `mise run e2e:unit` and `mise run e2e:lifecycle` and reports their results

#### Scenario: A change without tests or covered code

- **WHEN** a change edits only documentation
- **THEN** it owes no `mise run test`

#### Scenario: Repeating one failing test

- **WHEN** a developer fixes a test that `mise run test` reported
- **THEN** the developer can repeat that test directly with the Go test command, without another `mise run test`

#### Scenario: An agent does not start the full run on its own

- **WHEN** an agent finishes implementing a change and the user has not directly asked for `mise run test-full`
- **THEN** it runs `mise run test` and the specific long or end-to-end tests the change touches, and it does not run `mise run test-full`

#### Scenario: The removed browser command

- **WHEN** a developer runs `mise run e2e`
- **THEN** mise reports that the task does not exist

#### Scenario: A removed selection command

- **WHEN** a developer runs `mise run backend:test-package ./internal/lib/...`
- **THEN** mise reports that the task does not exist

### Requirement: The routine run includes the frontend tests

`mise run test` SHALL install the frontend dependencies from the lockfile, as `mise run run` does, and SHALL run the frontend unit and screen tests at the same time as the backend packages. It SHALL wait for every test process, and a failure in one SHALL NOT stop the others. It SHALL exit with a non-zero status when any fails, and its output SHALL name each failed process and test. `mise run test` SHALL NOT start the upgrade test, and `mise run test-full` SHALL start it only after all have passed. The browser suites SHALL stay out of `mise run test`.

#### Scenario: A fresh worktree

- **WHEN** a developer runs `mise run test` in a worktree without `frontend/node_modules`
- **THEN** the dependencies are installed from the lockfile and the frontend tests run

#### Scenario: A frontend test fails

- **WHEN** a frontend test fails during `mise run test-full` while backend packages still run
- **THEN** the backend packages finish, the upgrade test does not start, the output names the failed frontend test, and the command exits with a non-zero status

#### Scenario: A backend package fails

- **WHEN** a package fails in one backend test process during `mise run test-full` while the other backend process and the frontend tests still run
- **THEN** they finish, the upgrade test does not start, the output names the failed process and test, and the command exits with a non-zero status

#### Scenario: Browser suites stay out

- **WHEN** a developer runs `mise run test`
- **THEN** no browser test runs

#### Scenario: Interrupting every process

- **WHEN** a developer presses Ctrl+C while both backend test processes and the frontend tests run
- **THEN** all of them exit, and no test process or container of this run remains

### Requirement: Slow backend packages do not hold back the others

`mise run test` and `mise run test-full` SHALL run the backend packages named in a repository list of slow packages separately from all other selected backend packages, in concurrent test processes, so that fast packages start without waiting for slow ones. The list SHALL be ordered by measured time, longest first, and the slow process SHALL receive the packages in that order, with a fixed package limit lower than that of the other process. Both limits SHALL be chosen by measurement. The run SHALL take slow packages only from those it selects anyway, so an entry the run does not select SHALL neither fail the run nor add a package. A script test SHALL fail and name an entry that no longer exists or that neither command selects. A selected package missing from the list SHALL run with the other packages. Every test of every selected package SHALL run exactly once, including when CI runs a package as several shards.

#### Scenario: The order and the limits

- **WHEN** `mise run test` starts its backend packages
- **THEN** the slow process receives the list's packages in the list's order with the lower package limit
- **AND** the other process starts at the same time with the remaining packages

#### Scenario: A new package

- **WHEN** a developer adds a test package without adding it to the list
- **THEN** `mise run test` runs it with the other packages

#### Scenario: A stale list entry

- **WHEN** the list names a package that no longer exists, or the upgrade packages the run excludes
- **THEN** `mise run test` ignores the entry and the script test fails and names it

#### Scenario: A package runs as local shards

- **WHEN** CI runs a package as several shards and one of its tests has no measured weight
- **THEN** that test still runs, in exactly one shard, and the package is not also run whole

### Requirement: A hung wait fails its test in minutes

One `testwait` wait for a condition in a backend test SHALL end after two minutes, unless the test states its own limit because it exists to observe a longer period of real time. Container start-up waits, including those of the upgrade test, keep their own limits. A wait that ends this way SHALL fail its test and name what it waited for and the last state. `mise run test`, `mise run test-full` and a local upgrade test SHALL give each package a timeout of 15 minutes. In CI the upgrade test SHALL keep 30 minutes, because it runs with the race detector.

#### Scenario: A condition never holds

- **WHEN** a test waits for a condition that never holds
- **THEN** the test fails within two minutes and names what it waited for and the last state

#### Scenario: A test that observes real time

- **WHEN** a test states its own limit for a wait
- **THEN** the wait ends at that limit
