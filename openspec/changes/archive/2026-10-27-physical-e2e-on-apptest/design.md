## Context

Why the change is needed is in proposal.md, section Why. This document covers what shapes the approach. Paths are under `backend/internal/` unless they start with another top-level directory. `TP` stands for `features/tests/physical/postgresql`, `BP` for `features/backups/backups/backuping/physical` and `UP` for `features/backups/backups/usecases/physical/postgresql`.

**The old scenarios.** `TP/pg17/backup_restore_test.go` and `TP/pg18/backup_restore_test.go` are one-line wrappers around the 14 bodies in `TP/shared/bodies.go:29-455`, and pg18 leaves out the two bastion bodies. Each version is its own test binary, and `TP/pg17/setup_test.go` calls `TP/shared/setup.go:23`, which starts a fresh scheduler and WAL stream supervisor for the whole process through `BP/testing.go:146` and `BP/testing.go:189`. The cleaner is not started. Two bodies run one cleaner sweep through `BP/testing.go:233`, which sweeps one database because the old suite shares one internal database across packages. A body:
- starts its source with `containers.StartPhysicalPostgres`, `StartPhysicalPrimaryWithStandby` or `StartPhysicalPostgresBehindSshBastion` (`util/testing/containers/postgres_physical.go:154-222`), which need no build tag;
- seeds the workspace, a local storage, a notifier and the physical database with SQL through `UP/testing.go:159-240`, with the superuser `testuser`/`testpassword` and the database `testdb` (`features/databases/testing.go:153`);
- builds its own Gin router around the old physical, database and config controllers (`TP/shared/helpers.go:205-229`);
- switches the database to a replication-only user through `/limited-users` and `PUT /databases/{id}`, then checks it with `pgphysical.NewSource(...).ShouldSuggestReplicationOnlyUser` (`TP/shared/helpers.go:298-336`);
- reads the catalog directly: `FindByChainSpan` for committed segments (`helpers.go:685`, `sshtunnel.go:166`), `DeleteByID` to punch a WAL gap (`bodies.go:311`), `UP.WaitForWalGap` (`bodies.go:313`), `chain_view.ResolveRestoreSet` (`helpers.go:777-823`), the history repository (`helpers.go:1122`), three backup repositories for the re-anchoring (`helpers.go:1199-1223`) and the streamer repository (`sshtunnel.go:157`);
- parses LSNs with `util/walmath` and decodes the old routes into `dto/physical` types;
- waits with `time.Sleep` loops, each with its own timeout, and sleeps on purpose three times: two seconds on each side of a PITR target (`helpers.go:855-857`, `bodies.go:357`) and 75 seconds of bastion outage (`bodies.go:477`).

**The old loops.** `BP/di.go:45-105` builds the scheduler, the cleaner and the supervisor as `sync.OnceValue` singletons from positional struct literals. Each `Run` panics when it is called a second time on the same instance (`BP/scheduler.go:69-72`, `BP/cleaner.go:42-45`, `BP/wal_supervisor.go:107-110`). `internal/app/background_loops.go:17-36` runs the singletons only from `Serve` (`internal/app/server.go:69`). The facade `services` keeps the singleton supervisor for the live state route (`features/backups/backups/services/di.go:31`). The scheduler starts each backup with a bare `go s.backuper.MakeBackup(ctx, ...)` (`BP/scheduler.go:385`), and the backuper runs it on `context.Background()` registered in the process-wide cancellation registry (`BP/backuper.go:106-107`, `:211-212`). WAL streamers derive from the supervisor's context (`BP/wal_supervisor.go:285`). The cleaner's tick runs the retention, the orphan sweep and the sample expiry (`BP/cleaner.go:58-70`), and nothing outside the loop calls them. The old code reads `storage.GetDb()` at call time (`storage/storage.go:22`), so a loop started for one test application follows the database of whichever application is installed.

