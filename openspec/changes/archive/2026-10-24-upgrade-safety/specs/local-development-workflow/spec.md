## MODIFIED Requirements

### Requirement: One command runs the full backend suite

The repository SHALL provide `just test` at its root. It SHALL assume a Fedora host and, when `libedit` is installed on that host, SHALL make the bundled database clients startable without any further manual step. When `libedit` is missing, it SHALL stop before running tests and name the package to install. It SHALL start the shared test fixtures, pull the database engine and object storage images the backend tests reference, reset the shared database servers as the `backend-test-isolation` capability allows, and then run every backend test package, each test getting its internal database as that capability describes. After the backend packages pass, it SHALL run the upgrade test as the requirement "The full run includes the upgrade test" describes. It SHALL NOT prepare internal databases ahead of the tests. It SHALL NOT run the Docker filesystem suite. Its exit status SHALL be non-zero when any step or any test fails.

#### Scenario: Developer runs the suite on Fedora

- **WHEN** a developer runs `just test` on a Fedora host with a `.env` file
- **THEN** every backend test package runs, each against internal databases with every migration of the checkout applied
- **AND** tests that start a bundled database client do not fail on a missing shared library

#### Scenario: A test fails

- **WHEN** any backend test fails during `just test`
- **THEN** the command exits with a non-zero status

#### Scenario: Filesystem suite is not part of the run

- **WHEN** a developer runs `just test`
- **THEN** no Docker filesystem case runs

## ADDED Requirements

### Requirement: The full run includes the upgrade test

After the backend test packages pass, `just test` SHALL run the upgrade test. When the developer names a Databasus image through `DATABASUS_IMAGE`, the upgrade test SHALL use it. Otherwise `just test` SHALL build the image from the checkout first. A failed upgrade test SHALL make `just test` exit with a non-zero status. When a backend package fails, `just test` SHALL stop without building the image or running the upgrade test. A single package run SHALL NOT build an image or run the upgrade test. When the developer interrupts `just test` during the upgrade test, the containers, the network and the volume of the upgrade test SHALL be removed, and the shared test fixtures SHALL keep running.

#### Scenario: Developer runs the full suite without an image

- **WHEN** a developer runs `just test` without `DATABASUS_IMAGE`
- **THEN** the image is built from the checkout after the backend packages pass, and the upgrade test runs against it

#### Scenario: Developer names an image

- **WHEN** a developer runs `just test` with `DATABASUS_IMAGE` set to an image they built
- **THEN** no image is built and the upgrade test runs against the named image

#### Scenario: The upgrade test fails

- **WHEN** the upgrade test fails during `just test`
- **THEN** the command exits with a non-zero status

#### Scenario: A backend package fails

- **WHEN** a backend package fails during `just test`
- **THEN** no image is built, the upgrade test does not run, and the command exits with a non-zero status

#### Scenario: A single package run

- **WHEN** an agent runs `just test-package` for one backend package
- **THEN** no Databasus image is built and no upgrade test container starts

#### Scenario: Interrupt during the upgrade test

- **WHEN** a developer presses Ctrl+C while the upgrade test runs during `just test`
- **THEN** none of the upgrade test's containers, its network or its volume remains
- **AND** the shared test fixtures are still running
