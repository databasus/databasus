# backup-storage-assignment Specification

## Purpose
Defines which storage a restore reads a backup from, what happens to existing backups when a backup configuration moves to another storage, and that saving a backup configuration never changes the storages themselves.

## Requirements

### Requirement: A restore reads the backup from the storage it was written to

A restore SHALL read the backup from the storage recorded with that backup when it was written. The storage that the database's backup configuration names at the time of the restore SHALL NOT decide where the restore reads from.

#### Scenario: The configuration names another storage

- **WHEN** a user restores a logical backup written to storage A, and the database's backup configuration now names storage B
- **THEN** the restore reads the backup from storage A and completes

#### Scenario: The configuration names no storage

- **WHEN** a user restores a logical backup, and the database's backup configuration names no storage
- **THEN** the restore reads the backup from the storage recorded with it and completes

### Requirement: Moving a physical backup configuration to another storage deletes its physical backups

When a user switches a physical backup configuration from one configured storage to another, including during a workspace transfer, the new storage, removal of the old backup history and durable requests to delete its files SHALL commit together. The old full and incremental backups SHALL leave the backup list, together with the WAL segments and timeline history files that only those backups need. WAL that a full backup kept in the new storage needs SHALL be kept, wherever it is stored.

Assigning a storage when the physical configuration currently names no storage but retains backup history SHALL follow the same atomic cleanup, configuration acknowledgement and fresh-FULL rules. Clearing the configured storage SHALL NOT make retained full or incremental backups, WAL or timeline history eligible to continue a chain after reassignment. Deletion requests SHALL identify each retained artifact's actual storage even though the configuration no longer names it. An initial assignment with no retained history SHALL remain an ordinary config save without new cleanup work.

The switch SHALL NOT contact the previous storage or wait for remote deletion. A database error before commit SHALL leave the previous config, catalog and deletion obligations unchanged. After commit the new storage SHALL stay active regardless of remote cleanup failure. A configuration save SHALL return 200 with the saved configuration and current cleanup counts; a workspace transfer SHALL return 204 after commit.

A physical backup running on the old storage SHALL be stopped after commit and SHALL NOT publish into the new history. An old WAL streamer SHALL stop and SHALL NOT publish old-destination files into the new history. When backups are enabled, the next physical backup SHALL be a new FULL in the new storage without waiting for the previous FULL schedule or for cleanup. WAL streaming, when expected, SHALL restart for the new storage. A disabled backup configuration SHALL stay disabled, and its first backup after being enabled SHALL be a new FULL in the new storage.

#### Scenario: Chain on the old storage

- **WHEN** a database has full and incremental backups and WAL in storage A and the user switches its physical configuration to storage B
- **THEN** the configuration save response is 200 with the saved configuration, the config names B, and the old backups are absent from the list
- **AND** old files are queued for deletion from A and the next enabled backup is a new FULL in B

#### Scenario: Old storage is unreachable

- **WHEN** the user switches from unreachable storage A to reachable storage B
- **THEN** the switch is accepted without waiting for A and backups can start in B
- **AND** the old files stay queued and retry until confirmed absent or the source storage is explicitly deleted with abandonment

#### Scenario: Physical storage is reassigned after being cleared

- **WHEN** a physical configuration used A, now names no storage, retains full and incremental backups and WAL in A, and is assigned B
- **THEN** B, removal of the old history and durable deletion requests against A commit together, and the configuration save response is 200 with the saved configuration
- **AND** the next enabled backup is a new FULL in B, and expected WAL streaming restarts for B without waiting for cleanup of A

#### Scenario: Physical reassignment cannot persist cleanup

- **WHEN** a physical configuration names no storage but retains history in A, and assigning B fails to persist the required cleanup
- **THEN** the configuration still names no storage and the old history remains, without creating deletion obligations or canceling old work

#### Scenario: Initial physical assignment has no history

- **WHEN** a physical configuration names no storage, has no retained history and is first assigned B through a config save
- **THEN** the response is 200 with the config, B is assigned and no new cleanup work is created

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
- **THEN** WAL and timeline history that no surviving target backup needs leave the catalog and are queued for deletion
- **AND** the target FULL, its required WAL boundary segment and its timeline history remain

#### Scenario: The storage does not change

- **WHEN** a user saves a physical backup configuration with the storage it already had
- **THEN** no backup is removed and no new cleanup work is created

#### Scenario: Backups remain disabled

