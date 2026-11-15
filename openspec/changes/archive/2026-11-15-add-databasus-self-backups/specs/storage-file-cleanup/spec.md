## MODIFIED Requirements

### Requirement: A stored file is kept only when a transaction claims it

The system SHALL record a pending deletion for a unique file name before sending bytes to storage, and SHALL remove that record only as part of the transaction that publishes the file as part of a backup. Every other outcome SHALL leave the record in place, so the file is eventually removed.

The explicit lost-server recovery procedure in `databasus-recovery` is an exception for copied source-instance obligations and interrupted-operation normalization. It SHALL not replay those stale obligations against surviving storages and SHALL explain the resulting manual-cleanup limitation. This exception SHALL NOT apply to an ordinary restart, upgrade or failed live operation; their cleanup obligations remain durable and retryable.

#### Scenario: File and backup are published together

- **WHEN** storage accepts a file and the owning backup catalog change commits
- **THEN** the system stops tracking the file for deletion
- **AND** the file remains in storage

#### Scenario: Catalog publication fails after upload

- **WHEN** storage accepts a file but the owning catalog transaction fails
- **THEN** the system keeps the catalog change uncommitted
- **AND** it eventually removes the uploaded file

#### Scenario: Application stops during a write

- **WHEN** the application stops after recording a pending deletion but before publishing the file
- **THEN** the record survives the restart
- **AND** the system eventually removes every storage artifact created by that write

#### Scenario: Provider rejects the write

- **WHEN** a provider returns an error partway through a write
- **THEN** the file becomes eligible for deletion immediately
- **AND** the caller receives the provider error

#### Scenario: Publication arrives too late

- **WHEN** a caller tries to publish a file after cleanup has already taken it
- **THEN** the publishing transaction fails instead of committing
- **AND** the backup is not recorded as successful

### Requirement: Every failed backup path discards its files

The system SHALL leave the pending deletion in place for every file produced by an unsuccessful logical or physical backup operation, even when storage accepted the file before another stage failed. The owning backup flow SHALL identify the base file and each metadata, manifest, WAL or history sidecar that belongs to that operation.

The explicit lost-server recovery procedure in `databasus-recovery` is an exception for copied source-instance obligations and interrupted-operation normalization. It SHALL not replay those stale obligations against surviving storages and SHALL explain the resulting manual-cleanup limitation. This exception SHALL NOT apply to an ordinary restart, upgrade or failed live operation; their cleanup obligations remain durable and retryable.

#### Scenario: Logical dump fails after producing output

- **WHEN** a logical backup tool produces bytes and then exits with an error
- **THEN** the backup is marked failed
- **AND** the partial or completed storage file is eventually absent

#### Scenario: Physical backup stream fails after producing output

- **WHEN** a physical FULL or INCR stream writes bytes and then ends unsuccessfully
- **THEN** the backup is not published as completed
- **AND** its base file, manifest and metadata files are eventually absent

#### Scenario: Sidecar creation fails

- **WHEN** a base artifact is stored but its required metadata or manifest cannot be produced or published
- **THEN** every file produced for that attempt is eventually absent

#### Scenario: WAL or history catalog commit fails

- **WHEN** a WAL segment or timeline history file reaches storage but its catalog transaction does not commit
- **THEN** the artifact and its sidecar are eventually absent
- **AND** a later retry uses a different per-attempt object name

#### Scenario: Active backup is canceled

- **WHEN** a user or scheduler cancels a backup that may still be writing
- **THEN** every file from the canceled attempt is eventually absent
- **AND** no file is removed while its write is still running

### Requirement: Catalog removal and deletion requests are atomic

The system SHALL commit the removal or terminal failure of catalog records together with durable deletion requests for all files those records own. It SHALL NOT perform remote storage deletion while holding that catalog transaction open.

The explicit lost-server recovery procedure in `databasus-recovery` is an exception for copied source-instance obligations and interrupted-operation normalization. It SHALL not replay those stale obligations against surviving storages and SHALL explain the resulting manual-cleanup limitation. This exception SHALL NOT apply to an ordinary restart, upgrade or failed live operation; their cleanup obligations remain durable and retryable.

#### Scenario: Backup is removed while storage is unavailable

- **WHEN** retention or a user removes a backup while its storage provider is unavailable
- **THEN** the catalog removal and its deletion requests commit together
- **AND** the system retries the remote deletion after storage becomes available

#### Scenario: Deletion request cannot be persisted

