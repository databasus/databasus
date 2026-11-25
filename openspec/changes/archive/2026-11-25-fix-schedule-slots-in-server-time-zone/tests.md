# Test plan

## Impact

| Changed | Tests it can break | Why |
|---|---|---|
| `backend/internal/lib/schedule` | the package's own tests | `IsDue` and `GetNextOccurrence` convert their inputs to UTC, and the cron helpers lose their own conversion |
| Callers of `IsDue`: logical backups, physical backups, verification | `TestGetDueBackups`, `TestDecidePhysicalBackup` and `TestDecidePhysicalBackupAfterReload`, `TestLogicalBackupScheduler`, `TestVerificationScheduler` | they decide runs through the changed entry point; their inputs are already UTC, so the result must not change |
| `backend/internal/lib/database` (`database.Open`) | the package's own tests, including the new `TestOpen` and the transaction tests | the pool is now built from a pgx config with an after-connect hook and handed to GORM, instead of GORM opening it from the DSN |
| `backend/internal/lib/testdb` | its tests | its template step opens the internal database through `database.Open` |
| `backend/internal/app` (`app.go`, the dev seed) | its tests, such as `TestNew` and the system health tests | the application opens its pool through `database.Open` at startup and runs migrations on it |
| `backend/internal/app/apptest` | its tests, such as the SQL transaction, physical catalog and isolation tests | the harness opens the internal database through `database.Open` for direct SQL, the physical catalog, recovery state and notifier helpers |
| `backend/internal/modules/instancebackups` root package | every scenario of the package | the overdue check calls `GetNextOccurrence` with times read from the database, and the new `TestMain` runs the whole package at UTC-3 |

## Not selected

- Other modules and libraries that read times from the internal database, such as audit paging, backups, restores and storages. Two parts of the change reach them:
  - The read location: the test environment and CI run with the system zone UTC and no `TZ`, so `time.Local` already has a zero offset there. The location changes from `Local` to `UTC` with the same instant, and JSON output stays `Z`. No production code compares times with `==` or uses them as map keys, so no comparison changes its result.
  - The pool construction: every module queries the same pool through `database.Conn` and the transactor, and GORM registers its callbacks and builds queries the same way whatever created the pool. With `Conn` set, GORM skips only its DSN parsing, the simple-protocol switch the backend does not set and its `TimeZone` hook for `timestamp` columns the schema does not use. The pgx config comes from the same `pgx.ParseConfig` call and opens through the same `stdlib.OpenDB`. Queries, inserts with `RETURNING`, raw SQL, transactions and migrations through the new pool are exercised by the selected `lib/database`, `app`, `app/apptest` and `instancebackups` packages.
- `backend/internal/app/systemhttpapi`: generated contract code that does not open the database.
- `backend/internal/engines/postgresql/internaldb`: its staging connections use `pgx.Connect` directly and do not change.
- `backend/internal/modules/instancebackups/e2e`, `backend/internal/modules/backups/e2e`, `suite upgrade` and `suite e2e-browser`: the candidate image runs without `TZ`, so its behavior is the same as before for the reasons above. The `Dockerfile` and the end-to-end code are unchanged.
- Frontend tests: no frontend code changes, and the API output is unchanged in the test environment.
- `docs/architecture.md`: prose only.

## Selection

```tests
backend ./internal/lib/schedule
backend ./internal/lib/database
backend ./internal/lib/testdb
backend ./internal/app
backend ./internal/app/apptest
backend ./internal/modules/instancebackups
backend ./internal/modules/backups/internal/logical -run TestGetDueBackups
backend ./internal/modules/backups/internal/physical/backuping -run TestDecidePhysicalBackup
backend ./internal/modules/backups -run TestLogicalBackupScheduler
backend ./internal/modules/verification -run TestVerificationScheduler
```
