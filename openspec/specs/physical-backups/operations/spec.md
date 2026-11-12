# Physical Backup Operations Specification

## Purpose

Defines the API operations on physical backups of a PostgreSQL database: listing, starting, cancelling and deleting them, issuing restore tokens, downloading a restore set and getting the recovery script, with the answer each operation gives.

## Requirements

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

Starting a physical backup SHALL answer `202` with no body. The request SHALL name the kind of backup: a full backup, an incremental backup, or the kind the scheduler would choose. That kind is an incremental backup when the database's backup type includes incremental backups and its newest chain can be extended, and a full backup otherwise. The backup SHALL appear in the list when the scheduler starts it. When an incremental backup is requested and the database has no chain to extend, the request SHALL get `409` with `code` `no_extendable_chain` and SHALL request nothing. A database that is not a physical PostgreSQL database SHALL get `400` with `code` `invalid_request`. A database whose backups are turned off, a database whose backup config was never saved included, or whose backup config names no storage SHALL get `400` with `code` `invalid_request` and SHALL request nothing, whichever kind the request names.

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

#### Scenario: Backups turned off

- **WHEN** a member starts a full backup of a physical database whose backups are turned off and that has no earlier backup request pending
- **THEN** the response is `400` with `code` `invalid_request`, and the WAL stream statistics do not report a full backup as pending

#### Scenario: An incremental backup with backups turned off and no chain

- **WHEN** a member starts an incremental backup of a physical database whose backups are turned off and that has no completed full backup
- **THEN** the response is `400` with `code` `invalid_request`, not `409`

#### Scenario: An incremental backup with backups turned off and a chain

- **WHEN** a member starts an incremental backup of a physical database whose backups are turned off and whose newest chain can be extended, and then turns its backups on while no backup is due by its schedules
- **THEN** the start answers `400` with `code` `invalid_request`, and no incremental backup appears in the list after the scheduler's next pass

#### Scenario: A backup config that was never saved

- **WHEN** a member starts a backup of a physical database whose backup config was never saved
- **THEN** the response is `400` with `code` `invalid_request`

#### Scenario: No storage

- **WHEN** a member starts a backup of a physical database whose backups are turned on, whose backup config names no storage and that has no earlier backup request pending
- **THEN** the response is `400` with `code` `invalid_request`, and the WAL stream statistics do not report a full backup as pending

### Requirement: Cancelling stops a running physical backup

Cancelling a running full or incremental backup SHALL answer `200` with the backup in its final status, in the same form the list gives it. By the time the response is sent, the backup SHALL already be recorded as `CANCELED`, and its process SHALL stop afterwards. A backup that is no longer running SHALL get `409` with `code` `not_running` and SHALL stay as it is. A backup ID that names no full or incremental backup SHALL get `404` with `code` `not_found`.

#### Scenario: Cancelling a running backup

- **WHEN** a member cancels a running full backup
- **THEN** the response is `200` with the backup as `CANCELED`, the list read right after it shows the same backup with the same values, and its `pg_basebackup` process ends

#### Scenario: Cancelling a running incremental backup

- **WHEN** a member cancels a running incremental backup whose parent is the root full backup
- **THEN** the response is `200` with the backup as `CANCELED`, and it equals the row the list returns right after, its `parentIncrementalBackupId` included

#### Scenario: Cancelling a backup whose process is gone

- **WHEN** a member cancels a backup that is recorded as running but has no live process in Databasus
- **THEN** the response is `200` with the backup as `CANCELED`, and the backup is listed as `CANCELED`

#### Scenario: A finished backup

- **WHEN** a member cancels a completed full backup
- **THEN** the response is `409` with `code` `not_running`, and the backup is still `COMPLETED`

### Requirement: Deleting removes a backup and the backups built on it

Deleting a full backup SHALL remove it and every incremental backup built on it. Deleting an incremental backup SHALL remove it and every later incremental backup of its chain. A backup that is running SHALL be removed before the response, and its process SHALL stop afterwards. Both SHALL answer `204`. An ID that names no full or incremental backup, the ID of a WAL segment included, SHALL get `404` with `code` `not_found` and SHALL delete nothing. The `physical-backup-wal-ranges` spec holds the scenario for a WAL segment.

