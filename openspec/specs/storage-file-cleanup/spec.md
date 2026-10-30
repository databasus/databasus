# storage-file-cleanup Specification

## Purpose
Defines how Databasus publishes backup files and eventually removes every file that belongs to a failed, canceled, interrupted or deleted backup operation.

## Requirements

### Requirement: A stored file is kept only when a transaction claims it

The system SHALL record a pending deletion for a unique file name before sending bytes to storage, and SHALL remove that record only as part of the transaction that publishes the file as part of a backup. Every other outcome SHALL leave the record in place, so the file is eventually removed.

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

### Requirement: A deletion request is always accepted

Requesting deletion of a file SHALL succeed whatever state that file is in, including while its writer is still running. A request made during an active write SHALL take the file away from that write, so the write cannot be published afterwards.

#### Scenario: Deletion is requested during an active write

- **WHEN** a catalog transaction requests deletion for a file whose writer has not stopped
- **THEN** the request succeeds and the transaction can commit
- **AND** the file is eventually absent

#### Scenario: A cascading removal meets an in-flight upload

- **WHEN** a cascading catalog removal covers a WAL segment that is uploading at that moment
- **THEN** the removal commits without waiting for the upload
- **AND** the uploaded object is eventually absent

#### Scenario: A write completes after its file was requested for deletion

- **WHEN** a write finishes successfully after deletion was requested for the same file
- **THEN** the write reports that the file was taken for deletion
- **AND** its caller cannot publish that file as part of a backup

### Requirement: Cleanup never removes a file that is still being written or already published

The system SHALL use an immutable, unique object name for each independent write attempt. It SHALL NOT delete a file whose write is still running in the application process, and it SHALL NOT start a write for a file name that already carries a pending deletion. A deletion attempt that has taken a file SHALL invalidate any outstanding permission to publish it, and an attempt that outlived its own timeout SHALL NOT overwrite the result of a newer attempt.

#### Scenario: A long write outlives the cleanup interval

- **WHEN** a write runs for longer than the cleanup worker's scan interval
- **THEN** the worker does not delete that file while the write is running
- **AND** the write completes normally

#### Scenario: Cleanup takes a file that was uploaded but never published

- **WHEN** cleanup takes a file whose upload finished but whose catalog transaction never committed
- **THEN** any later attempt to publish that file fails
- **AND** the file is removed from storage

#### Scenario: A file name already carries a pending deletion

- **WHEN** a write starts for a file name whose deletion was already requested
- **THEN** the write fails without sending bytes to the provider
- **AND** the existing cleanup obligation keeps its own schedule

#### Scenario: An attempt outlives its own timeout

- **WHEN** a deletion attempt finishes after its timeout and the file was already picked up again
- **THEN** the late attempt does not reschedule or overwrite the newer attempt's state

#### Scenario: Cleanup worker stops after remote deletion

- **WHEN** the worker stops after deleting the remote file but before recording completion
- **THEN** a later worker repeats the idempotent deletion
- **AND** the cleanup obligation is removed only after deletion is confirmed

### Requirement: Deletion removes the complete logical file

Every storage provider SHALL treat deletion as removal of all physical state derived from the requested logical file name. Missing state SHALL count as success only when the provider confirms that it is absent; connection, authorization and lookup failures SHALL remain errors, except where a protocol gives one reply for both absence and refusal.

#### Scenario: Provider write leaves partial state

- **WHEN** a write fails after creating a temporary file, partial remote file, staged block, chunk object, manifest or multipart upload
- **THEN** deletion removes every piece derived from the logical file name

#### Scenario: S3 manifest is missing

- **WHEN** an interrupted chunked S3 write leaves completed chunk objects or incomplete multipart uploads without a manifest
- **THEN** deletion finds and removes only the exact chunks and uploads derived from that logical file name
- **AND** similarly prefixed files remain untouched

#### Scenario: File is already absent

- **WHEN** the provider confirms that every representation of the requested file is missing
- **THEN** deletion succeeds

#### Scenario: Provider lookup fails

- **WHEN** a connection, permission or other lookup error prevents the provider from confirming absence
- **THEN** deletion returns an error
- **AND** the cleanup obligation remains pending

#### Scenario: A protocol cannot separate absence from refusal

