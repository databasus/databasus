## MODIFIED Requirements

### Requirement: Everyday commands are separate from internal tasks

The repository SHALL expose exactly nine everyday mise tasks: `run`, `test`, `test-change`, `lint`, `format`, `doctor`, `review`, `generate-api` and `verify-deps`. In `mise.toml`, everyday commands SHALL appear first, internal CI/CD commands second, and tool dependencies and environment settings last. Internal tasks SHALL appear in the CI/CD section or in `mise-tasks/`, SHALL be hidden from the default listing, and SHALL remain callable by name from the repository root, backend and frontend. The CI/CD section SHALL explain that its tasks also serve hooks. The repository SHALL NOT retain aliases for `dev`, `full`, `test-full` or the removed root `pull-testcontainers` wrapper.

#### Scenario: Listing commands from a module

- **WHEN** a developer lists mise tasks from the repository root, backend or frontend
- **THEN** the default list shows exactly the nine everyday tasks
- **AND** CI and hooks can still invoke the hidden tasks by name

#### Scenario: The removed full command

- **WHEN** a developer runs `mise run test-full`
- **THEN** mise reports that the task does not exist

### Requirement: The full command runs the Docker test suites

`mise run test` SHALL run in the Ubuntu development VM and SHALL start the bundled database clients with the shared libraries that the VM provides under their own names, without creating aliases for missing ones. It SHALL start the shared test PostgreSQL server and no other development service, pull the database engine and object storage images the backend tests reference, and install the frontend dependencies from the lockfile, as `mise run run` does. It SHALL run every test package of the backend module except the upgrade test with its comparison helper, which run in the upgrade suite. It SHALL also run the frontend unit and screen tests, script tests, review hook tests, browser e2e unit tests, frontend browser UI, the upgrade test, browser end-to-end scenarios, lifecycle checks and lifecycle interruption checks. Each test that asks for a migrated internal database SHALL get one as that capability describes; a migration test that asks for an empty internal database SHALL get one with no migration applied and SHALL apply the migrations it needs itself, and packages that need no database SHALL open none. A package in a new directory of the module SHALL join this run without any change to the selection. Each suite SHALL start as soon as what it needs is ready and SHALL NOT wait for an unrelated suite to finish. Suites that run the application image SHALL start once the run has that image. The crashed-run probes and then the lifecycle interruption checks SHALL run while no other browser end-to-end run of the same command is active: neither SHALL start while another browser end-to-end run of the command is active, and the command SHALL start no other browser end-to-end run until both have ended. A failed suite SHALL NOT stop or prevent any other suite: every suite that can start SHALL run to its end, and the lifecycle interruption checks SHALL run even when the crashed-run probes fail. Only a failed image build SHALL keep suites from starting, namely those that run the image. The command SHALL then exit with a non-zero status and name each failed suite and test and each suite that could not start. When the developer interrupts the command, every suite's process SHALL exit, and no compose project, container, network or volume that the run's suites started SHALL remain, nor the image the run built; the shared test PostgreSQL server SHALL keep running. It SHALL take the same machine-wide lock as `mise run test-change`, and SHALL NOT update browser visual references. CI SHALL run its individual suites rather than `mise run test`. Podman and filesystem tests SHALL remain separate because they require additional host configuration.

#### Scenario: All ordinary Docker checks pass

- **WHEN** every suite passes during `mise run test`
- **THEN** every included suite runs once and the command succeeds
- **AND** neither the Podman subset nor the filesystem suite runs

#### Scenario: Developer runs the suite in the VM

- **WHEN** a developer runs `mise run test` in the Ubuntu development VM with a `.env` file
- **THEN** every selected backend test package runs, and each test that asks for a migrated internal database gets one with every migration of the checkout applied
- **AND** a migration test that asks for an empty internal database gets one with no migration applied, and packages that need no database open none
- **AND** tests that start a bundled database client do not fail on a missing shared library

#### Scenario: Only the test PostgreSQL server starts

- **WHEN** a developer runs `mise run test`
- **THEN** the shared test PostgreSQL server is started
- **AND** the command starts no other service of the development compose file

#### Scenario: A fresh worktree

- **WHEN** a developer runs `mise run test` in a worktree without `frontend/node_modules`
- **THEN** the dependencies are installed from the lockfile and the frontend tests run

#### Scenario: A package in a new directory

- **WHEN** a developer adds a test package in a new top-level directory of the backend module
- **THEN** `mise run test` runs its tests without any change to the selection

