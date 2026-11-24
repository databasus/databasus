## Context

See proposal.md, Why. The current state that shapes the approach:

- The server computes one number per backup, archive + database + gap, where the gap is 50% of the database clamped to 1536–5120 MB (`backend/internal/modules/verification/internal/verifications/capacity.go:15-19`, `:48-58`). The same number decides admission (`capacity.go:28-46`, called from `claim_service.go:82`) and becomes the agent's disk budget (`claim_service.go:100`).
- The server ignores CPU. The agent picks `pg_restore -j` from its own CPU per job (`backend/internal/agent/runner/job_execution.go:304`), capped at 8 and forced to 1 for TimescaleDB (`backend/internal/engines/postgresql/logical/pgrestore/args.go:64-72`). The CPU per job is `--max-cpu / --max-concurrent-jobs` (`backend/internal/agent/config/config.go:137`), and the claim already sends both values (`backend/internal/agent/runner/runner.go:121-126`).
- The agent stops the restore when `du` of PGDATA and `/tmp` in the container reaches the budget (`backend/internal/agent/container/container.go:62`, `backend/internal/agent/runner/diskwatch.go:48`). The archive stays in `/tmp` for the whole restore.
- The restore container runs with `wal_level=minimal` and `max_wal_size=1GB` and leaves `maintenance_work_mem` at its default (`backend/internal/agent/container/manager.go:49-58`). A parallel `pg_restore` loads each table it created in the same transaction as a `TRUNCATE`, so under `wal_level=minimal` the data load writes almost no WAL. Index builds spill their sorts to `base/pgsql_tmp`, and those files live until the index is built.
- The size probe sums `pg_total_relation_size` of the dumped tables and applies the backup's exclusion rules (`backend/internal/engines/postgresql/logical/size.go:13-79`). The pipeline calls it before the dump starts (`backend/internal/modules/backups/internal/pipeline/backup.go:87`, `postgresql_engine.go:139`) and stores the result as `BackupRawDbSizeMb` (`backend/internal/modules/backups/internal/logical/logical_backup.go:47-48`).
- The server loads the agent's running backups outside the claim transaction (`claim_service.go:44-51`, `verification_repository.go:57-69`). One agent process claims serially, one claim at a time (`runner.go:110-128`).
- An agent of the previous release keeps calling the agent API of its own release until it updates itself (`docs/architecture.md:200-202`).

The approach answers to the root `AGENTS.md`, `backend/AGENTS.md` and `website/AGENTS.md`. The backend rules that constrain it:

- Engine code knows the database but no tables, and no code outside the engine family switches on the database type. The pipeline therefore keeps reaching the probe through its per-engine dump source interface.
- A repository changes columns with one `UPDATE`, and migrations put numeric bounds into the schema.
- The verification agent imports only `lib`, `engines` and the generated agent client.
- The contract changes first, and the generated code is regenerated.

The website rule that constrains it is the sync rule: the English page and all six copies change together.

## Goals / Non-Goals

**Goals:**

- The disk budget of a verification follows from the sizes the restore actually writes, and the number of jobs follows from the disk the agent has.
- The budgets of an agent's running verifications never add up to more than `--max-disk-gb`.
- One component decides both the number of jobs and the budget, so the two cannot disagree.

**Non-Goals:**

- Measuring the real peak at run time and adapting the restore while it runs.
- Exact accounting of the restored database size. `BackupRawDbSizeMb` stays the estimate it is today.

## Decisions

### 1. Measure index sizes on the source at backup time

The backup already connects to the source to estimate the database size. In the same probe it reads `pg_relation_size` of every index whose table is in the dump scope and keeps the eight largest, largest first. Eight is the largest number of jobs `pgrestore.GetParallelJobs` allows.

The index query joins `pg_index` to the table and applies the same scope (`dump_scope.go:18`) and the same exclusion matching (`size.go:68-71`) as the size query. It reads only `relkind = 'i'`. A partitioned table's index has no storage of its own, and the partitions' indexes are counted through the partitions, which are ordinary tables in the scope. Primary keys, unique constraints and exclusion constraints are indexes in `pg_index`, so they count without special handling.

The engine keeps `GetDumpedSizeMb` and gains `GetLargestDumpedIndexSizesMb(ctx, logger, filter) ([]float64, error)`, which returns an empty, non-nil list when the dumped tables have no indexes. In the pipeline, every dump source's `GetRawSizeMb(ctx) (float64, bool)` becomes `GetSizeEstimate(ctx) SizeEstimate`, with `SizeEstimate{RawDbSizeMb float64; LargestIndexSizesMb []float64}`. The old name would no longer state everything the method returns, and the old success flag goes away because the backup run ignored it (`pipeline/backup.go:87`): a raw size of 0 already means unknown. MySQL, MariaDB and MongoDB fill only `RawDbSizeMb`.