- **WHEN** the system cannot persist every required deletion request
- **THEN** the related catalog removal or terminal state change rolls back
- **AND** the catalog continues to identify the files that still exist

#### Scenario: A cascade rolls back after part of it succeeded

- **WHEN** a cascading removal fails partway and its transaction rolls back
- **THEN** every catalog record it touched is restored
- **AND** none of the files those records name have been removed from storage

#### Scenario: A database with physical backups is deleted

- **WHEN** database deletion cascades FULL, INCR, WAL, history, manifest or metadata catalog rows
- **THEN** exact deletion requests for every referenced artifact are committed before the cascade removes the rows that name them
- **AND** a failure to persist them stops the database removal

#### Scenario: Deletion is requested more than once

- **WHEN** the same storage identifier and file name are requested for deletion repeatedly
- **THEN** the system retains one cleanup obligation
- **AND** each request succeeds without creating competing deletions

### Requirement: Failed deletions remain retryable and observable

The system SHALL retry failed deletions with exponential delay, a maximum delay and jitter, grown from the number of deletion attempts alone. It SHALL persist that attempt count, the next attempt time and a sanitized bounded error message. Each worker pass SHALL have a stable job_name and a fresh job_id, and SHALL report how many obligations are pending, how many are overdue and the age of the oldest one. Every deletion attempt it logs SHALL identify the job, storage, file, attempt and next retry without exposing credentials.

Every deletion obligation, including one created by a storage change, SHALL remain retryable until the file is confirmed absent or the user explicitly abandons cleanup by deleting its storage. Five failures or 24 hours SHALL NOT stop retries or remove an obligation. A failed attempt SHALL remain pending, not become a terminal successful or unresolved operation result.

The explicit lost-server recovery procedure in `databasus-recovery` is an exception for copied source-instance obligations and interrupted-operation normalization. It SHALL not replay those stale obligations against surviving storages and SHALL explain the resulting manual-cleanup limitation. This exception SHALL NOT apply to an ordinary restart, upgrade or failed live operation; their cleanup obligations remain durable and retryable.

#### Scenario: Storage outage persists

- **WHEN** repeated ordinary deletion attempts fail because storage is unavailable
- **THEN** each failure remains recorded and retries continue at a bounded cadence

#### Scenario: Storage credentials cannot delete

- **WHEN** a provider rejects ordinary deletion because the configured credentials lack permission
- **THEN** the recorded sanitized error identifies the permission failure and the obligation stays pending

#### Scenario: A file is requested for deletion more than once before any attempt

- **WHEN** several callers request deletion of the same file before the worker attempts it
- **THEN** the file carries one obligation and its first failed attempt waits the base delay because a request is not an attempt

#### Scenario: Storage recovers

- **WHEN** a later deletion attempt succeeds after earlier failures
- **THEN** the remote file is absent and the cleanup obligation is removed

#### Scenario: Provider error contains sensitive material

- **WHEN** a provider returns an error containing credentials or a signed address
- **THEN** persisted state and logs omit the sensitive value

#### Scenario: A deletion attempt is logged

- **WHEN** the worker attempts or reschedules a deletion
- **THEN** its logs retain job_name, job_id, storage_id, file_name, attempt count and next retry time after centralized redaction

#### Scenario: An operator asks what cleanup is outstanding

- **WHEN** cleanup obligations remain
- **THEN** each worker pass reports pending count, overdue count and oldest age, and recorded state identifies the failing storage and file

### Requirement: Deleting a storage reports the files it leaves behind

After ordinary authorization, deleting a selected Databasus backup destination SHALL return 409 with code `databasus_backup_in_use` before checking database-backup references or cleanup, including when those blockers also apply. Otherwise a database backup configuration or retained database backup SHALL cause 409 with code `in_use`. Neither refusal SHALL change anything, even when the request explicitly abandons cleanup. The stored references SHALL also prevent accidental cascading loss of those configurations and backup records. Historical Databasus backup copies alone SHALL not prevent deletion after deselection; their storage reference SHALL be detached and marked unavailable without automatically deleting their remote files.

For an otherwise unused storage, the server SHALL check all outstanding file obligations, including delayed, failed and currently claimed work, when processing deletion. Without explicit abandonment, outstanding work SHALL return 409 with code `cleanup_pending`, preserving the storage, credentials and obligations and making no provider deletion calls. A stale earlier read SHALL NOT waive this check.

