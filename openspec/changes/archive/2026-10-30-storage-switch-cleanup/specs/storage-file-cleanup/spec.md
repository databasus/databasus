## MODIFIED Requirements

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

## ADDED Requirements

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