#### Scenario: Deleting a full backup with incrementals

- **WHEN** a member deletes a completed full backup that has two incremental backups
- **THEN** the response is `204`, and neither the full backup nor its incremental backups are listed

#### Scenario: Deleting a running backup

- **WHEN** a member deletes a full backup that is running
- **THEN** the response is `204`, the list read right after it no longer shows the backup, and its `pg_basebackup` process ends

### Requirement: A restore token opens one restore set

Issuing a restore token SHALL answer `201` with the token and the time it expires, 15 minutes after it is issued. A token SHALL name either a database with an optional target time, which restores to that time or to the latest restorable moment, or one full or incremental backup, which restores that backup and the backups it is built on without replaying WAL. The token SHALL be refused at issue, and no token issued, when:

- the database has no completed full backup, or the chosen backup is not completed or misses a backup it is built on: `409` with `code` `no_restorable_backup`;
- the target time is earlier than the earliest full backup, a WAL gap lies between the full backup and the target, or reaching the target requires archived WAL on a timeline above 1 whose verified history is not published: `409` with `code` `target_time_unreachable`;
- another download or restore stream of the same user is running: `409` with `code` `already_running`;
- the chosen backup does not exist: `404` with `code` `not_found`;
- the database is not a physical PostgreSQL database: `400` with `code` `invalid_request`.

#### Scenario: Restoring to the latest moment

- **WHEN** a viewer asks for a restore token of a database with a completed full backup and archived WAL after it, without a target time and with the required timeline history published
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

For archive-based restores on a timeline above 1, Databasus SHALL include only WAL whose timeline has verified history published under the current backup storage selection. Without that history, a latest restore SHALL stop at the newest completed backup in the selected chain and SHALL include neither later WAL nor unrelated history files. An explicit target beyond that backup SHALL get `409` with `code` `target_time_unreachable`. Restoring an explicitly selected completed backup SHALL remain independent of separately archived history. Timeline 1 SHALL require no history file.

#### Scenario: Latest restore while promoted timeline history is pending

- **WHEN** a completed backup on timeline 2 has subsequent archived WAL but its verified timeline history has not been published
- **THEN** a latest restore token is issued for recovery through that backup only
- **AND** the downloaded set excludes the subsequent WAL

#### Scenario: Target requires unpublished history

- **WHEN** an explicit restore target requires WAL after the newest included backup on timeline 2 and that timeline's verified history is not published
- **THEN** token issuance returns `409` with `code` `target_time_unreachable`
- **AND** no token is issued

#### Scenario: Explicit backup remains restorable

- **WHEN** a completed backup is selected by its identifier and separately archived history is unavailable
- **THEN** token issuance succeeds if that backup and its required backup ancestors remain available
- **AND** the restore set contains no additional archived WAL or history

#### Scenario: History becomes available

- **WHEN** the verified history for timeline 2 is published while its completed backups and contiguous archived WAL remain available
- **THEN** latest and targeted restore selection can include that WAL without creating another FULL

#### Scenario: First timeline needs no history

- **WHEN** a restore on timeline 1 has a completed backup and sufficient contiguous archived WAL
- **THEN** absence of a timeline history file does not restrict that restore

### Requirement: The restore set downloads once with its token

Downloading a restore set SHALL need no session. The restore token SHALL be its credential and SHALL be passed as a query parameter. The download SHALL stream the restore set as a tar archive. A token that is unknown, expired or already used SHALL get `401` with `code` `unauthenticated`. A token SHALL be spent on its first use, even when that use is refused because another download or restore stream of the same user runs, which gets `409` with `code` `already_running`. When the restore set the token names can no longer be built, because a backup it needs was deleted, a WAL gap appeared, or required timeline history became unavailable after the token was issued, the download SHALL get the answer that issuing the token would get now.

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

#### Scenario: Targeted restore loses its required history before download

- **WHEN** a token for a target requiring archived WAL is issued while the promoted timeline's history is published, that history is then removed from the catalog, and the token is used
- **THEN** the download returns `409` with `code` `target_time_unreachable` before streaming the archive

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