The PostgreSQL dump source combines the two engine results with one pure function:

- the raw size query fails → a raw size of 0, and no index sizes are recorded, so the backup gets the one-job plan with the 5120 MB gap of an unknown size;
- only the index query fails → the raw size is kept, `LargestIndexSizesMb` is `nil`, and a warning is logged;
- both succeed → both are recorded.

A real index query failure cannot be caused cheaply against a real server, because `pg_relation_size` needs no privilege. The combining function is therefore covered by a unit test, which the testing skill allows for pure logic.

Rejected alternatives:

- Reading sizes when the verification starts: the agent has only the archive, and the archive holds no sizes.
- Reading the TOC with `pg_restore -l`: it lists the indexes but not their sizes.
- Keeping a ratio of the database size: that heuristic is what fails in issue #929, because the peak depends on which indexes build together, not on the total.
- Storing one sum of the eight largest: the server needs the sum for every N from 1 to 8.

### 2. Store index sizes as a nullable array on the backup

`backups_logical` gains `backup_largest_index_sizes_mb DOUBLE PRECISION[]`. `NULL` means the sizes were not recorded, either because the backup is older than this change or because the probe failed. An empty array means the dumped tables have no indexes. The schema bounds the array to at most 8 nonnegative elements. `LogicalBackup` gains `BackupLargestIndexSizesMb []float64` beside `BackupRawDbSizeMb`, where `nil` means not recorded.

`docs/architecture.md:128` says "Lists are `text[]`". That rule exists for lists of names and enum values. This change amends it: a list of numbers is a typed numeric array, so the schema can bound its elements, here at most 8 nonnegative values. Storing numbers as `text[]` would move that check into application code and make every reader parse strings.

Rejected alternatives:

- A separate table of index sizes: it adds a join for eight numbers that are always read together.
- Zero meaning unknown, as `BackupRawDbSizeMb` does: an array has a natural unknown in `NULL` and a natural none in `{}`, and the two must stay apart, because they lead to different plans.

### 3. The server picks the number of jobs when it assigns the verification

At claim time the server computes the per-job CPU budget as `MaxCPU / MaxConcurrentJobs`, rounded down and at least 1. When the claim has zero concurrent jobs, or fewer CPUs than concurrent jobs, the budget is 1. Then `maxParallelJobs = pgrestore.GetParallelJobs(perJobCPUBudget, isTimescaleDB)`. Reusing `GetParallelJobs` keeps the cap of 8 and the TimescaleDB rule in one place.

The server then walks N from `maxParallelJobs` down to 1 and takes the first N whose restore peak fits the agent's free disk:

```
restorePeak(N) = ceil(archive + database + 1536 + sum of the N largest index sizes)
freeDisk       = MaxDiskGb * 1024 - sum of reserved disk of the agent's RUNNING verifications
```

The reserve of 1536 MB covers WAL, which `max_wal_size=1GB` keeps near 1 GB, and the empty cluster of a few tens of megabytes. It keeps the value of today's minimum gap. A backup without recorded index sizes, or with an unknown database size (`BackupRawDbSizeMb` of 0), gets N = 1 and today's estimate. Without the database size, `restorePeak(N)` would leave out the largest part of what the restore writes. Today's estimate is kept as the plan for such backups rather than as a fallback for some other shape. A candidate that does not fit even at N = 1 is skipped, and the loop moves to the next candidate as it does today (`claim_service.go:81-84`).

The assignment carries `parallelJobs` = N and `maxContainerDiskMb` = `restorePeak(N)`. The agent passes `parallelJobs` straight to `pg_restore`. The agent's own derivation from `CPUPerJob` (`job_execution.go:303-304`) is deleted, and `CPUPerJob` keeps sizing only the container's CPU limit.

Rejected alternatives:

- The agent picks N itself: the server then has to guess the agent's N to compute the budget, and the two can disagree, which is the bug in issue #929.
- A fixed share of the disk per job (`--max-disk-gb / --max-concurrent-jobs`): it gives the job more room without saying what the restore needs, so a high `-j` can still overflow, and a large job beside a small one packs worse.
- A fair-share N that leaves room for the next job: The per-job CPU budget already caps N at the agent's share, so a greedy N cannot take more than one job's share of the parallelism. The extra rule would only shrink restores on agents that have the disk.

