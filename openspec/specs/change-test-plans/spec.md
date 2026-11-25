# change-test-plans Specification

## Purpose
Lets each OpenSpec change be checked by the tests it can affect, chosen and justified while the change is planned, so that a finished change runs only those tests and leaves the whole test set to CI and to a person who asks for it.

## Requirements

### Requirement: Every OpenSpec change carries a test plan

Every OpenSpec change SHALL have a test plan, `tests.md`, written during planning after the task list and before implementation starts. The plan SHALL hold an impact map that says what the change touches and which tests that can break, a list of deliberate exclusions with a reason for each, and a selection of tests. OpenSpec SHALL report the plan as an artifact of the change.

#### Scenario: A new change is planned

- **WHEN** an agent writes the artifacts of a new change
- **THEN** the change gets `tests.md` with an impact map, exclusions and a selection
- **AND** the change's status lists the test plan among its artifacts

#### Scenario: An exclusion has a reason

- **WHEN** the plan leaves out PostgreSQL 12 to 16 for a change whose code path runs only from PostgreSQL 17
- **THEN** the exclusions name those versions and give that reason

### Requirement: The selection names what to run

The selection SHALL be the lines of the first code block fenced as `tests` under the plan's "Selection" heading, one selector per line. A selector SHALL be `backend` with a backend package pattern and an optional `-run` filter, `frontend` with a path under the frontend source tree and an optional `-t` test name, or `suite` with the name of a suite the runner knows. Selectors SHALL carry no flags, limits or timeouts.

#### Scenario: A package with a version filter

- **WHEN** the selection holds `backend ./internal/engines/postgresql/logical -run '^TestSource$/^postgres:1[78]$'`
- **THEN** the run executes the PostgreSQL 17 and 18 subtests of that package and no other version

#### Scenario: A frontend path

- **WHEN** the selection holds `frontend src/features/edit-notifier`
- **THEN** the run executes the frontend tests under that path and no other frontend test

#### Scenario: A named suite

- **WHEN** the selection holds `suite e2e-browser`
- **THEN** the run executes the browser end-to-end scenarios against an application image

### Requirement: The runner knows eight suites

A `suite` selector SHALL name one of `browser-ui`, `e2e-browser`, `e2e-unit`, `lifecycle`, `lifecycle-interruption-checks`, `upgrade`, `scripts` and `review-hooks`. Each name SHALL be the name the run's summary reports, except `lifecycle`, which the summary reports as the crashed-run probes, `lifecycle-crashed-runs`, and the parts of the other lifecycle probes, `lifecycle-<n>-of-<count>`.

#### Scenario: The lifecycle suite in the summary

- **WHEN** a plan run with `suite lifecycle` ends
- **THEN** its summary lists `lifecycle-crashed-runs` and every `lifecycle-<n>-of-<count>` part with their outcomes

### Requirement: The runner decides how selected tests run

The runner SHALL apply the same parallel limits, package timeouts and race-detector policy to a selected backend package as `mise run test` applies to it, including the separate limits for the slow packages. When the selection holds several suites that `mise run test` orders among themselves, such as the crashed-run probes and other browser runs, the plan run SHALL keep that order.

#### Scenario: A slow package is selected

- **WHEN** the selection holds the Patroni physical backup package
- **THEN** it runs with the package and parallel test limits `mise run test` gives the slow packages

#### Scenario: Lifecycle probes beside the browser scenarios

- **WHEN** the selection holds both `suite lifecycle` and `suite e2e-browser`
- **THEN** no other browser run of the plan starts while the crashed-run probes run

### Requirement: A selector that runs no test fails

A backend selector SHALL fail when its pattern matches or loads no package, or when no test at the depth of its `-run` filter passes, fails or skips. A frontend selector SHALL fail when its path does not exist or no test under it passes or fails. The failure SHALL name the selector, so that a mistyped filter cannot pass by running nothing.

#### Scenario: A version filter with the wrong subtest name