### Requirement: A database runs one physical backup at a time

A database SHALL have at most one full or incremental backup recorded as running. Databasus SHALL also start no backup of a database while the process of an earlier backup of that database is still running, including a backup that was cancelled, deleted or stopped because its backups were turned off and whose process has not ended yet. A requested or scheduled backup SHALL start on the scheduler's first pass after that process has ended. Turning a database's backups off SHALL record its running backup as `CANCELED` before the save is answered. A backup still recorded as running when Databasus starts is failed as the `api-contract` spec describes, and the database is then free for its next backup.

#### Scenario: Starting right after a cancel

- **WHEN** a member cancels a running full backup and immediately requests a new full backup of the same database
- **THEN** the new backup starts only after the cancelled backup's `pg_basebackup` has ended, and it completes

#### Scenario: Starting right after a delete

- **WHEN** a member deletes a running full backup and the full backup cadence of the database is due
- **THEN** the next full backup starts only after the deleted backup's `pg_basebackup` has ended, and then completes

#### Scenario: Turning backups off during a backup

- **WHEN** a member turns off the backups of a database while its full backup is running, and turns them back on with the full backup cadence due
- **THEN** the list read right after the first save shows the backup as `CANCELED`
- **AND** the next full backup starts only after the cancelled backup's `pg_basebackup` has ended, and then completes

#### Scenario: A running backup at startup

- **WHEN** Databasus starts and finds a backup recorded as running
- **THEN** the scheduler can start the next backup of that database

### Requirement: A pending backup request is evaluated without waiting for a full scheduler pass

The backup request job SHALL evaluate each pending backup request with the same decision as a full scheduler pass, whether the request came from a manual start, a WAL slot rebuild or a storage switch. A request the decision accepts SHALL start its backup on the first pass of the backup request job that finds the database free. While the process of an earlier backup of that database is running, no backup SHALL start for the request, and the first pass after that process has ended SHALL apply the decision to it. An incremental request SHALL be cleared when incremental backups are not enabled or the database has no extendable chain, and it SHALL stay pending while a system identifier mismatch is unresolved. The job SHALL act only on databases with a pending request, whose backups are turned on and whose backup config names a storage. A backup that is due only by its schedule SHALL start on the next full scheduler pass, which starts 15 seconds after the previous one ended.

#### Scenario: A requested full backup between full scheduler passes

- **WHEN** a member requests a full backup of a physical database and only the backup request job runs afterwards
- **THEN** the full backup starts, and it appears in the list

#### Scenario: A requested incremental backup between full scheduler passes

- **WHEN** a member requests an incremental backup of a physical database with an extendable chain and only the backup request job runs afterwards
- **THEN** an incremental backup on that chain starts, and it appears in the list

#### Scenario: A full request recorded by a WAL slot rebuild

- **WHEN** a WAL slot rebuild has recorded a full request for a database and only the backup request job runs afterwards
- **THEN** the full backup starts, and it appears in the list

#### Scenario: A full request recorded by a storage switch

- **WHEN** a member switches the storage of a database with backups turned on and only the backup request job runs afterwards
- **THEN** a full backup in the new storage starts, and it appears in the list

#### Scenario: A full request filed during a running backup

- **WHEN** a member requests a full backup while the process of an earlier backup of the same database is running, and a pass of the backup request job runs
- **THEN** no second backup starts, and the request stays pending
- **AND** after that process has ended, the next pass starts the requested backup

#### Scenario: A schedule that is due without a request

- **WHEN** the full cadence of a database becomes due, the database has no pending request, and only the backup request job runs
- **THEN** no backup starts until the next full scheduler pass

#### Scenario: An incremental request that the decision clears

- **WHEN** a database whose backup config does not enable incremental backups has a pending incremental request and no backup due, and a pass of the backup request job runs
- **THEN** no backup starts, and the request is gone

#### Scenario: A requested incremental backup with an unresolved mismatch