#### Scenario: Suites start without waiting for the backend packages

- **WHEN** `mise run test` starts
- **THEN** the image build, the browser UI suite, the script tests, the review hook tests, the browser e2e unit tests and the crashed-run probes start while the backend packages are still running

#### Scenario: An additional suite fails

- **WHEN** one suite fails during `mise run test`
- **THEN** every other suite still runs to its end
- **AND** the command exits with a non-zero status and its summary names the failed suite

#### Scenario: A frontend test fails while backend packages run

- **WHEN** a frontend test fails during `mise run test` while backend packages still run
- **THEN** the backend packages finish, the output names the failed frontend test, and the command exits with a non-zero status

#### Scenario: The image cannot be built

- **WHEN** the application image fails to build during `mise run test`
- **THEN** the suites that run that image do not start, and the other suites run to their end
- **AND** the command exits with a non-zero status and names the failed build and each suite that did not start

#### Scenario: Crashed-run probes and interruption checks run alone

- **WHEN** `mise run test` runs the crashed-run probes and then the lifecycle interruption checks
- **THEN** no other browser end-to-end run of that command is active when the first of them starts
- **AND** the command starts no browser end-to-end run until both have ended

#### Scenario: The crashed-run probes fail

- **WHEN** a crashed-run probe fails during `mise run test`
- **THEN** the lifecycle interruption checks still run, and the command exits with a non-zero status

#### Scenario: Interrupting the concurrent schedule

- **WHEN** a developer presses Ctrl+C while the backend packages, the image build or the upgrade test, the browser UI suite, the browser end-to-end scenarios and the lifecycle checks run at the same time
- **THEN** no process of any suite remains, and no compose project, container, network or volume of the run's suites remains, nor the image the run built
- **AND** the shared test PostgreSQL server is still running

### Requirement: The full run includes the upgrade test

`mise run test` SHALL run the upgrade test as soon as the run has the application image, while other suites may still be running. When the developer names a Databasus image through `DATABASUS_IMAGE`, the upgrade test and the browser end-to-end scenarios SHALL use it and the run SHALL build no application image for them. Otherwise `mise run test` SHALL build the image from the checkout once, at its start, under a tag that only `mise run test` and `mise run test-change` use, and both suites SHALL use that image. Because these runs take turns, that tag cannot collide with another run, and a later run replaces an image that a killed run left under it. The run SHALL remove the image it built when it ends after success, a failure or an interruption. A failed upgrade test SHALL make `mise run test` exit with a non-zero status. A failed backend package or frontend test SHALL NOT prevent the image build or the upgrade test. A single test package started directly with the Go test command SHALL NOT build an image or run the upgrade test. When the developer interrupts `mise run test` during the upgrade test, the containers, the network and the volume of the upgrade test SHALL be removed, and the shared test PostgreSQL server SHALL keep running.

#### Scenario: Developer runs the full suite without an image

- **WHEN** a developer runs `mise run test` without `DATABASUS_IMAGE`
- **THEN** the image is built once from the checkout, and the upgrade test and the browser end-to-end scenarios run against it
- **AND** the image the run built does not remain after the run ends

#### Scenario: Developer names an image

- **WHEN** a developer runs `mise run test` with `DATABASUS_IMAGE` set to an image they built
- **THEN** the upgrade test and the browser end-to-end scenarios run against the named image, the run builds no application image for them, and the named image remains after the run

#### Scenario: The upgrade test fails

- **WHEN** the upgrade test fails during `mise run test`
- **THEN** the command exits with a non-zero status

#### Scenario: The frontend tests fail

- **WHEN** a frontend test fails during `mise run test`
- **THEN** the image is still built and the upgrade test still runs to its end, and the command exits with a non-zero status

#### Scenario: A backend package fails

- **WHEN** a backend package fails during `mise run test`
- **THEN** the upgrade test still runs to its end, and the command exits with a non-zero status

#### Scenario: A single package run

- **WHEN** an agent runs one backend package directly with the Go test command
- **THEN** no Databasus image is built and no upgrade test container starts

#### Scenario: Interrupt during the upgrade test

- **WHEN** a developer presses Ctrl+C while the upgrade test runs during `mise run test`
- **THEN** none of the upgrade test's containers, its network or its volume remains, and the image the run built is removed
- **AND** the shared test PostgreSQL server is still running

### Requirement: Interrupting the backend suite leaves no test processes or containers

