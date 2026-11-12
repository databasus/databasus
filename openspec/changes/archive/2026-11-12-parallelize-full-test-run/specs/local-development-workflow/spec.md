## MODIFIED Requirements

### Requirement: The full command runs the Docker test suites

`mise run test-full` SHALL run the routine backend and frontend suites, adding all backup and verification agent end-to-end packages, together with script tests, review hook tests, browser e2e unit tests, frontend browser UI, the upgrade test, browser end-to-end scenarios, lifecycle checks and lifecycle interruption checks. Each suite SHALL start as soon as what it needs is ready and SHALL NOT wait for an unrelated suite to finish. Suites that run the application image SHALL start once the run has that image. The crashed-run probes and then the lifecycle interruption checks SHALL run while no other browser end-to-end run of the same command is active: neither SHALL start while another browser end-to-end run of the command is active, and the command SHALL start no other browser end-to-end run until both have ended. A failed suite SHALL NOT stop or prevent any other suite: every suite that can start SHALL run to its end, and the lifecycle interruption checks SHALL run even when the crashed-run probes fail. Only a failed image build SHALL keep suites from starting, namely those that run the image. The command SHALL then exit with a non-zero status and name each failed suite and each suite that could not start. When the developer interrupts the command, every suite's process SHALL exit, and no compose project, container, network or volume that the run's suites started SHALL remain, nor the image the run built; the shared test PostgreSQL server SHALL keep running. It SHALL take the same machine-wide lock as `mise run test`, and SHALL NOT update browser visual references. CI SHALL run its individual suites rather than `mise run test-full`. Podman and filesystem tests SHALL remain separate because they require additional host configuration.

#### Scenario: All ordinary Docker checks pass

- **WHEN** every suite passes during `mise run test-full`
- **THEN** every included suite runs once and the command succeeds
- **AND** neither the Podman subset nor the filesystem suite runs

#### Scenario: Suites start without waiting for the backend packages

- **WHEN** `mise run test-full` starts
- **THEN** the image build, the browser UI suite, the script tests, the review hook tests, the browser e2e unit tests and the crashed-run probes start while the backend packages are still running

#### Scenario: An additional suite fails

- **WHEN** one suite fails during `mise run test-full`
- **THEN** every other suite still runs to its end
- **AND** the command exits with a non-zero status and its summary names the failed suite

#### Scenario: The image cannot be built

- **WHEN** the application image fails to build during `mise run test-full`
- **THEN** the suites that run that image do not start, and the other suites run to their end
- **AND** the command exits with a non-zero status and names the failed build and each suite that did not start

#### Scenario: Crashed-run probes and interruption checks run alone

- **WHEN** `mise run test-full` runs the crashed-run probes and then the lifecycle interruption checks
- **THEN** no other browser end-to-end run of that command is active when the first of them starts
- **AND** the command starts no browser end-to-end run until both have ended

#### Scenario: The crashed-run probes fail

- **WHEN** a crashed-run probe fails during `mise run test-full`
- **THEN** the lifecycle interruption checks still run, and the command exits with a non-zero status

#### Scenario: Interrupting the concurrent schedule

- **WHEN** a developer presses Ctrl+C while the backend packages, the image build or the upgrade test, the browser UI suite, the browser end-to-end scenarios and the lifecycle checks run at the same time
- **THEN** no process of any suite remains, and no compose project, container, network or volume of the run's suites remains, nor the image the run built
- **AND** the shared test PostgreSQL server is still running

### Requirement: The full run includes the upgrade test

`mise run test-full` SHALL run the upgrade test as soon as the run has the application image, while other suites may still be running. When the developer names a Databasus image through `DATABASUS_IMAGE`, the upgrade test and the browser end-to-end scenarios SHALL use it and the run SHALL build no application image for them. Otherwise `mise run test-full` SHALL build the image from the checkout once, at its start, under a tag that only the full command uses, and both suites SHALL use that image. Because full runs take turns, that tag cannot collide with another full run, and a later full run replaces an image that a killed run left under it. The run SHALL remove the image it built when it ends after success, a failure or an interruption. A failed upgrade test SHALL make `mise run test-full` exit with a non-zero status. A failed backend package or frontend test SHALL NOT prevent the image build or the upgrade test. A single test package started directly with the Go test command SHALL NOT build an image or run the upgrade test. When the developer interrupts `mise run test-full` during the upgrade test, the containers, the network and the volume of the upgrade test SHALL be removed, and the shared test PostgreSQL server SHALL keep running.

#### Scenario: Developer runs the full suite without an image

- **WHEN** a developer runs `mise run test-full` without `DATABASUS_IMAGE`
- **THEN** the image is built once from the checkout, and the upgrade test and the browser end-to-end scenarios run against it
- **AND** the image the run built does not remain after the run ends

#### Scenario: Developer names an image

- **WHEN** a developer runs `mise run test-full` with `DATABASUS_IMAGE` set to an image they built
- **THEN** the upgrade test and the browser end-to-end scenarios run against the named image, the run builds no application image for them, and the named image remains after the run

#### Scenario: The upgrade test fails

- **WHEN** the upgrade test fails during `mise run test-full`
- **THEN** the command exits with a non-zero status

#### Scenario: The frontend tests fail

- **WHEN** a frontend test fails during `mise run test-full`
- **THEN** the image is still built and the upgrade test still runs to its end, and the command exits with a non-zero status

