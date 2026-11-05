## MODIFIED Requirements

### Requirement: One command starts the local app

The repository SHALL provide `mise run dev`. It SHALL start the development database, the shared backend test fixtures, the live WAL backup source with its load generator, the backend and the frontend development server, and SHALL show the database, the load generator, the backend, the frontend and the seed in one terminal window that lists every process separately, with each process's output available on its own. It SHALL work in a fresh worktree that has a `.env` file but has never generated the backend's API documentation or installed frontend dependencies. When a backend from an earlier run still holds the backend port, the command SHALL stop that process before starting the new backend, so that requests reach the code just started. It SHALL also stop a backend file watcher left over from an earlier run in the same worktree, so that no old watcher rebuilds and restarts a backend after the new run starts. A saved change to a frontend source file SHALL reach the open page without restarting any process.

#### Scenario: Developer starts the app

- **WHEN** a developer runs `mise run dev` on a Fedora host with a `.env` file
- **THEN** the development database, the shared test fixtures, the WAL backup source, its load generator, the backend and the frontend start
- **AND** the window lists the database, the load generator, the backend, the frontend and the seed as separate processes with their own output

#### Scenario: Fresh worktree

- **WHEN** a developer runs `mise run dev` in a worktree that has a `.env` file and nothing else prepared
- **THEN** the backend compiles and starts without a separate generation command
- **AND** the frontend development server starts without a separate dependency installation command

#### Scenario: Stale backend holds the port

- **WHEN** a backend from an earlier run still listens on the backend port
- **THEN** `mise run dev` stops that process and the new backend serves requests on that port

#### Scenario: Watcher from a killed run survives

- **WHEN** an earlier `mise run dev` in the same worktree was killed without cleanup and its backend file watcher is still running
- **THEN** `mise run dev` stops that watcher before starting its own
- **AND** a later saved Go file restarts only the backend of the new run

#### Scenario: Frontend file is saved

- **WHEN** a developer saves a change to a frontend source file while `mise run dev` is running
- **THEN** the open page shows the change without any process being restarted

### Requirement: Interrupting the local app stops what it started

When a developer interrupts `mise run dev` with Ctrl+C, the backend and the frontend development server SHALL exit, and the development database, the WAL backup source and its load generator SHALL stop. These containers SHALL be stopped rather than removed, so the database's data survives into the next run. The shared backend test fixtures SHALL keep running, because the backend suite and other worktrees use them.

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

The repository SHALL provide `mise run test`. It SHALL assume a Fedora host and, when `libedit` is installed on that host, SHALL make the bundled database clients startable without any further manual step. When `libedit` is missing, it SHALL stop before running tests and name the package to install. It SHALL start the shared test fixtures, pull the database engine and object storage images the backend tests reference, reset the shared database servers as the `backend-test-isolation` capability allows, and then run every backend test package, each test getting its internal database as that capability describes. After the backend packages pass, it SHALL run the upgrade test as the requirement "The full run includes the upgrade test" describes. It SHALL NOT prepare internal databases ahead of the tests. It SHALL NOT run the Docker filesystem suite. Its exit status SHALL be non-zero when any step or any test fails.

#### Scenario: Developer runs the suite on Fedora

- **WHEN** a developer runs `mise run test` on a Fedora host with a `.env` file
- **THEN** every backend test package runs, each against internal databases with every migration of the checkout applied
- **AND** tests that start a bundled database client do not fail on a missing shared library

#### Scenario: A test fails

- **WHEN** any backend test fails during `mise run test`
- **THEN** the command exits with a non-zero status

#### Scenario: Filesystem suite is not part of the run

- **WHEN** a developer runs `mise run test`
- **THEN** no Docker filesystem case runs

### Requirement: The full run includes the upgrade test

After the backend test packages pass, `mise run test` SHALL run the upgrade test. When the developer names a Databasus image through `DATABASUS_IMAGE`, the upgrade test SHALL use it. Otherwise `mise run test` SHALL build the image from the checkout first. A failed upgrade test SHALL make `mise run test` exit with a non-zero status. When a backend package fails, `mise run test` SHALL stop without building the image or running the upgrade test. A single package run SHALL NOT build an image or run the upgrade test. When the developer interrupts `mise run test` during the upgrade test, the containers, the network and the volume of the upgrade test SHALL be removed, and the shared test fixtures SHALL keep running.

