## ADDED Requirements

### Requirement: TimescaleDB leaves restoring mode after an interrupted restore

After successfully entering TimescaleDB restoring mode, a restore SHALL attempt to leave that mode on success, failure, ordinary cancellation or timeout. Ordinary cancellation SHALL NOT cancel cleanup. During normal application operation cleanup SHALL have its own 30-second maximum and SHALL finish before that restore execution is considered drained.

Graceful shutdown SHALL preserve the existing eight-second total exit limit, consisting of at most six seconds to drain and two seconds to flush logs. Cleanup SHALL be bounded by the remaining drain time minus a one-second reserve for recording the restore outcome; this smaller limit SHALL also apply when shutdown starts during an already-running cleanup. If no cleanup time remains, no new cleanup work SHALL start. The application SHALL NOT wait beyond its shutdown limit even when cleanup does not cooperate with cancellation.

A successful archive restore whose cleanup fails SHALL be reported as failed. Cleanup failure SHALL NOT replace an archive failure or an ordinary cancellation. An archive failure established before cleanup SHALL remain the recorded failure when user cancellation, the restore timeout or shutdown occurs during cleanup. When shutdown causes the interruption, the existing shutdown-failure outcome SHALL apply. When user cancellation is the first cause of the restore stopping before shutdown, the restore SHALL remain canceled while cleanup obeys the shortened shutdown budget. A cleanup failure or skipped cleanup SHALL report that the target may remain in restoring mode when logging can finish within the budget. If cleanup does not return before forced exit, the shutdown drain-timeout report SHALL identify the unfinished work; any restore without a recorded final outcome SHALL be failed by the existing next-start recovery. Neither case SHALL claim that the target left restoring mode.

#### Scenario: Cancellation after pre-restore

- **WHEN** a member cancels a restore after TimescaleDB entered restoring mode and the target remains reachable during ordinary application operation
- **THEN** the restore is canceled and the target leaves restoring mode before its execution is drained

#### Scenario: Restore failure

- **WHEN** restoring the archive fails after TimescaleDB entered restoring mode and cleanup can reach the target
- **THEN** the target leaves restoring mode and the restore retains the archive failure

#### Scenario: Cleanup fails after an otherwise successful restore

- **WHEN** archive restoration succeeds but leaving restoring mode fails
- **THEN** the restore is failed and the cleanup error is recorded

#### Scenario: Target is unreachable during cleanup

- **WHEN** the target becomes unreachable after entering restoring mode and the restore is canceled during ordinary application operation
- **THEN** cleanup stops within its 30-second maximum, the restore remains canceled, and the log states that the target may still be in restoring mode

#### Scenario: Shutdown cleanup succeeds within the budget

- **WHEN** shutdown starts after pre-restore and cleanup can finish within the remaining cleanup time
- **THEN** the target leaves restoring mode, the restore records the existing shutdown failure and the application exits within eight seconds

#### Scenario: Shutdown interrupts slow cleanup

- **WHEN** cleanup after pre-restore exceeds the remaining shutdown cleanup budget
- **THEN** cleanup is interrupted at its cutoff and, when it returns cooperatively, the remaining drain time is used to record the restore's shutdown failure and warning
- **AND** the application exits within eight seconds and the target may remain in restoring mode

#### Scenario: Shutdown begins during an ordinary cleanup attempt

- **WHEN** an ordinary cleanup attempt with a 30-second maximum is still running when shutdown starts
- **THEN** its limit is shortened to the shared shutdown cleanup cutoff rather than retaining or restarting a 30-second window
- **AND** the application exits within the unchanged eight-second total limit

#### Scenario: Shutdown budget is already exhausted

- **WHEN** cleanup becomes ready only after its shutdown cutoff
- **THEN** the application starts no new cleanup SQL, reports possible remaining restoring mode when budget permits, and preserves the existing exit limit

#### Scenario: Cleanup does not return before forced exit

- **WHEN** cleanup ignores cancellation and does not return before the drain deadline
- **THEN** the application reports unfinished draining and exits within eight seconds without claiming target cleanup succeeded
- **AND** the target may remain in restoring mode and a restore without a final recorded outcome is failed on the next start


#### Scenario: Shutdown follows user cancellation

- **WHEN** a member cancels an archive restoration that has not already failed and shutdown begins while its cleanup is still running
- **THEN** cleanup obeys the shutdown cutoff and the restore remains canceled when its outcome can be recorded

#### Scenario: User cancellation follows an archive failure

- **WHEN** archive restoration fails and a member cancels the restore while cleanup is still running
- **THEN** the restore is failed with the archive failure rather than canceled

#### Scenario: Shutdown follows an archive failure

- **WHEN** archive restoration fails and shutdown begins while cleanup is still running
- **THEN** cleanup observes the shutdown budget and the recorded restore failure retains the archive error rather than the shutdown interruption message

## MODIFIED Requirements

### Requirement: A running restore can be cancelled

Cancelling a running restore SHALL answer `202`, stop its restore tool together with every process the tool started, including while the backup is still being read from storage before the tool starts, and record the restore as `CANCELED`, unless the archive had already failed before cleanup as the TimescaleDB cleanup requirement states. A cancellation received during cleanup SHALL NOT replace that established failure. Cancelling a restore that is no longer running SHALL get `409` with `code` `not_running` and SHALL change nothing.

#### Scenario: Cancelling a running restore

- **WHEN** a member cancels a running restore
- **THEN** the response is `202`, reading the restore later shows `CANCELED`, and no process of its restore tool is left running

#### Scenario: Cancelling a restore while its archive downloads

- **WHEN** a member cancels a parallel PostgreSQL restore while its archive is still being downloaded into a file
- **THEN** the response is `202`, and reading the restore later shows `CANCELED`
- **AND** no restore tool process was started, and the temporary archive file is gone

#### Scenario: Cancelling a finished restore

- **WHEN** a member cancels a restore that has completed
- **THEN** the response is `409` with `code` `not_running`, and the restore is still `COMPLETED`
