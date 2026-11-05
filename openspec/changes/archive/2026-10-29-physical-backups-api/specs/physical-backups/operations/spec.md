## Purpose

Defines the API operations on physical backups of a PostgreSQL database: listing, starting, cancelling and deleting them, issuing restore tokens, downloading a restore set and getting the recovery script, with the answer each operation gives.

## ADDED Requirements

### Requirement: The physical backup operations are part of the documented API

Every operation on physical backups, restore tokens, restore sets, the recovery script and WAL streams SHALL be described in the documented API with its fields, its statuses and its error codes. Every error SHALL carry a stable `code` and SHALL NOT carry the text of an internal error. The operations that existed before this change under `/api/v1/backups/physical/` SHALL no longer answer under those paths, and a request to one of them SHALL get `404` with `code` `not_found`.

#### Scenario: An internal failure

- **WHEN** a physical operation fails for a reason the client cannot correct
- **THEN** the response is `500` with `code` `internal`, and its body holds no text of the internal error

#### Scenario: An old path

- **WHEN** a client calls the list of physical backups under `/api/v1/backups/physical/`
- **THEN** the response is `404` with `code` `not_found`, and it holds no backup

### Requirement: The physical backup list shows backups and WAL ranges with the storage they use

Listing the physical backups of a database SHALL return its full backups, its incremental backups and its WAL ranges, newest first, each with its type, its status, its timeline, its positions in the WAL, its size and its times, together with the total storage the database's physical backups use. The list SHALL filter by type, by status and by a time before which the rows were created. A page SHALL hold 50 rows unless the request asks for between 1 and 1000. A request for a database that has no physical backups because it is not a physical PostgreSQL database SHALL get `400` with `code` `invalid_request`.

#### Scenario: Listing a database's backups

- **WHEN** a viewer lists the physical backups of a database with one completed full backup and archived WAL after it
- **THEN** the response holds a WAL range row and the full backup row, and the total storage the backups use

#### Scenario: A page size out of range

- **WHEN** a client asks for a page of 1001 rows
- **THEN** the response is `400` with `code` `invalid_request`

### Requirement: A manual start asks the scheduler for a backup

Starting a physical backup SHALL answer `202` with no body. The request SHALL name the kind of backup: a full backup, an incremental backup, or the kind the scheduler would choose. That kind is an incremental backup when the database's backup type includes incremental backups and its newest chain can be extended, and a full backup otherwise. The backup SHALL appear in the list when the scheduler starts it. When an incremental backup is requested and the database has no chain to extend, the request SHALL get `409` with `code` `no_extendable_chain` and SHALL request nothing. A database that is not a physical PostgreSQL database SHALL get `400` with `code` `invalid_request`.

#### Scenario: Starting a full backup

- **WHEN** a member starts a full backup of a physical database with backups turned on
- **THEN** the response is `202`, and a full backup of the database appears in the list in `IN_PROGRESS` or `COMPLETED` after the scheduler's next pass

#### Scenario: The scheduler's choice for a full-only backup type

- **WHEN** a member starts a backup of the scheduler's choice for a physical database whose backup type takes only full backups and whose newest full backup is completed
- **THEN** the response is `202`, and the backup that appears in the list after the scheduler's next pass is a full backup

#### Scenario: An incremental backup without a chain

- **WHEN** a member starts an incremental backup of a physical database that has no completed full backup
- **THEN** the response is `409` with `code` `no_extendable_chain`, and no backup appears in the list after the scheduler's next pass

#### Scenario: A logical database

- **WHEN** a member starts a physical backup of a MySQL database
- **THEN** the response is `400` with `code` `invalid_request`

### Requirement: Cancelling stops a running physical backup

Cancelling a running full or incremental backup SHALL answer `202` with no body, and the backup SHALL then show the status `CANCELED`. A backup that is no longer running SHALL get `409` with `code` `not_running` and SHALL stay as it is. A backup ID that names no full or incremental backup SHALL get `404` with `code` `not_found`.

#### Scenario: Cancelling a running backup

- **WHEN** a member cancels a running full backup
- **THEN** the response is `202`, and the list later shows the backup as `CANCELED`

#### Scenario: A finished backup

- **WHEN** a member cancels a completed full backup
- **THEN** the response is `409` with `code` `not_running`, and the backup is still `COMPLETED`

### Requirement: Deleting removes a backup and the backups built on it

Deleting a full backup SHALL remove it and every incremental backup built on it. Deleting an incremental backup SHALL remove it and every later incremental backup of its chain. A backup that is running SHALL be stopped and then removed. Both SHALL answer `204`. An ID that names no full or incremental backup, the ID of a WAL segment included, SHALL get `404` with `code` `not_found` and SHALL delete nothing. The `physical-backup-wal-ranges` spec holds the scenario for a WAL segment.