#### Scenario: Developer runs the full suite without an image

- **WHEN** a developer runs `mise run test` without `DATABASUS_IMAGE`
- **THEN** the image is built from the checkout after the backend packages pass, and the upgrade test runs against it

#### Scenario: Developer names an image

- **WHEN** a developer runs `mise run test` with `DATABASUS_IMAGE` set to an image they built
- **THEN** no image is built and the upgrade test runs against the named image

#### Scenario: The upgrade test fails

- **WHEN** the upgrade test fails during `mise run test`
- **THEN** the command exits with a non-zero status

#### Scenario: A backend package fails

- **WHEN** a backend package fails during `mise run test`
- **THEN** no image is built, the upgrade test does not run, and the command exits with a non-zero status

#### Scenario: A single package run

- **WHEN** an agent runs `mise run backend:test-package` for one backend package
- **THEN** no Databasus image is built and no upgrade test container starts

#### Scenario: Interrupt during the upgrade test

- **WHEN** a developer presses Ctrl+C while the upgrade test runs during `mise run test`
- **THEN** none of the upgrade test's containers, its network or its volume remains
- **AND** the shared test fixtures are still running

### Requirement: Interrupting the backend suite leaves no test processes or containers

When a developer interrupts `mise run test` with Ctrl+C, every test process of that run SHALL exit, and every test container and test network that run started SHALL be removed. Test containers and networks started by another run or by a single package run, in this worktree or another one, SHALL NOT be removed. The shared test fixtures SHALL keep running, so that the next run and other worktrees can use them.

#### Scenario: Developer presses Ctrl+C mid-suite

- **WHEN** a developer presses Ctrl+C while `mise run test` runs tests
- **THEN** no test process from that run remains
- **AND** no test container that run started remains

#### Scenario: Another worktree's tests are running

- **WHEN** a developer interrupts `mise run test` while a test package of another worktree runs with its own test container
- **THEN** that container is still running afterwards and that package's tests continue

#### Scenario: Shared fixtures survive the interrupt

- **WHEN** a developer interrupts `mise run test`
- **THEN** the shared test fixtures are still running afterwards

### Requirement: Full backend runs take turns

Only one `mise run test` SHALL run tests on a machine at a time. A `mise run test` started while another one runs, in any worktree, SHALL say that it waits for the other run and SHALL start its tests after the other run ends. A single test package run SHALL NOT wait for a full run, apart from the wait for the reset of the shared test PostgreSQL servers. A full run SHALL NOT remove containers or networks of a package run, SHALL NOT reset or delete an internal database a package run uses, and SHALL reset the shared test PostgreSQL servers only after the package run's process has ended. Tests of old code that change shared objects on those servers can still affect each other across runs, as they can today; this requirement does not cover that.

#### Scenario: Two worktrees start a full run

- **WHEN** an agent in one worktree starts `mise run test` while a full run of another worktree is running
- **THEN** the second run prints that it waits for another full run
- **AND** its tests start after the first run ends

#### Scenario: A package run during a full run

- **WHEN** an agent runs one backend test package while a full run of another worktree is running its tests
- **THEN** the package run starts its tests without waiting for the full run to end
- **AND** its containers, networks and internal database stay in place until the package run ends

### Requirement: Test images can be pulled ahead of a run

The repository SHALL provide `mise run pull-testcontainers`. It SHALL pull the database engine images (PostgreSQL, MySQL, MariaDB and MongoDB) and the object storage image that the backend tests reference, SHALL retry each failed pull a bounded number of times, and SHALL exit with a non-zero status when an image still cannot be pulled. Other test images, such as the FTP, SFTP, SMB, mail and Azure storage emulators, are pulled by the tests on first use.

#### Scenario: Pulling succeeds

- **WHEN** a developer runs `mise run pull-testcontainers` with registry access
- **THEN** every database engine and object storage image the backend tests reference is present locally afterwards

#### Scenario: An image cannot be pulled

- **WHEN** an image still fails to pull after its retries
- **THEN** the command names that image and exits with a non-zero status

### Requirement: Root commands lint and format every module

