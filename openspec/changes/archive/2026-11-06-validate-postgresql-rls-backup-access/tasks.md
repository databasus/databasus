## 1. Logical connection validation

- [x] 1.1 Define row-level security and its RLS abbreviation in `docs/glossary.md` as PostgreSQL policies that restrict which table rows a role can access, and use that term consistently when documenting effective RLS validation in the existing create, update, saved-test and unsaved-test operation descriptions in `backend/api/openapi.yaml`. Keep response shapes, statuses and codes unchanged; run `mise run generate-api`. Verify contract and generated consistency with `mise run backend:lint` and check that the descriptions use the glossary's term and abbreviation.
- [x] 1.2 Add a `TestConnection RLS privileges` group to the existing `TestSource` matrix, with an isolated source database and a non-owner SELECT role restricted by RLS. Assert an error identifying the role, affected table and RLS. Before changing production code, run `mise exec -- go -C backend test -count=1 -timeout=15m -run '^TestSource$/postgres:16/TestConnection_RLS' ./internal/engines/postgresql/logical` and confirm that the new assertion fails because validation currently accepts the role.
- [x] 1.3 Implement effective RLS inspection in a dedicated logical-engine unit and call it from `Source.TestConnection` after the ordinary read-privilege check. Reuse PostgreSQL's effective-access inquiry, the existing RLS sentinel, bounded name formatting and current-user lookup; do not grant privileges or change provisioning policy. Verify the regression now passes with `mise exec -- go -C backend test -count=1 -timeout=15m -run '^TestSource$/postgres:16/TestConnection_RLS' ./internal/engines/postgresql/logical`.
- [x] 1.4 Extend the engine group across the existing PostgreSQL 12–18 matrix with BYPASSRLS, superuser, owner, inherited ownership, forced RLS, an all-rows policy, included schemas, whole-table and data exclusions, unqualified search-path matching, and partition parent/leaf cases. Preserve missing-SELECT rejection for data-excluded tables, and ensure RLS diagnostics contain no credentials. Verify all engine scenarios with `mise exec -- go -C backend test -count=1 -timeout=15m -run '^TestSource$' ./internal/engines/postgresql/logical`.

## 2. Database HTTP behavior

- [x] 2.1 Add `TestPostgresqlRLSConnectionValidation` with real PostgreSQL fixtures prepared through source SQL and application state prepared through the API. Cover refused create and update, failed saved and unsaved tests, an authorized role and a data exclusion that removes only the RLS requirement. Assert HTTP 200 plus `isConnected: false` and an RLS diagnostic for tests, and HTTP 400 `database_connection_failed` for refused saves; compare the stored configuration after refusal and all database fields before and after a saved test. Verify with `mise exec -- go -C backend test -count=1 -timeout=15m -run '^TestPostgresqlRLSConnectionValidation$' ./internal/modules/databases`.

## 3. Logical backup after RLS changes

- [x] 3.1 Add a scenario under the existing `TestBackupAndRestorePostgresql` version matrix using an isolated database with at least two tenants' rows and an ordinary readable table. Validate and complete a baseline backup with a non-owner SELECT role, then enable restrictive RLS without updating the saved connection. Confirm fewer rows are visible to that role and start another real backup without a new preflight. Assert FAILED with the dump's RLS diagnostic and table name. Wait for the backup run, execute the storage deletion pass, assert both failed-attempt files are absent, and assert the earlier completed backup and its files remain available. Verify the real-tool scenarios with `mise exec -- go -C backend test -count=1 -timeout=15m -run '^TestBackupAndRestorePostgresql$' ./internal/modules/backups/e2e/postgresql`.

## 4. Physical backup and restoration with RLS

- [x] 4.1 Add `TestPhysicalBackupWithRLS` scenarios to the existing PostgreSQL 17 and 18 end-to-end packages, sharing RLS restore fixtures under `physicaltest`. Use the existing replication-only user and explicitly verify no SELECT on the marker table, no superuser and no BYPASSRLS. With RLS present, require a successful physical connection test and full backup, restore it, and compare every marker row and the RLS configuration. In the incremental scenario, introduce RLS after a full backup, add more rows, take an incremental and verify the restored chain contains both stages and the RLS configuration. Keep physical production privileges unchanged. Verify with `mise exec -- go -C backend test -count=1 -timeout=15m -run '^TestPhysicalBackupWithRLS$' ./internal/modules/backups/e2e/postgresqlphysical/pg17 ./internal/modules/backups/e2e/postgresqlphysical/pg18`.

## 5. Integration checks

- [x] 5.1 Format touched backend files with `mise run backend:format` and verify them with `mise run backend:lint`. Search affected documentation and comments for claims that SELECT alone guarantees dump access or that BYPASSRLS is required for every RLS table; align descriptions within this change while preserving role-provisioning policy. Run the change-verification command below and retain results from the targeted end-to-end commands above before mandatory implementation review.

## Verification results

- Before the production fix, the PostgreSQL 16 RLS regression failed because `TestConnection` returned no error for a SELECT role subject to RLS. The same command passed after the fix.
- The complete `TestSource` matrix passed on PostgreSQL 12–18, including the new RLS group.
- `TestPostgresqlRLSConnectionValidation` passed, including preservation of nonempty health and backup outcome fields.
- `TestBackupAndRestorePostgresql` passed across PostgreSQL 12–18, including real dump failure after RLS drift, failed-file cleanup and preservation of the baseline backup.
- `TestPhysicalBackupWithRLS` passed in both PostgreSQL 17 and 18 packages for full-only and incremental restoration.
- `mise run test` passed: both backend groups and all 187 frontend test files (2560 tests).
- `mise run backend:format`, `mise run backend:lint`, `mise run frontend:lint`, `pnpm --dir frontend build` and `openspec validate validate-postgresql-rls-backup-access --strict` passed. API generation is current. The generated frontend schema is excluded from the frontend formatter.

## Change verification

```sh
mise run test
```