### 4. The verification stores its reserved disk

`verification_runs` gains `reserved_disk_mb BIGINT NOT NULL DEFAULT 0` with a nonnegative check. `UpdateClaim` writes it in the same `UPDATE` that sets the status, the agent and the start time. Admission sums `reserved_disk_mb` of the agent's `RUNNING` rows inside the claim transaction. That replaces `ListRunningBackupIDsByAgentID` and the backup lookup (`claim_service.go:44-51`). A row that leaves `RUNNING` stops counting without being reset, and the next claim of a requeued row overwrites it.

N depends on the free disk at the moment of the claim, so the budget cannot be recomputed later from the backup alone. Storing it is the only way to sum it.

Rejected alternatives:

- Storing N and recomputing the restore peak: it works, but admission needs megabytes, and a later change of the formula would silently change what running verifications hold.
- Reading the sum outside the transaction, as today: it is cheap to move inside, and that removes a window in which a cancellation and a claim interleave.

Locking the agent row is not needed, because one agent process claims serially (`runner.go:110-128`).

### 5. The agent builds the disk budget failure message

The agent already builds the message (`job_execution.go:79`) and now also knows N. The message becomes `restore exceeded the per-job disk budget of <budget> MB with <N> parallel jobs; raise --max-disk-gb or lower --max-cpu of the agent`. The server keeps storing the agent's message as it does (`retry.go:42-46`).

Rejected alternative: building the message on the server from stored values. It would split one message between two components, and the agent already has every value.

### 6. Two terms go into the glossary

The change makes two concepts visible to users, in the failure message and on the website, and stores one of them. `docs/glossary.md` gains **Disk budget**: the disk a verification may use on its agent, reserved from its assignment until it stops running. It also gains **Restore peak**: the most disk a restore is expected to occupy at once, from which the disk budget is set. The artifacts use only these two terms.

Rejected alternative: leaving them undefined. The glossary's entry for assignment eligibility already speaks of a "resource budget", and a third wording would follow.

### 7. The agent contract gains a required field

`VerificationAssignment` in `backend/api/agent.yaml` gains the required integer `parallelJobs` with minimum 1. This adds a field to a response, so an agent of the previous release decodes the assignment and ignores it. Until that agent updates itself it keeps choosing N from its CPU, as `docs/architecture.md:200-202` allows. `docs/architecture.md` needs no change, because the decision follows its compatibility rule.

## Risks / Trade-offs

- [The temporary files of an index build can be larger than the index on the source] → The source size includes bloat, which works in the restore's favor, and a sort spills roughly the tuples it sorts. This is still an expectation. Before the planner is built, a task measures the peak of a real parallel restore against `restorePeak(N)`. If the peak is above it, that task adds a measured multiplier to this design and to the example budgets of the `backup-verification` delta spec, so the planner and its tests are written against the corrected formula. Measured on PostgreSQL 18 with four jobs and four indexes of about 32 MB each, every one of whose sorts outgrows `maintenance_work_mem`: the restore peaked 122.5 MB above the archive and the restored database, the empty cluster included, while the four indexes total 128.6 MB. No multiplier is needed.
- [The restored database is larger than the source, for example because of a different TOAST compression] → This risk exists today and the change does not affect it. The budget failure message tells the user what to change.
- [An agent whose disk only just fits a backup now refuses it, because the largest index costs more than the old gap] → The proposal marks this as breaking. The verification stays pending until another agent takes it or it times out as unclaimed, and the website page explains how much disk a verification needs.
- [An agent of the previous release keeps its own N] → It fails as it does today until it updates itself. Self-update closes the gap.
- [A greedy N leaves less room for the next verification] → The per-job CPU budget bounds N, so a single verification cannot take more than its share of parallelism. The next verification waits only when the disk is genuinely short.
- [Backups made before the change are verified with one job] → They are slower, but they pass as they do today with `--max-cpu=1`, and new backups replace them as retention rotates.

## Migration Plan

1. A migration adds `backup_largest_index_sizes_mb` to `backups_logical` as nullable, with the length and nonnegativity check. Existing rows stay `NULL`.
2. A migration adds `reserved_disk_mb` to `verification_runs` and fills it for rows in `RUNNING` with today's estimate computed in SQL from their backup. A verification that runs across the upgrade therefore still counts against its agent's disk.
3. The backend and the agent ship in one image. Agents update themselves on their next version check.

Rollback: the down migrations drop both columns. The code of the previous release reads neither column, so it runs on the rolled-back schema.
