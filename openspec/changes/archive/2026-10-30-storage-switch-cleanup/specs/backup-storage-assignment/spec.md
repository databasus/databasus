## MODIFIED Requirements

### Requirement: Moving a physical backup configuration to another storage deletes its physical backups

When a user switches a physical backup configuration from one configured storage to another, including during a workspace transfer, the new storage, removal of the old backup history and durable requests to delete its files SHALL commit together. The old full and incremental backups SHALL leave the backup list, together with the WAL segments and timeline history files that only those backups need. WAL that a full backup kept in the new storage needs SHALL be kept, wherever it is stored.

Assigning a storage when the physical configuration currently names no storage but retains backup history SHALL follow the same atomic cleanup, operation response and fresh-FULL rules. Clearing the configured storage SHALL NOT make retained full or incremental backups, WAL or timeline history eligible to continue a chain after reassignment. Deletion requests SHALL identify each retained artifact's actual storage even though the configuration no longer names it. An initial assignment with no retained history SHALL remain an ordinary config save without a switch operation.

The switch SHALL NOT contact the previous storage or wait for remote deletion. A database error before commit SHALL leave the previous config, catalog and deletion obligations unchanged. After commit the new storage SHALL stay active regardless of remote cleanup failure. The request SHALL return 202 with an observable cleanup operation.

A physical backup running on the old storage SHALL be stopped after commit and SHALL NOT publish into the new history. An old WAL streamer SHALL stop and SHALL NOT publish old-destination files into the new history. When backups are enabled, the next physical backup SHALL be a new FULL in the new storage without waiting for the previous FULL schedule or for cleanup. WAL streaming, when expected, SHALL restart for the new storage. A disabled backup configuration SHALL stay disabled, and its first backup after being enabled SHALL be a new FULL in the new storage.

#### Scenario: Chain on the old storage

- **WHEN** a database has full and incremental backups and WAL in storage A and the user switches its physical configuration to storage B
- **THEN** the response is 202 with an operation, the config names B, and the old backups are absent from the list
- **AND** old files are queued for deletion from A and the next enabled backup is a new FULL in B

#### Scenario: Old storage is unreachable

- **WHEN** the user switches from unreachable storage A to reachable storage B
- **THEN** the switch is accepted without waiting for A and backups can start in B
- **AND** the cleanup operation eventually reports complete or incomplete according to its finite cleanup policy

#### Scenario: Physical storage is reassigned after being cleared

- **WHEN** a physical configuration used A, now names no storage, retains full and incremental backups and WAL in A, and is assigned B
- **THEN** B, removal of the old history and durable deletion requests against A commit together, and the response is 202 with a cleanup operation
- **AND** the next enabled backup is a new FULL in B, and expected WAL streaming restarts for B without waiting for cleanup of A

#### Scenario: Physical reassignment cannot persist cleanup

- **WHEN** a physical configuration names no storage but retains history in A, and assigning B fails to persist the required cleanup
- **THEN** the configuration still names no storage and the old history remains, without creating an operation or canceling old work

#### Scenario: Initial physical assignment has no history

- **WHEN** a physical configuration names no storage, has no retained history and is first assigned B through a config save
- **THEN** the response is 200 with the config, B is assigned and no switch operation is created

#### Scenario: Retry after a failed removal

- **WHEN** persisting the switch or any required deletion obligation fails
- **THEN** the request fails and the previous config, backup catalog and cleanup state are unchanged
- **AND** submitting the switch again retries the atomic change rather than repairing a partially committed config

#### Scenario: Full backup running during the change

- **WHEN** a full backup writes into A while the configuration switches to B
- **THEN** the old backup stops after commit and cannot reappear as a completed backup
- **AND** the new FULL waits only for old execution to drain, not for deleting A's files

#### Scenario: WAL streaming continues on the new storage

- **WHEN** the configuration expects WAL streaming and switches from A to B
- **THEN** the old streamer stops and subsequent accepted WAL uploads belong to B
- **AND** a late upload by the old streamer cannot reintroduce A's catalog entries

#### Scenario: A running old FULL is removed while a target FULL survives