**The test application.** `apptest.New` builds the server's application with `app.New`, including the old physical routes (`internal/app/routes.go:60-68`) and the old physical event handlers (`internal/app/wiring.go:52-85`), but it never calls `startBackgroundLoops` (`internal/app/apptest/apptest.go:71-180`, `docs/testing.md:37`). One application lives per process at a time (`apptest.go:50`). `CallOldAPI` always sends the session's token and buffers the whole answer (`internal/app/apptest/old_api.go:19-53`). `enginetest.Team` builds its application without options (`internal/app/apptest/enginetest/team.go:36-48`), and `enginetest.NewDatabaseSettings` knows only the logical engines (`enginetest/database_settings.go:19-33`).

**The rules the new code falls under.** The new packages are in the target layout, so `forbidigo` rejects `time.Sleep`, `Eventually`, `context.Background` and, in a non-test file, `os/exec` (`.golangci.yml:67-87`, `:168-180`, `:204-206`). `archlint` lets a production file of a module import only lower layers (`tools/archlint/layers.go:100-130`), so no non-test package under `internal/modules` may import `apptest`. `testwait.Until` polls up to the test's deadline and has no per-call timeout (`lib/testwait/testwait.go:26-70`).

**What the API returns.** The physical database settings in the contract carry no slot names (`api/openapi.yaml:3971-4021`). The old list returns FULL, INCREMENTAL and WAL rows with their timeline, `startLsn`, `stopLsn`, the chain links and, for WAL rows, the segment count and `hasGapBefore` (`features/backups/backups/dto/physical/dto.go:11-46`). It carries no typed error reason. The statistics report the streamer's state, `RUNNING` only while the row is `RUNNING` and its heartbeat is younger than 90 seconds (`core/physical/wal_health/streamer_state.go:12-22`, `wal_health/constants.go:34`). The supervisor heartbeats every streamer it owns on each reconcile pass (`BP/wal_supervisor.go:198`, `:567-579`). The restore token answers 422 with the gap in the error text (`controllers/physical/controller.go:361-368`).

**Creation on a standby.** `createDatabase` detects the cluster and runs the replication test (`modules/databases/internal/userdatabase/service.go:455-477`). For a WAL-streaming type the test checks only the privilege to call `pg_switch_wal` (`engines/postgresql/physical/forced_wal_rotation.go:18-30`), not the call itself, so it should pass on a read-only standby. No test has shown that yet.

**CI.** `mise-tasks/backend/test-group-packages:13-14` maps the `physical` group to `features/tests/physical/...` and the `logical` group to all of `modules/backups/e2e/...`. `mise-tasks/backend/test-group-tests` shards a group by the top-level test names it finds in the test files, weighted by `backend/test-shard-weights.txt`. `mise run backend:test-package` gives a run 15 minutes, and one physical version takes about 7 (`mise.toml:92-104`).

## Goals / Non-Goals

**Goals:**
- Each of the 14 scenarios checks the same outcome as before, and a table below maps every old check to its new one.
- The scenarios depend on nothing that `engines-physical` or `physical-backups-module` move, apart from the internal-table queries listed in decision 8.
- The production changes add constructors, the cleaner's single pass and two methods of `internal/app` without changing backup execution, and repair WAL list grouping under out-of-order uploads.

**Non-Goals:**
- Deciding what the live-loops option becomes in `physical-backups-module`. Decision 4 records the direction.
- Deleting tagged physical helpers that old package tests still use.

## Decisions

### 1. A shared scenario package in the app layer, one test package per version

The bodies and their helpers move into `internal/app/apptest/physicaltest`, a non-test package like `enginetest` that only the end-to-end tests import. Two test packages, `internal/modules/backups/e2e/postgresqlphysical/pg17` and `.../pg18`, hold the top-level tests and call it with their image and version.

Alternatives rejected:
- *A non-test package under `modules/backups/e2e`.* `archlint` rejects a module package that imports `apptest`, because the app layer is above the modules (`tools/archlint/layers.go:118-130`).
- *One test package that runs both versions as subtests.* `apptest` allows one application per process, so the versions would run one after the other. At about 7 minutes each, the package would outlast the 15-minute limit of `backend:test-package`.
- *A copy of the bodies per version.* Two copies drift, and the review of this change could no longer compare one old body with one new one.

The shared package runs the recovery script and `docker` through `lib/process`, because `forbidigo` rejects `os/exec` in its non-test files. This changes nothing in what the script receives.

