## Why

The restore dialog opens its target form as the backed-up database itself: the form keeps that database's ID, port, database name, SSH bastion and client certificate, and clears only the host, user and password. Its connection test therefore fills omitted secrets from the source database, so the test can pass with the source's password or SSH key while the restore itself, which receives only what the form holds, fails. The same test needs the member role, so a workspace viewer, who may restore logical backups, cannot get past it in the interface.

## What Changes

- Open the restore target form as a new database of the backup's engine: no saved database behind it, the engine's default settings, no setting of the backed-up database and no secret shown as stored. The target requires the same fields as adding a database of that engine, including the password.
- Start the restore directly from the target form, without a separate connection test. The restore start already connects to the target and checks its version before it answers; a refused start keeps showing the server's reason inline while the form keeps every entered value.
- Prevent a second submission while a restore start is pending, because the start now carries the connection check and can take seconds.
- Give the MySQL, MariaDB and MongoDB target forms the restore behavior the PostgreSQL form already receives, so all four engines follow the same flow.
- No API, backend or access-rule change. A viewer can restore from the interface because the flow no longer calls the member-only settings test.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `restores`: the interface enters the restore target from scratch, never uses the backed-up database's stored settings or secrets for it, and checks the target by starting the restore while keeping the entered values after a refusal.

## Impact

First milestone-5 change. Root `AGENTS.md` and `frontend/AGENTS.md` apply, with the testing and frontend-design skills. No copy changes, so the six interface dictionaries stay as they are.

Affected areas are the restore dialog under `frontend/src/features/restores/`, the logical database forms under `frontend/src/features/databases/ui/edit/` that the dialog reuses, and their screen tests. The edit, create and copy flows of saved databases keep their current connection-test behavior. No database migration, endpoint, dependency or generated code changes.

### Out of Scope

- Changing `testDatabaseSettings`, its member-only rule or how it fills secrets for a saved database.
- Moving the restore form to its final owner or removing the restore entries of the FSD migration baseline; the cross-feature import of the database form stays as it is.
- Hiding backup-only fields, such as table exclusions, from the MySQL, MariaDB and MongoDB target forms.
- New copy, a confirmation dialog before the restore starts, or remembering targets between dialog openings beyond what the dialog already keeps.
- Physical restore, which has no target form.
