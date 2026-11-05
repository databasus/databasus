## ADDED Requirements

### Requirement: Removing a backup removes its verifications

Whenever a logical backup is removed, whether a member deletes it, retention drops it, its database's backup config moves to another storage or its database is deleted, its verifications SHALL be removed in the same step, whatever their status, a running one included. When that removal is refused, the backup and all its verifications SHALL stay. Deleting a database SHALL also remove its verification config.

#### Scenario: A member deletes a verified backup

- **WHEN** a member deletes a logical backup that has a completed verification, a running one and a pending one
- **THEN** the response is `204`, and none of the three verifications is listed for the database any more

#### Scenario: Retention drops a verified backup

- **WHEN** a database's retention no longer keeps a backup that has a verification
- **THEN** after the next retention check the backup is not listed, and its verification is no longer listed for the database

#### Scenario: The storage of a verified backup changes

- **WHEN** a member saves a database's backup config with another storage while one of its backups has a verification
- **THEN** the save succeeds, the backup is not listed, and its verification is no longer listed for the database

#### Scenario: A database with verifications is deleted

- **WHEN** a member deletes a database that has a verification config and verified backups
- **THEN** the response is `204`, and no verification or verification config of the database remains

#### Scenario: A refused deletion keeps the verifications

- **WHEN** a member deletes a database while one of its logical backups is in progress, and another backup of it has a verification
- **THEN** the response is `409` with `code` `in_use`, the backup is still listed, the verification is still listed for the database, and the database's verification config is still returned
