## Why

A PostgreSQL logical connection can pass Databasus's read-privilege checks even though row-level security (RLS) prevents a complete dump. Detect that problem during connection validation and prove that later RLS changes fail a logical backup safely, while physical backups preserve all rows without RLS bypass privileges.

## What Changes

- Check whether RLS applies to the connecting role on tables whose data the logical backup includes. Return a diagnostic naming the role and affected tables before accepting the connection.
- Allow roles that already bypass RLS through PostgreSQL's effective privileges or table ownership, and respect schema, whole-table and table-data exclusions.
- **BREAKING (validation behavior):** replace SELECT-only acceptance with validation of complete row access during logical database creation, updates and saved or unsaved connection tests. Configurations that cannot produce their requested dump will be refused; there is no compatibility mode that retains the incomplete validation. Existing saved configurations are not migrated or deleted.
- Add real PostgreSQL tests for connection validation and for RLS enabled after a successful connection and backup. Verify a subsequent dump fails, records `FAILED`, and leaves no partial backup after storage cleanup.
- Add physical backup and restore coverage with RLS and a replication role without `BYPASSRLS`, proving full and incremental backups retain all rows.

### Out of scope

- Automatically granting `BYPASSRLS`, changing read-only role provisioning, or adding an RLS checkbox or setup wizard.
- Adding a new preflight before every logical dump, or replacing the dump tool's runtime protection.
- Changing physical backup privileges, backup formats, role policies, or restore access rules.
- Permanent development databases, new PostgreSQL server versions, or new client tool bundles.

## Capabilities

### New Capabilities

- `postgresql-rls-backup-access`: validate effective RLS access for logical backups, reject incomplete logical backup attempts after permission drift, and preserve complete physical backups independently of row visibility.

### Modified Capabilities

None. The new capability specifies RLS behavior alongside the existing database, logical backup outcome, backup scope and physical replication requirements.

## Impact

- Governed by root `AGENTS.md` and `backend/AGENTS.md`.
- Production change in `backend/internal/engines/postgresql/logical`; existing database providers and connection validation callers reuse it.
- Tests in the logical engine, database module HTTP tests, PostgreSQL logical backup end-to-end suite, and physical backup restore suites and their shared fixtures.
- Update relevant descriptions in `backend/api/openapi.yaml` and regenerate API artifacts. Keep existing request and response shapes, HTTP statuses, and error codes; RLS diagnostics use the existing connection failure fields.
- Define row-level security and its RLS abbreviation in `docs/glossary.md` alongside the new API descriptions.
- No database migration, new dependency, frontend change, or architecture change.