After that refusal the interface SHALL explain that cleanup is still running, recommend waiting, and warn that deleting now stops automatic cleanup and leaves any remaining files for manual removal. It SHALL offer Wait and Delete anyway. Only the explicit Delete anyway action SHALL acknowledge abandonment. Wait SHALL preserve the storage and cleanup work.

An authorized deletion with `shouldAbandonCleanup=true` SHALL remove the unused storage setting and all its outstanding obligations atomically and return 204, without a final remote drain or deleting the external bucket or directory. The readable audit log SHALL record abandonment, the number of outstanding files and their names up to a fixed limit, without claiming those files were confirmed present or deleted. A failed transaction SHALL preserve the storage and every obligation. A deletion with no queued files SHALL succeed without abandonment.

Queue registration and storage deletion SHALL be ordered so a new obligation is either included in the server's check or cannot be registered against the removed storage. An attempt already in progress can finish, but its late answer SHALL NOT recreate an abandoned obligation, undo storage deletion or start another attempt for that storage.

#### Scenario: Storage is deleted while backups still reference it

- **WHEN** a user deletes a storage that still has database backup records but is not a selected Databasus backup destination, with or without explicit abandonment
- **THEN** the response is 409 with code `in_use`, every record and obligation remains, and no file is removed by the request

#### Scenario: Storage is deleted while a backup configuration uses it

- **WHEN** a user explicitly abandons cleanup while deleting a storage named by a database backup configuration and not selected as a Databasus backup destination
- **THEN** the response is 409 with code `in_use`, and the configuration, storage and obligations remain

#### Scenario: Storage row deleted directly in the database

- **WHEN** a storage is removed directly while a database backup configuration or retained database backup references it
- **THEN** the stored references refuse the deletion rather than cascading away the records that identify its files

#### Scenario: Storage with reachable pending cleanup is deleted

- **WHEN** an otherwise unused storage has outstanding files and deletion does not acknowledge abandonment
- **THEN** the response is 409 with code `cleanup_pending`, the warning is offered and nothing changes or contacts the provider

#### Scenario: User chooses to wait

- **WHEN** a user chooses Wait in the cleanup warning
- **THEN** the warning closes without an abandoning deletion request, and background cleanup continues

#### Scenario: User explicitly deletes during cleanup

- **WHEN** an authorized user chooses Delete anyway for an unused storage with queued files
- **THEN** the storage and its obligations are removed together, the response is 204, and abandonment is recorded in the audit log
- **AND** the request does not drain files or delete the external bucket or directory

#### Scenario: Storage with an unreachable provider is deleted

- **WHEN** the provider is unreachable and the user acknowledges abandonment
- **THEN** deletion succeeds without waiting for the provider, and the audit records the outstanding work

#### Scenario: A previously empty queue receives work

- **WHEN** an obligation is registered after a client read an empty queue but before its ordinary deletion request is checked
- **THEN** deletion returns `cleanup_pending` without losing the new work

#### Scenario: A worker returns after abandonment

- **WHEN** a claimed provider attempt answers after explicit deletion committed
- **THEN** it does not recreate its queue entry or schedule another attempt

#### Scenario: Storage deletion fails after removing obligations

- **WHEN** the storage-deletion transaction fails after changing its obligations
- **THEN** the storage, credentials, obligations and absence of a successful abandonment audit event are restored together

#### Scenario: An unused storage has no obligations

- **WHEN** a user deletes an unused storage whose queue is empty without acknowledging abandonment
- **THEN** deletion returns 204 without contacting the provider

#### Scenario: Storage is deleted while a write to it is still running

- **WHEN** an otherwise unused storage has a protected writer and its user requests deletion
- **THEN** deletion without explicit abandonment returns `cleanup_pending` without touching the file
- **AND** explicitly acknowledged deletion can remove the storage and its obligations while warning that the writer's files may remain

#### Scenario: Workspace containing such a storage is deleted

- **WHEN** a workspace without databases, selected Databasus backup destinations or retained database-backup references is deleted while one of its storages has outstanding cleanup
- **THEN** the whole deletion returns `cleanup_pending` without implicitly abandoning files or exposing a raw constraint error

#### Scenario: Databasus destination and database references both prevent deletion

- **WHEN** an authorized editor deletes a selected Databasus backup destination that also has retained database backups and queued cleanup
- **THEN** the response is 409 with code `databasus_backup_in_use`, no record changes and no provider deletion is called
- **AND** after deselection, the database-backup references still cause `in_use` before cleanup is considered
