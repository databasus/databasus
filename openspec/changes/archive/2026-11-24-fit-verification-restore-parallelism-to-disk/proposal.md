## Why

Restore verification of a large PostgreSQL backup fails when the agent has more CPU, even though the agent has free disk to spare (issue #929). The agent stops the restore at a disk budget of archive + database + a fixed gap of at most 5 GB, and `--max-disk-gb` does not change that budget. While `pg_restore -j N` builds indexes, each of the N concurrent builds spills its sort into temporary files roughly the size of that index, so the peak grows with N while the gap stays fixed. A user with a 221 GB database and 500 GB of disk passes with `--max-cpu=2` in 1h13m and fails with `--max-cpu=6` or `8`.

## What Changes

- A PostgreSQL logical backup records the sizes of the eight largest indexes it dumps, eight being the most parallel jobs a verification restore uses. Indexes of tables left out of the backup, or whose rows are left out, are not counted.
- When an agent claims work, the server picks the number of parallel restore jobs N for the verification. N is the largest value from 1 up to the agent's per-job CPU budget (at least 1, at most 8, always 1 for TimescaleDB) whose restore peak fits the agent's free disk. The restore peak is archive + database + a fixed reserve for WAL and the empty cluster + the sizes of the N largest indexes. The agent's free disk is `--max-disk-gb` minus the disk budgets of its running verifications.
- When not even N = 1 fits, the agent does not get that verification, as a verification that does not fit is skipped today.
- The verification keeps its disk budget reserved while it runs, so concurrent verifications on one agent never reserve more than `--max-disk-gb` together.
- The assignment tells the agent N, and the agent restores with exactly that many jobs. Its disk budget is the restore peak computed for N.
- A backup made before this change, or one whose index sizes or database size could not be read, is verified with one job and the disk budget it gets today.
- The message of a verification that ran out of its disk budget names the budget and the number of parallel jobs, and says that raising `--max-disk-gb` or lowering `--max-cpu` helps.
- The disk budget paragraph of the website's restore verification page explains the new rule in all seven languages.
- **BREAKING** (agent API): `VerificationAssignment` in `backend/api/agent.yaml` gains the required field `parallelJobs`. The agent no longer derives its parallel jobs from its own CPU budget, and that code is deleted. An agent of the previous release ignores the field and keeps choosing its jobs from its CPU budget until it updates itself, so until then it can still reach its disk budget as it does today.
- **BREAKING** (disk planning): for backups with recorded index sizes, the gap of 50% of the database clamped to 1.5–5 GB is deleted. A backup with a large index now needs more free disk to be admitted even with one job, so an agent whose disk only just held the old estimate can stop receiving that verification.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `backup-verification`: a verification gets the number of parallel restore jobs and the disk budget that fit the agent's free disk, keeps that disk budget reserved while it runs, and reports a disk budget failure with what to change.
- `verification-container-resources`: on a host without a CPU limit, the CPU budget no longer sizes restore parallelism on the agent; it bounds the parallelism the server assigns.
- `postgresql-logical-backup-scope`: the index sizes that plan a verification's disk leave out excluded tables and tables whose rows are excluded, as the estimated database size does.

## Out of scope

- Building indexes one at a time, restoring by sections, or tuning `maintenance_work_mem` and `max_parallel_maintenance_workers` in the restore container.
- Restores into the user's own database server. They have no disk budget of ours.
- The retry policy of a verification that ran out of its disk budget.
- Index sizes for MySQL, MariaDB and MongoDB backups, which verification does not restore.
- Measuring index sizes again for backups made before this change.
- Showing the chosen number of jobs in the interface.
- Failing a verification early when its backup fits no agent at all.

## Impact

- Governed by: the root `AGENTS.md`, `backend/AGENTS.md` and `website/AGENTS.md`. The change does not edit them.
- Backend, backups: the PostgreSQL size probe in `backend/internal/engines/postgresql/logical/size.go`, the size estimate of every engine's dump source in the backup pipeline, `LogicalBackup` and its repository, and a migration adding the index sizes to `backups_logical`.
- Backend, verification: `capacity.go` and `claim_service.go` in `backend/internal/modules/verification/internal/verifications`, the verification repository, and a migration adding the reserved disk to `verification_runs`. It also fills that column for verifications running at upgrade time.
- Agent API: `backend/api/agent.yaml`, generated servers and clients, `backend/internal/modules/verification/internal/agentapi/handler.go`.
- Agent: `backend/internal/agent/runner/job_execution.go` takes `parallelJobs` from the assignment and builds the new failure message.
- Website: `website/app/(en)/restore-verification/page.tsx` and its six copies in `website/app/[lang]/restore-verification/content/`.
- Docs: `docs/glossary.md` gains the terms disk budget and restore peak, and the Types rule in `docs/architecture.md` allows a typed numeric array for a list of numbers.
