# browser-e2e Specification

## Purpose

Defines the independent browser end-to-end suite: what it proves about the path a new user depends on when the application image runs as a black box, and how each run isolates and removes everything it starts.

## Requirements

### Requirement: The suite treats the application as a black box

The suite SHALL live in the root `e2e/` directory and SHALL import nothing from the backend or the frontend. Its scenarios and their fixtures SHALL hold no path into those trees and SHALL reach the application only through its address, its HTTP responses, the browser storage the application uses and its screens. A lifecycle probe MAY also write into the log of a container of its own run, because it tests the suite and not the application. The code that starts a run SHALL use the checkout only as the context of an image build. Requests for application operations SHALL reach the real application. Only requests to external services the application calls from the browser, such as GitHub metadata, MAY be answered by the suite.

#### Scenario: Backend internals change

- **WHEN** a backend or frontend internal unit that is not visible through screens, HTTP or browser storage is renamed
- **THEN** the suite still runs without any change

#### Scenario: A screen breaks an operation

- **WHEN** a form sends settings that the real application refuses
- **THEN** the scenario using that form fails and shows the refusal

### Requirement: One command starts everything the suite needs

`mise run e2e:browser` SHALL start a supervisor, the process that owns one run. The run SHALL start everything it needs and SHALL require no running development application, no environment file and no shared database server. It SHALL install the suite's dependencies from their lockfile; this ignored dependency directory is the only part of the checkout a run writes to. The output SHALL name the image under test. Its exit status SHALL be zero only when every scenario passed and cleanup succeeded.

#### Scenario: A fresh worktree

- **WHEN** a developer runs `mise run e2e:browser` in a fresh checkout with Docker and the pinned tools, with no environment file and no installed dependencies
- **THEN** the suite installs its dependencies from the lockfile, builds the image and runs

#### Scenario: A run leaves the checkout unchanged

- **WHEN** a run finishes in a checkout
- **THEN** the checkout's tracked, untracked and ignored files differ only by the ignored dependency directory

### Requirement: The suite runs the application image like a deployment

A run SHALL run the application image through a compose file that mirrors the documented production deployment: the image's own startup, its internal database on an empty data volume and its health check. It SHALL publish no port of the application on the host, so a development application on port `4005` does not conflict with it. It SHALL use the image named by `DATABASUS_IMAGE` when set and otherwise SHALL build one from the checkout. When the `CI` environment variable is set and `DATABASUS_IMAGE` is not, the run SHALL fail before any scenario.

#### Scenario: A named image

- **WHEN** `mise run e2e:browser` runs with `DATABASUS_IMAGE` set
- **THEN** the output names that image, and the suite tests it, builds none, publishes no application port on the host and leaves the image in place

#### Scenario: No image is named locally

- **WHEN** `mise run e2e:browser` runs on a developer's machine without `DATABASUS_IMAGE`
- **THEN** the suite builds an image from the checkout under a name unique to the run, names it in the output, tests it and removes that name afterwards

#### Scenario: CI without an image

- **WHEN** the suite starts in CI without `DATABASUS_IMAGE`
- **THEN** the command fails and names the missing setting, builds no image, and no scenario runs

### Requirement: The critical path is verified from the first account to the restored rows

On an instance holding no account, a run SHALL register the first account through the interface and see that the interface says this account will administer the instance. A scenario SHALL then, through the interface alone: sign in as that instance admin, create a workspace and a local storage, add a PostgreSQL logical database through the database wizard with the connection test passing, wait until the backup the wizard starts is shown as completed, start a restore of that backup into an empty database of the same server and wait until the restore is shown as completed. The run SHALL write rows into the database it backs up and SHALL check after the restore that this database still holds them, so two empty databases cannot pass. It SHALL then read the restored database directly and compare its rows with the rows it wrote. A status shown on a screen SHALL NOT be enough to pass.

After the logical restore passes, the same scenario SHALL continue on the same running instance, account, workspace and storage. Through the interface, it SHALL add a physical PostgreSQL 18 connection configured for full backups, incremental backups and WAL streaming, see its connection test pass, and complete a full backup followed by an incremental backup on that full backup's chain. The run SHALL write distinguishable rows before the full backup, between the full and incremental backups, and in two separate stages after the incremental backup. The latter two stages SHALL finish different WAL segments, and the run SHALL wait until both segments and the WAL needed to replay through the final write are archived. Two rows in the backup list SHALL NOT substitute for evidence of two distinct archived segments.

The scenario SHALL restore the physical chain to its latest archived state twice, first through the Host tab and then through the Docker tab of the Script restore method. For each restore, it SHALL obtain a fresh command through the interface, fill a distinct non-default restore directory containing a space, fill the tab's PostgreSQL tools or image field, and execute the rendered recovery command unchanged. The scenario SHALL also execute the startup commands shown by that tab in the prepared recovery environment. It SHALL NOT reconstruct the commands from an API response or application code, replace their URLs or arguments, or repair a displayed command in the test. Each restored cluster SHALL finish recovery and contain exactly the recorded rows from all four write stages. The source SHALL still contain those rows after both restores.

