## Context

See `proposal.md` for the motivation and `specs/` for the required behavior.

A scheduler pass lists the enabled configs in two queries (`backend/internal/modules/backups/internal/physical/backuping/scheduler.go:48`, `backend/internal/modules/backups/internal/config/service.go:170`) and then evaluates each database. Evaluation calls `loadSchedulingState` (`scheduler_state.go:26`), which reads the newest full backup, the mismatch state and the newest incremental backup in three queries (`scheduler_state.go:35-45`). Chain facts load only when the first decision step asks for them (`scheduler_state.go:50-54`). `claimAndInsert` calls the same function again under the advisory lock (`scheduler.go:265`), so a decision is always repeated on fresh data before a row is inserted.

A pending backup request is a timestamp in the database's config, `ForceFullRequestedAt` or `ForceIncrementalRequestedAt`. It is written by `RequestFullBackup` and `RequestIncrementalBackup` (`backend/internal/modules/backups/internal/config/service.go:313-326`) and by the storage switch (`config_repository.go:569`). The HTTP start and the WAL slot rebuild write it inside a transaction (`physical/backup_request_service.go:37`, `physical/walstreaming/manager.go:417`).

A pass has several outcomes for a request, all from the existing decision:

- A full request starts its backup when the database is free. While an earlier run is alive, it stays pending (`backuper.go:95`).
- An incremental request starts its backup on an extendable chain (`scheduler_decision.go:69-79`).
- An incremental request is cleared when incremental backups are not enabled or no extendable chain exists (`scheduler_decision.go:48`, `:67-68`, `scheduler.go:271`).
- When the full cadence is also due, the same evaluation starts that scheduled full (`scheduler_decision.go:48-52`, `:82-86`). So the backup request job can start a due full before the next full scheduler pass, but only for a database with a pending request.
- An incremental request stays pending with no backup while a system identifier mismatch is unresolved (`scheduler_decision.go:33-35`), and that wait ends only with a completed full backup. It also stays pending after a chain or parent read error (`scheduler_decision.go:69`).

`background.Runner` starts each pass one interval after the previous one ended and has no way to wake a job (`backend/internal/lib/background/runner.go:120-134`). One process owns an installation, as the spec `single-instance-runtime` says.

The approach is constrained by these rules: periodic work goes through `background.Runner`, a repository runs every query through `database.Conn(ctx, r.db)`, code stays inside its module with `module.go` as the boundary (`backend/AGENTS.md`), and tests observe behavior through the API with direct SQL only for a state the API cannot create (`.agents/skills/testing/SKILL.md`).

## Goals / Non-Goals

**Goals:**

- Scheduling queries stop growing with the number of databases on a pass with nothing due.
- A pending backup request reaches the existing decision as fast as it does today, including one that waits for a running backup. The new job only reruns that decision, so a request can also be cleared or stay waiting.
- Every decision stays the same, with no change to `scheduler_decision.go`.

**Non-Goals:**

- Reducing the row lookups PostgreSQL does per database on a full scheduler pass. The batch reads save round trips; the longer interval saves the work.
- Batching chain reads. A pass with something due or requested still reads chains per database.

## Decisions

### Two jobs instead of a wake-up channel

`Scheduler` gets a second job, `backups_physical_request_scheduler`, with an interval of one second, returned by `GetRequestJob`. Its pass calls `ListPhysicalBackupConfigsWithPendingRequest`, which lists configs that are enabled, name a storage and have a request timestamp, then runs the shared evaluation for those databases. `backups_physical_scheduler` moves to 15 seconds and still evaluates every enabled config, requests included.

`ListPhysicalBackupConfigsWithPendingRequest` passes its rows through `fitToBackupTypes` (`config/service.go:551`), like `ListEnabledPhysicalBackupConfigs`. A raw row has an empty `BackupType`, so `isIncrementalEnabled` would return false and the job would clear every incremental request with no backup. An empty list stays at one query, because `GetBackupTypes` returns early on an empty ID list (`postgresqlphysical/provider.go:56`). The query counter test pins this.

The job reads committed rows, so it sees every writer without any change to them, and it finds a request left waiting by a running backup with no extra signal.

Alternatives rejected:

- **A channel that wakes the scheduler.** It saves about one query per second and needs four other changes: a wake option in `background.Runner`, a call after commit from each request writer, a signal when a run ends, and a test for each of them. A writer that forgets the signal delays its backup by up to 15 seconds with no error.
- **One job at one second with a counter that runs the full evaluation every 15th pass.** It hides a second rhythm inside one job, and instance health could not report either rhythm separately. `backend/AGENTS.md` also rules out hand-written pass guards of this kind.
- **A single job at 5 seconds with no fast path.** It is the simplest option and sends about 0.8 queries per second. After a click, the user could wait up to 6 seconds with no reaction, because the start button releases as soon as the server answers and the list polls every second. The WAL slot rebuild and the storage switch would wait the same way.

A config with no storage stays out of the pending list because `evaluateConfig` logs an error for it on every evaluation. The full scheduler pass keeps that log at a rate of one per 15 seconds.

### Batch reads as the only way to read the newest state

`catalog` gets two reads that take a list of database IDs:

