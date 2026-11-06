## Context

See `proposal.md` for the motivation and `specs/postgresql-rls-backup-access/spec.md` for observable behavior. This change includes security decisions and real backup/restore coverage, so it needs a design.

Current behavior relevant to the change:

- Logical `Source.TestConnection` checks schema/SELECT access and user mappings, but not effective RLS access (`backend/internal/engines/postgresql/logical/source.go:44`, `backend/internal/engines/postgresql/logical/dump_privileges.go:36`).
- Database creation and updates call the provider's connection test; saved and unsaved tests reach the same engine method (`backend/internal/modules/databases/internal/userdatabase/service.go:455`, `backend/internal/modules/databases/internal/connection/service.go:93`). The logical provider already supplies all three scope filters (`backend/internal/modules/databases/internal/providers/postgresqllogical/connected_database.go:35`).
- RLS discovery currently serves read-only role suggestion and provisioning, where any discovered RLS table suppresses or refuses generated roles. It does not answer whether the supplied role can dump the table (`backend/internal/engines/postgresql/logical/read_only_user.go:57`, `backend/internal/engines/postgresql/logical/read_only_user.go:85`).
- Scope matching already accounts for included schemas, extension membership, the role's default search path, whole-table exclusions and data exclusions (`backend/internal/engines/postgresql/logical/dump_scope.go:16`, `backend/internal/engines/postgresql/logical/size.go:44`).
- The pipeline propagates dump errors instead of returning a successful receipt. The logical service records failure and requests file deletion without publishing the backup (`backend/internal/modules/backups/internal/pipeline/backup.go:156`, `backend/internal/modules/backups/internal/pipeline/backup.go:220`, `backend/internal/modules/backups/internal/logical/backup_run_service.go:67`, `backend/internal/modules/backups/internal/logical/backup_run_service.go:247`).
- The engine test matrix already runs PostgreSQL 12 through 18 with an isolated database per scenario (`backend/internal/engines/postgresql/logical/source_test.go:12`). The logical end-to-end matrix has an analogous privilege-drift scenario for sequence access (`backend/internal/modules/backups/e2e/postgresql/engine_postgresql_test.go:846`).
- Physical restore fixtures already use a replication-only role and assert it is neither superuser nor BYPASSRLS (`backend/internal/app/apptest/physicaltest/team.go:69`). Full and incremental restore scenarios compare rows from successive write stages (`backend/internal/app/apptest/physicaltest/restore_chain.go:11`, `backend/internal/app/apptest/physicaltest/restore_chain.go:37`).

The approach follows root `AGENTS.md`, `backend/AGENTS.md`, and the testing and API design skills. Privilege inspection belongs to engines under `docs/architecture.md#engines`; HTTP adaptation stays in the database module. Cleanup uses the existing file store under `docs/architecture.md#stored-files`. No architectural boundary changes.

## Goals / Non-Goals

**Goals:** make one engine-level RLS check cover every connection-validation caller, preserve the existing diagnostic response formats, and establish real database evidence for both failure and complete restoration.

**Non-Goals:** see the proposal's out-of-scope list. In particular, keep role provisioning separate from validation and keep runtime dump protection independent of an earlier check.

## Decisions

### 1. Ask PostgreSQL whether RLS applies to the current role

Add a dedicated RLS privilege unit under the logical engine, such as `rls_privileges.go`, invoked by `Source.TestConnection` after the existing read-access check. Use `pg_catalog.row_security_active(c.oid)` on tables whose data is dumped, on the connection authenticated as the supplied role. It accounts for superuser, BYPASSRLS, effective ownership and FORCE ROW LEVEL SECURITY without duplicating PostgreSQL's authorization rules.

Return a contextual error using the package's existing RLS sentinel, the server's `current_user`, and the affected qualified names. Use the existing bounded name formatter for large lists. Explain that RLS blocks a complete dump and recommend credentials authorized for full access. Do not imply that every table owner bypasses forced RLS, that SELECT alone suffices, or that Databasus will confer privileges. No password appears in the message.