### 2. Top-level tests grouped by what they test

The testing skill names a Go function after what it tests and each scenario after its result and condition. CI shards a group by top-level test names, so a single function would put every scenario into one shard. The scenarios therefore form six functions, and each version package declares the ones it runs:

| Function | Scenario sentence | Old body |
|---|---|---|
| `TestRestorePhysicalBackupChain` | recovers rows written before the full backup with only a full backup | `RunFullOnlyRecoversBaseRows` |
| | recovers rows of every backup with a full backup and two incrementals | `RunFullPlusTwoIncrementalsRecoversAllRows` |
| `TestRestorePhysicalBackupToTargetTime` | recovers rows up to the target and drops later ones with streamed wal | `RunFullTwoIncrementalsPlusWalRecoversToTarget` |
| | returns 422 for the restore token when a wal gap lies before the target | `RunWhenWalGapBeforeTargetTokenRequestReturns422` |
| | keeps a target past the full backup restorable after the orphan sweep | `RunWhenOrphanSweepRunsAfterFullTargetPastFullStaysRestorable` |
| `TestRestorePhysicalBackupOnHost` | serves the restored rows through the image entrypoint after a host restore | `RunBootViaEntrypointVolumeMountRecoversBaseRows` |
| | replays to the target through the image entrypoint after a host restore | `RunPitrBootViaEntrypointVolumeMountRecoversToTarget` |
| | refuses to restore when the output directory holds a postgresql directory | `RunRejectsMisaimedRestoreTarget` |
| `TestPhysicalBackupAfterPromotion` | catalogs the new timeline history on the database after the standby is promoted | `RunWalStreamCatalogsHistoryOnTimelineSwitch` |
| | starts a full backup on the new timeline after the standby is promoted | `RunPromotionReanchorsIncrementalChain` |
| `TestPhysicalBackupReplicationSlot` | creates the slot when wal streaming starts and drops it when the database is deleted | `RunWalSlotAppearsWhenBackupingStartsRemovedWhenDatabaseDeleted` |
| | drops the slot when a database that streamed wal is deleted | `RunWalSlotWhenDatabaseDeletedWithStreamedWalSlotRemovedSoNoWalStuck` |
| `TestPhysicalBackupOverSshTunnel` | recovers rows up to the target with a chain built through the bastion | `RunOverSshTunnelRecoversToTarget` |
| | keeps the slot and resumes streaming after the bastion dies mid-stream | `RunWhenBastionDiesMidStreamKeepsTheSlotAndResumes` |

pg18 does not declare `TestPhysicalBackupOverSshTunnel`, as today. The bastion outage scenario, the slowest by far, shares its function only with the other tunnel scenario, so the sharding can still give it a shard of its own. `backend/test-shard-weights.txt` replaces the 14 old names with the six new ones, using the durations of one local run of each.

Alternatives rejected:
- *One function per version with 14 subtests.* All 14 scenarios would run on one CI shard.
- *Keeping the 14 old function names.* They break the naming rule of the testing skill, and this change rewrites every function anyway.

### 3. Constructors and a cleaner pass in the old loop package

`BP/di.go` gains `NewPhysicalBackupsScheduler`, `NewPhysicalBackupCleaner` and `NewPhysicalWalStreamSupervisor`, which build fresh instances from the same dependencies as today. The three singletons become `sync.OnceValue` over them, so the server builds the same objects as before. The scheduler of a fresh instance uses the singleton backuper, which keeps no run state. `PhysicalBackupCleaner` gains `RunPass(ctx) error`: the body of its tick, with the errors of the retention and orphan steps joined. `Run` calls it on each tick and logs its error, as it logs each step's error today. `CreateTestPhysicalScheduler` and `CreateTestPhysicalCleaner` in `BP/testing.go` call the constructors instead of repeating the literals.

`internal/app` gains two methods in `background_loops.go`:
- `StartPhysicalBackupLoops(ctx)` starts a fresh scheduler and supervisor in a `background.Group` of its own on `ctx`. It returns a stop function that waits, bounded by the context it is given, for both loops to return, and then for the process-wide cancellation registry to hold no work. The registry is where the detached backuper and the streamers register (`BP/backuper.go:107`, `BP/wal_supervisor.go:294`), and the shutdown of `Serve` waits on it the same way (`internal/app/server.go:86`).
- `RunPhysicalBackupCleanerPass(ctx)` runs `RunPass` of a fresh cleaner.