- `GetNewestBackupsByDatabases` returns the newest full and the newest incremental backup of each database. The query joins `unnest(?::uuid[])` with the two kinds and uses `CROSS JOIN LATERAL ... ORDER BY created_at DESC LIMIT 1`, so each lookup uses `idx_backups_physical_database_id_kind_created_at` (`backend/migrations/20261028225903_merge_physical_full_and_incremental_backups.sql:140`).
- `GetDatabasesWithUnresolvedMismatch` returns the databases with an unresolved mismatch, using `GROUP BY database_id` with `FILTER`. Its `HAVING` clause holds the rule the deleted `HasUnresolvedSystemIdentifierMismatch` applied in Go: a mismatch stays unresolved until a full backup completes after it.

`claimAndInsert` calls the same reads with one ID, inside its transaction. `FindLastFullAnyStatusByDatabase`, `FindLastIncrementalByDatabase` and `HasUnresolvedSystemIdentifierMismatch` have no other callers and are deleted.

Alternatives rejected:

- **`DISTINCT ON (database_id, kind)`.** It is shorter, but it reads the whole history of every database on each pass, which is worse than today's indexed lookups.
- **Merging both reads into one SQL statement.** The task text asks for one query, but the two result shapes differ. One statement would return a backup row with extra mismatch columns, only to save a fixed one query per 15 seconds. What matters is that the count does not grow.
- **Keeping the per-database reads for the claim path.** Two ways to read one fact can drift apart, and the decision under lock would then disagree with the decision before it. This is also why the task's test comparing batch and per-database decisions is replaced: nothing is left to compare against. The new test runs several databases with different histories through one pass, and the existing scheduler API tests now run through the batch reads.

### Decision functions stay and receive complete state

Reading all three facts for every database removes the early exits of `loadSchedulingState` (`scheduler_state.go:36` and `:41`). This does not change a decision, because `decidePhysicalBackupFromNewest` checks the forced full first and the mismatch second, before it uses any other fact (`scheduler_decision.go:23-35`).

`loadSchedulingState` splits into a batch step that returns the newest state per database and a chain step for one database. The pass uses the batch step once and then the chain step only where `needsChain` is true. `claimAndInsert` runs both for one database.

The alternative was to keep the early exits by reading mismatch only for databases without a forced full. It saves nothing, because the batch query runs anyway, and it adds a filter to get wrong.

### A failed batch read fails the pass

Today a failed read skips one database, and the pass continues. With a batch read, an error returns from the pass, `background.Runner` logs it and the next pass retries. Handling it per database has no meaning here, because the statement covers all of them and fails only when the internal database does. A failed chain read keeps its per-database handling.

### Query count is measured through a test helper on the application

The test application gets `CountInternalDatabaseQueries`, which counts the statements sent to the internal database while a function runs, through a GORM callback registered on the application's database handle. This follows `App.NewJobRunner` (`backend/internal/app/app.go:406`), an existing method for tests.

Alternatives rejected:

- **`pg_stat_database` counters.** PostgreSQL publishes them with a delay, so the count would be flaky.
- **Counting inside a repository test only.** A single query in a repository method proves nothing about the pass that calls it, and the bug is in that pass.

### The live job is checked through the runner's snapshots

A live test checks that the job snapshot of the live job holds a completed pass, before and after `RestartModules`, through `Runner.GetSnapshots` (`backend/internal/lib/background/job_snapshot.go:29`).

The alternative was to assert that a backup starts well inside 15 seconds. The testing rules forbid a time limit on a wait, and on a loaded machine such a bound would flake. A live `backups_physical_scheduler` also serves requests, so without the snapshot check a missing job would stay invisible. The one-second interval itself is pinned by the instance health scenario.

## Risks / Trade-offs

- Scheduled backups and the replacement FULL after a failover start on the next full scheduler pass, about 15 seconds later plus the length of the previous pass. → The replacement FULL writes no request, so a database with no pending request waits for a full scheduler pass, as the delta for `postgresql-physical-backup-failover` says. A database with a pending incremental request gets the replacement FULL on the next pass of the backup request job, because the unchanged decision checks the replacement before that request (`scheduler_decision.go:37-40`). The wait is short next to the length of a backup. A storage switch and a WAL slot rebuild write a request and are not affected.
- Tests with live jobs that wait for a scheduled backup can take about 15 seconds longer. → Their waits use `testwait.Until` with no per-wait limit, so they slow down rather than fail. If a test becomes too slow, it triggers a pass with `RunJobPass`, which the live mode already allows.
- Two jobs can evaluate one database at the same moment. → `registerRun` refuses a second run in the process, and `claimAndInsert` decides again under the advisory lock from fresh data. A request cleared by one job is not started again by the other. `scheduler_race_test.go` covers the repeated decision under the lock.
- A forced incremental that stays pending reads chains every second for that database when the decision needs them, for example during a running backup. A forced incremental with an unresolved mismatch reads no chain, but it stays in the pending list until a full backup completes, so each pass sends the batch reads for it. → This is no more work than today's pass for that database, and it is limited to databases with a pending request.

## Migration Plan

No migration is needed: no schema or data changes. Rollback is a revert of the code. A pending backup request written before an upgrade is served by the first pass after it.
