## ADDED Requirements

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

- **WHEN** a developer runs `mise run agent:e2e` on a host with Docker and the shared test fixtures
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

## MODIFIED Requirements

### Requirement: Test images can be pulled ahead of a run

The repository SHALL provide `mise run pull-testcontainers`. It SHALL pull the database engine images (PostgreSQL, MySQL, MariaDB and MongoDB) and the object storage image that the backend tests reference, and the images the verification agent restores into during its end-to-end scenarios. It SHALL retry each failed pull a bounded number of times, and SHALL exit with a non-zero status when an image still cannot be pulled. Other test images, such as the FTP, SFTP, SMB, mail and Azure storage emulators, are pulled by the tests on first use.

#### Scenario: Pulling succeeds

- **WHEN** a developer runs `mise run pull-testcontainers` with registry access
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

## REMOVED Requirements

### Requirement: Commits are checked before they are created
**Reason**: It exempts commits that touch only the old end-to-end suite of the verification agent, which this change deletes, and one of its scenarios covers that suite. A delta cannot drop one scenario of a requirement, so the requirement is replaced as a whole.
**Migration**: "A commit is checked before it is created" holds the same checks without the exemption. The agent's end-to-end scenarios belong to the backend module, so a commit that touches them runs the backend checks.
