## MODIFIED Requirements

### Requirement: A database is deleted completely or not at all

Deleting a database while one of its logical backups or a restore of one of them is in progress SHALL get `409` with `code` `in_use` and SHALL delete nothing. Otherwise deleting a database SHALL remove, in one step, the database, its settings, its notifier attachments, its backup configs, its healthcheck config and attempts, and the records of its logical backups with their restores and download tokens, of its physical backups and of its WAL. It SHALL ask for the deletion of every stored file those records describe. When any part fails, nothing SHALL be removed. Only after the deletion has succeeded SHALL a running backup of the database be cancelled, its WAL streaming stopped and its replication slots dropped on the source server.

#### Scenario: Deleting a database with backups

- **WHEN** a member deletes a database that has logical backups in a storage
- **THEN** the response is `204`, the database and its backups are no longer listed, and the backup files are removed from the storage

#### Scenario: Deleting a database with physical backups

- **WHEN** a member deletes a physical PostgreSQL database that has, in a storage, a full backup, incremental backups that continue each other, archived WAL segments and a timeline history file, and also has a WAL streamer, an in-flight claim and WAL stream samples
- **THEN** the response is `204`, no record of any of them remains, and the deletion of every file they name is requested

#### Scenario: Deleting a database with healthcheck attempts

- **WHEN** a member deletes a database that has a healthcheck config and recorded attempts
- **THEN** the response is `204`, and no healthcheck config or attempt of the database remains

#### Scenario: A logical backup is running

- **WHEN** a member deletes a database while its logical backup is in progress
- **THEN** the response is `409` with `code` `in_use`, and the database and its backups are still listed

#### Scenario: A restore of the database is running

- **WHEN** a member deletes a database while a restore of one of its backups is running
- **THEN** the response is `409` with `code` `in_use`, and the database and its backups are still listed

#### Scenario: A failure during the deletion

- **WHEN** the deletion of a database's backup records fails
- **THEN** the database and all its backups are still listed, and no backup file is removed

#### Scenario: Deleting a database that streams WAL

- **WHEN** a member deletes a physical PostgreSQL database with WAL streaming
- **THEN** the streaming stops, and the replication slots Databasus created on the source server are dropped after the database is gone

### Requirement: A copy starts with no backup history

Copying a database SHALL create a database in the same workspace named after the original with ` (Copy)` appended, with the same type, settings, secrets and notifiers. The copy SHALL get a copy of the original's verification config when the original has one, and a copy of the original's healthcheck config. The copy of a logical database SHALL also get a copy of the original's backup config when the original has one, and the copy of a physical PostgreSQL database SHALL get no backup config. The copy SHALL show no last backup time, no last backup error and the original's health status, and SHALL have no healthcheck attempts. A copied physical PostgreSQL database SHALL use its own replication slot.

#### Scenario: Copying a logical database

- **WHEN** a member copies a PostgreSQL database that has backups
- **THEN** a new database named `<name> (Copy)` is listed with the same settings and notifiers, no last backup time, and a backup config with the original's settings

#### Scenario: Copying a physical database

- **WHEN** a member copies a physical PostgreSQL database that has a backup config
- **THEN** a new database named `<name> (Copy)` is listed with the same settings, its own replication slot and no backup config

#### Scenario: Copying a database with a changed healthcheck config

- **WHEN** a member copies a database whose healthcheck config has checks switched off and an interval of 10 minutes
- **THEN** the copy's healthcheck config has checks switched off and an interval of 10 minutes, and the copy has no attempts
