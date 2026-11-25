# Tasks

Backend test commands run with `TESTCONTAINERS_RYUK_DISABLED=true`, as `docs/testing.md` says under "Commands".

## 1. Reproduce the bug in the Databasus backup module

- [x] 1.1 Add `backend/internal/modules/instancebackups/server_time_zone_test.go` with a `TestMain` that assigns `time.Local` a fixed UTC-3 zone before `m.Run()`, as design decision 3 says. Add scenarios to `TestTrackInstanceBackupHealth` in `backup_health_test.go`, labeled with issue 933, for the two `databasus-backup-alerts` scenarios: a daily 04:00 UTC schedule backs up at 04:00 UTC, and the incident check at 07:16 UTC finds the destination healthy, opens no incident and leaves no message to root admin in the test application's mail server (`owner.App.Mail.GetMessagesTo`); with no other copy, the status at 04:01 UTC the next day shows an overdue time of 04:15 UTC. Add a scenario that the status response returns the destination's `lastSuccessfulSnapshotAt` in `time.UTC`, for the `runtime-configuration` delta. Run `go -C backend test -count=1 -run 'TestTrackInstanceBackupHealth' ./internal/modules/instancebackups` on the unfixed code and verify that the new scenarios fail with an OVERDUE incident and an incident email at 07:16 UTC and a UTC-3 offset in the response.

## 2. Schedule slots in UTC

- [x] 2.1 Add scenarios to `backend/internal/lib/schedule/schedule_test.go` for every `recurring-schedules` delta scenario, passing inputs in `time.FixedZone` zones at UTC-3, UTC-5 and UTC+3 to `IsDue` and `GetNextOccurrence` for daily, weekly and monthly schedules, including a daily schedule without a time of day. Verify that `go -C backend test -count=1 ./internal/lib/schedule` fails on the new scenarios only.
- [x] 2.2 Convert `now` and `lastRun` in `IsDue` and `after` in `GetNextOccurrence` to UTC before any helper runs, per design decision 2, and drop the conversions in the cron helpers that the entry points now cover. Verify with `go -C backend test -count=1 ./internal/lib/schedule` and with `go -C backend test -count=1 -run 'TestTrackInstanceBackupHealth' ./internal/modules/instancebackups`, where the issue 933 schedule scenarios now pass and the response-offset scenario still fails.

## 3. Database times in UTC

- [x] 3.1 Add a test in `backend/internal/lib/database`, in the test file that mirrors `database.go`, that stores a time through a connection from `database.Open`, reads it back and checks that its location is `time.UTC`. Verify that `go -C backend test -count=1 -run 'TestOpen' ./internal/lib/database` fails on the unfixed code.
- [x] 3.2 Change `database.Open` per design decision 1: parse the DSN into a pgx connection config, open the pool with `stdlib.OpenDB` and an after-connect hook that registers a `TIMESTAMPTZ` codec with `ScanLocation: time.UTC`, and hand the pool to GORM with `postgres.New(postgres.Config{Conn: ...})`, keeping the pool size settings and the ping. Verify with `go -C backend test -count=1 ./internal/lib/database`, `go -C backend test -count=1 ./internal/lib/testdb`, `go -C backend test -count=1 ./internal/app` and `go -C backend test -count=1 ./internal/app/apptest`.
- [x] 3.3 Extend the time rule at `docs/architecture.md:160` so it also covers times read back from the internal database. Verify by searching `docs/` and the `AGENTS.md` files for `UTC` and checking that every mention agrees with the new wording.

## 4. Callers on a non-UTC server

- [x] 4.1 Run `go -C backend test -count=1 ./internal/modules/instancebackups` and fix every scenario that fails because it assumed a UTC server zone, fixing the code or the test, whichever holds the assumption. Verify that the whole root package passes under the UTC-3 zone.
- [x] 4.2 Verify that the other schedule callers still pass with `go -C backend test -count=1 -run 'TestGetDueBackups' ./internal/modules/backups/internal/logical`, `go -C backend test -count=1 -run 'TestDecidePhysicalBackup' ./internal/modules/backups/internal/physical/backuping`, `go -C backend test -count=1 -run 'TestLogicalBackupScheduler' ./internal/modules/backups` and `go -C backend test -count=1 -run 'TestVerificationScheduler' ./internal/modules/verification`.
- [x] 4.3 Run `mise run backend:format`, then verify with `mise run backend:lint`.