- **WHEN** a provider's only reply for a missing file is also its reply for a refused one
- **THEN** deletion treats that reply as absence
- **AND** every other failure the protocol can distinguish is still an error

### Requirement: Failed deletions remain retryable and observable

The system SHALL retry failed deletions with exponential delay, a maximum delay and jitter, grown from the number of deletion attempts alone. It SHALL persist that attempt count, the next attempt time and a sanitized bounded error message. Each worker pass SHALL have a stable job_name and a fresh job_id, and SHALL report how many obligations are pending, how many are overdue and the age of the oldest one. Every deletion attempt it logs SHALL identify the job, storage, file, attempt and next retry without exposing credentials.

Ordinary deletion obligations SHALL remain retryable until the file is confirmed absent or storage deletion reports the leftovers. Cleanup explicitly owned by a storage switch SHALL instead obey the finite policy below. Exhausting switch cleanup SHALL NOT stop retries of an independent ordinary obligation for the same file.

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

The system SHALL reject deleting a storage configuration while a backup configuration uses it or backup records still reference it, because those records are the only place its file names exist. The database itself SHALL refuse the removal, so that a missing check in the application cannot delete a backup configuration or a backup record together with the storage. A rejected deletion SHALL get `409` with `code` `in_use`, SHALL change nothing and SHALL remove no file from the provider. When nothing references the storage, deletion SHALL attempt the storage's outstanding cleanup and SHALL then succeed. The cleanup SHALL skip a file whose write is still running in the application process. Deletion SHALL record how many files it could not remove or skipped, and their names up to a fixed limit. That record SHALL be somewhere the user can read. Outstanding cleanup SHALL NOT make a storage or a workspace permanently undeletable.

#### Scenario: Storage is deleted while backups still reference it

- **WHEN** a user deletes a storage that still has backup records
- **THEN** the response is `409` with `code` `in_use`
- **AND** the backups stay readable and removable through their own deletion path
- **AND** no file is removed from the provider, the files of the storage's pending cleanup included

#### Scenario: Storage is deleted while a backup configuration uses it

- **WHEN** a user deletes a storage that the backup configuration of a database names
- **THEN** the response is `409` with `code` `in_use`
- **AND** the database keeps its backup configuration with that storage, and no file is removed from the provider

#### Scenario: Storage row deleted directly in the database

- **WHEN** the row of a storage is deleted directly in the database while a backup configuration, a logical backup record or a physical backup record references it
- **THEN** the database refuses the statement
- **AND** the configuration and the records remain

#### Scenario: Storage with reachable pending cleanup is deleted

- **WHEN** a storage with no backup records, no backup configuration using it and reachable pending cleanup is deleted
- **THEN** the pending files are removed from the provider before the deletion commits, so a reader still sees the storage until its files are gone
- **AND** deletion succeeds

#### Scenario: Storage with an unreachable provider is deleted

- **WHEN** a storage is deleted while its provider cannot be reached
- **THEN** deletion still succeeds
- **AND** the audit log names the storage, how many files were left behind and their names up to a fixed limit

#### Scenario: Storage is deleted while a write to it is still running

- **WHEN** a storage that nothing references is deleted while a write of a file to it is still running in the application
- **THEN** deletion succeeds without removing that file
- **AND** the audit log counts that file among the files left behind and names it

#### Scenario: Workspace containing such a storage is deleted

- **WHEN** a workspace is deleted whose storages still hold pending cleanup
- **THEN** deletion follows the same path as deleting each storage
- **AND** the caller never sees a raw database constraint error

### Requirement: Cleanup state contains no provider credentials

Durable cleanup state SHALL contain file identity, scheduling data, retry data and sanitized errors only. It SHALL NOT duplicate provider credentials, tokens, private keys, signed addresses or complete provider configuration.

#### Scenario: Cleanup state is inspected at rest

- **WHEN** an operator or database tool reads durable cleanup state
- **THEN** the record contains no provider authentication material or reusable signed address

### Requirement: Storage-switch cleanup has a finite durable outcome

A storage switch SHALL durably associate every artifact to delete with its operation and original storage before removing the catalog that names it. A successful deletion or confirmed absence SHALL count once as deleted. At most five failed attempts per artifact SHALL be made for switch-owned cleanup; a 24-hour deadline from acceptance SHALL also end cleanup for unresolved artifacts. Each provider attempt SHALL be bounded by its own timeout and the operation's remaining deadline.

