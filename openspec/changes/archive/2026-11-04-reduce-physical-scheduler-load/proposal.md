## Why

The physical backup scheduler wakes up every second and sends three history queries per database, so an idle instance with 50 databases sends about 150 queries per second to its internal database. No user notices a scheduled backup starting a few seconds late, so most of this work buys nothing. Only a manual start needs a fast reaction.

## What Changes

- A full scheduler pass runs every 15 seconds instead of every second, the interval the logical scheduler already uses.
- A full scheduler pass reads the newest backups and the system identifier mismatch state of all databases with a fixed number of queries, instead of three queries per database.
- The reload under lock before a backup is inserted uses the same reads with one database, and the per-database read methods are deleted.
- A new background job, the backup request job, runs every second and reads only the configs with a pending backup request. It sends the databases it finds through the same steps and the same decision as a full scheduler pass, so a manual start, a WAL slot rebuild and a storage switch that the decision accepts keep starting a backup within about a second.
- The replacement FULL after a timeline failure writes no request, so it starts on the next full scheduler pass, about 15 seconds later. A database that also has a pending incremental request gets it on the next pass of the backup request job.
- The decision functions that choose a full backup, an incremental backup or nothing do not change.

Nothing here breaks a caller, a schema, a config or an API. The new job appears in the instance health response next to the others, which that response already does for every job.

### Out of scope

- Chain reads in the second decision step. They stay per database and run only for a database whose incremental backup is due or requested.
- A new index for the mismatch check. Its cost per database stays the same and it runs 15 times less often.
- The logical scheduler, the retention cleaner and the WAL stream manager.
- A wake-up mechanism for background jobs. `background.Runner` does not change.
- Frontend changes. The backup list keeps its one-second poll.

### Module rules

The change answers to the root `AGENTS.md` and `backend/AGENTS.md`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `physical-backups/background-load`: the scheduler's idle work stops growing in queries with the number of databases, and instance health reports the two intervals.
- `physical-backups/operations`: a pending backup request is evaluated without waiting for a full scheduler pass, and a scheduled backup starts on the next full scheduler pass.
- `postgresql-physical-backup-failover`: an immediate replacement FULL starts on the next pass that evaluates its database, unless a system identifier mismatch is unresolved.
- `api-contract`: the live physical jobs of a test application include the backup request job, and a test can run one pass of it by name.

## Impact

- `backend/internal/modules/backups/internal/physical/backuping`: scheduler passes and state loading.
- `backend/internal/modules/backups/internal/physical/catalog`: batch reads replace three per-database reads.
- `backend/internal/modules/backups/internal/config`: a list of configs with a pending backup request.
- `backend/internal/modules/backups/internal/physical/facade_service.go`: the job list.
- `backend/internal/app` and `backend/internal/app/apptest`: the live job list and a query counter for tests.
- `docs/architecture.md`, `docs/testing.md` and `docs/glossary.md`.
- Internal database load at 50 idle databases drops from about 152 queries per second to about 1.3, by calculation. No measurement was taken.
