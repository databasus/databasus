## Purpose

Keeps the status a background process records on a database, such as its last backup time or health, from overwriting settings that a user saved at about the same moment.

## ADDED Requirements

### Requirement: Background status updates change only the status

When the application records the last backup time, the last backup error or the health status of a database, it SHALL change only that status. Recording a successful backup's time SHALL also clear the last backup error, because the error no longer describes the latest backup. The database's name, connection settings, credentials and every other setting SHALL keep the values stored at the moment of the update, including values a user saved after the background process had read the database. A status update for a database that no longer exists SHALL report an error, so the caller can log it.

#### Scenario: User changes the password while a backup finishes

- **WHEN** a user saves a new connection password for a database, and a backup of that database that started earlier finishes right after the save
- **THEN** the database keeps the new password
- **AND** its last backup time is the time of that backup

#### Scenario: Health check runs while the user renames the database

- **WHEN** a user renames a database, and a health check that read the database before the rename records its result after it
- **THEN** the database keeps the new name
- **AND** its health status is the result of that check

#### Scenario: Backup failure is recorded

- **WHEN** a backup of a database fails
- **THEN** the database shows the backup error
- **AND** none of its other settings change

#### Scenario: Successful backup after a failed one

- **WHEN** a database shows a backup error, and its next backup succeeds
- **THEN** the database shows the new last backup time and no backup error
- **AND** none of its other settings change

#### Scenario: Database deleted before the update

- **WHEN** a database is deleted while its backup runs, and the backup then records its time
- **THEN** the update reports an error that the database was not found
