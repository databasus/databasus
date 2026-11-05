## Why

The scheduled availability check of user databases is the last feature with its own tables, user operations and screens that still lives in the old code. `backend/internal/features/healthcheck` holds about 1.1 thousand lines of code and 1.1 thousand lines of tests. Its three Gin routes are outside the contract and answer every error as `400` with the internal text. Its loop starts a bare goroutine per database every minute (`attempt/background_service.go:66`), and stopping Databasus does not wait for them. It checks access through `workspaces_bridge.Access.IsActorAllowed` and `databases_bridge.Access`, which have no other consumers left, and its two tables cascade from `databases`, which `docs/architecture.md`, "Data", forbids between modules. No spec describes the feature.

Milestone 13a of the backend rewrite plan moves it into `internal/modules/healthcheck`, so milestone 13b can move telemetry and the system endpoints, and 13c can delete the bridges.

## What Changes

- **The module.** `internal/modules/healthcheck` appears with `module.go`, `Deps`, `New` and its code under `internal/`: configs, attempts with the probe job, and the HTTP handlers. Repositories run through `database.Conn(ctx)`. Saving a config and its audit event share one transaction. Access goes through `workspaces.Authorize`. Databases are read, probed and given their health status through `databases.Module`, and notifications go through `notifiers.Module`.
- **The probe job.** The checks run as the `background.Runner` job `healthcheck_probe` once a minute. A pass picks the databases that are due by their interval and probes them concurrently with a fixed limit, through `background.Group` and a semaphore. The pass waits for its probes, so shutdown waits for them too. A probe that the shutdown interrupts records nothing. Every time comes from the clock in `Deps`.
- **The rules of a check stay as they are**: the threshold of failed attempts before a database counts as unavailable, the return to available on the first success, the notifications and the retention of attempts. A new database starts as available, so its first successful check notifies nobody. A database whose checks were switched off has no status, so the first successful check after they are switched on again still notifies that the database is online, which confirms that Databasus reached it.
- **Config lifecycle.** A database gets the default config in its creation transaction, as today. A copy of a database gets the original's config, which is new: today a copy gets the default. Deleting a database deletes its config and attempts in the deletion transaction. Reading a config no longer writes one: a migration gives the default config to every database that lacks one.
- **Three operations enter `backend/api/openapi.yaml` under the tag `healthcheck`**, designed by the `api-design` skill: read and save the config of a database, and list the attempts of a database from a given time, newest first, with a cursor. Errors are `Problem` bodies with a stable `code`.
- **The schema.** One migration renames three config columns to the names of the contract, repairs rows that would break the new constraints, adds `CHECK` constraints on the positive numbers and on the attempt status, gives missing configs the default, and replaces the cascading references to `databases` with `RESTRICT`.
- **The frontend** calls the three operations through the generated client, uses the contract's types instead of the hand-written models, follows the attempts cursor for the chosen period, and shows errors through `translateApiError`. The healthcheck screens move from the `features/healthcheck` slice into the `features/databases` slice, their only consumer, so the database screens no longer import another slice of the same layer. `HealthckeckAttemptsComponent` becomes `HealthcheckAttemptsComponent`.
- **Tests.** The old controller and use-case tests become API tests of the module on `apptest`, with probed databases in testcontainers instead of the shared test server. A `coverage.md` in the change maps every old test to its replacement. A migration test checks the repairs and constraints. The upgrade test writes a config with non-default values before the upgrade and reads it after.
- **Removed:** `internal/features/healthcheck`, `workspaces_bridge.Access`, `databases_bridge.Access`, the `features/audit_logs` wrapper and the healthcheck entry of `internal/app/background_loops.go`.
- **BREAKING** for users' scripts: `POST /api/v1/healthcheck-config`, `GET /api/v1/healthcheck-config/{databaseId}` and `GET /api/v1/healthcheck-attempts/{databaseId}` give way to the operations of design.md. The config fields `isSentNotificationWhenUnavailable`, `attemptsBeforeConcideredAsDown` and `storeAttemptsDays` are renamed to `isNotificationEnabled`, `failedAttemptsBeforeUnavailable` and `attemptRetentionDays`, the attempt list takes a required `since` and a cursor instead of an optional `afterDate`, and errors are `Problem` bodies.

## Capabilities

### New Capabilities

- `database-healthcheck`: when a database is checked, when it counts as unavailable and available again, when notifications go out, what switching the checks off resets, how long attempts are kept, what a stop of Databasus does to a running check, and how the attempts are listed.

### Modified Capabilities

- `databases`: a copy gets the original's healthcheck config, and deleting a database also removes its healthcheck config and attempts.
- `workspace-access`: a new requirement that the healthcheck operations of a workspace's databases follow the workspace roles.
- `upgrade-test`: a new requirement that a healthcheck config written before the upgrade is there after it.

## Out of scope

- Telemetry, the system health endpoint, `/disk/usage` and the transfer of `maxRamGb`. That is milestone 13b.
- Deleting the remaining bridges, `di.go`, the `testhelpers` tag and the linter exceptions. That is milestone 13c.
- New rules for checks or notifications. The first-success notification, the threshold and the retention keep today's behavior.
- The speed of the attempts chart on long periods. The screen keeps drawing one mark per attempt, so a week at one check a minute stays about ten thousand marks, as today.
- Moving the interval to `lib/schedule`. The interval stays a number of minutes, because the shortest calendar schedule there is hourly, and the default check runs every minute.
- The health status column and its writes, which belong to `databases`.
- The other same-layer imports of `frontend/src/features/databases/ui/DatabaseComponent.tsx`, from `features/backups` and `features/verification`.

## Impact

- **Backend, new:** `internal/modules/healthcheck`; one migration and its test in `backend/migrations`.
- **Backend, changed:**
  - `backend/api/openapi.yaml` and the generated code;
  - `internal/app`: construction of the module after `databases` and `notifiers`, its routes, its job, its subscriptions in `wiring.go`, and the removal of the old routes, the old loop and the old getters;
  - `backend/e2e/upgrade`: a previous-release writer and a current-release reader for the config;
  - `backend/tools/archlint` needs no change: it already lists `healthcheck` as a module.
- **Backend, deleted:** `internal/features/healthcheck`, `internal/features/workspaces/access.go` with its test, `internal/features/databases/access.go`, and `internal/features/audit_logs`.
- **Frontend:** `frontend/src/entity/healthcheck`; `frontend/src/features/healthcheck`, which moves into `frontend/src/features/databases/ui/healthcheck`; and its two consumers, `DatabaseComponent.tsx` and `DatabaseConfigComponent.tsx`. No new error code, so the six dictionaries need no new text.
- **Running instances:** configs and attempts are kept. Databases without a config get the default at the upgrade instead of at the first opening of their page. Rows that would break the new constraints are repaired as design.md describes.
- **Documents:** `docs/glossary.md` (rows for the healthcheck config and the healthcheck attempt, which it lacks), and `temp/refactor/plan.md` (the checklist and the table of temporary relaxations). `docs/architecture.md` already lists the module and its place in the order.
- **Rules the change answers to:** the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