- **WHEN** a database with an extendable chain has an unresolved system identifier mismatch and a pending incremental request, and a pass of the backup request job runs
- **THEN** no backup starts, and the request stays pending
- **AND** a requested full backup of that database starts on the next pass

#### Scenario: A request of a database with backups turned off

- **WHEN** a database has a pending full request and its backups are turned off, and a pass of the backup request job runs
- **THEN** no backup starts, and the WAL stream statistics still report a full backup as pending
- **AND** after the backups are turned on, the next pass starts the requested backup

#### Scenario: A request of a database with no storage

- **WHEN** a database has a pending full request and its backup config names no storage, and a pass of the backup request job runs
- **THEN** no backup starts, and the WAL stream statistics still report a full backup as pending

### Requirement: Script recovery instructions preserve the selected restore settings

For both Host and Docker recovery, the Script method SHALL show a recovery command and startup instructions that consistently use the restore directory and PostgreSQL version selected in the form. A restore directory containing spaces SHALL remain one shell argument. Relative directories SHALL resolve from the execution directory, and absolute directories SHALL remain absolute.

For Host recovery, a supplied PostgreSQL tools directory SHALL apply to both recovery preparation and the displayed PostgreSQL startup command. For Docker recovery, the selected PostgreSQL image SHALL apply to both recovery preparation and the displayed startup command. The displayed container mount SHALL make PostgreSQL serve the cluster prepared in the selected restore directory, including PostgreSQL 18's version-specific data layout. Following these commands on a prepared recovery machine SHALL start the restored cluster without editing the displayed paths, image or arguments.

#### Scenario: Host recovery with a custom directory

- **WHEN** a user selects Host recovery, a relative restore directory containing a space and a PostgreSQL tools directory outside the default command search path
- **THEN** the recovery and startup commands use those directories, and following the commands starts the restored cluster

#### Scenario: Docker recovery with an absolute directory

- **WHEN** a user selects Docker recovery, an absolute restore directory containing a space and a matching PostgreSQL image identified by digest
- **THEN** both commands use the selected image, and startup mounts the selected directory without prepending the execution directory
- **AND** PostgreSQL serves the restored cluster rather than initializing an empty cluster

### Requirement: Physical recovery preserves committed schema changes

On PostgreSQL 17 and 18, restoring to the latest available point on one timeline SHALL preserve the committed schema and rows represented by the selected backups and contiguous archived WAL. This SHALL hold both for a FULL followed only by WAL and for a FULL followed by incremental backups and WAL. Changes made after the last completed backup SHALL be recovered from the subsequent WAL. Rolled-back transactional DDL SHALL leave no changes in the recovered schema.

The preserved schema SHALL include created schemas and tables, added and renamed columns, column types and defaults, indexes, constraints, and committed object renames and deletions. A column type conversion that rewrites stored rows SHALL preserve both the converted values and the resulting definition.

#### Scenario: Schema changes recovered only from WAL after a FULL

- **WHEN** a FULL completes, schema changes and corresponding rows are committed afterwards, and the following WAL is archived without taking an incremental backup
- **THEN** restoring to the latest available point starts a writable cluster with those rows and the committed object definitions

#### Scenario: Schema changes span incrementals and subsequent WAL

- **WHEN** a FULL and two successive incremental backups contain different stages of schema changes, further changes are committed after the last incremental, and contiguous WAL through those changes is archived
- **THEN** restoring to the latest available point preserves the rows and schema changes from every stage, including those present only in the later WAL

#### Scenario: Renamed and deleted objects retain their committed state

- **WHEN** columns or tables are renamed and other objects are deleted before the selected recovery point
- **THEN** the recovered schema contains the renamed objects under their new names and excludes their old names and the deleted objects

#### Scenario: Transactional DDL is rolled back

- **WHEN** a transaction creates, alters or drops schema objects and is rolled back before the selected recovery point
- **THEN** recovery preserves the previously committed definitions and excludes objects created only in that transaction

### Requirement: Physical recovery preserves sequence configuration and usable ID generation

Physical recovery on PostgreSQL 17 and 18 SHALL preserve logged sequences used by `serial`, identity columns and explicit sequence defaults, including objects created after the FULL. It SHALL preserve their column defaults, identity generation mode, column ownership dependencies and sequence type, start, increment, minimum, maximum, cycle and cache settings at the selected recovery point.

