## ADDED Requirements

### Requirement: The shared test server gives up durability

The shared PostgreSQL server that holds internal test databases SHALL run without flushing its writes to durable storage, as the engine test containers do. A crash of that server or of the host MAY destroy its contents. Recreating the server with an empty data directory SHALL be enough to recover: the next test SHALL get a migrated database without any separate preparation step. The development database SHALL keep its durability.

#### Scenario: The test server runs without durability

- **WHEN** the shared test server is started from the compose file
- **THEN** `SHOW fsync`, `SHOW synchronous_commit` and `SHOW full_page_writes` each return `off`

#### Scenario: New settings keep the existing databases

- **WHEN** the running server holds a migrated template and `docker compose up -d backend-db-test` recreates it with a changed compose command
- **THEN** the template still exists and `SHOW fsync` returns `off`

#### Scenario: The test server is recreated

- **WHEN** the shared test server is recreated with an empty data directory and a developer runs a backend test package
- **THEN** the test gets a database with every migration of its checkout applied

#### Scenario: The development database is unaffected

- **WHEN** a developer starts the local environment
- **THEN** `SHOW fsync` on the development database returns `on`