When a developer interrupts `mise run test` or `mise run test-change` with Ctrl+C, every test process of that run SHALL exit, and every test container and test network that run started SHALL be removed. Test containers and networks started by another run or by a single package run, in this worktree or another one, SHALL NOT be removed. The shared test PostgreSQL server SHALL keep running, so that the next run and other worktrees can use it.

#### Scenario: Developer presses Ctrl+C mid-suite

- **WHEN** a developer presses Ctrl+C while `mise run test` or `mise run test-change` runs tests
- **THEN** no test process from that run remains
- **AND** no test container that run started remains

#### Scenario: Another worktree's tests are running

- **WHEN** a developer interrupts `mise run test-change` while a test package of another worktree runs with its own test container
- **THEN** that container is still running afterwards and that package's tests continue

#### Scenario: Shared fixtures survive the interrupt

- **WHEN** a developer interrupts `mise run test-change`
- **THEN** the shared test PostgreSQL server is still running afterwards

### Requirement: Test runs take turns

Only one `mise run test` or `mise run test-change` SHALL run tests on a machine at a time. Such a command started while another one runs, in any worktree, SHALL print `waiting for another test run to finish` and SHALL start its tests after the other run ends. A single test package run SHALL NOT wait for either command, apart from the wait for the reset of the shared test PostgreSQL servers. Neither command SHALL remove containers or networks of a package run, reset or delete an internal database a package run uses, or reset the shared test PostgreSQL servers before the package run's process has ended. Tests of old code that change shared objects on those servers can still affect each other across runs, as they can today; this requirement does not cover that.

#### Scenario: Two worktrees start a full run

- **WHEN** a developer in one worktree starts `mise run test` while `mise run test` or `mise run test-change` of another worktree is running
- **THEN** the second run prints `waiting for another test run to finish`
- **AND** its tests start after the first run ends

#### Scenario: Two worktrees start the routine run

- **WHEN** an agent in one worktree starts `mise run test-change` while `mise run test` or `mise run test-change` of another worktree is running
- **THEN** the second run prints `waiting for another test run to finish`
- **AND** its tests start after the first run ends

#### Scenario: A package run during a full run

- **WHEN** an agent runs one backend test package while `mise run test` of another worktree is running its tests
- **THEN** the package run starts its tests without waiting for that run to end
- **AND** its containers, networks and internal database stay in place until the package run ends

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

### Requirement: Slow backend packages do not hold back the others

`mise run test` and `mise run test-change` SHALL run the backend packages named in a repository list of slow packages separately from all other selected backend packages, in concurrent test processes, so that fast packages start without waiting for slow ones. The list SHALL be ordered by measured time, longest first, and the slow process SHALL receive the packages in that order, with a fixed package limit lower than that of the other process. Both limits SHALL be chosen by measurement. The run SHALL take slow packages only from those it selects anyway, so an entry the run does not select SHALL neither fail the run nor add a package. A script test SHALL fail and name an entry that no longer exists or that `mise run test` does not select. A selected package missing from the list SHALL run with the other packages. In `mise run test` and in CI, every test of every selected package SHALL run exactly once, including when CI runs a package as several shards. In `mise run test-change`, a test that two overlapping selectors of the plan select runs once for each of them, so that each selector's filter is checked on its own.

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

#### Scenario: A plan selects one slow and one fast package

- **WHEN** a test plan selects one package from the list and one package outside it
- **THEN** `mise run test-change` runs the first with the slow process's limits and the second with the other process's limits

#### Scenario: A package runs as local shards

- **WHEN** CI runs a package as several shards and one of its tests has no measured weight
- **THEN** that test still runs, in exactly one shard, and the package is not also run whole

### Requirement: The full run finishes within twenty minutes on the reference machine

The reference developer machine has at least 10 performance cores and 32 GB of memory free for tests, out of about 64 GB in total. On that machine, `mise run test` started while no other test run is active, with the test images already present and Docker's build cache holding an earlier build of the image, SHALL finish within twenty minutes, with about ten minutes as the target. The limit leaves room above the measured runs of about fourteen minutes, so a slightly slower run still meets it. Running more work at once than the machine has processor cores is acceptable; the memory and disk space free for tests are the bounds. No suite SHALL limit how many of its tests run at once with a constant of its own below what the runner gives it; the runner SHALL set each limit for the machine by measurement and state the reason beside it. The slow backend group's package limit SHALL stay below the other backend group's, as the requirement on slow backend packages says. The browser UI suite SHALL run locally with more than the two workers each CI shard uses, with a count derived from the host's processors, while CI SHALL keep exactly two workers per shard. When the run ends, after success or failure, it SHALL print how long the image build, each suite and the whole run took; when the developer named an image, the summary SHALL mark the build as skipped.