- **WHEN** the storage of a disabled physical backup configuration changes
- **THEN** the new storage is saved and cleanup is accepted without enabling or starting backups

#### Scenario: Backups are enabled after a switch

- **WHEN** a physical backup configuration was disabled during its storage switch and is later enabled
- **THEN** its first backup is a new FULL in the selected storage, regardless of the old chain or its previous schedule

#### Scenario: A switch rolls back during an active physical backup

- **WHEN** a physical backup or WAL stream is running and persisting its storage switch fails
- **THEN** the old configuration and catalog remain, and the failed switch does not cancel the active work

### Requirement: Saving a backup configuration never creates or changes a storage

Saving a logical or physical backup configuration SHALL NOT create a storage or change any field of an existing storage. A configuration SHALL refer only to the identifier of an existing storage in the database's workspace, or name no storage. Storage fields beside that identifier SHALL be ignored. A storage SHALL change only through its own endpoints and access checks. This is a security rule: changing a backup configuration SHALL NOT bypass permission checks for editing storage settings or credentials.

A storage identifier that does not exist or belongs to another workspace SHALL get 400 with code invalid_request and SHALL leave the configuration unchanged. This check SHALL hold against a concurrent transfer: either saving succeeds and the storage transfer gets 409 with code in_use, or transfer succeeds and saving gets 400 with code invalid_request without a config change.

Every successful save, including a switch or reassignment with retained history, SHALL answer 200 with the saved config and its current cleanup counts. Clearing storage SHALL preserve the retained catalog and SHALL NOT itself enqueue that catalog for deletion. Initial assignment without retained history SHALL create no new cleanup work. These acknowledgements SHALL NOT change storage ownership, secret preservation or concurrent-transfer guarantees.

#### Scenario: The request names an existing storage

- **WHEN** a user saves a config naming a storage of the database's workspace
- **THEN** the config names that storage and its settings, name and credentials stay unchanged
- **AND** the response is 200 with the acknowledged config, including for a switch or reassignment with retained history

#### Scenario: The request carries changed storage fields

- **WHEN** a user saves a config with an existing storage identifier and extra fields changing that storage's name or credentials
- **THEN** the extra fields are ignored and the storage retains its name and credentials
- **AND** the response is 200 with the config, including for a switch or reassignment with retained history

#### Scenario: The request carries a storage without an identifier

- **WHEN** a user saves a config with a storage object in place of an identifier
- **THEN** the configuration names no storage and no storage is created

#### Scenario: Clearing storage preserves history until reassignment

- **WHEN** a user clears the configured storage while the database has backups in A
- **THEN** the response is 200 with a config naming no storage, the old backup history remains and no new cleanup work is created
- **AND** a later assignment of storage, including A again, applies the reassignment cleanup rules

#### Scenario: The request carries an unknown storage

- **WHEN** a user saves a config with an identifier that no storage has
- **THEN** the response is 400 with code invalid_request, the config is unchanged and no storage is created

#### Scenario: The request names a storage of another workspace

- **WHEN** a user saves a config naming a storage of another workspace
- **THEN** the response is 400 with code invalid_request and the config retains its storage

#### Scenario: A save races a transfer of its storage

- **WHEN** a user saves a config naming S while another request transfers S to another workspace
- **THEN** exactly one succeeds: either the save answers 200, the transfer answers 409 with code in_use and S stays in the database's workspace; or transfer answers 204 and saving answers 400 with code invalid_request, leaving the previous config

#### Scenario: A database transfer commits before a configuration save

- **WHEN** a database transfer commits while a config save waits for its database and config locks
- **THEN** the save checks access and storage ownership against the database's workspace read under those locks
- **AND** a former-workspace storage or a caller without access to the current workspace is refused without changing deletion obligations
- **AND** new cleanup is attributed to the database and the workspace authorized for that committed change

### Requirement: Moving a logical backup configuration to another storage deletes its logical backups

When a user switches a logical backup configuration from one configured storage to another, including during a workspace transfer, the new storage, removal of the database's old logical backups and durable file-deletion requests SHALL commit together. A configuration save SHALL answer 200 with the saved configuration and current cleanup counts; a workspace transfer SHALL answer 204 after commit. Remote deletion SHALL run in the background without delaying the switch.

These rules SHALL also apply when the configuration currently names no storage but still has logical backups from an earlier assignment. Each deletion request SHALL address the backup's actual storage. An initial assignment with no retained logical backups SHALL remain an ordinary config save without new cleanup work.

