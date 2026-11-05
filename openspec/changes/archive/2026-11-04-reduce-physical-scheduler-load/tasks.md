## 1. Batch reads

- [x] 1.1 Add `CountInternalDatabaseQueries` to the test application, which counts the statements sent to the internal database while a function runs, using a GORM callback on the application's database handle. Verify with `go test -count=1 -run 'TestPhysicalSchedulerLoad' ./internal/modules/backups/physicalapitests/backuping/` from `backend/` once 1.2 exists.
- [x] 1.2 Add `TestPhysicalSchedulerLoad` with the scenario "reads the state of many databases with a fixed number of queries when nothing is due", comparing a pass over two databases with a pass over five. Run it on the unchanged scheduler and see it fail because the count grows. Verify with the same command as 1.1.
- [x] 1.3 Add `GetNewestBackupsByDatabases` and `GetDatabasesWithUnresolvedMismatch` to `catalog`, and split `loadSchedulingState` into a batch step and a per-database chain step. `runPendingBackups` uses the batch step once per pass and returns its error; `claimAndInsert` uses both steps for one database. `scheduler_decision.go` stays unchanged. Verify that `TestPhysicalSchedulerLoad` and `TestDecidePhysicalBackup` pass with `go test -count=1 -run 'TestPhysicalSchedulerLoad|TestDecidePhysicalBackup' ./internal/modules/backups/...` from `backend/`.
- [x] 1.4 Delete `FindLastFullAnyStatusByDatabase`, `FindLastIncrementalByDatabase` and `HasUnresolvedSystemIdentifierMismatch`, and verify that nothing still references them with `mise run backend:lint`.
- [x] 1.5 Add `TestPhysicalScheduler` with the scenarios "decides each database from its own history when one pass reads many databases", where the first two databases have a due full cadence, the second also has an unresolved mismatch and the third has no due cadence, and "starts a full backup only for the database whose mismatch a completed full backup resolved". Verify with `go test -count=1 -run 'TestPhysicalScheduler' ./internal/modules/backups/physicalapitests/...` from `backend/`.

## 2. Backup request job and intervals

- [x] 2.1 Add `TestPhysicalBackupRequestPass` with these scenarios, and run it before 2.2 to see it fail because the job `backups_physical_request_scheduler` does not exist. Verify with `go test -count=1 -run 'TestPhysicalBackupRequestPass' ./internal/modules/backups/physicalapitests/...` from `backend/`.
  - "starts a requested full backup without a full scheduler pass"
  - "starts a requested incremental backup on the extendable chain without a full scheduler pass"
  - "starts a backup requested during a running backup after that backup ends"
  - "leaves a due schedule to the full scheduler pass when the database has no request"
  - "keeps an incremental request pending and starts a requested full backup when a mismatch is unresolved"
  - "leaves a replacement full to the full scheduler pass after a timeline failure when no request is pending"
  - "starts a replacement full instead of a pending incremental request after a timeline failure"
  - "starts one full and clears a pending full request after a timeline failure"
  - "leaves a replacement full to the full scheduler pass after an interrupted full when no request is pending"
  - "starts no replacement full after an interrupted full when a mismatch is unresolved"
  - "clears an incremental request and starts nothing when incrementals are disabled"
  - "starts a full backup in the new storage when a storage switch recorded the request"
  - "starts a full backup when a WAL slot rebuild recorded the request", which records the request with SQL, because a real rebuild needs a lost replication slot and writes only this column
  - "starts a requested full backup after the backups of the database are turned back on", where the first pass with backups turned off starts nothing and the WAL stream statistics report a full backup as pending
  - "starts nothing and keeps the request when the backup config names no storage"
- [x] 2.2 Add `ListPhysicalBackupConfigsWithPendingRequest` to the config service, passing its rows through `fitToBackupTypes`, and add `GetRequestJob` with the job `backups_physical_request_scheduler` and a one-second interval to `Scheduler`, using the shared evaluation of 1.3. Return from the pass before any other read when the list is empty. Register the job in `GetJobs`. Add the scenario "sends one query when no request is pending" to `TestPhysicalSchedulerLoad`, with two databases and then five. Verify with `go test -count=1 -run 'TestPhysicalBackupRequestPass|TestPhysicalSchedulerLoad' ./internal/modules/backups/physicalapitests/...` from `backend/`.
- [x] 2.3 Change the interval of `backups_physical_scheduler` to 15 seconds. Add a scenario to `TestInstanceHealth` that checks the reported intervals of both jobs. Verify with `go test -count=1 -run 'TestInstanceHealth' ./internal/app/` from `backend/`.
- [x] 2.4 Add `backups_physical_request_scheduler` to the live job list and the live route of `RunJobPass` in `apptest`, and add a way for a test to read the live runner's snapshots. Add these scenarios to `internal/app/apptest/apptest_test.go`, and verify with `go test -count=1 -run 'TestPhysicalBackupLoops|TestRunJobPass' ./internal/app/apptest/` from `backend/`.
  - `TestPhysicalBackupLoops`: "runs the backup request job live and again after modules restart", which reads the job's completed pass from the snapshots before and after `RestartModules`, then requests a full backup and sees it in the list without an explicit pass
  - `TestPhysicalBackupLoops`: "runs one pass of the backup request job by name with live jobs"
  - `TestPhysicalBackupLoops`: "leaves a requested backup unstarted without the option and without a pass"
  - `TestRunJobPass`: "starts a requested full backup when one pass of the backup request job runs"
  - `TestRunJobPass`: "starts a requested full backup when one scheduler pass runs"

## 3. Documentation

- [x] 3.1 Update the scheduler paragraph of `docs/architecture.md` with the two jobs, their intervals and why a job replaces a wake-up signal, and update the live job description in `docs/testing.md`. Add to `docs/glossary.md` the terms pending backup request (a request timestamp in the backup config, such as `ForceFullRequestedAt`), full scheduler pass and backup request job (`backups_physical_request_scheduler`). Search the repository for other mentions of the one-second interval and the deleted methods and bring them in line. Verify with `mise run lint`.

## 4. Finish

- [x] 4.1 Run `mise run backend:format` and `mise run backend:lint` and verify that both finish with no errors.

## Change verification

```sh
mise run test
```