- **WHEN** a FULL is running in A, a completed FULL already exists in B, and the configuration switches to B
- **THEN** WAL and timeline history that no surviving target backup needs leave the catalog and enter the cleanup operation
- **AND** the target FULL, its required WAL boundary segment and its timeline history remain

#### Scenario: The storage does not change

- **WHEN** a user saves a physical backup configuration with the storage it already had
- **THEN** no backup is removed and no new switch operation is created

#### Scenario: Backups remain disabled

- **WHEN** the storage of a disabled physical backup configuration changes
- **THEN** the new storage is saved and cleanup is accepted without enabling or starting backups

#### Scenario: Backups are enabled after a switch

- **WHEN** a physical backup configuration was disabled during its storage switch and is later enabled
- **THEN** its first backup is a new FULL in the selected storage, regardless of the old chain or its previous schedule

#### Scenario: A switch rolls back during an active physical backup

- **WHEN** a physical backup or WAL stream is running and persisting its storage switch fails
- **THEN** the old configuration and catalog remain, and the failed switch does not cancel the active work

### Requirement: Moving a logical backup configuration to another storage deletes its logical backups

When a user switches a logical backup configuration from one configured storage to another, including during a workspace transfer, the new storage, removal of the database's old logical backups and durable file-deletion requests SHALL commit together. The request SHALL answer 202 with an observable cleanup operation. Remote deletion SHALL run in the background without delaying the switch.

These rules SHALL also apply when the configuration currently names no storage but still has logical backups from an earlier assignment. Each deletion request SHALL address the backup's actual storage. An initial assignment with no retained logical backups SHALL remain an ordinary config save without a switch operation.

When a logical backup or a restore using those backups is running, the switch SHALL get 409 with code in_use and SHALL change nothing. When the catalog removal or recording deletion obligations fails, the request SHALL get 500 with code internal and SHALL leave the previous configuration, catalog and operation state unchanged. Remote cleanup failure after commit SHALL NOT revert or fail the already accepted switch.

#### Scenario: Backups on the old storage

- **WHEN** a database has logical backups in A and a user switches its logical configuration to B
- **THEN** the response is 202 with an operation, the config names B and the old backups leave the list
- **AND** deletion of their files from A is requested

#### Scenario: Logical storage is reassigned after being cleared

- **WHEN** a logical configuration used A, now names no storage, retains backups in A, and is assigned B
- **THEN** B, removal of the old backups and durable deletion requests against A commit together, and the response is 202 with a cleanup operation

#### Scenario: Logical reassignment conflicts with active work

- **WHEN** a logical configuration names no storage but retains backups in A, and a backup or restore is active when B is assigned
- **THEN** the reassignment is refused with 409 and code in_use without changing the configuration or backups

#### Scenario: Logical reassignment cannot persist cleanup

- **WHEN** a logical configuration names no storage but retains backups in A, and assigning B fails to persist the required cleanup
- **THEN** the configuration still names no storage and the old backups remain listed, without creating an operation

#### Scenario: Initial logical assignment has no history

- **WHEN** a logical configuration names no storage, has no retained backups and is first assigned B through a config save
- **THEN** the response is 200 with the config, B is assigned and no switch operation is created

#### Scenario: A backup is running

- **WHEN** a logical backup runs and a user switches its configuration to another storage
- **THEN** the response is 409 with code in_use, the config still names the old storage and its backups stay listed

#### Scenario: A restore is running

- **WHEN** a restore reads one of the old logical backups and the user switches storage
- **THEN** the response is 409 with code in_use and the config and backups are unchanged

#### Scenario: The storage does not change

- **WHEN** a user saves a logical configuration with its current storage
- **THEN** no backup is removed and no new switch operation is created

#### Scenario: The removal fails

- **WHEN** the old catalog removal or any required deletion request cannot be persisted
- **THEN** the response is 500 with code internal and the previous configuration and backup list remain

#### Scenario: Old storage is unreachable

- **WHEN** a user switches a logical configuration from unreachable A to B
- **THEN** B becomes active without contacting A and the accepted operation tracks deletion separately

### Requirement: Saving a backup configuration never creates or changes a storage