`Serve` keeps the singletons, because the live state route reads the singleton supervisor.

Alternatives rejected:
- *Running the singletons once per test process.* The scheduler and the supervisor would outlive the test that started them. A backup still running from test N would write its outcome into the database of test N+1, because the old code reads `storage.GetDb()` at call time, and the streamers of test N would keep dialling its stopped containers.
- *Keeping the option in a file behind the `testhelpers` tag that uses `CreateTest*`.* The new packages would depend on the tag again, and `physical-backups-module` would delete those factories under the option.
- *Building the instances in `apptest`.* The fields are unexported and the literals positional, so only the package itself can build them.

### 4. The live-loops option and the cleaner pass in `apptest`

`apptest.WithPhysicalBackupLoops()` makes `New` call `StartPhysicalBackupLoops` after the modules start and the server listens. Its cleanup is registered last, so it runs first: it cancels the loops' context and calls the stop function with a two-minute bound. A loop or a registered run that is still alive after that fails the test with `tb.Errorf`. The containers a test starts register their cleanup after `New`, so they stop before this cleanup runs, and a backup still talking to its source fails fast instead of holding the wait. `(*App).RunPhysicalBackupCleanerPass(tb)` runs one pass and fails the test on its error. Without the option nothing physical starts, which the delta spec of `api-contract` states. `docs/testing.md` names the exception in "The test application".

The cleaner stays out of the option. The old suite never ran it live, two scenarios need its sweep at a known point, and a one-minute tick would make them depend on time. The test databases are now one per test, so a full pass is safe, and the old per-database sweep is no longer needed.

In `physical-backups-module` the loops become `background.Runner` jobs. The option then probably starts the scheduler and the WAL stream reconciliation as jobs on a real clock. That change decides the form and removes the exception from `docs/testing.md` if it no longer applies.

Alternatives rejected:
- *Starting the cleaner live as well, as the rewrite plan first suggested.* Covered above.
- *A per-call `Start/Stop` method instead of an option.* Every physical scenario needs the loops from its first request, and an option keeps the cleanup inside `New`.

The application tests cover shutdown with two consecutive live applications and check that no registered work or active source replication slot remains after each application ends. A registered run that outlives a short internal stop bound must report a test error.

### 5. Data through the API

The scenarios build their team with `enginetest.NewTeam(t, apptest.WithPhysicalBackupLoops())`, so `NewTeam` takes options. The team creates the physical database with `createDatabase`, with the source's address, the superuser, `sslMode` `disable`, the backup type and, for the bastion scenarios, `enginetest.NewBastionSSHTunnelSettings`. `enginetest.NewDatabaseSettings` gains the `PostgresqlPhysicalSettings` branch. The replication-only user comes from `Team.CreateLimitedUser` and `Team.UpdateDatabaseSettings`, passing the new role’s password when it replaces the superuser credentials. Backups are enabled with `updateBackupConfig` with the once-a-year cron and the 64 MiB lag threshold the old helper uses (`TP/shared/helpers.go:231-279`). No mTLS setting is needed, because every old fixture passes `TLS: nil`.

Two details differ from the old fixture, and neither is something a scenario checks. The owner is a non-admin member, as in `enginetest.Team`, where the old fixture used an admin. No notifier is attached, because no scenario checks a notification.

The two promotion scenarios create the database on the standby. Context explains why the replication test should pass there. Task 2.1 builds this case first, before the scenarios depend on it.

Alternatives rejected:
- *Keeping the SQL fixture.* It is behind the tag and is deleted in `physical-backups-module`.

### 6. Old routes through `apptest`, answers in test-local types

The protected old routes go through `CallOldAPI`. `apptest` gains `CallOldPublicAPI(tb, call)`, which sends no token, for the recovery script, and `DownloadOldPublicAPI(tb, path, destinationPath)`, which streams the restore bundle into a file and returns the status. The scenario package decodes the answers into small structs of its own with only the fields it reads.

