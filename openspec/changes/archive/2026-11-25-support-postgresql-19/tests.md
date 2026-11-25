# Test plan

## Impact

| Changed | Tests it can break | Why |
|---|---|---|
| Version number rules and the bundle list in `engines/postgresql/shared` | `./internal/engines/postgresql/shared/...`, `TestShippedBundles` in `./internal/engines/clienttools` | The bundle list decides which client directories the startup check requires and which programs each must hold. The downgrade comparison gains a PostgreSQL 19 case. |
| Logical version parsing and the zstd rule in `engines/postgresql/logical` | The `Version` parser tests, `TestDump`, `TestSource` on 12 and 19, the logical end-to-end matrix on 15, 16 and 19 | Parsing decides which servers are accepted. zstd starts at 16, so 15 and 16 sit on either side of the boundary, and 19 is the new version. |
| Logical acceptance and refusal through the API | `TestTestDatabase`, `TestCreateLogicalDatabaseVersions` | A PostgreSQL 19 logical database is tested, saved and read back with version 19, and a PostgreSQL 11 server is refused with the version found and the supported range. |
| Physical version parsing and its two refusals in `engines/postgresql/physical` | Physical engine tests on 17 and 19, the PostgreSQL 16 cases of `TestInspectCluster` and `TestReadLiveTimeline`, `TestTestPhysicalDatabase` and `TestCreatePhysicalDatabaseVersions` in `./internal/modules/databases` | 17 is the lower bound of acceptance and 19 is new. The connection test returns the refusal messages to the user. The split error changes what `TestInspectCluster` and `TestReadLiveTimeline` expect from a PostgreSQL 16 server. |
| Physical API loops gaining 19 in `physicalapitests/backuping` and `walstreaming` | The five loop tests on 17 and 19 | The same physical code runs against a 19 server for slots, artifacts, WAL summaries, archiving and standby sources. |
| WAL summarization check gaining 19 | `TestPhysicalDatabaseWalSummarization` | It reads `summarize_wal` from a 19 server through the physical connection setup. |
| New `e2e/postgresqlphysical/pg19` package | The whole `pg19` package | Full, incremental, PITR, promotion, lost-tail, schema, sequence, RLS, PgBouncer and reconnect recovery on 19 back the extended `physical-backups/operations` and failover requirements. |
| Stale recovery target planted by the Patroni independent recovery fixture | `TestPatroniIndependentRecoveryPostgres17` | The fixture's value changes for every major, and 17 is the oldest that runs it. |
| Patroni `Postgres19` variants and fixture matrix | `TestPatroni*Postgres19`, `TestPatroniFixtureLifecycle` on 19 | They back the Patroni requirements extended to 19 and show whether Patroni 4.1.5 runs a 19 cluster. |
| `PostgresqlVersion` enum and generated code | `TestGetPostgresqlLogicalVersions`, and every API test that returns a PostgreSQL database | API tests validate responses against the contract, so a 19 database requires the new enum value. |
| Agent runner reading majors from its pinned versions, `GetPinnedPostgresMajors` and the start wiring | `./internal/agent/runner/...`, `./internal/agent/container/...`, `./internal/agent/e2e/...` | The runner decides whether to restore or fail an assignment, the getter exposes the pinned majors, and the agent end-to-end tests start the wired agent and verify backups of every pinned major. |
| PostgreSQL 19 case in the recovery command tests | `restoreCommands.test.ts` | It pins the Docker data layout the extended script recovery requirement claims for 19. |
| Verification image recipe, its release matrices and the pinned 19.0 | `TestVerifyBackup` on 19, `TestHasOnlyIgnorableErrorsForVerification`, `./internal/agent/container/...`, `suite scripts` | The PostgreSQL 19 case builds and restores into the 19 image, and a missing 19 extension counts as ignorable. The older majors restore into published catalog images, so only the task 5.1 builds for 12 and 18 check the reworked extension conditions. The scripts read the pinned versions. |
| CI database client step, the `doctor` bundle check and their tests | `suite scripts` | `backend_test_preparation_test.py` checks the commands of the CI step, and `development_environment_check_test.py` checks how `ldd -r` output becomes a result. |
| Shard weights of the Patroni 19 tests and the cluster shard check | `suite scripts` | `test_group_test.py` requires a weight for every Patroni scenario test and a backup scenario in every shard. |
| `tools/dependencies/clients.json` and the new bundle files | `suite scripts` | The dependency verification scripts read the client records and hash the committed files. |

