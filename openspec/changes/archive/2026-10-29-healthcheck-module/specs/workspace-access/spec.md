## ADDED Requirements

### Requirement: Healthchecks follow the workspace roles

The healthcheck operations on a workspace's databases SHALL apply the workspace roles:

- a viewer MAY read the healthcheck config of a database and list its attempts;
- a member MAY also save the healthcheck config;
- an instance admin MAY do all of these in every workspace.

A viewer who saves a healthcheck config SHALL get `403` with `code` `permission_denied`, and nothing SHALL change. A user who is neither a member of the database's workspace nor an instance admin SHALL get `404` with `code` `not_found` for every one of these operations. That answer SHALL be the same as for a database that does not exist, and SHALL reveal nothing of the database.

#### Scenario: Viewer reads healthchecks

- **WHEN** a viewer of a workspace reads the healthcheck config of one of its databases and lists its attempts
- **THEN** each request succeeds

#### Scenario: Viewer saves a healthcheck config

- **WHEN** a viewer of a workspace saves the healthcheck config of one of its databases
- **THEN** the response is `403` with `code` `permission_denied`, and the config is unchanged

#### Scenario: Another team's healthchecks

- **WHEN** a user who is not a member of a workspace reads and saves the healthcheck config of one of its databases and lists its attempts
- **THEN** each response is `404` with `code` `not_found`, and the config is unchanged

#### Scenario: Same answer as a missing database

- **WHEN** a user who is not a member of a workspace reads the healthcheck config of one of its databases and of a database ID that does not exist
- **THEN** both responses are `404` with `code` `not_found` and have the same body

#### Scenario: Instance admin outside the workspace

- **WHEN** an instance admin who is not a member of a workspace saves the healthcheck config of one of its databases
- **THEN** the response is `200`
