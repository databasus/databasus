## Why

Milestone 11 of the backend rewrite plan simplifies the physical backups now that milestone 10 has moved them into the `backups` module unchanged. Full and incremental backups live in two tables with one column apart, so every model, repository, terminal write and query exists twice, and five queries glue the tables back together with `UNION ALL`. Because a unique index cannot span two tables, a third table, `backups_physical_in_flight`, records which backup of a database is running, and three places release that record on their own. One of them, the cancel, releases it while the backup's `pg_basebackup` is still stopping. The next backup of the database can then start beside it. Both use the same replication slot name, so the new backup can fail: either the old process still holds the slot when the new one drops it before creating its own, or the old process's cleanup drops the new slot before the new `pg_basebackup` attaches to it.

The scheduler decides what to start before it takes the database's lock, then checks the decision again under the lock with a second, separately written copy of the same conditions. The spec also asks Databasus to remember which compression worked for a database, which the code has never done and which would keep a rebuilt source on gzip forever.

## What Changes

- **One table for full and incremental backups.** `backups_physical` replaces `backups_physical_full` and `backups_physical_incremental`, with a `kind` column, a reference to the direct parent and a reference to the chain's full backup. The code has one backup type and one repository. The API contract stays as it is: the incremental's root and parent are derived from the new columns.
- **Invariants in the schema.** A `CHECK` requires the parent and chain root exactly for incrementals, and the file, both LSNs and the completion time for a completed backup. A partial unique index allows at most one running backup per database.
- **`backups_physical_in_flight` is deleted.** The running row is the claim, so no code releases a claim anymore and the startup sweep has one pass instead of two.
- **One owner of status changes.** One service creates the running row and performs every transition to completed, failed, chain broken and cancelled. Each transition is one conditional `UPDATE` on the running status, made together with the file obligations it carries in one transaction.
- **Cancel and delete record the outcome at once.** Cancelling a running backup records `CANCELED` before the response, and deleting one removes it before the response. The process stops after the commit. The backuper remembers which databases still have a live backup process, and the scheduler starts nothing for such a database until that process has ended.
- **The scheduler decides with one function, before and under the database's lock.** A pass with nothing due ends without a transaction, as today. Otherwise the scheduler loads the state again under the lock, calls the same pure decision function and inserts what it decides. The separate recheck of an immediate replacement is deleted. The decision function gets the chain facts in a second step, so the loading of the state no longer repeats the order of its conditions.
- **Compression falls back on every backup.** The requirement to remember the compression that worked is removed from the spec and from `docs/architecture.md`. The code keeps trying `zstd`, then `gzip`, then none on each backup, as it does today.
- **Dead code is deleted:** `PhysicalSlotCleanupListener` and the methods of the catalog, the chains, the WAL streamer and the LSN range that nothing in production calls or passes as a value.
- **"In-flight" leaves the backup code.** The glossary term becomes "running backup", and the identifiers that named the claim take that word, such as `HasRunningFullBackup` and `CancelRunningForDatabase`.
- **BREAKING** for the schema: `backups_physical_full`, `backups_physical_incremental` and `backups_physical_in_flight` are replaced by `backups_physical`, with the stored backup IDs kept. A user's own SQL against the old tables stops working. The API, the frontend and every stored file stay as they are.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `physical-backups/operations`: a cancelled backup shows `CANCELED` as soon as the request is answered; a deleted running backup disappears at once and its process stops afterwards; a new requirement says a database runs one physical backup process at a time, so the next backup waits for a stopped one to end.
- `physical-backups/backup-method`: the compression requirement no longer asks to remember the compression that worked; every backup starts with zstd and falls back.
- `upgrade-test`: the physical data written before the upgrade gains an incremental backup, and after the upgrade it is listed with its chain and streamed in the restore set.

## Out of scope

- The health of a WAL stream. `streamhealth.GetVerdict` already gives the API and the Monitoring tab one verdict, and the notifications come from the live trackers as the `wal-stream-monitoring` spec requires.
- The database deletion handler, which is already one handler with an explicit order.
- Simplifying the chain, WAL ownership and retention code (`chains`, `retention`), beyond what one table changes in their queries.
- The API contract, the frontend, the WAL streaming, the restore stream and the recovery script.
- Any change to which backup the scheduler chooses, to the retry and escalation rules, or to the shutdown behavior of a running backup.
- A per-database compression setting or a remembered codec.

## Impact

- **Migrations:** one new migration in `backend/migrations` that creates `backups_physical`, copies both tables with their IDs, repairs rows the new constraints would reject, and drops the three old tables; an extension of `migrations/physical_backups_migration_test.go`.
- **Backend:** `internal/modules/backups/internal/physical` with its subpackages `catalog`, `backuping`, `chains`, `retention`, `restorestream`, `timelines`, `streamhealth` and `walstreaming`, the facade services of `physical`, `internal/modules/backups/module.go` (`PhysicalFullBackup` becomes `PhysicalBackup`) and its telemetry reader in `internal/features/telemetry`.
- **Tests:** the seed helpers and `WaitForPhysicalBackups` of `internal/app/apptest`, the SQL in `internal/app/apptest/physicaltest/promotion.go`, the API tests under `internal/modules/backups/physicalapitests/*` and `internal/modules/backups/*_test.go` that read or seed the old tables, `internal/features/workspaces/testing/testing.go`, and the upgrade test in `backend/e2e/upgrade`.
- **Not affected:** `backend/api/openapi.yaml`, the generated clients, the frontend, the stored backups, WAL segments and history files.
- **Running instances:** the migration moves every full and incremental backup into one table with its ID. A backup that was still marked running when the old release stopped is failed at startup, as it is today.
- **Documents:** `docs/architecture.md` ("Physical backups": the table, the running-backup index, the owner of status changes, the compression, and the health bullet, which no longer matches the code and the spec), `docs/glossary.md` (the in-flight claim term) and the rewrite plan `temp/refactor/plan.md`.
- **Rules the change answers to:** the root `AGENTS.md` and `backend/AGENTS.md`.
