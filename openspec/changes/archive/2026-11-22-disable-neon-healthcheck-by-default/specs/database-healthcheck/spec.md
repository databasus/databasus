## RENAMED Requirements

- FROM: `### Requirement: Every database has a healthcheck config that starts switched on`
- TO: `### Requirement: Every database has a healthcheck config with a default chosen by its host`

## MODIFIED Requirements

### Requirement: Every database has a healthcheck config with a default chosen by its host

Every database SHALL have one healthcheck config. A new database SHALL get the default config when it is created: checks switched on, notifications switched on, a check every minute, three failed checks in a row before the database counts as unavailable, and attempts kept for seven days. A new database whose configured database host, not the SSH tunnel's address, is a Neon host SHALL get the same config with checks switched off, because Neon bills compute only while it is awake and a check every minute keeps it awake. A Neon host is the name `neon.tech` or any name ending in `.neon.tech`, compared without regard to case, surrounding whitespace and a trailing dot. A copy of a database SHALL keep the healthcheck config of the original, whatever its host. A database that existed before the upgrade without a config SHALL have the default config with checks switched on after the upgrade, whatever its host. Reading a config SHALL change nothing.

#### Scenario: A new database

- **WHEN** a member creates a database and reads its healthcheck config
- **THEN** the config has checks switched on, notifications switched on, an interval of 1 minute, 3 failed attempts before unavailable and 7 days of retention

#### Scenario: A new database on Neon

- **WHEN** a member creates a PostgreSQL database with the host `ep-young-dew-123456.us-east-2.aws.neon.tech` and reads its healthcheck config
- **THEN** the config has checks switched off, notifications switched on, an interval of 1 minute, 3 failed attempts before unavailable and 7 days of retention

#### Scenario: A new database on a Neon pooler host

- **WHEN** a member creates a PostgreSQL database with the host `ep-young-dew-123456-pooler.us-east-2.aws.neon.tech`
- **THEN** its healthcheck config has checks switched off

#### Scenario: A host that only contains the Neon name

- **WHEN** a member creates a PostgreSQL database with the host `neon.tech.example.com`
- **THEN** its healthcheck config has checks switched on

#### Scenario: A Neon database is never checked until a member switches the checks on

- **WHEN** a member creates a database on a Neon host, and Databasus looks for due checks
- **THEN** no attempt is added for that database

#### Scenario: A copy of a Neon database whose checks were switched on

- **WHEN** a member switches on the checks of a Neon database and copies the database
- **THEN** the copy's healthcheck config has checks switched on

#### Scenario: A database without a config before the upgrade

- **WHEN** Databasus is upgraded on an instance where a database has no healthcheck config
- **THEN** after the upgrade the config of that database is the default config with checks switched on, whatever its host

## ADDED Requirements

### Requirement: The config tells whether switching the checks on needs a confirmation

Every healthcheck config answer, after a read and after a save, SHALL carry a read-only flag that tells whether switching the checks on needs the user's confirmation. The flag SHALL be true when, and only when, the database's current configured database host, not the SSH tunnel's address, is a Neon host, as the default config requirement defines it. The flag SHALL follow the host stored at the time of the answer, so a database edited to point at Neon answers true from then on. The flag SHALL NOT restrict saving: a save that switches the checks on SHALL succeed for a Neon database as for any other, and a save request SHALL NOT be refused for leaving the flag out.

#### Scenario: Reading the config of a Neon database

- **WHEN** a viewer reads the healthcheck config of a database on a Neon host
- **THEN** the answer's confirmation flag is true

#### Scenario: Reading the config of another database

- **WHEN** a viewer reads the healthcheck config of a database whose host is not a Neon host
- **THEN** the answer's confirmation flag is false

#### Scenario: Switching the checks on for a Neon database through the API

- **WHEN** a member saves the config of a Neon database with checks switched on
- **THEN** the response is `200` with checks switched on and the confirmation flag true, and Databasus checks the database from then on

#### Scenario: A database moved to Neon

- **WHEN** a member changes the host of a database from another server to a Neon host and reads its healthcheck config
- **THEN** the confirmation flag is true, and the checks keep the state they had before the edit