When a logical backup or a restore using those backups is running, the switch SHALL get 409 with code in_use and SHALL change nothing. When the catalog removal or recording deletion obligations fails, the request SHALL get 500 with code internal and SHALL leave the previous configuration, catalog and deletion obligations unchanged. Remote cleanup failure after commit SHALL NOT revert or fail the already accepted switch.

#### Scenario: Backups on the old storage

- **WHEN** a database has logical backups in A and a user switches its logical configuration to B
- **THEN** the configuration save response is 200 with the saved configuration, the config names B and the old backups leave the list
- **AND** deletion of their files from A is requested

#### Scenario: Logical storage is reassigned after being cleared

- **WHEN** a logical configuration used A, now names no storage, retains backups in A, and is assigned B
- **THEN** B, removal of the old backups and durable deletion requests against A commit together, and the configuration save response is 200 with the saved configuration

#### Scenario: Logical reassignment conflicts with active work

- **WHEN** a logical configuration names no storage but retains backups in A, and a backup or restore is active when B is assigned
- **THEN** the reassignment is refused with 409 and code in_use without changing the configuration or backups

#### Scenario: Logical reassignment cannot persist cleanup

- **WHEN** a logical configuration names no storage but retains backups in A, and assigning B fails to persist the required cleanup
- **THEN** the configuration still names no storage and the old backups remain listed, without changing deletion obligations

#### Scenario: Initial logical assignment has no history

- **WHEN** a logical configuration names no storage, has no retained backups and is first assigned B through a config save
- **THEN** the response is 200 with the config, B is assigned and no new cleanup work is created

#### Scenario: A backup is running

- **WHEN** a logical backup runs and a user switches its configuration to another storage
- **THEN** the response is 409 with code in_use, the config still names the old storage and its backups stay listed

#### Scenario: A restore is running

- **WHEN** a restore reads one of the old logical backups and the user switches storage
- **THEN** the response is 409 with code in_use and the config and backups are unchanged

#### Scenario: The storage does not change

- **WHEN** a user saves a logical configuration with its current storage
- **THEN** no backup is removed and no new cleanup work is created

#### Scenario: The removal fails

- **WHEN** the old catalog removal or any required deletion request cannot be persisted
- **THEN** the response is 500 with code internal and the previous configuration and backup list remain

#### Scenario: Old storage is unreachable

- **WHEN** a user switches a logical configuration from unreachable A to B
- **THEN** B becomes active without contacting A and the queued files remain retryable independently of the committed change

### Requirement: A storage switch warns about deletion and exposes its cleanup outcome

Before confirming a storage change in backup settings or a workspace transfer, the interface SHALL explain that existing backups become unavailable, old files are queued for deletion, the new storage becomes active immediately and cleanup runs in the background. It SHALL NOT describe this as copying or migrating the old backups. It SHALL warn that unreachable storage delays cleanup and explicitly deleting that storage abandons remaining cleanup.

After the change, the selected database's Storage section SHALL show only its currently visible outstanding cleanup, with remaining-file counts and a retry warning while files cannot yet be removed. The notice SHALL disappear when there are no outstanding files in that authorized scope. There SHALL be no empty history, completed-switch history or workspace-level cleanup banner. Missing files SHALL count as removed only after confirmed absence; explicit abandonment SHALL NOT be presented as a saved successful cleanup result.

A reload SHALL recover current cleanup. A failed status read SHALL preserve the last known notice, provide a read retry and SHALL NOT resubmit the configuration change or transfer. Further switches SHALL NOT make earlier queued files address a newer storage, and an old writer's later file SHALL still be queued even if it increases the outstanding count. No fixed total or per-switch percentage SHALL be promised.

#### Scenario: User confirms a switch

- **WHEN** a user chooses another storage in backup settings or a transfer
- **THEN** the deletion warning appears before submission, and the new storage becomes active after commit
- **AND** any visible outstanding cleanup appears beside Storage when that database's settings are shown

#### Scenario: Cleanup completes

- **WHEN** the worker confirms absence for every outstanding file in the visible scope
- **THEN** its current count becomes zero and the notice disappears without creating a completed-history entry

#### Scenario: Cleanup gives up

- **WHEN** old storage remains unavailable through five failures or longer than 24 hours
- **THEN** outstanding files remain queued, the notice identifies delayed cleanup and retries continue with backoff

#### Scenario: Page is reloaded

- **WHEN** a user reloads database settings during cleanup
- **THEN** the current outstanding count is read again and the notice resumes beside Storage