Sequence values after recovery SHALL follow native PostgreSQL recovery semantics. Recovery SHALL NOT promise gapless IDs, an exact next value based on the largest restored ID, or transactional rollback of `nextval` and `setval`. For an ascending, noncycling sequence used consistently to generate IDs, with any explicit repositioning moving it forward, new generated IDs SHALL avoid the restored committed IDs and respect the recovered configuration. A forward `setval` represented by the recovered WAL SHALL not be silently lost even when losing it would not immediately cause a duplicate ID.

#### Scenario: Serial and identity columns remain usable

- **WHEN** a backup and subsequent WAL contain tables with `serial` and identity columns, including a table created after the FULL
- **THEN** recovery preserves the generation definitions and sequence dependencies
- **AND** the first insert that omits a generated ID succeeds and allocates an ID distinct from the restored committed IDs
- **AND** subsequent allocations also produce IDs distinct from the restored committed IDs

#### Scenario: An explicit sequence is repositioned and altered

- **WHEN** a logged sequence used by a column default is moved forward with `setval`, its increment and cache are changed, and those operations are included in the selected recovery point
- **THEN** recovery preserves the column default and the committed sequence parameters
- **AND** newly generated values respect the forward position and the recovered increment, allowing native recovery gaps

#### Scenario: Both cache settings survive recovery

- **WHEN** the recovered state includes a sequence with `CACHE 1` and another with a larger cache
- **THEN** both retain their cache settings and generate distinct usable IDs after recovery without requiring either sequence to resume at the largest restored ID plus one

#### Scenario: A sequence call occurs in a rolled-back transaction

- **WHEN** a transaction calls `nextval` or `setval` on an existing logged sequence, writes a row and then rolls back
- **THEN** the recovered table excludes that row
- **AND** the sequence remains usable according to native PostgreSQL recovery semantics without a guarantee that the call was undone

### Requirement: PITR applies the transaction boundary to schema changes

For a reachable target after the end of the selected base backup on one timeline, PITR SHALL preserve the transactional schema changes committed before the target and SHALL exclude those committed after it, including commands executed before the target by a transaction that commits later. This SHALL cover object creation, alteration, rename and deletion, as well as transactional sequence definition changes. It SHALL NOT impose that transaction boundary on the effects of `nextval` or `setval` beyond PostgreSQL's native recovery semantics.

#### Scenario: The target precedes the schema transaction commit

- **WHEN** a transaction executes schema changes before the target time but commits after it, and WAL containing the later commit is available
- **THEN** restoring to that target preserves the schema as it was before the transaction, including objects that the transaction later renames or deletes
- **AND** objects created only by that transaction are absent

#### Scenario: The target follows the schema transaction commit

- **WHEN** a transaction commits schema and sequence definition changes before the target, a later transaction changes the schema after the target, and WAL containing both commits is available
- **THEN** restoring to the target includes the first transaction's definitions and rows and excludes the later transaction's changes
- **AND** generated-ID inserts succeed under the recovered definitions and native sequence semantics

### Requirement: Physical recovery is independent of inherited replication settings

For PostgreSQL 17 and 18, preparing a physical recovery set SHALL prevent inherited source replication settings, competing recovery targets and recovery signal files from overriding the newly selected recovery operation. An explicit target SHALL govern replay to that target; recovery without an explicit target SHALL NOT stop at a stale inherited target. Startup SHALL use the restored data directory and local archived WAL without depending on the former primary's connection settings or archive command. When local authentication files are copied, source-specific data-directory and authentication-file path overrides SHALL NOT prevent the independent target from using them. This SHALL apply to assignments with or without `=` and to locally included configuration regardless of filename extension, including nested `include`, `include_if_exists` and `include_dir` directives. This SHALL also hold when an explicitly selected completed backup ships no additional archived WAL.

If the source keeps configuration outside its data directory, the recovery script SHALL retain its existing diagnostic and operator-controlled configuration preparation. The test environment SHALL supply an independent recovery configuration without weakening the source's access rules or transport security.

