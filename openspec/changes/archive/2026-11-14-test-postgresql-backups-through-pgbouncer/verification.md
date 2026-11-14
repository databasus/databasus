## Verification

Executed locally on amd64 on November 14, 2026. PostgreSQL client runs used the Fedora libedit shim and `TESTCONTAINERS_RYUK_DISABLED=true` as documented in `docs/testing.md`.

| Check | Result |
| --- | --- |
| `mise run doctor` | Passed; Codex hooks trusted and enabled |
| `go -C backend test -count=1 ./internal/lib/testimages` | Passed |
| `go -C backend test -count=1 -timeout=20m -run '^(TestBackupAndRestorePostgresqlThroughPgBouncer\|TestRestorePhysicalBackupThroughPgBouncer)$' ./internal/modules/backups/e2e/postgresql ./internal/modules/backups/e2e/postgresqlphysical/pg18` | Passed; logical 7.305s, physical 26.507s; neither skipped |
| `go -C backend test -count=1 -timeout=20m -run '^TestBackupAndRestorePostgresql$/^PostgreSQL_18$/restores the rows with' ./internal/modules/backups/e2e/postgresql` | Passed; 2.028s |
| `go -C backend test -count=1 -timeout=20m -run '^(TestRestorePhysicalBackupChain\|TestRestorePhysicalBackupToTargetTime)$/^recovers rows' ./internal/modules/backups/e2e/postgresqlphysical/pg18` | Passed; 24.635s |
| `mise run backend:format` | Passed; no unrelated files changed |
| `mise run backend:lint` | Passed; 0 issues |
| `openspec validate test-postgresql-backups-through-pgbouncer --strict` | Passed |
| `git diff --check` | Passed |

The first logical run exposed lib/pq's `extra_float_digits` startup parameter being rejected by PgBouncer. The fixture now explicitly allows that parameter using `ignore_startup_parameters`; replication startup parameters remain intact. The new scenarios passed after that fixture correction.

Both tests use isolated PostgreSQL 18 sources without published database ports and a real PgBouncer 1.26.1 endpoint with session pooling. The physical test checks the API chain links, the downloaded full/incremental manifests and backup archives, multiple archived WAL files, and the exact marker set after booting the recovered cluster. The logical test checks four restored tables, columns, primary keys, rows and backup size with four restore jobs.

The registry index and both platform digests were resolved for `edoburu/pgbouncer:v1.26.1-p0`. Runtime verification was on amd64; no arm64 runtime run was performed. No production code, existing test scenarios or shared documentation changed. Automatic implementation review runs when the agent finishes the turn.
