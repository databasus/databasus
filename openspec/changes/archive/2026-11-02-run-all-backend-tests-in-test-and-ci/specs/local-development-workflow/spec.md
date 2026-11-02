## MODIFIED Requirements

### Requirement: One command runs the full backend suite

The repository SHALL provide `mise run test`. It SHALL assume a Fedora host and, when `libedit` is installed on that host, SHALL make the bundled database clients startable without any further manual step. When `libedit` is missing, it SHALL stop before running tests and name the package to install. It SHALL start the shared test fixtures, pull the database engine and object storage images the backend tests reference, reset the shared database servers as the `backend-test-isolation` capability allows, and then run every test package of the backend module except the real-API browser packages and the upgrade test with its comparison helper, each test that asks for a migrated internal database getting one as that capability describes. A migration test that asks for an empty internal database SHALL get one with no migration applied and SHALL apply the migrations it needs itself, and packages that need no database SHALL open none. A package in a new directory of the module SHALL join this run without any change to the selection. After the backend packages pass, it SHALL run the upgrade test as the requirement "The full run includes the upgrade test" describes. It SHALL NOT prepare internal databases ahead of the tests. It SHALL NOT run the Docker filesystem suite. Its exit status SHALL be non-zero when any step or any test fails.

#### Scenario: Developer runs the suite on Fedora

- **WHEN** a developer runs `mise run test` on a Fedora host with a `.env` file
- **THEN** every selected backend test package runs, and each test that asks for a migrated internal database gets one with every migration of the checkout applied
- **AND** a migration test that asks for an empty internal database gets one with no migration applied, and packages that need no database open none
- **AND** tests that start a bundled database client do not fail on a missing shared library

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

#### Scenario: The upgrade and browser packages stay out of the package step

- **WHEN** `mise run test` selects its backend packages
- **THEN** it selects neither the real-API browser packages nor the upgrade test and its comparison helper
