# Proposal

## Why

When the Databasus container runs with a `TZ` other than UTC, the Databasus backup opens a false OVERDUE incident every day and emails root admin about it, although every backup ran on time (issue 933). The backend stores every time in UTC, but after reading a time from the internal database it holds it in the server's time zone. The schedule code then builds the expected slot on the server's calendar instead of the UTC calendar the schedule is saved in.

## What Changes

- The backend holds every time it reads from the internal database in UTC, whatever the server's time zone. Stored values do not change, because every time column is already `TIMESTAMPTZ`.
- Schedule slots of daily, weekly and monthly schedules are computed on the UTC calendar whatever time zone the input time carries, as cron schedules already are. This covers both the decision that a run is due and the next expected occurrence.
- The Databasus backup no longer reports a backup that ran on schedule as overdue when the server's time zone is not UTC.
- On an installation with a non-UTC `TZ`, the API returns times read from the database with a UTC offset (`Z`) instead of the server's offset. The moment is the same, and the interface already shows every time in the user's local time, so nothing changes on screen. A client of the API that compared the offset text would see the difference; the contract defines these fields as date-times, not as text in a particular offset.
- No data migration. An overdue deadline already stored with the wrong slot is cleared by the next successful copy, as any outstanding deadline is.

This change answers to the root `AGENTS.md` and `backend/AGENTS.md`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `recurring-schedules`: a new requirement that slots of every schedule type stay on the UTC calendar whatever the server's time zone, for logical backups, physical backups, verification and Databasus backups.
- `databasus-backup-alerts`: a new requirement that due times and overdue deadlines follow the schedule's UTC slots, so a backup that ran on schedule is not overdue on a server whose time zone is not UTC.
- `runtime-configuration`: a new requirement that times read back from the internal database, and the API's copies of them, stay in UTC whatever the server's time zone.

## Out of scope

- Time zones of the user databases Databasus backs up, and the verification agent's connections to them.
- The environment of the tools the application starts. They keep the operator's `TZ`, as `runtime-configuration` already requires.
- Log timestamps, which are already written in UTC.
- How the interface shows and converts schedule times. It already converts between local time and UTC.
- A per-schedule time zone or any setting to choose one.
- Repairing stored overdue deadlines with a migration.

## Impact

- `backend/internal/lib/database`: the internal database connection reads `TIMESTAMPTZ` values in UTC. Every module reads through it, so every time loaded from the database changes its location from the server's zone to UTC. No code compares times by location or uses them as map keys without converting to UTC first.
- `backend/internal/lib/schedule`: `IsDue` and `GetNextOccurrence` convert their time arguments to UTC.
- Callers of the schedule library: the Databasus backup's production and overdue check, the logical and physical backup schedulers and the verification scheduler. Only the overdue check changes behavior; the others already pass UTC.
- API responses carry `Z` offsets on installations with a non-UTC `TZ`.
- `docs/architecture.md`: the time rule under "Data" also covers times held in memory after reading them.