#### Scenario: The full run on the reference machine

- **WHEN** a developer runs `mise run test` on the reference machine with no other test run active, the test images present and a warm build cache
- **THEN** it finishes within twenty minutes

#### Scenario: Durations are reported

- **WHEN** `mise run test` ends, whether it passed or failed, without a named image
- **THEN** its output lists the duration of the image build, of each suite and of the whole run

#### Scenario: Durations with a named image

- **WHEN** `mise run test` ends after the developer named an image through `DATABASUS_IMAGE`
- **THEN** its output marks the image build as skipped and lists the duration of each suite and of the whole run

#### Scenario: The browser UI suite uses the machine locally

- **WHEN** the browser UI suite runs locally on a host with more than four logical processors
- **THEN** it runs with more than two workers, a count derived from the host's processors

#### Scenario: CI keeps two browser UI workers

- **WHEN** a CI shard of the browser UI suite runs
- **THEN** it runs with exactly two workers

### Requirement: Suites of the full run do not disturb each other

Every suite of `mise run test` and `mise run test-change` SHALL pass or fail as it does when run alone. No suite SHALL stop, remove or change the containers, networks, volumes, images, databases or temporary files of another suite. Docker networks that the upgrade test and browser end-to-end runs create themselves SHALL have subnets inside the test network range that the testing documentation names, as the backend tests' networks do, so that Docker's default address pools stay free for the verification agents, which cannot choose their range. A test that fails only because other suites load the machine SHALL be fixed in that test, by waiting for the condition it depends on or by a time limit that matches the operation, and SHALL NOT be taken out of the concurrent schedule or hidden by a retry.

#### Scenario: The verification agent finds free networks

- **WHEN** browser end-to-end runs and the upgrade test hold their own networks while a verification agent of the same full run creates its networks
- **THEN** the agent gets the networks it asks for from Docker's default address pools

#### Scenario: Another worktree runs the upgrade test directly

- **WHEN** another worktree runs the upgrade command directly while `mise run test` runs its suites against the image it built
- **THEN** the full run's image is neither replaced nor removed, and its suites keep testing it

#### Scenario: A suite fails only under load

- **WHEN** a test passes alone but fails while the other suites of `mise run test` run
- **THEN** the defect is fixed in that test, and the suite stays in the concurrent schedule

### Requirement: The CloudNativePG suite owns its Kubernetes clusters

Each run of the CloudNativePG suite SHALL create its own disposable Kubernetes clusters, with names, networks, volumes, credentials and files no other run uses. Their networks SHALL take subnets from the test network range that the testing documentation names. The suite SHALL NOT read or change the developer's Kubernetes configuration or selected context, and SHALL NOT connect to a Kubernetes cluster it did not create.

After success, a failure during startup or cancellation, the run SHALL remove the containers, networks, volumes and temporary files of its Kubernetes clusters, and only those. When the test process of `mise run test` or `mise run test-change` is killed, the surviving script SHALL remove the run's labelled containers, networks and volumes when it exits. A killed direct package run carries no run label, so it leaves its resources and its temporary directories behind. The testing documentation SHALL name the fixture label that finds those resources and SHALL say that this label also matches resources of runs still active in other worktrees.

The suite needs a Docker daemon that runs privileged containers and makes container addresses reachable from the host, which is rootful Docker on Linux. A missing prerequisite, such as an unavailable Docker daemon, a refused privileged container, an unreachable container address or an image that cannot be obtained, SHALL fail the suite with a message that names it instead of skipping scenarios.

#### Scenario: Two worktrees run the suite at once

- **WHEN** two worktrees on one machine run the CloudNativePG suite at the same time
- **THEN** each run uses only its own Kubernetes clusters and both pass as they do alone
- **AND** neither run's cleanup removes a resource of the other

#### Scenario: The developer's Kubernetes configuration is left alone

- **WHEN** a developer with a configured Kubernetes context runs the suite
- **THEN** their Kubernetes configuration file and selected context are unchanged after the run
- **AND** no scenario connects to a Kubernetes cluster from that configuration

#### Scenario: Startup fails part way

- **WHEN** a Kubernetes cluster or the CloudNativePG operator fails to start after some of the run's resources were created
- **THEN** the suite fails with the cause and diagnostics
- **AND** every resource the run created is removed