#### Scenario: A status request fails

- **WHEN** a background status read fails while the cleanup notice is visible
- **THEN** the last known count remains with a read-retry action, without claiming completion or resubmitting a change

#### Scenario: Switch with no old artifacts

- **WHEN** a storage change creates no outstanding file work and no earlier files remain outstanding in the authorized database/workspace scope
- **THEN** its acknowledgement contains no switch resource and the Storage section shows no cleanup notice

#### Scenario: A later switch adds no files while earlier cleanup is pending

- **WHEN** A-to-B cleanup remains outstanding in the authorized database/workspace scope and a later B-to-C change adds no file work
- **THEN** the acknowledgement includes the aggregate outstanding count and the earlier cleanup notice remains visible beside Storage
- **AND** the notice disappears only after that authorized scope has no outstanding files

#### Scenario: Two changes overlap

- **WHEN** a database changes from A to B and then to C while files remain queued in A
- **THEN** each file is still deleted only from its actual original storage, and the interface exposes current outstanding work rather than separate switch histories

#### Scenario: A database is deleted during cleanup

- **WHEN** a database is deleted with files still queued against a surviving storage
- **THEN** their cleanup continues independently of the deleted database and there is no database switch history to retain

### Requirement: A finished physical backup that lost its storage is not published

A physical backup whose execution succeeds after its storage assignment changed SHALL NOT be recorded as completed, and its database's last backup time SHALL NOT move. Its files SHALL be handed to the deletion of stored files, and it SHALL NOT send a success or a failure notification, whatever state its record is in. A record the storage change has already removed SHALL stay removed, and a record it has already ended SHALL keep its status and fail message. A record still running SHALL end as cancelled when the storage change's stop reaches it, as for a backup the change stopped earlier.

#### Scenario: Execution finishes just before the stop reaches the backup

- **WHEN** a physical backup still recorded as running finishes its execution after the database's storage was cleared and before the stop is recorded
- **THEN** the backup is not recorded as completed, its files are requested for deletion, the last backup time does not move and no notification is sent
- **AND** once the stop is recorded the backup is listed as cancelled

#### Scenario: The storage change already removed the backup

- **WHEN** a physical backup finishes its execution after a storage switch removed its record
- **THEN** no record of the backup returns to the list, its files are requested for deletion and no notification is sent

#### Scenario: The storage change already ended the backup

- **WHEN** a physical backup finishes its execution after the storage change recorded it as cancelled
- **THEN** it stays cancelled with its fail message unchanged, its files are requested for deletion and no notification is sent
### Requirement: Current cleanup counts obey database and workspace access

Reading current cleanup SHALL require a session and permission to view the database's current workspace. An instance administrator SHALL retain existing workspace-access privileges. A non-admin outside the workspace SHALL receive 404 with code `not_found`; no session SHALL receive 401 with code `unauthenticated`.

Counts SHALL include only that database's obligations attributed to its current workspace. Later transfer SHALL NOT move earlier cleanup attribution or reveal previous-workspace cleanup to new viewers. Responses SHALL contain aggregate pending and failed-file counts only, without provider errors, file names, credentials, signed addresses or inaccessible storage identities. Duplicate deletion requests SHALL NOT move a file's original attribution to a later workspace. Source-storage deletion SHALL continue to follow that storage's own editing permissions.

#### Scenario: Member of another workspace

- **WHEN** a non-admin with no membership in the database's current workspace requests its cleanup counts
- **THEN** the response is 404 with code `not_found` and no counts are exposed

#### Scenario: Listing by storage does not grant database access

- **WHEN** a non-admin can view a storage but cannot view the current workspace of a listed database
- **THEN** listing configurations by that storage returns 404 with code `not_found` without exposing cleanup counts

#### Scenario: Instance admin without workspace membership

- **WHEN** an instance admin reads a database's configuration without workspace membership
- **THEN** its safe current cleanup counts are available under the existing administrator privileges

#### Scenario: No session

- **WHEN** a caller without a session reads cleanup counts
- **THEN** the response is 401 with code `unauthenticated`

#### Scenario: A database is transferred again

- **WHEN** a database with work attributed to workspace B moves later to workspace C
- **THEN** that earlier work continues without being exposed through C's configuration summary

#### Scenario: A later request repeats an earlier file

- **WHEN** a new workspace's change requests deletion of a file already attributed to an earlier workspace
- **THEN** the same obligation is retained without moving its original attribution or revealing it to the new scope