- **WHEN** the selection holds `backend ./internal/modules/backups/e2e/postgresql -run '/^postgres:17$'` and that package names its version subtests `PostgreSQL 17`
- **THEN** the run reports the selector as one that ran no test and exits with a non-zero status, even though the package's top-level tests ran

#### Scenario: A pattern that matches no package

- **WHEN** the selection holds `backend ./internal/modules/notifierz/...`
- **THEN** the run reports the selector as one that matched no package

#### Scenario: A frontend path without tests

- **WHEN** the selection holds a frontend path that contains no test file
- **THEN** the run reports the selector as one that ran no test

#### Scenario: A selector whose tests skip

- **WHEN** the selection holds `backend ./internal/modules/backups/e2e/postgresql/neon` and the run has no Neon connection string
- **THEN** the run prints the skip with its reason, names the selector as one whose tests all skipped and does not fail because of it

### Requirement: A plan never selects the whole test set

A test plan SHALL NOT select `mise run test`; the backend patterns `./...`, `./internal/...`, `./internal/modules/...`, `./internal/engines/...` or `./internal/lib/...`; a backend pattern that reaches the upgrade test, such as `./e2e/...`; or the frontend path `src` or its layer `src/pages`, `src/widgets`, `src/features`, `src/entities` or `src/shared`. A change that can break no test SHALL select only `none` and state the reason in its exclusions.

#### Scenario: The whole backend module

- **WHEN** the selection holds `backend ./...` or `backend ./internal/...`
- **THEN** the review hook refuses the plan and names the selector

#### Scenario: A pattern that reaches the upgrade test

- **WHEN** the selection holds `backend ./e2e/...`
- **THEN** the review hook refuses the plan and says that the upgrade test runs as `suite upgrade`

#### Scenario: A narrow recursive pattern

- **WHEN** the selection holds `backend ./cmd/...` or `backend ./internal/modules/notifiers/...`
- **THEN** the review hook accepts the selector

#### Scenario: A whole frontend layer

- **WHEN** the selection holds `frontend src/features`
- **THEN** the review hook refuses the plan and names the selector

#### Scenario: A documentation change

- **WHEN** a change edits only documentation
- **THEN** its selection holds `none` and its exclusions say that no code or test changes

### Requirement: Engine versions are selected by behavior

When a change touches code that behaves differently across engine versions, the plan SHALL select the versions where the behavior under test differs. When no such difference exists, it SHALL select the oldest and the newest supported version of each affected engine.

#### Scenario: A version-independent change to the PostgreSQL dump

- **WHEN** a change alters how the PostgreSQL dump names its output and nothing in that path depends on the server version
- **THEN** the plan selects PostgreSQL 12 and 18 and excludes the versions in between with that reason

### Requirement: Changes to the browser end-to-end suite select its suites

A change that adds or changes browser end-to-end scenarios SHALL select `suite e2e-browser`. A change to that suite's other code SHALL select `suite e2e-unit`. When it changes how a run starts or cleans up, it SHALL also select `suite lifecycle`, and when it changes the command that drives the lifecycle probes, `suite lifecycle-interruption-checks` as well.

#### Scenario: A changed browser scenario

- **WHEN** a change edits a scenario of the browser end-to-end suite
- **THEN** its selection holds `suite e2e-browser`

#### Scenario: A changed lifecycle command

- **WHEN** a change edits the command that drives the lifecycle probes
- **THEN** its selection holds `suite e2e-unit`, `suite lifecycle` and `suite lifecycle-interruption-checks`

### Requirement: Changes to backend end-to-end code select their tests

A change to the verification agent's code or its end-to-end tests SHALL select `backend ./internal/agent/e2e/...`. A change to tests under `internal/modules/backups/e2e` or `internal/modules/instancebackups/e2e` SHALL select the changed packages there. A change to the upgrade test SHALL select `suite upgrade`, and a change to the `Dockerfile` SHALL select `suite upgrade` and `suite e2e-browser`.

#### Scenario: A changed verification agent

- **WHEN** a change edits the verification agent's restore code
- **THEN** its selection holds `backend ./internal/agent/e2e/...`

#### Scenario: A changed Dockerfile