Absolute includes below the source data directory recorded in copied configuration SHALL be rebased into the restored data directory. Their contents SHALL receive the same sanitization as relative includes, and the rewritten paths SHALL remain usable when the prepared directory is moved into its runtime container.

Source-directory discovery SHALL follow the local include graph before sanitization and honor configuration order. A source directory declared only in an included file SHALL support rebasing absolute includes, including those appearing before that declaration.

Source-local `hba_file` and `ident_file` paths to copied authentication files SHALL be rebased to their restored locations, including subdirectories without default root-level authentication files. A referenced copied file SHALL take precedence over an unrelated default root-level file.

Authentication paths SHALL target the prepared local PGDATA for native startup or the selected image's PGDATA for `--combine-image`. Copied authentication files at those paths SHALL NOT trigger the missing-configuration diagnostic merely because their default root-level names are absent.

#### Scenario: Absolute source-local configuration includes

- **WHEN** copied configuration records its source data directory and includes an absolute path below it, including a nested optional include or include directory
- **THEN** recovery preparation rewrites the source-local path to the copied file or directory, removes inherited recovery settings from its contents and starts PostgreSQL without the source filesystem

#### Scenario: Source directory is declared in a local settings file

- **WHEN** the main configuration includes a local file that records the source data directory and includes another file by an absolute source-local path
- **THEN** recovery preparation discovers the source directory through the include graph before removing its declaration
- **AND** PostgreSQL starts with the sanitized copied files without depending on the original source paths

#### Scenario: Targeted recovery from a Patroni source

- **WHEN** a downloaded recovery set from a promoted Patroni source contains inherited replication settings or older recovery targets and the former primary is unavailable
- **THEN** preparing and starting the recovery reaches the newly selected target using the local archive
- **AND** the recovered cluster contains transactions through that target, excludes later transactions and accepts new writes with generated IDs

#### Scenario: Copied authentication files reside in a subdirectory

- **WHEN** copied configuration references source-local HBA and ident files in a subdirectory and no default root-level authentication files were copied
- **THEN** recovery preparation rebases both paths and PostgreSQL starts using the copied authentication files
- **AND** their contents and the source's access rules remain unchanged

#### Scenario: Referenced authentication files take precedence over root defaults

- **WHEN** copied configuration references source-local HBA and ident files in a subdirectory and distinct default root-level authentication files were also copied
- **THEN** recovery preparation retains the referenced files and PostgreSQL starts using their unchanged contents
- **AND** both native startup and image-entrypoint startup use those files without changing source access rules

#### Scenario: Latest recovery ignores a stale target

- **WHEN** latest recovery is requested from a Patroni source whose copied configuration contains an older recovery target or a standby signal
- **THEN** recovery includes the available later archived transactions rather than stopping at the inherited target or remaining a standby
- **AND** the recovered cluster becomes writable without contacting the former primary

#### Scenario: Explicit backup recovery ships no additional WAL

- **WHEN** a completed physical backup from a Patroni source is selected by its identifier and its copied configuration contains source replication or recovery settings
- **THEN** recovery starts a writable independent cluster at the selected backup's consistency point without requiring an additional archive or the former primary

#### Scenario: External configuration is prepared for recovery

- **WHEN** the downloaded data directory lacks configuration stored externally by the source
- **THEN** the recovery diagnostic identifies the missing configuration and the required preparation
- **AND** supplying independent configuration according to that diagnostic allows the restored cluster to start while source access rules and transport security remain unchanged

### Requirement: Cleanup preserves retained recovery points after repeated Patroni switches

On PostgreSQL 17 and 18, physical backup cleanup and storage file deletion SHALL preserve the files, manifests, WAL and timeline histories required by recovery points still protected by retention after repeated Patroni switches. This SHALL include a reachable target on an older timeline as well as a reachable target after a replacement FULL on the newest timeline. Cleanup SHALL NOT make a target between promotion and completion of its replacement FULL reachable.

#### Scenario: Retained old and new targets survive both cleanup stages