Saving a logical or physical backup configuration SHALL NOT create a storage or change any field of an existing storage. A configuration SHALL refer only to the identifier of an existing storage in the database's workspace, or name no storage. Storage fields beside that identifier SHALL be ignored. A storage SHALL change only through its own endpoints and access checks. This is a security rule: changing a backup configuration SHALL NOT bypass permission checks for editing storage settings or credentials.

A storage identifier that does not exist or belongs to another workspace SHALL get 400 with code invalid_request and SHALL leave the configuration unchanged. This check SHALL hold against a concurrent transfer: either saving succeeds and the storage transfer gets 409 with code in_use, or transfer succeeds and saving gets 400 with code invalid_request without a config change.

A successful save that switches from one configured storage to another, or assigns storage to a configuration with no current storage but retained backup history, SHALL answer 202 with the storage switch operation. Ordinary edits with the same storage and initial storage assignment without retained history SHALL answer 200 with the config. Clearing storage SHALL preserve the retained catalog and SHALL NOT itself create a switch operation. These response distinctions SHALL NOT change the storage ownership, secret-preservation or concurrent-transfer guarantees.

#### Scenario: The request names an existing storage

- **WHEN** a user saves a config naming a storage of the database's workspace
- **THEN** the config names that storage and its settings, name and credentials stay unchanged
- **AND** a switch from another configured storage or reassignment with retained history answers 202, while an ordinary save or initial assignment without retained history answers 200

#### Scenario: The request carries changed storage fields

- **WHEN** a user saves a config with an existing storage identifier and extra fields changing that storage's name or credentials
- **THEN** the extra fields are ignored and the storage retains its name and credentials
- **AND** the response is 202 with an operation for a switch from another configured storage or reassignment with retained history, otherwise 200 with the config

#### Scenario: The request carries a storage without an identifier

- **WHEN** a user saves a config with a storage object in place of an identifier
- **THEN** the configuration names no storage and no storage is created

#### Scenario: Clearing storage preserves history until reassignment

- **WHEN** a user clears the configured storage while the database has backups in A
- **THEN** the response is 200 with a config naming no storage, the old backup history remains and no switch operation is created
- **AND** a later assignment of storage, including A again, applies the reassignment cleanup rules

#### Scenario: The request carries an unknown storage

- **WHEN** a user saves a config with an identifier that no storage has
- **THEN** the response is 400 with code invalid_request, the config is unchanged and no storage is created

#### Scenario: The request names a storage of another workspace

- **WHEN** a user saves a config naming a storage of another workspace
- **THEN** the response is 400 with code invalid_request and the config retains its storage

#### Scenario: A save races a transfer of its storage

- **WHEN** a user saves a config naming S while another request transfers S to another workspace
- **THEN** exactly one succeeds: either the save answers 202 for a switch or 200 for an ordinary save, the transfer answers 409 with code in_use and S stays in the database's workspace; or transfer answers 204 and saving answers 400 with code invalid_request, leaving the previous config

#### Scenario: A database transfer commits before a configuration save

- **WHEN** a database transfer commits while a config save waits for its database and config locks
- **THEN** the save checks access and storage ownership against the database's workspace read under those locks
- **AND** a former-workspace storage or a caller without access to the current workspace is refused without creating an operation
- **AND** an accepted switch belongs to the current workspace

## ADDED Requirements

### Requirement: A storage switch warns about deletion and exposes its cleanup outcome

Before confirming a storage switch in a backup configuration or workspace transfer, the interface SHALL explain that existing backups become unavailable, their files will be deleted from the previous storage, the new storage becomes active immediately, and file deletion continues in the background. It SHALL warn that files may remain if the old storage cannot be reached. It SHALL NOT promise that physical deletion finishes before the new storage is used.

The accepted operation SHALL expose its identity, database, previous and target storage identifiers, creation and completion times, cleanup deadline, and total, pending, deleted and remaining artifact counts. Its state SHALL be CLEANUP_PENDING, COMPLETED or COMPLETED_WITH_REMAINDERS. The total SHALL equal the sum of pending, deleted and remaining artifacts; remaining SHALL count artifacts with a terminal unresolved outcome. A terminal operation SHALL have no pending artifacts and SHALL expose its completion time. Completion with remainders SHALL explicitly mean that switching succeeded but some old files could not be confirmed deleted.