Alternatives rejected:
- *A flag on `OldAPICall` that drops the token.* `CallOldAPI` still takes a session, so the call would read as authenticated when it is not.
- *Decoding into `dto/physical`.* `physical-backups-module` moves that package, and the scenarios would change with it.
- *Buffering the bundle in memory.* The old recorder did, but a file matches what the recovery script reads and keeps the test's memory flat.

### 7. No imports of what the next changes move

| Old dependency | Replacement | Moved by |
|---|---|---|
| `util/walmath` for LSNs | a local parser of the `X/Y` text into an integer | `engines-physical` |
| `ShouldSuggestReplicationOnlyUser` | a query of `pg_roles` on the source: `rolreplication` set, `rolsuper`, `rolcreaterole`, `rolcreatedb` and `rolbypassrls` clear | `engines-physical` changes the engine's settings type |
| `UP.GenerateWalActivity`, `ForceWalRotation`, `WaitForWalSummaries`, `SlotExists`, `OpenAdminConn` | source helpers of the scenario package over `pgx` | `physical-backups-module` deletes `UP/testing.go` |
| slot names from the fixture | `pg_replication_slots` on the source, filtered by the `databasus_slot_` and `databasus_basebackup_` prefixes | the contract does not return slot names; each test starts its own source, so every such slot belongs to its database |
| `dto/physical`, `physical_enums` | test-local types and string constants | `physical-backups-module` |

The engine keeps its own tests of `ShouldSuggestReplicationOnlyUser`. The scenario checks the role the backups run as, which is what the old assertion protected.

### 8. Catalog reads become API reads, with four queries left

| Old read | New read |
|---|---|
| completed FULL and incrementals, failing on `ERROR` and `CHAIN_BROKEN` | the old list filtered by type |
| committed segments past the chain tip | WAL rows of the old list on the tip's timeline, by `stopLsn` |
| committed segment count before and after the outage | the sum of `walSegmentsCount` over the WAL rows on timeline 1 |
| `WaitForWalGap` | a WAL row with `hasGapBefore` |
| `ResolveRestoreSet(nil)` reaching an LSN | WAL rows on the tip's timeline with no `hasGapBefore` after the tip and a `stopLsn` at or past it |
| `ResolveRestoreSet(target)` | the restore token for the target answering 200 |
| streamer not `FAILED` | the statistics reporting the streamer `RUNNING` |
| completed FULL on timeline 2 and the newest FULL of any status | FULL rows of the old list with their status and timeline |

The streamer check is the same condition in other words. A `FAILED` row reports `NOT_RUNNING` at once. A healthy owned streamer is heartbeated on every reconcile pass, so it stays `RUNNING` well inside the 90-second staleness.

The four internal-table accesses that remain, which `physical-backups-module` updates when it renames the tables:

1. `QuerySQL` on `physical_wal_segments`: the IDs of the committed segments (`file_name IS NOT NULL`) of the database on the tip's timeline, ordered by `start_lsn`. The list groups segments into ranges and hands out no segment ID.
2. `ExecuteSQL` on `physical_wal_segments`: delete the middle one of those segments. The API cannot create a WAL gap.
3. `QuerySQL` on `physical_wal_history_files`: whether a row exists for the database's `databases.id` and timeline 2. The API does not return history files, and the row on the parent ID is what the issue #643 regression test pins.
4. `QuerySQL` on `physical_incremental_backups`: `status` and `error_reason` of the incrementals whose `root_full_backup_id` is the stale FULL. The list carries only the free-text failure, and the scenario must see `TIMELINE_SWITCH_DETECTED`.

### 9. Waiting on conditions

Every poll becomes `testwait.Until`. A condition that sees a backup in `ERROR` or `CHAIN_BROKEN` calls `t.Fatalf` itself, so a broken chain still fails at once. The per-step timeouts disappear, because `testwait.Until` waits up to the test's deadline. A failure that is not terminal now takes longer to report, and the message keeps the last state, as the re-anchoring wait already does (`TP/shared/helpers.go:1153-1197`).