An artifact that exhausts its attempts, whose original storage configuration is deleted, or whose deadline expires SHALL be recorded as unresolved and SHALL NOT be retried on behalf of that switch. The operation SHALL end when every artifact has a deleted or unresolved outcome. It SHALL be complete with remainders when any artifact is unresolved. Terminal outcomes and counts SHALL remain unchanged if an independent obligation later deletes a remaining file. Restarting the application SHALL preserve deadlines, failed-attempt counts and outcomes. A duplicate or late attempt SHALL NOT overwrite a newer outcome, extend the deadline or count an artifact twice.

The operation's artifact set SHALL contain the old catalog's files and known unfinished writes recorded at acceptance. The result SHALL describe that captured set, without claiming that unknown files in the provider were discovered or deleted. An old write that creates another file after acceptance SHALL remain subject to its own recorded cleanup obligation and SHALL NOT extend the switch's artifact set or deadline.

Files of a running write SHALL remain protected. If an old writer outlives the cleanup deadline, its known artifacts SHALL be reported unresolved rather than deleted while still being written. A later independent cleanup obligation remains governed by its own policy.

#### Scenario: Five deletion failures

- **WHEN** deletion of a switch-owned artifact fails five times
- **THEN** that artifact is recorded unresolved and receives no sixth attempt on behalf of the switch

#### Scenario: A repeated request supersedes an active attempt's queue generation

- **WHEN** a deletion request changes the pending queue's generation during a failing provider call and the worker retries before the old lease expires
- **THEN** the failed call counts once toward its attributed switch's five-attempt limit
- **AND** a second worker cannot overwrite the unsettled attribution while that call is still active

#### Scenario: Deadline is reached

- **WHEN** 24 hours have elapsed since acceptance and some switch-owned artifacts remain pending
- **THEN** the next deletion-worker pass records those artifacts unresolved and finishes the operation without further provider calls

#### Scenario: Application restarts during cleanup

- **WHEN** the application restarts after three failed attempts
- **THEN** that artifact has at most two failed attempts left and the original deadline remains

#### Scenario: Old storage configuration is deleted

- **WHEN** the previous storage configuration is deleted while its switch cleanup is pending
- **THEN** confirmed deletions retain their outcome and all other associated artifacts become unresolved
- **AND** deleting that storage or workspace does not leave an eternally pending switch

#### Scenario: An ordinary obligation also exists

- **WHEN** a switch gives up on a file that an independent ordinary cleanup obligation also names
- **THEN** the ordinary obligation keeps its retry policy and the switch retains its incomplete outcome

#### Scenario: An independent obligation later deletes a remaining file

- **WHEN** an independent obligation confirms deletion after the switch recorded that file as unresolved
- **THEN** the switch keeps its terminal incomplete status, original completion time and artifact counts

#### Scenario: An old writer creates another file after acceptance

- **WHEN** an old writer creates a file after the switch captured its cleanup artifacts
- **THEN** the file cannot enter the new backup history and its independent cleanup obligation remains
- **AND** the switch's artifact total and deadline stay unchanged

#### Scenario: Another application instance cleans during an active upload

- **WHEN** one application instance is still uploading a captured artifact and the accepting instance runs cleanup before the deadline
- **THEN** cleanup does not call the provider to delete that artifact or count an attempt, and the artifact remains pending
- **AND** after the upload finishes, cleanup can delete the discarded file and count its confirmed absence once

#### Scenario: A deleted backup writer drains after storage is cleared

- **WHEN** a running backup is deleted, the storage selection is cleared, and a storage switch is accepted while its writer still drains
- **THEN** known unfinished files from the last selected storage are captured with their original storage even without surviving backup catalog rows
- **AND** clearing the selection or restarting the application does not erase that attribution

#### Scenario: A late provider result arrives

- **WHEN** an attempt finishes after its lease or operation deadline and a newer state has already committed
- **THEN** it cannot change the terminal outcome or double-count the artifact

#### Scenario: Another switch occurs before cleanup finishes

- **WHEN** the database switches from A to B and then from B to C while cleanup of A is pending
- **THEN** cleanup of A continues against A, cleanup of B addresses only B, and neither operation deletes C's files