- **WHEN** a change edits the `Dockerfile`
- **THEN** its selection holds `suite upgrade` and `suite e2e-browser`

### Requirement: Changes to repository tooling select its suites

A change to the review hooks SHALL select `suite review-hooks`. A change to the repository's Python scripts or their tests SHALL select `suite scripts`.

#### Scenario: A changed review hook

- **WHEN** a change edits the review hooks
- **THEN** its selection holds `suite review-hooks`

#### Scenario: A changed script

- **WHEN** a change edits the test runner's script tests
- **THEN** its selection holds `suite scripts`

### Requirement: The plan follows the implementation

While implementing a change, the agent SHALL update the test plan when it finds an affected area the plan missed or a selected area the change no longer affects. The plan at the end of implementation SHALL match the diff.

#### Scenario: A missed module

- **WHEN** during implementation the agent changes a backups service that the impact map does not mention
- **THEN** the agent adds the area to the impact map and its tests to the selection before the final run

### Requirement: One command runs a change's selection

`mise run test-change <change>` SHALL run the selection of the named change's test plan on the schedule of `mise run test`: each suite starts as soon as that schedule allows, and a failed suite SHALL NOT stop another. The command SHALL print each suite's outcome and duration, name each failed test and each selector that ran no test, copy its output to a new file under `temp/` and exit with a non-zero status on any failure.

#### Scenario: Two selectors, one fails

- **WHEN** a plan selects a backend package and a frontend path, and a frontend test fails
- **THEN** the backend package still runs to its end
- **AND** the summary names the failed suite and the command exits with a non-zero status

#### Scenario: A plan that selects nothing

- **WHEN** the selection holds only `none`
- **THEN** the command runs no test, says that the plan selects no tests and exits with status zero

#### Scenario: An unknown change

- **WHEN** a developer runs `mise run test-change` with a change name that has no test plan
- **THEN** the command runs no test, names the missing plan and exits with a non-zero status

### Requirement: A plan run takes turns with other runs

`mise run test-change` SHALL take the same machine-wide lock as `mise run test`. Started while another of these runs is active, in any worktree, it SHALL print `waiting for another test run to finish` and start its tests after that run ends.

#### Scenario: Two agents finish at once

- **WHEN** an agent in one worktree starts `mise run test-change` while another worktree's `mise run test-change` runs
- **THEN** the second run prints `waiting for another test run to finish`
- **AND** its tests start after the first run ends

### Requirement: Suites that run the application image share one build

When the selection holds suites that run the application image, the plan run SHALL build the image from the checkout once and run those suites against it, or use the image named by `DATABASUS_IMAGE` without building. It SHALL remove an image it built when it ends. A failed build SHALL keep only those suites from starting.

#### Scenario: Two image suites

- **WHEN** the selection holds `suite upgrade` and `suite e2e-browser` and `DATABASUS_IMAGE` is unset
- **THEN** the image is built once and both suites run against it
- **AND** the image does not remain after the run

### Requirement: Interrupting a plan run removes what it started

When a developer interrupts `mise run test-change`, every process of the run SHALL exit and every container, network, volume and image the run started SHALL be removed. Resources of other runs and the shared test PostgreSQL server SHALL remain.

#### Scenario: Ctrl+C during a plan run

- **WHEN** a developer presses Ctrl+C while a plan run's backend package and browser scenarios run
- **THEN** no process, container or network of that run remains
- **AND** the shared test PostgreSQL server is still running

### Requirement: An agent checks its work with selected tests only

An agent SHALL check a finished change with `mise run test-change` and SHALL NOT run `mise run test`, even when asked to check broadly. While fixing a failing test, it MAY repeat that test directly with the Go test command or the frontend test runner; such a repeat SHALL take no lock.

#### Scenario: Finishing a change

- **WHEN** an agent finishes implementing a change
- **THEN** it runs `mise run test-change` for that change and reports the command and its result
- **AND** it does not run `mise run test`

#### Scenario: Repeating one failing test

- **WHEN** an agent fixes a test that the plan run reported
- **THEN** it can repeat that test with the Go test command while another worktree's run holds the lock
