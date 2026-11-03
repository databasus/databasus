## 1. Foreign key indexes

- [x] 1.1 Create the stub with `goose -dir backend/migrations create add_foreign_key_indexes sql`, add `backend/migrations/foreign_key_indexes_migration_test.go` with `TestForeignKeyIndexesMigration` requiring an index whose first column is each of the four columns after the migration and no such index after rolling this migration back with `provider.ApplyVersion(ctx, version, false)`, and verify with `go test ./migrations -run TestForeignKeyIndexesMigration` from `backend/` that it fails because those indexes are missing
- [x] 1.2 Fill the migration with the four `CREATE INDEX` statements and a Down section of four plain `DROP INDEX` statements, and verify that `go test ./migrations -run TestForeignKeyIndexesMigration` from `backend/` passes
- [x] 1.3 Add `idx_backups_physical_storage_id` to the expected index list in `TestMergePhysicalBackupsMigration`, and verify with `go test ./migrations -run TestMergePhysicalBackupsMigration` from `backend/`

## 2. Scheduler query

- [x] 2.1 Add a subtest to `TestLogicalBackupScheduler` where two databases have long histories and only the one whose newest backups stay under its tries gets a retry, and verify with `source ../scripts/fedora.sh && apply_libedit_shim && TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 -run 'TestLogicalBackupScheduler' ./internal/modules/backups/...` from `backend/` that it passes on the current query
- [x] 2.2 Rewrite `listNewestPerDatabase` to use `LATERAL` with `unnest` and `pq.Array` as `design.md` describes, and verify that `source ../scripts/fedora.sh && apply_libedit_shim && TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 -run 'TestLogicalBackupScheduler' ./internal/modules/backups/...` from `backend/` still passes

## 3. Format and lint

- [x] 3.1 Run `mise run backend:format` and verify that `mise run backend:lint` passes

## Change verification

```sh
mise run test
```