The deliberate sleeps become conditions:
- *The margin around a PITR target.* The scenario reads `clock_timestamp()` on the source and waits until the source's clock is two seconds past it before it takes the target. After the target, it waits until the clock is two more seconds past. The target is compared with commit times on the source, so its clock is the one that matters.
- *The bastion outage.* This is a negative check, so it first waits for a sign that the outage was noticed: the statistics report the source unreachable or more receiver restarts than before the outage. It then watches the statistics until 75 seconds have passed since the bastion stopped, failing as soon as the streamer is not `RUNNING`. The watch uses a timer and a ticker, which the linter allows.

Alternatives rejected:
- *Keeping `time.Sleep` with a `nolint`.* The testing skill forbids fixed waits, and the conditions above say what the test actually waits for.

### 10. CI groups and the old package

`test-group-packages` maps `physical` to `./internal/modules/backups/e2e/postgresqlphysical/...` and leaves that subtree out of `logical`. The shared package under `internal/app/apptest` has no tests, so its place in the `other` group costs nothing. The Go packages under `backend/internal/features/tests/physical` are deleted in the same change. The `testdata/mtls` assets remain at their existing paths because the old mTLS usecase tests still consume them and `.gitleaksignore` fingerprints include both the path and the introducing commit. Keeping these existing assets avoids adding a new private-key exception. Each old helper it used is deleted too if nothing else uses it any more, for example `SubscribeToBackupConfigEventsWithoutApp` (`BP/database_deleting_subscription_testhelpers.go:25`). The ones that old package tests still use stay until `physical-backups-module`.

### 11. WAL ranges preserve gaps under concurrent uploads

Receive timestamps can differ from LSN order. Grouping by receive time and starting a group only at a segment marked as following a gap can let a later-arriving segment with a lower LSN join that group, hiding the gap. The list first finds display boundaries around FULL, INCREMENTAL and timeline switches after applying filters. Within each WAL display group it orders the selected segments by LSN and starts a new range wherever consecutive LSNs do not connect. The gap marker remains computed against every committed segment before filtering, so a filter does not invent a catalog gap.

Deterministic API tests supply receive times in different orders, check that contiguous segments still merge, that real gaps remain visible, and that date filtering does not group across an excluded segment. Existing controller tests continue to cover display boundaries, paging and stable range identity. The repair enforces `physical-backup-wal-ranges` without changing its requirements or the API shape.

## Constraints from the module rules

- `backend/AGENTS.md`, "Long-running work": the new loop starter runs its goroutines through `background.Group`, and the shared package starts tools through `lib/process`. The old loops keep their own ticker loops and `hasRun` guards until `physical-backups-module`, as the rewrite plan says.
- `backend/AGENTS.md`, "Where code goes": wiring stays in `internal/app`, and `apptest` only calls it.
- The root `AGENTS.md`, "Naming" and the testing skill: function and scenario names, domain variable names, data through the API, no fixed waits.
- `docs/architecture.md` does not change. The change adds no module, layer or boundary.

## Risks / Trade-offs

- [The databases module refuses to create a database on a standby] → Task 2.1 creates one before any scenario depends on it. If the module refuses, the change stops and the decision goes to the user, because keeping those two scenarios would need either a fixture outside the API or a change to the module.
- [The statistics show no sign of the bastion outage] → The scenario then has no sign to wait for before watching its window. If neither the unreachable source nor the restart count changes during the outage, the watch starts when the bastion stops, as the old sleep did, and the design note in the scenario says why.
- [A failing step waits until the test's deadline] → Terminal backup states fail at once, and every wait reports its last state.
- [A detached backup keeps running after its test] → The stop function waits for the cancellation registry, and the containers stop before it waits, so the backup fails against its dead source and records its outcome before the database closes.
- [The fresh supervisor is not the one the live state route reads] → No scenario calls that route, and `Serve` keeps one instance for both.
- [Shard durations are guessed until measured] → The weights come from a local run of each new function, and a wrong weight only unbalances the shards.

## Migration Plan

Backup execution stays unchanged. The physical backup list exposes real WAL gaps even when receive times differ from LSN order. No migration is needed. Reverting the change restores the old suite and the faulty list grouping.
