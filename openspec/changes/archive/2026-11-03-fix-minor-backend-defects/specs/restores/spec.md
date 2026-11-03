## ADDED Requirements

### Requirement: A failed archive download does not report a partially restored target

When a PostgreSQL restore that needs the archive as a file fails while it reads the backup from storage, the restore tool has not started and has written no backup data to the target. The restore's fail message SHALL say that reading the backup failed before the restore tool started, SHALL contain the storage error, and SHALL NOT say that the target was partially restored. A restore that streams the backup directly into the tool and loses the stream SHALL keep reporting that the target is left partially restored.

#### Scenario: Storage fails during a parallel restore's download

- **WHEN** a PostgreSQL restore with parallel jobs loses its storage connection while downloading the archive into a file
- **THEN** the restore is recorded as failed, its fail message contains the storage error and says the restore tool did not start
- **AND** its fail message does not say the target is partially restored

#### Scenario: Storage fails during a streamed restore

- **WHEN** a PostgreSQL restore that streams the backup into the restore tool loses its storage connection
- **THEN** the restore is recorded as failed and its fail message says the target is left partially restored

### Requirement: A failed version comparison is not reported as a too-old target

When a MySQL, MariaDB or MongoDB restore cannot compare the backup's server version with the target's, starting it SHALL answer `500` with `code` `internal`, close the connection to the target and create no restore. The answer SHALL NOT say that the target's version is lower than the backup's.

#### Scenario: The recorded version of the backup's database cannot be read

- **WHEN** a member starts a restore of a MySQL, MariaDB or MongoDB backup whose database has an unreadable recorded server version, into a reachable target of that engine
- **THEN** the response is `500` with `code` `internal`, no restore is created, and the response does not say the target is too old

#### Scenario: The target is older than the backup

- **WHEN** a member starts a restore of a MySQL, MariaDB or MongoDB backup into an older target of that engine
- **THEN** the response is still `400` with `code` `invalid_request` and names both versions