#### Scenario: A backend package fails

- **WHEN** a backend package fails during `mise run test-full`
- **THEN** the upgrade test still runs to its end, and the command exits with a non-zero status

#### Scenario: A single package run

- **WHEN** an agent runs one backend package directly with the Go test command
- **THEN** no Databasus image is built and no upgrade test container starts

#### Scenario: Interrupt during the upgrade test

- **WHEN** a developer presses Ctrl+C while the upgrade test runs during `mise run test-full`
- **THEN** none of the upgrade test's containers, its network or its volume remains, and the image the run built is removed
- **AND** the shared test PostgreSQL server is still running

### Requirement: The routine run includes the frontend tests

`mise run test` SHALL install the frontend dependencies from the lockfile, as `mise run run` does, and SHALL run the frontend unit and screen tests at the same time as the backend packages. It SHALL wait for every test process, and a failure in one SHALL NOT stop the others. It SHALL exit with a non-zero status when any fails, and its output SHALL name each failed process and test. `mise run test` SHALL NOT start the upgrade test. The browser suites SHALL stay out of `mise run test`.

#### Scenario: A fresh worktree

- **WHEN** a developer runs `mise run test` in a worktree without `frontend/node_modules`
- **THEN** the dependencies are installed from the lockfile and the frontend tests run

#### Scenario: A frontend test fails

- **WHEN** a frontend test fails during `mise run test` while backend packages still run
- **THEN** the backend packages finish, the output names the failed frontend test, and the command exits with a non-zero status

#### Scenario: A backend package fails

- **WHEN** a package fails in one backend test process during `mise run test` while the other backend process and the frontend tests still run
- **THEN** they finish, the output names the failed process and test, and the command exits with a non-zero status

#### Scenario: Browser suites stay out

- **WHEN** a developer runs `mise run test`
- **THEN** no browser test runs

#### Scenario: Interrupting every process

- **WHEN** a developer presses Ctrl+C while both backend test processes and the frontend tests run
- **THEN** all of them exit, and no test process or container of this run remains

## ADDED Requirements

### Requirement: The full run finishes within twenty minutes on the reference machine

The reference developer machine has at least 10 performance cores and 32 GB of memory free for tests, out of about 64 GB in total. On that machine, `mise run test-full` started while no other test run is active, with the test images already present and Docker's build cache holding an earlier build of the image, SHALL finish within twenty minutes, with about ten minutes as the target. The limit leaves room above the measured runs of about fourteen minutes, so a slightly slower run still meets it. Running more work at once than the machine has processor cores is acceptable; the memory and disk space free for tests are the bounds. No suite SHALL limit how many of its tests run at once with a constant of its own below what the runner gives it; the runner SHALL set each limit for the machine by measurement and state the reason beside it. The slow backend group's package limit SHALL stay below the other backend group's, as the requirement on slow backend packages says. The browser UI suite SHALL run locally with more than the two workers each CI shard uses, with a count derived from the host's processors, while CI SHALL keep exactly two workers per shard. When the run ends, after success or failure, it SHALL print how long the image build, each suite and the whole run took; when the developer named an image, the summary SHALL mark the build as skipped.

#### Scenario: The full run on the reference machine

- **WHEN** a developer runs `mise run test-full` on the reference machine with no other test run active, the test images present and a warm build cache
- **THEN** it finishes within twenty minutes

#### Scenario: Durations are reported

- **WHEN** `mise run test-full` ends, whether it passed or failed, without a named image
- **THEN** its output lists the duration of the image build, of each suite and of the whole run

#### Scenario: Durations with a named image

- **WHEN** `mise run test-full` ends after the developer named an image through `DATABASUS_IMAGE`
- **THEN** its output marks the image build as skipped and lists the duration of each suite and of the whole run

#### Scenario: The browser UI suite uses the machine locally

- **WHEN** the browser UI suite runs locally on a host with more than four logical processors
- **THEN** it runs with more than two workers, a count derived from the host's processors

#### Scenario: CI keeps two browser UI workers

- **WHEN** a CI shard of the browser UI suite runs
- **THEN** it runs with exactly two workers

### Requirement: Suites of the full run do not disturb each other

Every suite of `mise run test-full` SHALL pass or fail as it does when run alone. No suite SHALL stop, remove or change the containers, networks, volumes, images, databases or temporary files of another suite. Docker networks that the upgrade test and browser end-to-end runs create themselves SHALL have subnets inside the test network range that the testing documentation names, as the backend tests' networks do, so that Docker's default address pools stay free for the verification agents, which cannot choose their range. A test that fails only because other suites load the machine SHALL be fixed in that test, by waiting for the condition it depends on or by a time limit that matches the operation, and SHALL NOT be taken out of the concurrent schedule or hidden by a retry.

#### Scenario: The verification agent finds free networks

- **WHEN** browser end-to-end runs and the upgrade test hold their own networks while a verification agent of the same full run creates its networks
- **THEN** the agent gets the networks it asks for from Docker's default address pools

#### Scenario: Another worktree runs the upgrade test directly

- **WHEN** another worktree runs the upgrade command directly while `mise run test-full` runs its suites against the image it built
- **THEN** the full run's image is neither replaced nor removed, and its suites keep testing it

#### Scenario: A suite fails only under load

- **WHEN** a test passes alone but fails while the other suites of `mise run test-full` run
- **THEN** the defect is fixed in that test, and the suite stays in the concurrent schedule