#### Scenario: A scenario is cancelled

- **WHEN** a run's context is cancelled while scenarios hold their CloudNativePG clusters
- **THEN** the run's Kubernetes clusters, networks, volumes and temporary files are removed

#### Scenario: The test process of a full run is killed

- **WHEN** the test process of `mise run test` is killed while CloudNativePG scenarios run
- **THEN** after the script exits, no container, network or volume with that run's label remains

#### Scenario: A prerequisite is missing

- **WHEN** Docker is unavailable, a privileged container is refused, the container address of a Kubernetes cluster is unreachable from the host or a pinned image cannot be obtained
- **THEN** the suite fails with a message naming the missing prerequisite
- **AND** no scenario is reported as skipped or passed

### Requirement: A hung wait fails its test in minutes

One `testwait` wait for a condition in a backend test SHALL end after two minutes, unless the test states its own limit because it exists to observe a longer period of real time. Container start-up waits, including those of the upgrade test, keep their own limits. A wait that ends this way SHALL fail its test and name what it waited for and the last state. `mise run test` and `mise run test-change` SHALL give each package a timeout of 15 minutes, and 30 minutes to the slow packages, which share the Docker daemon with every other suite of the run; a local upgrade test SHALL give each package 15 minutes. In CI the upgrade test SHALL keep 30 minutes, because it runs with the race detector.

#### Scenario: A condition never holds

- **WHEN** a test waits for a condition that never holds
- **THEN** the test fails within two minutes and names what it waited for and the last state

#### Scenario: A test that observes real time

- **WHEN** a test states its own limit for a wait
- **THEN** the wait ends at that limit

## ADDED Requirements

### Requirement: Fixed suites keep their own commands

The fixed suites `mise run agent:test`, `mise run agent:e2e`, `mise run agent:e2e-podman`, `mise run backend:test-upgrade` and `mise run test-filesystem` SHALL remain, along with the selectors CI uses. The browser end-to-end suite SHALL keep three hidden commands: `mise run e2e:browser`, `mise run e2e:unit` and `mise run e2e:lifecycle`. Neither the fixed suites nor these commands SHALL take the machine-wide lock. `mise run e2e` SHALL NOT exist.

#### Scenario: The removed browser command

- **WHEN** a developer runs `mise run e2e`
- **THEN** mise reports that the task does not exist

#### Scenario: A fixed suite runs on its own

- **WHEN** a developer runs `mise run agent:e2e`
- **THEN** the verification agent's Docker scenarios run without taking the machine-wide lock

### Requirement: No command takes packages as arguments

The repository SHALL NOT provide a command that takes backend packages, test names or shard names as arguments and runs their tests. Tests are selected only through a change's test plan, the Go test command or the frontend test runner run directly.

#### Scenario: A removed selection command

- **WHEN** a developer runs `mise run backend:test-package ./internal/lib/...`
- **THEN** mise reports that the task does not exist

#### Scenario: The plan command takes a change name

- **WHEN** a developer runs `mise run test-change ./internal/lib/...`
- **THEN** the command finds no change of that name, runs no test and exits with a non-zero status

## REMOVED Requirements

### Requirement: One command runs the routine backend suite

**Reason**: The routine run that selected every backend package outside the end-to-end ones is deleted. `mise run test` now names the full run, and an agent checks a change with its own test plan.

**Migration**: The guarantees about bundled clients, the shared test PostgreSQL server, migrated internal databases and new package directories move to "The full command runs the Docker test suites". An agent runs `mise run test-change <change>`; a person runs `mise run test`.

### Requirement: One routine test command checks a change

**Reason**: Checking a change moves from one fixed command to the change's test plan, described in the `change-test-plans` capability. An agent no longer runs the whole test set.

**Migration**: The agent's check is "An agent checks its work with selected tests only" in `change-test-plans`. The fixed suites and the ban on package-argument commands move to "Fixed suites keep their own commands" and "No command takes packages as arguments". The commands owed by changes to end-to-end code move to "Changes to the browser end-to-end suite select its suites", "Changes to backend end-to-end code select their tests" and "Changes to repository tooling select its suites" in `change-test-plans`.

### Requirement: The routine run includes the frontend tests

**Reason**: The routine run is deleted. The frontend unit and screen tests run in `mise run test` and through frontend selectors of a test plan.

**Migration**: Installing frontend dependencies from the lockfile and running the frontend tests beside the backend packages are part of "The full command runs the Docker test suites".
