# Design

## Context

See `proposal.md` for the problem. The design rests on these facts about the current code:

- Every time column of the internal database is `TIMESTAMPTZ`. `docs/architecture.md:160` states "Time is `TIMESTAMPTZ`, and all time is UTC", and `backend/AGENTS.md` adds that the current time comes from the clock in `Deps`. The clock returns UTC (`backend/internal/lib/clock/clock.go:18`).
- `database.Open` lets GORM open the pool from the DSN (`backend/internal/lib/database/database.go:17`). Under GORM the driver is pgx v5. Its `TIMESTAMPTZ` codec returns a scanned value in `time.Local` unless the codec has a `ScanLocation` (`pgx/v5@v5.13.0/pgtype/timestamptz.go:291-293`). With `TZ` set, `time.Local` is the operator's zone, so every time the backend reads back carries that zone. The moment is right; the calendar fields (`Year`, `Day`, `Hour`, `Weekday`) are the local ones.
- Every backend connection that reads application data from the internal database goes through `database.Open`, in the application, the dev seed and the test harness. The one exception is the staging step of internal database recovery, which connects to the maintenance database with `pgx.Connect` (`backend/internal/engines/postgresql/internaldb/staging_database.go:76`, `:106`, `:126`). It only creates, renames and drops databases and reads no stored times, so it stays as it is.
- The schedule library documents `TimeOfDay` as UTC (`backend/internal/lib/schedule/schedule.go:22`). Its daily, weekly and monthly helpers build slots with `time.Date(..., <input>.Location())` (`schedule.go:163`, `:241`, `:277`, `:325`, `:347`, `:351`), so a slot follows the location of the time passed in. The cron path converts to UTC first (`schedule.go:220`, `:295`).
- `IsDue` callers pass `now` from the clock, so run decisions are right today: the Databasus backup production (`backend/internal/modules/instancebackups/internal/catalog/production_service.go:76`), logical backups (`backend/internal/modules/backups/internal/logical/backup_scheduler_service.go:119`, `:131`), physical backups (`backend/internal/modules/backups/internal/physical/backuping/scheduler_decision.go:42`, `:56`, `:82`, `:89`) and verification (`backend/internal/modules/verification/internal/verifications/scheduler_service.go:93`).
- The one `GetNextOccurrence` caller is the Databasus backup's overdue check. `Destination.GetOutstandingDueAt` passes the last successful snapshot time or the selection time read from the database (`backend/internal/modules/instancebackups/internal/configs/config.go:115`, `:123-129`). On a server at UTC-3, a 04:00 UTC snapshot becomes 01:00 local, the next slot becomes 04:00 local, which is 07:00 UTC, and the destination turns overdue at 07:15 UTC. `recordOutstanding` stores that due time (`backend/internal/modules/instancebackups/internal/configs/service.go:328-345`).

The approach is constrained by these `backend/AGENTS.md` rules: code without domain knowledge stays in `internal/lib/<name>`; no package-level variable holds a dependency; take the current time from the clock and keep all time in UTC; every error of a package sits in its `errors.go`.

## Goals / Non-Goals

**Goals:**

- Every time the backend reads from the internal database is in UTC, whatever `TZ` the process runs with.
- The schedule library honors its UTC contract for any input, not only for inputs that happen to be UTC.
- A module API test runs the Databasus backup module on a server zone other than UTC, so this class of bug fails a test.

**Non-Goals:**

- Auditing times parsed from other sources, such as user database servers, the verification agent or external tools. They do not feed schedule slots today.
- Changing how the interface converts schedule times.

## Decisions

### 1. The internal database connection reads `TIMESTAMPTZ` in UTC

`database.Open` parses the DSN into a pgx connection config and opens the pool with `stdlib.OpenDB`. An after-connect hook registers a `TIMESTAMPTZ` type whose codec has `ScanLocation: time.UTC` on every new connection. GORM gets that pool through `postgres.New(postgres.Config{Conn: pool})`. The pool size settings and the ping stay as they are. Queries, repositories and stored values do not change.

This puts the UTC rule where times enter the backend, so every module gets it at once, including code written later.

Alternatives rejected:

- **Pass the hook through GORM's `OptionOpenDB` and keep GORM opening the pool.** The pgx `stdlib` connector keeps a single after-connect hook. When the DSN carries a `TimeZone` parameter, GORM appends its own hook after ours (`gorm.io/driver/postgres@v1.6.3/postgres.go:105-121`), and that hook replaces ours. An operator DSN would then silently bring the bug back.
- **Add `TimeZone=UTC` to the DSN.** GORM applies that parameter only to `timestamp` columns, not to `TIMESTAMPTZ` (`postgres.go:113-117`). The PostgreSQL session time zone also does not change how pgx scans binary `TIMESTAMPTZ` values. The DSN also belongs to the operator.
- **Set `time.Local = time.UTC` at startup.** It changes process-wide state of the standard library from one place in the application. Each entry point that opens the database, such as console commands and the test harness, would need the same line. It also hides the zone from code that might need it later, instead of making the database package own its output.
- **Convert with `.UTC()` in each repository or entity.** There are dozens of time fields. Each new one would be a new chance to forget, and the bug would come back one field at a time.