The repository SHALL provide `mise run lint` and `mise run format`, covering the backend, the verification agent and the frontend. `mise run lint` SHALL exit with a non-zero status when any module fails its checks, and SHALL report the failures of every failing module, not only the first. `mise run format` SHALL rewrite files in all three modules to their formatter's output. Each module SHALL also have its own lint and format tasks, `mise run backend:lint`, `mise run agent:lint`, `mise run frontend:lint` and the matching `format` tasks, and every task of a module SHALL run in that module's directory whichever directory of the repository it is started from.

#### Scenario: One module fails lint

- **WHEN** the frontend has a lint error and the backend and the verification agent are clean
- **THEN** `mise run lint` exits with a non-zero status and reports the frontend error

#### Scenario: Two modules fail lint

- **WHEN** both the backend and the frontend have a lint error
- **THEN** `mise run lint` exits with a non-zero status and reports the errors of both modules

#### Scenario: Developer formats everything

- **WHEN** a developer runs `mise run format`
- **THEN** a later `mise run lint` reports no formatting difference in any of the three modules

#### Scenario: Module task started from another directory

- **WHEN** a developer runs `mise run backend:lint` from the frontend directory
- **THEN** the backend checks run against the backend module and pass or fail as they would from the backend directory

### Requirement: Tool versions are pinned in the repository

The repository SHALL pin the exact versions of a defined tool set, so that two hosts set up from the same commit run the same versions of those tools. The pinned set SHALL cover the backend and frontend toolchains, the commit hook runner, the process window, the migration tool, the Go linter, the API contract linter, the backend file watcher and the browser automation server agents use. The version manager that installs these tools also runs the repository's tasks, and the repository SHALL name the minimum version of it that the tasks need. An older version manager SHALL refuse to run a task and name the minimum version. Docker, the shell, the Chrome browser and system libraries such as `libedit` come from the host and are not pinned.

#### Scenario: New host setup

- **WHEN** a developer installs the pinned tools on a new Fedora host and runs `mise run lint`
- **THEN** every tool from the pinned set that the task calls resolves to its pinned version

#### Scenario: The version manager is too old

- **WHEN** a developer whose version manager is older than the minimum the repository names runs `mise run lint`
- **THEN** no check runs, and the output names the minimum version

#### Scenario: Browser server starts at its pinned version

- **WHEN** an agent client starts the browser automation server registered in the repository
- **THEN** the server runs at the version pinned in the repository, without downloading a newer one

### Requirement: Commits are checked before they are created

Creating a commit SHALL check that each staged file of each module the commit touches matches that module's formatter, SHALL run that module's lint checks, and for the frontend its type check as well, and SHALL refuse the commit when any of them fails. When a staged file is not formatted, the refusal SHALL name the file and the command that formats it. The check reads each staged file as it is in the working tree, because it leaves the working tree and the index as they are; content staged for the commit that differs from the working tree is not checked separately. A module the commit does not touch SHALL NOT be checked.

#### Scenario: Unformatted file is staged

- **WHEN** a developer commits a staged frontend file that is not formatted
- **THEN** no commit is created, and the output names the file and the command that formats it

#### Scenario: Lint error in a staged module

- **WHEN** a developer commits a backend change that fails the backend lint checks
- **THEN** no commit is created and the lint output is shown

#### Scenario: Untouched module

- **WHEN** a commit touches only the backend
- **THEN** no frontend or verification agent check runs

#### Scenario: Staged content differs from the working tree

- **WHEN** a developer stages an unformatted Go file, then formats it in the working tree without staging the result, and commits
- **THEN** the formatting check passes, because it reads the formatted working-tree file

### Requirement: The backend restarts when its source changes

While `mise run dev` is running, saving a change to a backend Go source file SHALL rebuild the backend and replace the running backend with the new build, without the developer touching the process window. A change to a test file alone SHALL NOT restart the backend. When the new code fails to compile, the backend that was already running SHALL keep serving requests, and the compile error SHALL appear in the backend's output in the process window. Interrupting `mise run dev` SHALL leave no backend process from that run, including one started by a restart.

#### Scenario: Developer saves a backend change

- **WHEN** a developer changes the response of an endpoint and saves the Go file while `mise run dev` is running
- **THEN** a request to that endpoint returns the new response once the rebuild finishes, without any manual restart

#### Scenario: The change does not compile

- **WHEN** a developer saves a Go file that does not compile
- **THEN** the backend output shows the compile error
- **AND** the previous backend still answers requests on the backend port

#### Scenario: Only a test file changes