- **WHEN** two Patroni switchovers have completed, retention still protects a reachable target before the first switch and another after the newest replacement FULL, and physical cleanup and storage deletion have processed their work
- **THEN** newly downloaded sets for both targets start independent PostgreSQL clusters with their respective rows, schema definitions and sequence definitions
- **AND** new generated-ID writes succeed in both recovered clusters
- **AND** each archive-based set includes the published history required for its own timeline in its FULL's storage

#### Scenario: Cleanup does not hide the promotion recovery gap

- **WHEN** both cleanup stages have run after the replacement FULL and a target between promotion and that FULL's completion still requires replay across timelines
- **THEN** requesting that target returns `409` with code `target_time_unreachable` and issues no restore token

### Requirement: Delayed WAL publication does not extend a source timeline's PITR time

An explicit PITR target SHALL require source-time evidence from complete transaction records in the selected timeline's contiguous published WAL, or a completed-backup consistency point. Receipt, claim and upload timestamps SHALL NOT establish source-time reachability. With inclusive recovery, a confirming WAL transaction SHALL be strictly later than the target. Old WAL without source-time evidence SHALL NOT use receipt time as a substitute. Latest recovery and explicit completed-backup recovery SHALL remain available without claiming an unconfirmed explicit target.

User-controlled descriptive payload, including multiline prepared-transaction identifiers, SHALL NOT be interpreted as COMMIT/ABORT records or extend source-time evidence. Genuine transaction completion after PREPARE SHALL remain usable as evidence.

Confirmed source time SHALL retain the maximum timestamp among trusted complete transaction records. A later record with an older timestamp, including an independently verified completion after PREPARE, SHALL NOT reduce already confirmed coverage.

When an explicit target equals the selected completed backup's consistency point and contiguous WAL contains no later confirming transaction, recovery SHALL use the completed backup and its required ancestors without additional archived WAL. Supplying that advertised target to the recovery script SHALL start a writable cluster without requiring a later transaction. Latest recovery SHALL retain eligible WAL, and targets beyond the confirmed bound SHALL remain refused.

#### Scenario: A completed backup is the only confirmed time

- **WHEN** a completed FULL or INCREMENTAL has subsequent archived WAL but no transaction later than its completion, and the owner requests the completion point advertised by statistics
- **THEN** the downloaded set contains the selected backup and its required ancestors without additional archived WAL
- **AND** starting recovery with that explicit target succeeds and permits new writes
- **AND** latest recovery retains eligible WAL without advertising an unconfirmed later PITR target

#### Scenario: Transaction timestamps decrease in WAL order

- **WHEN** a later complete transaction record has a timestamp older than an already confirmed record
- **THEN** statistics and target validation retain the greatest confirmed source timestamp as their time evidence
- **AND** an independently verified completion after PREPARE cannot reduce an earlier trusted bound

#### Scenario: A prepared identifier imitates transaction output

- **WHEN** a prepared-transaction identifier contains a multiline COMMIT description or a fabricated WAL record header with a future timestamp
- **THEN** decoding retains only time confirmed by actual completed WAL records, and the fabricated time cannot extend recovery coverage or authorize a target
- **AND** a subsequent genuine COMMIT or ABORT remains observable

#### Scenario: Old timeline WAL is published after promotion

- **WHEN** old-timeline WAL is published after promotion and its receipt timestamp is later than a requested target in the replacement-FULL gap
- **THEN** token issuance and download validation refuse that target with `409 target_time_unreachable`, both before and after the replacement FULL completes
- **AND** a pre-promotion target confirmed by that timeline's archived transactions remains recoverable

#### Scenario: A final transaction is not a later confirmation

- **WHEN** a requested target equals the last decoded transaction timestamp and no later transaction is published
- **THEN** token issuance refuses that explicit target
- **AND** a target one microsecond earlier remains eligible when its WAL is contiguous

#### Scenario: Confirming WAL disappears after token issuance

- **WHEN** a valid target token was issued while confirming WAL was available, that WAL is removed, and remaining late-received WAL has insufficient source-time evidence
- **THEN** downloading with that token returns `409 target_time_unreachable` before archive streaming begins