#### Scenario: Deleting a full backup with incrementals

- **WHEN** a member deletes a completed full backup that has two incremental backups
- **THEN** the response is `204`, and neither the full backup nor its incremental backups are listed

#### Scenario: Deleting a running backup

- **WHEN** a member deletes a full backup that is running
- **THEN** the response is `204`, the backup stops, and it is no longer listed

### Requirement: A restore token opens one restore set

Issuing a restore token SHALL answer `201` with the token and the time it expires, 15 minutes after it is issued. A token SHALL name either a database with an optional target time, which restores to that time or to the latest restorable moment, or one full or incremental backup, which restores that backup and the backups it is built on without replaying WAL. The token SHALL be refused at issue, and no token issued, when:

- the database has no completed full backup, or the chosen backup is not completed or misses a backup it is built on: `409` with `code` `no_restorable_backup`;
- the target time is earlier than the earliest full backup, or a WAL gap lies between the full backup and the target: `409` with `code` `target_time_unreachable`;
- another download or restore stream of the same user is running: `409` with `code` `already_running`;
- the chosen backup does not exist: `404` with `code` `not_found`;
- the database is not a physical PostgreSQL database: `400` with `code` `invalid_request`.

#### Scenario: Restoring to the latest moment

- **WHEN** a viewer asks for a restore token of a database with a completed full backup and archived WAL after it, without a target time
- **THEN** the response is `201` with a token and its expiry time

#### Scenario: A target time behind a WAL gap

- **WHEN** a viewer asks for a restore token to a time that lies after a gap in the archived WAL
- **THEN** the response is `409` with `code` `target_time_unreachable`, and its detail names the latest restorable position

#### Scenario: No full backup yet

- **WHEN** a viewer asks for a restore token of a database that has no completed full backup
- **THEN** the response is `409` with `code` `no_restorable_backup`

#### Scenario: A running download

- **WHEN** a user asks for a restore token while a backup download of the same user runs
- **THEN** the response is `409` with `code` `already_running`

### Requirement: The restore set downloads once with its token

Downloading a restore set SHALL need no session. The restore token SHALL be its credential and SHALL be passed as a query parameter. The download SHALL stream the restore set as a tar archive. A token that is unknown, expired or already used SHALL get `401` with `code` `unauthenticated`. A token SHALL be spent on its first use, even when that use is refused because another download or restore stream of the same user runs, which gets `409` with `code` `already_running`. When the restore set the token names can no longer be built, because a backup it needs was deleted or a WAL gap appeared after the token was issued, the download SHALL get the answer that issuing the token would get now.

#### Scenario: Downloading with a fresh token

- **WHEN** a client downloads the restore set with a token issued a minute ago
- **THEN** the response is `200` with a tar archive that holds the full backup and the WAL up to the target

#### Scenario: Using a token twice

- **WHEN** a client downloads a restore set with a token and then downloads again with the same token
- **THEN** the second response is `401` with `code` `unauthenticated`

#### Scenario: A backup deleted after the token was issued

- **WHEN** the only full backup of a database is deleted after a restore token of the database was issued, and the token is then used
- **THEN** the response is `409` with `code` `no_restorable_backup`

#### Scenario: The chosen backup deleted after the token was issued

- **WHEN** a full or incremental backup is deleted after a restore token naming that backup was issued, and the token is then used
- **THEN** the response is `404` with `code` `not_found`, as a new token request for that backup would receive

### Requirement: The restore set and the recovery script are public but reveal nothing else

The restore set download and the recovery script SHALL be the only physical operations that need no session. The recovery script SHALL be the same for every caller and SHALL hold no address of the instance, no token and no credential. A restore token SHALL NOT appear in the server's request log, which records the path of a request without its query.

#### Scenario: Getting the recovery script

- **WHEN** a client without a session gets the recovery script
- **THEN** the response is `200` with a shell script that takes the restore set URL or path and the output directory as arguments

#### Scenario: The same script for every caller

- **WHEN** a signed-in user issues a restore token and then gets the recovery script, and a client without a session gets it too
- **THEN** both responses have the same body, and it contains neither the instance's address nor the issued token

#### Scenario: Downloading leaves no token in the request log

- **WHEN** a client downloads a restore set with its token
- **THEN** the request log records the path of the download without the query, so the token is not in it

#### Scenario: Listing without a session

- **WHEN** a client without a session lists the physical backups of a database
- **THEN** the response is `401` with `code` `unauthenticated`