### 2. The schedule library converts its inputs to UTC

`IsDue` converts `now` and `lastRun` to UTC, and `GetNextOccurrence` converts `after`, before any helper runs. The helpers then build every slot in UTC, as the cron path already does. The two lines make the documented contract hold for every caller.

Alternatives rejected:

- **Rely on decision 1 alone.** It fixes issue 933, but the library would still be correct only for UTC inputs, and nothing in its signature says so. A future caller could pass a time from another source, such as an API request with an offset, and get a shifted slot.
- **Convert in the Databasus backup caller.** That fixes one caller and leaves the trap for the next one.

### 3. Tests reproduce the bug on a non-UTC server

- **Schedule unit tests.** The schedule library is pure logic, which `docs/testing.md` allows as a unit test. New scenarios pass inputs in `time.FixedZone` zones at UTC-3, UTC-5 and UTC+3 and check that daily, weekly and monthly `IsDue` and `GetNextOccurrence` give the UTC slots from the `recurring-schedules` delta spec. A fixed zone needs no tzdata in the test environment and has no daylight saving time to make results depend on the date.
- **Database test.** A test in `backend/internal/lib/database` stores a time and reads it back through `database.Open`, then checks that its location is `time.UTC`. It fails on the current code under any `TZ`, because the default codec returns `time.Local`, which is never the `time.UTC` location.
- **Databasus backup module API tests.** The module's root test package gets a `TestMain` in `backend/internal/modules/instancebackups/server_time_zone_test.go` that sets `time.Local` to a fixed UTC-3 zone before any test runs. Every scenario in the package then runs on a server west of UTC, as in the issue. The new regression test, labeled with issue 933, follows the `databasus-backup-alerts` scenarios: a daily 04:00 UTC schedule backs up at 04:00 UTC, the incident check at 07:16 UTC finds the destination healthy, the test application's mail server, which the package's apps have by default, holds no incident email to root admin, and the status at 04:01 UTC the next day shows an overdue time of 04:15 UTC. The same package checks that the status response returns `lastSuccessfulSnapshotAt` in UTC, as the `runtime-configuration` delta requires.

Alternatives rejected for the module test:

- **Set `TZ` in `TestMain`.** Go reads `TZ` once, the first time anything uses `time.Local`, and an imported package may already have done so before `TestMain`. Assigning `time.Local` before `m.Run()` is deterministic, and no test goroutine exists yet.
- **A separate test package only for the regression.** Running the whole module on a non-UTC zone also checks every existing scenario of the module, at no cost in test time.

### 4. No data migration

A wrong due time already stored in `instancebackups_destinations.outstanding_due_at` lies between two real slots. The delivery update clears every due time at or before the new snapshot (`backend/internal/modules/instancebackups/internal/configs/destination_repository.go:95`), so the first successful copy after the upgrade clears it, and the incident resolves as any overdue incident does. A migration would repair data that heals itself within one schedule period.

### 5. Documentation

`docs/architecture.md:160` gains the in-memory half of the rule: times read back from the internal database are UTC too. `backend/AGENTS.md` already says "All time is UTC" and does not change. The change does not touch the architecture itself, so no other section changes.

## Risks / Trade-offs

- [API responses change their offset on installations with a non-UTC `TZ`, from the server's offset to `Z`] → The moment is unchanged and the contract types these fields as `date-time`. The frontend parses them as dates and shows local time; it does not compare offset text (checked: no string slicing of timestamps under `frontend/src`).
- [Code that formats a time read from the database without `.UTC()` changes its output on such installations, for example the WAL archive time in a chain alert (`backend/internal/modules/backups/internal/physical/walstreaming/manager.go:476`)] → The output becomes UTC, which matches the rule. No production code compares times with `==` or uses them as map keys without converting to UTC first, so no comparison changes its result.
- [Opening the pool ourselves could drop a GORM default] → With `Conn` set, GORM skips only its own DSN parsing and its `TimeZone` hook for `timestamp` columns, which the schema does not use. Query execution mode and the other pgx defaults stay the same. The module API tests exercise the whole pool.
- [Setting `time.Local` in a test package could leak into another package] → Each Go test package is its own process, so the change stays inside the Databasus backup module's root test package.
- [An existing module scenario fails under UTC-3] → That scenario holds the same assumption the bug did. It is fixed as part of this change.

## Migration Plan

Deploy as a normal release. No schema or data change. A rollback restores the false incidents on non-UTC installations and nothing else.