#### Scenario: The path works

- **WHEN** the critical path scenario runs
- **THEN** it passes, and the restored database holds exactly the rows of the backed-up database
- **AND** both physical restore environments serve the recorded full, incremental and WAL-stage rows

#### Scenario: A restore loses rows

- **WHEN** the restore is shown as completed but the restored database does not hold the backed-up database's rows
- **THEN** the run fails and names the difference

#### Scenario: The first account takes the instance

- **WHEN** the run opens its empty instance
- **THEN** the sign-up form says the account will administer the instance, and after submitting it the account can open the instance settings

#### Scenario: A command shown by the interface is wrong

- **WHEN** a rendered recovery or startup command uses an incorrect path, image, tool directory or argument
- **THEN** the run fails at command execution or at the restored-data check instead of correcting the command

#### Scenario: WAL is missing from a restore

- **WHEN** the restored cluster lacks either of the two write stages made after the incremental backup
- **THEN** the run fails and identifies the missing rows, even if backup and restore statuses look successful

#### Scenario: Both tabs restore the same chain

- **WHEN** the Host restore finishes and the scenario starts the Docker restore
- **THEN** it obtains a new single-use restore command through the interface and writes to a different directory
- **AND** it keeps the same Databasus instance, account, workspace, storage and physical backup chain

### Requirement: Sign-in and workspace behavior stay covered

The suite SHALL verify through the real application that a user signs in, sees their own email in the profile and signs out, that signing out clears the account's stored selections while keeping the language, the theme and the version reload marker, that an instance without a mail server offers no password reset, and that saved storages persist across a reload and are isolated between workspaces.

#### Scenario: Signing in

- **WHEN** a user signs in on an instance without a mail server
- **THEN** the sign-in form has offered no password reset, and the profile shows the user's own email

#### Scenario: Signing out

- **WHEN** a signed-in user signs out
- **THEN** the sign-in form is shown, the browser's session token, selected workspace, stored selections of databases and backup filters are gone, and the language, the theme and the version reload marker remain

#### Scenario: Another workspace

- **WHEN** a user saves a storage, reloads the page and switches to a workspace without storages
- **THEN** the storage is still selected after the reload, neither the screen nor the API lists it in the other workspace, and returning shows it again

### Requirement: A run removes what it started after every outcome

A run SHALL stop and remove every container, network and volume it started, its temporary directory, and any image it built under its own name, after success, a failed scenario, a failed start-up and an interruption. An interrupted or failed run SHALL exit with a non-zero status. Cleanup failure SHALL fail the run and name what remains. This cleanup SHALL NOT remove an image the run did not build.

#### Scenario: A scenario fails after start-up

- **WHEN** a probe deliberately fails after the application and the browser are ready
- **THEN** the command fails with the intended diagnostic, and nothing the run started remains

#### Scenario: The application never becomes ready

- **WHEN** the application does not answer during start-up
- **THEN** the command fails, prints the end of the application's log, and nothing the run started remains

#### Scenario: A run is interrupted

- **WHEN** a run receives an interrupt or termination signal after starting its resources
- **THEN** it exits non-zero, and nothing the run started remains
- **AND** the supervisor removes the scenario child's temporary root after terminating that child, even if pytest's cleanup callbacks did not run

#### Scenario: Cleanup cannot finish

- **WHEN** a resource of the run cannot be removed
- **THEN** the command fails and names that resource, even when every scenario passed
- **AND** it prints the masked log tails captured before teardown, including logs of containers already removed

### Requirement: A run removes what crashed runs left behind

Before it starts its own resources, a run SHALL remove the resources of earlier runs whose supervisor no longer exists: their containers, networks, volumes, temporary directories and the images they built under their own names. It SHALL NOT touch the resources of a run whose supervisor is alive, and SHALL NOT touch anything the suite did not create. The only images it removes are those a run or `mise run e2e:lifecycle` recorded as its own build.

#### Scenario: An earlier run was killed

- **WHEN** an earlier run was killed before its cleanup ran, and a new run starts
- **THEN** the new run removes the earlier run's containers, networks, volumes, temporary directory and built image before it starts
- **AND** an exited supervisor counts as dead even while its parent has not reaped its process entry

#### Scenario: Another worktree is running the suite

- **WHEN** a run starts while another worktree's run is in progress
- **THEN** the other run continues and passes

#### Scenario: Crashed resources cannot be removed

- **WHEN** removal of a crashed run's project or built image fails
- **THEN** the new run fails before building its image or starting its resources and names what could not be removed
- **AND** it preserves the crashed run's supervisor record so a later run can recover the resources

### Requirement: Overlapping runs do not disturb each other

Runs SHALL be able to overlap on one machine, from one worktree or several. Each run SHALL own its names, ports, data and temporary directories, and SHALL NOT take any machine-wide lock. A run SHALL NOT stop or change the shared test PostgreSQL server, the development services or another run's resources.

#### Scenario: Two runs overlap

- **WHEN** two runs execute concurrently
- **THEN** both pass, and their accounts, data, ports and temporary directories are different

