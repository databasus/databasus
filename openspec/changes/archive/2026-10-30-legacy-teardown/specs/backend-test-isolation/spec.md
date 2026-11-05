## MODIFIED Requirements

### Requirement: Abandoned test databases are removed

Test databases are the copies allocated to individual tests. A test database that has no connections and was created more than an hour ago SHALL be removed by the next test process that asks for a test database, once per process. The migrated templates the copies are made from SHALL NOT be removed. A test database that has an open connection SHALL NOT be removed, whatever its age. Databases on the test server that are not test databases SHALL NOT be removed.

#### Scenario: A killed run left databases behind

- **WHEN** a test run was killed two hours ago and left its test databases, and a new test package starts
- **THEN** those databases are removed

#### Scenario: A long test is still connected

- **WHEN** a test database was created more than an hour ago and a test still holds a connection to it
- **THEN** another test process does not remove it

## ADDED Requirements

### Requirement: Test runs preserve unrelated database sources

Local and CI test runs SHALL NOT reset shared source servers or remove databases, roles, permissions or replication slots belonging to another test, worktree or development session. Source fixtures that a test changes SHALL belong to that test or use uniquely named objects with cleanup limited to those objects. The existing cleanup of abandoned internal test databases remains governed by its separate requirement.

#### Scenario: A local run starts beside a development session
- **WHEN** a local backend run starts while a development source holds a database, a role and a replication slot
- **THEN** preparing the run leaves those objects unchanged and issues no shared-source reset

#### Scenario: CI prepares a backend selection
- **WHEN** CI prepares a backend test selection
- **THEN** it does not run a shared-source reset or wait for fixed source ports that selection does not use

#### Scenario: Seed tests overlap
- **WHEN** two development-seed tests run concurrently using their own backend instances and source fixtures
- **THEN** both can create their configured backups without changing the other test's source or internal data

## REMOVED Requirements

### Requirement: Tests of code outside the target layout get a database per test process

**Reason**: The legacy process-wide database accessor and the packages that require it are removed.

**Migration**: Every test that needs an internal database uses its own migrated database; seed tests build the real test application.

### Requirement: Resetting the shared database servers does not break running tests

**Reason**: No test requires a destructive reset of shared source servers after the remaining seed fixtures are isolated.

**Migration**: Remove the reset command and reset coordination from local and CI runners. Preserve the independent lock that serializes full local runs and the cleanup limited to resources owned by an interrupted run.