The interface SHALL poll the persisted status while cleanup is pending, stop polling at a terminal outcome and recover pending operations after a reload. A status-read failure SHALL retain the last known state, offer retry and SHALL NOT report the switch as failed or silently submit it again. Multiple switches SHALL have independent, discoverable operations. A cleanup result SHALL never be presented as successful deletion unless absence was confirmed.

#### Scenario: User confirms a switch

- **WHEN** the user selects another storage in the configuration or transfer form
- **THEN** the deletion and background-cleanup warning appears before submission
- **AND** after acceptance the interface displays the active new storage and cleanup progress

#### Scenario: Cleanup completes

- **WHEN** all files associated with a switch are confirmed absent
- **THEN** its operation becomes COMPLETED with zero pending and remaining artifacts, and polling stops

#### Scenario: Cleanup gives up

- **WHEN** the finite cleanup policy ends while some files are not confirmed absent
- **THEN** its operation becomes COMPLETED_WITH_REMAINDERS and the interface states that the switch succeeded but some old files may remain

#### Scenario: Page is reloaded

- **WHEN** a user reloads the database screen while cleanup is pending
- **THEN** the screen discovers that operation and resumes displaying its persisted state

#### Scenario: A status request fails

- **WHEN** an operation-status read fails after the switch was accepted
- **THEN** the interface keeps the last known state and offers another status read without submitting another switch

#### Scenario: Switch with no old artifacts

- **WHEN** a switch has no files requiring cleanup
- **THEN** its returned operation is already COMPLETED with zero artifact counts

### Requirement: Storage switch history remains discoverable

The system SHALL provide individual operation reads and a cursor-paged operation history for a required workspace, optionally filtered by database. History SHALL include pending and terminal operations, ordered newest first with a stable order for operations accepted at the same time. Each operation SHALL retain the workspace in which its switch was accepted, which is the target workspace for a transfer.

Deleting a database SHALL NOT erase its switch operations or interrupt their cleanup. Authorized viewers SHALL still be able to read those operations and discover them through the acceptance workspace's history. A later database transfer SHALL NOT move an earlier operation into the new workspace or grant that workspace access to it. All history and individual reads SHALL obey the authorization requirement below.

#### Scenario: A database is deleted during cleanup

- **WHEN** a database is deleted while its storage switch cleanup is pending
- **THEN** the cleanup continues and its operation remains readable and listed in the acceptance workspace's history

#### Scenario: A database is transferred again

- **WHEN** a database with an existing switch operation moves to another workspace
- **THEN** the earlier operation remains scoped to its acceptance workspace and the later move grants no additional access to it

#### Scenario: Several operations share an acceptance time

- **WHEN** a viewer reads successive pages containing operations accepted at the same time
- **THEN** their order is stable and each operation appears once in that traversal

#### Scenario: A viewer filters operation history

- **WHEN** a viewer lists the acceptance workspace's history with a database filter
- **THEN** the response contains only that database's operations within that workspace, including operations whose database was subsequently deleted

### Requirement: Only an authorized user can inspect a storage switch

Reading or listing switch operations SHALL require a session and permission to view the operation's workspace, the target workspace at acceptance. An instance admin SHALL have that permission without workspace membership, as required by workspace-access. A nonmember who is not an instance admin SHALL receive 404 with code not_found, and a missing session SHALL receive 401 with code unauthenticated. Responses SHALL expose safe state and counts only, without provider errors, credentials, file names or signed addresses. Editing permission in both workspaces SHALL still be required to initiate a workspace transfer, with the existing instance-admin permissions preserved.

#### Scenario: Member of another workspace

- **WHEN** a user who is not an instance admin and has no membership in the operation's workspace requests its status
- **THEN** the response is 404 with code not_found and reveals no switch details

#### Scenario: Instance admin without workspace membership

- **WHEN** an instance admin who is not a member of the operation's workspace reads the operation and lists that workspace's operations
- **THEN** both requests succeed and return the same safe operation state available to an authorized workspace viewer

#### Scenario: No session

- **WHEN** a caller without a session reads switch status
- **THEN** the response is 401 with code unauthenticated