- **WHEN** a developer saves a change to a Go test file and nothing else
- **THEN** the backend is not restarted

#### Scenario: Interrupt after restarts

- **WHEN** the backend has restarted several times and the developer presses Ctrl+C in the `mise run dev` window
- **THEN** no backend process from that run remains

### Requirement: The local app is seeded with a dev account and databases

When the backend started by `mise run dev` begins answering requests, the seed SHALL make sure the local app contains:

- the dev account `test@email.com` with password `Q1234567`;
- a workspace owned by that account;
- a local storage in that workspace;
- a PostgreSQL database with logical backups of the shared logical test fixture;
- a PostgreSQL database with WAL-streaming physical backups of the live WAL backup source.

The seed SHALL go through the same public API and validation as the web interface. It SHALL find each item by its name, SHALL create only the items that are missing and SHALL NOT change an item that already exists. When the seed creates a database, it SHALL switch its backups on with the local storage and SHALL start the first backup. The logical database SHALL keep its 3 newest backups, and the WAL-streaming database SHALL keep its 2 newest backup chains.

The seed command, `go run ./cmd/seed_dev` in the backend directory, SHALL also run on its own against the app already answering on the backend port, with the same behavior. The repository SHALL NOT provide a separate task for it. When nothing answers on the backend port, the seed command SHALL exit with a non-zero status and say that the app has to be started with `mise run dev`. When the backend answers its health check with an error status, for example because disk usage is too high, the seed SHALL exit with a non-zero status and show the reason the backend gave, and SHALL create nothing.

When the seed cannot sign in as the dev account and cannot create it, for example because the account exists with a different password or registration is closed, it SHALL exit with a non-zero status and a message that names the account and the reason the backend gave, and SHALL create nothing. When the instance requires two-factor sign-in, the seed SHALL exit with a non-zero status and a message that names the account and says that two-factor sign-in is required, and SHALL create nothing. When the backend refuses to create a seeded item after sign-in, for example because the dev account is a member that may not create workspaces, the seed SHALL exit with a non-zero status, show the name of that item and the reason the backend gave, and SHALL NOT change any item that already exists. A seed that fails SHALL NOT stop the other processes of `mise run dev`.

#### Scenario: First run on an empty app

- **WHEN** a developer runs `mise run dev` against a development database that has no users
- **THEN** after the backend starts, signing in as `test@email.com` with `Q1234567` succeeds
- **AND** the account's workspace lists the local storage and both databases, each with backups switched on and a first backup started
- **AND** the logical database's backup settings keep 3 backups, and the WAL-streaming database's keep 2 chains

#### Scenario: Second run

- **WHEN** a developer runs `mise run dev` again with the seeded items already present
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
- **THEN** the command exits with a non-zero status and tells the developer to start the app with `mise run dev`

### Requirement: Seed credentials exist only in local development

The dev account, its password and the seeded databases SHALL be created only by `mise run dev` and by the seed command run on its own in a development checkout. The production image and the production backend SHALL neither contain the seed nor run it, whatever the environment variables, and SHALL create no account with known credentials. The seed SHALL NOT change the password, the role or the settings of any existing account, including the dev account.

#### Scenario: Production image starts

- **WHEN** the production image starts with an empty data directory
- **THEN** no account exists until someone signs up, and `test@email.com` cannot sign in

#### Scenario: Another account exists

- **WHEN** the development database holds an account other than the dev account and the seed runs
- **THEN** that account's password, role and settings are unchanged afterwards

## ADDED Requirements

### Requirement: The commit check leaves the working tree as it is

The commit check SHALL NOT rewrite any file, SHALL NOT add any file or change to the commit, and SHALL NOT stash, restore or otherwise change unstaged changes in the working tree, whether it passes or refuses the commit. Several agents may edit one working tree at once, and a change one of them has not staged belongs to it alone.

#### Scenario: Another agent has unstaged edits

- **WHEN** a developer commits a staged backend file while another backend file has unstaged edits
- **THEN** after the check, passed or refused, the other file holds exactly the unstaged edits it held before
- **AND** the commit contains only what was staged

#### Scenario: A staged file has further unstaged edits

- **WHEN** a staged file has further unstaged edits and the commit check runs
- **THEN** the file in the working tree keeps both its staged and its unstaged content, and nothing is added to the index