Do not count rows, execute application queries or alter source permissions during validation. A policy with `USING (true)` still leaves the role subject to RLS; default `pg_dump` rejects that situation too.

Alternatives rejected: requiring the `rolbypassrls` attribute rejects valid superusers and table owners; parsing policies cannot reliably establish the dump's authorization semantics; running an extra dump during connection testing adds unnecessary work and cannot replace runtime protection.

PostgreSQL references: [effective RLS inquiry](https://www.postgresql.org/docs/18/functions-info.html#FUNCTIONS-INFO-ACCESS-TABLE), [default dump handling of RLS](https://www.postgresql.org/docs/18/app-pgdump.html#APP-PGDUMP-OPTIONS), and [owner and FORCE behavior](https://www.postgresql.org/docs/18/ddl-rowsecurity.html).

### 2. Check the actual table-data scope without weakening ordinary privileges

Reuse `dumpedRelationScope`, `queryDumpedRelations`, normalized exclusions and `isRelationExcluded`. Inspect ordinary row-bearing tables, including leaf partitions; do not treat partitioned parents, sequences or foreign tables without dumped data as row sources. Preserve the existing extension-membership behavior, including extension configuration tables whose rows are dumped.

Skip relations matched by either whole-table or table-data exclusion for the RLS check only. Continue to require schema USAGE and SELECT for included structures, including data-excluded tables. Reuse the same default-search-path visibility and pattern semantics as the dump. Excluding a partitioned parent does not implicitly exclude its children.

Do not replace the existing role-provisioning RLS helper with this effective-access query: provisioning evaluates whether a newly created restricted role would work, not whether the administrator currently connected can bypass RLS. Correct any misleading RLS comments in touched code while keeping that provisioning policy.

Alternatives rejected: reusing `refuseRowLevelSecurityTables` denies valid roles and ignores data exclusions; scanning every database table produces false failures outside the requested scope; building another pattern matcher risks divergence from the backup arguments.

### 3. Preserve the HTTP shapes and use existing refusal paths

Define row-level security and its abbreviation RLS in `docs/glossary.md` as PostgreSQL policies that restrict which table rows a role can access. Use that common term when documenting logical RLS validation in the existing creation, update and connection-test descriptions in `backend/api/openapi.yaml`, then regenerate API artifacts. Saved and unsaved tests continue to answer HTTP 200 with `isConnected: false` and an `error` diagnostic. Do not add a reason enum solely for RLS. Creation and updates continue to answer HTTP 400 with `database_connection_failed`, and failed updates keep the saved settings. Connection tests themselves do not update database state.

This is stricter validation, as marked BREAKING in the proposal, not a response-shape migration. Generated schema descriptions may change; frontend behavior and dictionaries do not need edits. Follow the contract-first rule in `docs/architecture.md#http-api`.

Alternatives rejected: a new response enum would require frontend copy and handling outside this task; rejecting at the HTTP handler duplicates engine rules and misses non-HTTP callers; treating RLS as an internal error hides an actionable source-access refusal.

### 4. Keep the real dump as protection against permission drift

Add a logical end-to-end scenario to the existing PostgreSQL 12–18 matrix. Create an isolated source database with ordinary data and an RLS candidate table containing at least two tenants' rows. Grant a non-owner login schema access and SELECT on the required tables and sequences, without BYPASSRLS. Validate the connection and complete a baseline backup before enabling RLS.

Enable a policy that permits only one tenant and verify through the restricted connection that fewer rows are visible while SELECT remains granted. Start another backup through the API without updating the saved connection or invoking a new validation preflight. Require FAILED with the real dump's RLS diagnostic and table name. The real tool produces a partial stream; no fabricated artifact or tool stub stands in for it.

Wait for the backup run to end, execute storage cleanup through the existing job helper, and verify the failed attempt's file and metadata are absent. Verify the baseline backup remains completed and its files remain present. Do not publish the failed artifact or weaken cleanup assertions to accept leftovers.

Alternatives rejected: a stubbed exit code proves only error plumbing; a test of the engine alone cannot prove the published status or file cleanup; adding another preflight to the dump path would prevent the test from establishing the existing runtime protection.

### 5. Prove physical preservation through restoration

Use the existing physical full and incremental restore infrastructure on PostgreSQL 17 and 18, with isolated scenario fixtures. Add dedicated RLS scenarios using the existing replication-only user rather than a superuser. Explicitly confirm no SELECT on the marker table, no BYPASSRLS and no superuser privileges.

For the full-only scenario, enable RLS with a restrictive policy before the connection test and full backup, then restore and compare every inserted marker row as the restore administrator. Confirm the RLS flag and policy survive restoration. For the incremental scenario, take the full backup first, enable RLS and insert additional rows, then take an incremental and restore the chain. Compare rows from both stages and the restored RLS configuration.

Keep physical production code unchanged: physical replication copies cluster files rather than selecting rows. Missing replication privileges remain covered by existing connection and backup failure tests. [PostgreSQL's physical backup description](https://www.postgresql.org/docs/18/app-pgbasebackup.html) defines this distinction.

Alternatives rejected: expecting physical backups to fail without BYPASSRLS confuses replication privileges with row access; checking COMPLETED alone cannot establish that protected rows survived; a new permanent development server increases setup without improving isolated test coverage.

### 6. Put each check at its observable boundary

- Engine scenarios exercise `Source.TestConnection` against every supported logical PostgreSQL version, including bypass/superuser, owner/FORCE, an all-rows policy, scope exclusions, unqualified patterns and parent/leaf partition cases. Include effective ownership through role membership so an attribute-only implementation cannot pass.
- Database HTTP scenarios use real PostgreSQL to cover create, update, saved test and unsaved test outcomes. Compare persisted settings after a refused update and all database fields after a saved test. Keep credentials out of diagnostics.
- Logical end-to-end scenarios prove real dump failure and file cleanup; physical end-to-end scenarios prove restored rows. All source database state is prepared with SQL because the application API does not create source schemas or policies. Application state is prepared through its API.

Use `TestPostgresqlRLSConnectionValidation` for the database HTTP scenarios and `TestPhysicalBackupWithRLS` in the existing PostgreSQL 17 and 18 end-to-end packages. Register the logical drift scenario under the existing `TestBackupAndRestorePostgresql` matrix so it shares that matrix's running version fixture. Reuse version lists and fixtures rather than copying version inventories.

Alternatives rejected: HTTP-only coverage repeats a large application setup for every engine edge case; engine-only coverage misses API behavior; reusing one mutable database across scenarios makes role and RLS changes interfere with unrelated tests.

## Risks / Trade-offs

- **Permission changes after validation:** the check is only a snapshot. The real dump must still fail when access changes; the drift scenario verifies that independent protection.
- **False refusals from scope differences:** reuse the existing relation scope and pattern matcher, then compare exclusions and partition behavior with real dumps. Data exclusion lifts only the RLS data requirement, not SELECT.
- **Partially written files:** cleanup is asynchronous. Assert absence after the run finishes and the deletion pass completes; retain the baseline backup to detect overbroad deletion.
- **Longer test runs:** reuse existing version fixtures and small marker datasets. Run the affected end-to-end packages explicitly in addition to `mise run test`, which excludes them.
- **Stricter validation also reaches health probing:** the logical provider's liveness probe delegates to its connection test. A source whose saved role can no longer dump all included rows may become unhealthy; the check does not rewrite its stored credentials or back up only visible rows.

## Migration Plan

No schema or data migration is required. Deploy the engine check and its contract description together. Existing saved databases stay intact, while subsequent validation applies the stronger check. Operators of newly refused configurations must supply an appropriately authorized role or exclude the affected data; Databasus does not automatically grant BYPASSRLS.

Reverting the implementation restores the earlier validation behavior without a data rollback. Existing complete backups and physical backup formats are unaffected.
