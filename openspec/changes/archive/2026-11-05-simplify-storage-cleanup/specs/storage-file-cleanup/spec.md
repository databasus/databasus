## MODIFIED Requirements

### Requirement: Failed deletions remain retryable and observable

The system SHALL retry failed deletions with exponential delay, a maximum delay and jitter, grown from the number of deletion attempts alone. It SHALL persist that attempt count, the next attempt time and a sanitized bounded error message. Each worker pass SHALL have a stable job_name and a fresh job_id, and SHALL report how many obligations are pending, how many are overdue and the age of the oldest one. Every deletion attempt it logs SHALL identify the job, storage, file, attempt and next retry without exposing credentials.

Every deletion obligation, including one created by a storage change, SHALL remain retryable until the file is confirmed absent or the user explicitly abandons cleanup by deleting its storage. Five failures or 24 hours SHALL NOT stop retries or remove an obligation. A failed attempt SHALL remain pending, not become a terminal successful or unresolved operation result.

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

Deleting a storage referenced by a backup configuration or retained backup SHALL return 409 with code `in_use` and SHALL change nothing, even when the request explicitly abandons cleanup. The stored references SHALL also prevent accidental cascading loss of those configurations and backup records.

For an otherwise unused storage, the server SHALL check all outstanding file obligations, including delayed, failed and currently claimed work, when processing deletion. Without explicit abandonment, outstanding work SHALL return 409 with code `cleanup_pending`, preserving the storage, credentials and obligations and making no provider deletion calls. A stale earlier read SHALL NOT waive this check.

After that refusal the interface SHALL explain that cleanup is still running, recommend waiting, and warn that deleting now stops automatic cleanup and leaves any remaining files for manual removal. It SHALL offer Wait and Delete anyway. Only the explicit Delete anyway action SHALL acknowledge abandonment. Wait SHALL preserve the storage and cleanup work.

An authorized deletion with `shouldAbandonCleanup=true` SHALL remove the unused storage setting and all its outstanding obligations atomically and return 204, without a final remote drain or deleting the external bucket or directory. The readable audit log SHALL record abandonment, the number of outstanding files and their names up to a fixed limit, without claiming those files were confirmed present or deleted. A failed transaction SHALL preserve the storage and every obligation. A deletion with no queued files SHALL succeed without abandonment.

Queue registration and storage deletion SHALL be ordered so a new obligation is either included in the server's check or cannot be registered against the removed storage. An attempt already in progress can finish, but its late answer SHALL NOT recreate an abandoned obligation, undo storage deletion or start another attempt for that storage.

#### Scenario: Storage is deleted while backups still reference it

- **WHEN** a user deletes a storage that still has backup records, with or without explicit abandonment
- **THEN** the response is 409 with code `in_use`, every record and obligation remains, and no file is removed by the request

#### Scenario: Storage is deleted while a backup configuration uses it

- **WHEN** a user explicitly abandons cleanup while deleting a storage named by a backup configuration
- **THEN** the response is 409 with code `in_use`, and the configuration, storage and obligations remain

#### Scenario: Storage row deleted directly in the database

- **WHEN** a storage is removed directly while a backup configuration or retained backup references it
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

- **WHEN** a workspace is deleted while one of its storages has outstanding cleanup
- **THEN** the whole deletion returns `cleanup_pending` without implicitly abandoning files or exposing a raw constraint error

### Requirement: Cleanup state contains no provider credentials

Durable cleanup state SHALL contain file identity, database/workspace attribution where needed for authorized current-state reads, scheduling data, retry data and sanitized errors only. It SHALL NOT duplicate provider credentials, tokens, private keys, signed addresses or complete provider configuration.

#### Scenario: Cleanup state is inspected at rest

- **WHEN** an operator or database tool reads durable cleanup state
- **THEN** the record contains no provider authentication material or reusable signed address

## REMOVED Requirements

### Requirement: Storage-switch cleanup has a finite durable outcome

**Reason**: All unfinished files use one retryable queue; historical per-switch outcomes, captured totals and deadlines are removed.

**Migration**: Discard unfinished deletion state from the unreleased implementation without copying or restoring it. Read current cleanup counts; use explicit storage deletion to abandon retries.
