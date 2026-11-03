## Why

Ten small backend defects give users a wrong status or message, lose a Slack notification, leave a security event out of the audit log, or restart a slow-starting pod. One of them can leave an instance unable to roll back: a broken image migrates the schema first and only then refuses to start, and the previous image then refuses the newer schema. Each fix is small, so they travel together in one change.

## What Changes

- A server with a broken PostgreSQL client refuses to start before it migrates the schema.
- The Helm chart gets a startup probe, so a slow start does not end in a liveness restart loop.
- A logical backup stopped during its setup ends without waiting for a silent SSH bastion and records the real reason: cancelled for a user's request, the existing shutdown or time-limit message otherwise.
- A physical backup that finished just as its storage changed hands its files to deletion, instead of rolling that request back.
- A failed archive download reports that the restore tool has not started, instead of a partially restored target.
- One failed removal no longer stops retention from removing the other expired backups of that database.
- A MySQL, MariaDB or MongoDB restore whose versions cannot be compared reports an internal error, instead of calling the target too old.
- A Slack delivery waits out a rate limit up to 3 minutes. Every other delivery keeps its 30-second limit.
- A sent password reset code writes an audit event.
- `-h` and `--help` exit with code 0.

## Out of scope

- Failure notifications on application restart stay as they are. `application-shutdown` requires them for physical backups, and logical backups send them too. Silencing them could hide a lost full backup.
- The order of `.env` and the published connection string stays as it is. It matters only for an operator who mounts a `.env` into the image, and preferring the published value would silently move that operator from their database to an empty embedded one.
- `Content-Length` on backup downloads needs nothing: it is already sent and tested for encrypted and unencrypted backups.
- Orphan slot cleanup stays synchronous before the port opens. Running it in the background lets it drop the slot of a backup that has just started.
- The PostgreSQL version comparison, which allows a restore when a version cannot be parsed, stays as it is.
- Automatic retry of backups interrupted by a restart, retention of physical backups, and any other change to the HTTP contract.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `audit-log`: a sent password reset code writes an event without an author.
- `backup-storage-assignment`: a finished physical backup that lost its storage is not published, and its files go to deletion.
- `client-tool-bundles`: a broken PostgreSQL client stops the server before the schema changes.
- `host-console-commands`: asking for help exits with code 0.
- `logical-backup-outcome`: the outcome of a stopped backup does not depend on the stage it reached.
- `logical-backups`: one failed removal does not stop retention for the database.
- `notifiers`: a rate-limited Slack delivery gets a longer limit than other deliveries.
- `restores`: the failure message of an archive download, and a version comparison failure reported apart from a too-old target.
- `single-instance-runtime`: the Helm chart lets a slow start finish before liveness applies.

## Impact

- This change answers to the root `AGENTS.md` and `backend/AGENTS.md`.
- Backend: `cmd/databasus`, `internal/app`, `lib/sshtunnel`, the logical and physical parts of the backups module, the restore pipeline, the Slack provider and notifier delivery, and password reset in the users module.
- Documentation: the audit rule in `docs/architecture.md`, `docs/glossary.md` and `backend/AGENTS.md` gains the case of an effect outside the database.
- Deployment: `deploy/helm` gains the `startupProbe` values and its documentation. Existing values keep working.
- The HTTP contract, the database schema and the frontend do not change.