#### Scenario: A nested test process finishes

- **WHEN** a browser-suite pytest process finishes while another process is using temporary files
- **THEN** it removes only its own temporary root and leaves the other process's files usable

### Requirement: Secrets stay out of a run's output

Each run SHALL generate its own account and database passwords, and SHALL pass them to what it starts through the environment of those processes and not through a file in the repository. A run SHALL record no browser trace, video or screenshot and SHALL retain no file of its own after it ends other than the ignored dependency directory. Docker's layer cache, pulled images and the package manager's cache are shared caches and stay. Everything a run prints, including its test output, the application's and containers' logs, and recovery command diagnostics, SHALL have the run's generated passwords, any bearer token and any physical restore token masked. A failed run SHALL print the end of the log of each of its containers, the application's and recovery environments' included. Failure output SHALL NOT include the values of local variables or reveal a restore token through a URL or an echoed command.

#### Scenario: A failed run

- **WHEN** a run fails after signing in
- **THEN** its printed output, including test output, the application's log and container logs, contains neither the run's passwords nor any bearer token or physical restore token, and no file of the run remains on disk other than the ignored dependency directory

#### Scenario: A recovery command fails

- **WHEN** a recovery command fails and its diagnostic includes the restore URL
- **THEN** the output identifies the failing recovery stage and exit status while masking the restore token

#### Scenario: Recovery log collection stops responding

- **WHEN** a failed run's private recovery engine stops responding during log collection
- **THEN** the run prints the available outer container logs and partial recovery output with secrets masked
- **AND** it identifies the log collection timeout and removes its resources

### Requirement: The suite's lifecycle guarantees are themselves tested

`mise run e2e:lifecycle` SHALL run lifecycle probes, each of which drives the public entry point. The probes SHALL go through a failing scenario, a failure with secrets in its output, a refused form, a start-up failure, a failure outside a scenario, an interruption, a cleanup failure, a leftover of a killed run next to a live run, and two overlapping runs. It SHALL also drive the refusal in CI and a named image. When no image is named, it SHALL build one image for its probes and remove it after every outcome, and it SHALL also drive a run that builds its own image, a killed run whose built image the next run removes, and a fresh checkout, including that a run leaves that checkout unchanged. When one is named, it SHALL skip those three and build nothing. With the `CI` environment variable set and no image named, it SHALL refuse and build nothing. It SHALL start an unrelated container of its own outside every run and query it after each probe. It SHALL check the exit status, the diagnostic, the absence of secrets in the captured output, that nothing of those runs remains and that the unrelated container still answers. It SHALL NOT be part of `mise run e2e:browser`.

The probes SHALL also cover refusal to remove a crashed run's project and its built image before startup.

Probes that kill a run or deliberately prevent its cleanup SHALL register unconditional teardown of that exact run's project, owned image and temporary directory before the destructive step. This teardown SHALL still run if a later recovery command or assertion fails.

#### Scenario: Cleanup regresses

- **WHEN** a change makes an interrupted run leave a container behind
- **THEN** the lifecycle command fails and names that container

#### Scenario: The lifecycle command in CI

- **WHEN** `mise run e2e:lifecycle` runs with `DATABASUS_IMAGE` set
- **THEN** every probe that applies runs against that image, and no image is built

#### Scenario: The lifecycle command in CI without an image

- **WHEN** `mise run e2e:lifecycle` runs with the `CI` environment variable set and no `DATABASUS_IMAGE`
- **THEN** the command fails and names the missing setting, and no image is built

#### Scenario: A run leaves other services alone

- **WHEN** a lifecycle probe finishes in any outcome
- **THEN** the unrelated container the command started is still running and answering

#### Scenario: The lifecycle command is interrupted

- **WHEN** `mise run e2e:lifecycle` is interrupted after building its image
- **THEN** it removes that image, and if it was killed instead, the next run removes it

### Requirement: Physical recovery environments belong to the run

A run SHALL prepare the tools, empty directories and isolated container engine needed by both restore environments without requiring a running development application or installing PostgreSQL on the developer's host. Commands shown by the interface SHALL resolve the same application address from both recovery environments as from the browser. The run SHALL leave that address unchanged in the commands and SHALL publish no application or restored-database port on the host. Every recovery process, container, engine state and directory SHALL belong to that run and SHALL be removed after success, failure, interruption and recovery of a crashed run. Overlapping runs SHALL NOT share writable recovery data or a recovery engine.

#### Scenario: Two complete critical paths overlap

- **WHEN** two browser runs execute logical and physical recovery concurrently
- **THEN** each restores its own recorded rows, and cleaning up either run leaves the other one's recovery resources usable

#### Scenario: A run fails during physical recovery

- **WHEN** a run fails or is interrupted after its recovery engine has started a restore container
- **THEN** cleanup removes the recovery processes, containers, directories and engine state without changing the shared host engine or another run

#### Scenario: A physical recovery run is killed

- **WHEN** a run is killed while its recovery resources exist and a later run starts
- **THEN** the later run removes those resources before starting its own and leaves live runs untouched