## Not selected

- PostgreSQL 13, 14, 17 and 18 in the logical source tests, and 12, 13, 14, 17 and 18 in the logical end-to-end matrix: the parsing path is the same membership check for every bundled major, and the selected versions cover the lower bound, both sides of the zstd boundary and the new version.
- PostgreSQL 18 in the physical engine tests, and the `pg17` and `pg18` packages: 18 takes the same "at least 17 and bundled" path as 17 and 19, and no physical code beyond version parsing changes. The selected API loop tests still run 18, because their loops cannot be filtered by version.
- The other Patroni 17 and 18 variants, and the Patroni 18 independent recovery, and the CloudNativePG suite: their code path changes only in version parsing, which the physical engine tests cover, and CloudNativePG gains no version.
- The other frontend tests: the frontend only receives a wider generated enum type, which `pnpm typecheck` checks.
- `README.md`, its translations, `assets/tools/README.md` and the website copy: text only, and `mise run lint` covers the website build rules.
- `scripts/image_pins.py` and the catalog record of the published 19 verification image: this change does not modify them, and the follow-up commit after the release adds them.
- The internal database dump and restore in `engines/postgresql/internaldb`: they use the fixed PostgreSQL 17 client, which the rules leave unchanged.

## Selection

```tests
backend ./internal/engines/postgresql/shared/...
backend ./internal/engines/clienttools -run TestShippedBundles
backend ./internal/engines/postgresql/logical -run Version
backend ./internal/engines/postgresql/logical -run TestDump
backend ./internal/engines/postgresql/logical -run TestSource/postgres:(12|19)$
backend ./internal/modules/backups/e2e/postgresql -run TestBackupAndRestorePostgresql/PostgreSQL_(15|16|19)$
backend ./internal/engines/postgresql/physical -run Version
backend ./internal/engines/postgresql/physical -run /postgres:(17|19)$
backend ./internal/engines/postgresql/physical -run (TestInspectCluster|TestReadLiveTimeline)/(returns_the_too_old_error|reads_identity_without_rejecting)
backend ./internal/modules/databases -run TestTestDatabase|TestTestPhysicalDatabase|TestCreatePhysicalDatabaseVersions|TestCreateLogicalDatabaseVersions|TestPhysicalDatabaseWalSummarization|TestGetPostgresqlLogicalVersions
backend ./internal/engines/postgresql/logical/pgrestore -run TestHasOnlyIgnorableErrorsForVerification
backend ./internal/modules/backups/physicalapitests/backuping -run TestPhysicalBackupSlots$|TestPhysicalBackupArtifacts|TestIncrementalBackupWithPendingWalSummaries
backend ./internal/modules/backups/physicalapitests/walstreaming -run TestArchivePhysicalWal$|TestPhysicalWalSourceInRecovery
backend ./internal/modules/backups/e2e/postgresqlphysical/pg19
backend ./internal/modules/backups/e2e/postgresqlphysical/patroni -run Postgres19
backend ./internal/modules/backups/e2e/postgresqlphysical/patroni -run TestPatroniIndependentRecoveryPostgres17$
backend ./internal/modules/backups/e2e/postgresqlphysical/patroni -run TestPatroniFixtureLifecycle/.*postgres:19
backend ./internal/agent/runner/...
backend ./internal/agent/container/...
backend ./internal/agent/e2e/...
suite scripts
frontend src/pages/databases/physical-backups/lib/restoreCommands.test.ts
```
